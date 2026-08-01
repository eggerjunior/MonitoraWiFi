package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"egger/api/internal/auth"
	"egger/api/internal/store"
)

func TestAgentEnrollment_FullFlow(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)

	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	// 1. Admin gera token de enrolamento para o site.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/agent-enrollment-tokens", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("esperava 201 ao criar token, recebeu %d: %s", rec.Code, rec.Body.String())
	}
	var tokenResp struct {
		EnrollmentToken string `json:"enrollment_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &tokenResp); err != nil {
		t.Fatalf("resposta inválida: %v", err)
	}
	if tokenResp.EnrollmentToken == "" {
		t.Fatal("esperava um enrollment_token não vazio")
	}

	// 2. Agente troca o token por uma credencial de longa duração.
	enrollBody, _ := json.Marshal(map[string]string{
		"enrollment_token": tokenResp.EnrollmentToken,
		"hostname":         "gateway-residencia",
		"platform":         "linux_amd64",
		"version":          "0.1.0",
	})
	enrollReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/enroll", bytes.NewReader(enrollBody))
	enrollRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(enrollRec, enrollReq)
	if enrollRec.Code != http.StatusCreated {
		t.Fatalf("esperava 201 ao enrolar, recebeu %d: %s", enrollRec.Code, enrollRec.Body.String())
	}
	var enrollResp struct {
		AgentID     string `json:"agent_id"`
		AgentSecret string `json:"agent_secret"`
	}
	if err := json.Unmarshal(enrollRec.Body.Bytes(), &enrollResp); err != nil {
		t.Fatalf("resposta de enroll inválida: %v", err)
	}
	if enrollResp.AgentID == "" || enrollResp.AgentSecret == "" {
		t.Fatal("esperava agent_id e agent_secret não vazios")
	}

	// 3. Reusar o mesmo token de enrolamento deve falhar (uso único).
	reuseReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/enroll", bytes.NewReader(enrollBody))
	reuseRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(reuseRec, reuseReq)
	if reuseRec.Code != http.StatusUnauthorized {
		t.Fatalf("esperava 401 ao reusar token já usado, recebeu %d", reuseRec.Code)
	}

	// 4. Heartbeat autenticado com a credencial do agente.
	hbBody, _ := json.Marshal(map[string]any{"status": "ok", "queued_items": 0})
	hbReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+enrollResp.AgentID+"/heartbeat", bytes.NewReader(hbBody))
	hbReq.Header.Set("Authorization", "Bearer "+enrollResp.AgentSecret)
	hbRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(hbRec, hbReq)
	if hbRec.Code != http.StatusOK {
		t.Fatalf("esperava 200 no heartbeat, recebeu %d: %s", hbRec.Code, hbRec.Body.String())
	}
	if len(deps.agentHeartbeats.entries) != 1 {
		t.Fatalf("esperava 1 heartbeat registrado, encontrou %d", len(deps.agentHeartbeats.entries))
	}

	// 5. Heartbeat com credencial errada deve ser rejeitado.
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+enrollResp.AgentID+"/heartbeat", bytes.NewReader(hbBody))
	badReq.Header.Set("Authorization", "Bearer credencial-errada")
	badRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("esperava 401 com credencial errada, recebeu %d", badRec.Code)
	}
}

func TestAgentTelemetry_IdempotentReplay(t *testing.T) {
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)

	agentID := uuid.New()
	secret, secretHash, _ := auth.GenerateAgentSecret()
	_ = deps.agents.Create(context.Background(), store.Agent{
		ID:         agentID,
		SiteID:     uuid.New(),
		Hostname:   "gateway",
		Platform:   "linux_amd64",
		AuthMethod: "rotating_credential",
		SecretHash: secretHash,
		EnrolledAt: time.Now().UTC(),
	})

	payload := agentTelemetryRequest{
		PingTests: []pingTestPayload{
			{
				Target:         "1.1.1.1",
				Protocol:       "icmp",
				ExecutedAt:     time.Now().UTC().Format(time.RFC3339),
				IdempotencyKey: "batch-1-seq-1",
			},
		},
	}
	body, _ := json.Marshal(payload)

	sendTelemetry := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID.String()+"/telemetry", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+secret)
		rec := httptest.NewRecorder()
		deps.server.Routes().ServeHTTP(rec, req)
		return rec
	}

	rec1 := sendTelemetry()
	if rec1.Code != http.StatusAccepted {
		t.Fatalf("esperava 202 no primeiro envio, recebeu %d: %s", rec1.Code, rec1.Body.String())
	}

	// Reenvio do mesmo lote (simulando reconexão após falha de rede) não
	// deve duplicar o registro — mesma idempotency_key.
	rec2 := sendTelemetry()
	if rec2.Code != http.StatusAccepted {
		t.Fatalf("esperava 202 no reenvio, recebeu %d: %s", rec2.Code, rec2.Body.String())
	}

	if len(deps.pingTests.byKey) != 1 {
		t.Fatalf("esperava exatamente 1 ping_test após reenvio idempotente, encontrou %d", len(deps.pingTests.byKey))
	}
}

func TestAgentTelemetry_SpeedTestAndListEndpoint(t *testing.T) {
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)

	siteID := uuid.New()
	agentID := uuid.New()
	secret, secretHash, _ := auth.GenerateAgentSecret()
	_ = deps.agents.Create(context.Background(), store.Agent{
		ID:         agentID,
		SiteID:     siteID,
		Hostname:   "gateway",
		Platform:   "linux_amd64",
		AuthMethod: "rotating_credential",
		SecretHash: secretHash,
		EnrolledAt: time.Now().UTC(),
	})

	download := 87.5
	payload := agentTelemetryRequest{
		SpeedTests: []speedTestPayload{
			{
				Mode:           "http",
				DownloadMbps:   &download,
				ExecutedAt:     time.Now().UTC().Format(time.RFC3339),
				IdempotencyKey: "speedtest-1",
			},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID.String()+"/telemetry", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+secret)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("esperava 202, recebeu %d: %s", rec.Code, rec.Body.String())
	}

	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/speed-tests", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("esperava 200 ao listar speed tests, recebeu %d: %s", listRec.Code, listRec.Body.String())
	}

	var listResp struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("resposta inválida: %v", err)
	}
	if listResp.Total != 1 {
		t.Fatalf("esperava 1 speed test listado, recebeu %d", listResp.Total)
	}
}
