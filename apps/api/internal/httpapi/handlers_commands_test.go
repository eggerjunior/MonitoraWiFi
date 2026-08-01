package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"egger/api/internal/store"
)

// enrollTestAgent registra e enrola um agente de verdade contra o site
// (via o fluxo HTTP real, não inserção direta no fake) para que os testes de
// comando exerçam a mesma resolução de "agente ativo do site" que produção.
func enrollTestAgent(t *testing.T, deps agentTestDeps, siteID uuid.UUID) (agentID, agentSecret string) {
	t.Helper()

	tokenReq := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/agent-enrollment-tokens", nil)
	admin := firstAdminUser(t, deps)
	cookie := loginAndGetCookie(t, deps.server, admin.Email, "senha12345")
	tokenReq.AddCookie(cookie)
	tokenRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(tokenRec, tokenReq)
	if tokenRec.Code != http.StatusCreated {
		t.Fatalf("esperava 201 ao criar token de enrolamento, recebeu %d: %s", tokenRec.Code, tokenRec.Body.String())
	}
	var tokenResp struct {
		EnrollmentToken string `json:"enrollment_token"`
	}
	if err := json.Unmarshal(tokenRec.Body.Bytes(), &tokenResp); err != nil {
		t.Fatalf("resposta de token inválida: %v", err)
	}

	enrollBody, _ := json.Marshal(map[string]string{
		"enrollment_token": tokenResp.EnrollmentToken,
		"hostname":         "agente-teste",
		"platform":         "linux_amd64",
		"version":          "0.1.0",
	})
	enrollReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/enroll", bytes.NewReader(enrollBody))
	enrollRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(enrollRec, enrollReq)
	if enrollRec.Code != http.StatusCreated {
		t.Fatalf("esperava 201 ao enrolar agente, recebeu %d: %s", enrollRec.Code, enrollRec.Body.String())
	}
	var enrollResp struct {
		AgentID     string `json:"agent_id"`
		AgentSecret string `json:"agent_secret"`
	}
	if err := json.Unmarshal(enrollRec.Body.Bytes(), &enrollResp); err != nil {
		t.Fatalf("resposta de enroll inválida: %v", err)
	}
	return enrollResp.AgentID, enrollResp.AgentSecret
}

func firstAdminUser(t *testing.T, deps agentTestDeps) store.User {
	t.Helper()
	for _, u := range deps.users.byEmail {
		return u
	}
	t.Fatal("nenhum usuário registrado no fake de teste")
	return store.User{}
}

func TestAgentCommand_FullFlow(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	// Sem agente ativo no site, criar comando deve falhar com 409.
	createBody, _ := json.Marshal(map[string]any{
		"type":   "ping",
		"params": map[string]string{"target": "1.1.1.1", "protocol": "icmp"},
	})
	noAgentReq := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/commands", bytes.NewReader(createBody))
	noAgentReq.AddCookie(cookie)
	noAgentRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(noAgentRec, noAgentReq)
	if noAgentRec.Code != http.StatusConflict {
		t.Fatalf("esperava 409 sem agente ativo, recebeu %d: %s", noAgentRec.Code, noAgentRec.Body.String())
	}

	agentID, agentSecret := enrollTestAgent(t, deps, siteID)

	// 1. Usuário cria um comando de ping.
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/commands", bytes.NewReader(createBody))
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("esperava 202 ao criar comando, recebeu %d: %s", createRec.Code, createRec.Body.String())
	}
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("resposta de criação inválida: %v", err)
	}
	if created.Status != "pending" {
		t.Fatalf("status = %q, esperado pending", created.Status)
	}

	// 2. Usuário consulta o status — ainda pending.
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/commands/"+created.ID, nil)
	getReq.AddCookie(cookie)
	getRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("esperava 200 ao buscar comando, recebeu %d: %s", getRec.Code, getRec.Body.String())
	}

	// 3. Agente reivindica comandos pendentes.
	claimReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/"+agentID+"/commands", nil)
	claimReq.Header.Set("Authorization", "Bearer "+agentSecret)
	claimRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(claimRec, claimReq)
	if claimRec.Code != http.StatusOK {
		t.Fatalf("esperava 200 ao reivindicar comandos, recebeu %d: %s", claimRec.Code, claimRec.Body.String())
	}
	var claimed struct {
		Items []struct {
			ID     string `json:"id"`
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(claimRec.Body.Bytes(), &claimed); err != nil {
		t.Fatalf("resposta de claim inválida: %v", err)
	}
	if len(claimed.Items) != 1 || claimed.Items[0].ID != created.ID {
		t.Fatalf("esperava 1 comando reivindicado com id %s, obtive %+v", created.ID, claimed.Items)
	}
	if claimed.Items[0].Status != "claimed" {
		t.Fatalf("status pós-claim = %q, esperado claimed", claimed.Items[0].Status)
	}

	// 4. Reivindicar de novo não deve retornar o mesmo comando (já claimed).
	claimAgainReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/"+agentID+"/commands", nil)
	claimAgainReq.Header.Set("Authorization", "Bearer "+agentSecret)
	claimAgainRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(claimAgainRec, claimAgainReq)
	var claimedAgain struct {
		Items []any `json:"items"`
	}
	json.Unmarshal(claimAgainRec.Body.Bytes(), &claimedAgain)
	if len(claimedAgain.Items) != 0 {
		t.Fatalf("esperava 0 comandos na segunda reivindicação, obtive %d", len(claimedAgain.Items))
	}

	// 5. Agente reporta o resultado.
	resultBody, _ := json.Marshal(map[string]any{
		"status": "completed",
		"result": map[string]any{"latency_ms_p50": 12.3, "packet_loss_pct": 0},
	})
	resultReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/commands/"+created.ID+"/result", bytes.NewReader(resultBody))
	resultReq.Header.Set("Authorization", "Bearer "+agentSecret)
	resultRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(resultRec, resultReq)
	if resultRec.Code != http.StatusOK {
		t.Fatalf("esperava 200 ao reportar resultado, recebeu %d: %s", resultRec.Code, resultRec.Body.String())
	}

	// 6. Usuário consulta de novo — agora completed, com o resultado.
	finalReq := httptest.NewRequest(http.MethodGet, "/api/v1/commands/"+created.ID, nil)
	finalReq.AddCookie(cookie)
	finalRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(finalRec, finalReq)
	var final struct {
		Status string         `json:"status"`
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal(finalRec.Body.Bytes(), &final); err != nil {
		t.Fatalf("resposta final inválida: %v", err)
	}
	if final.Status != "completed" {
		t.Fatalf("status final = %q, esperado completed", final.Status)
	}
	if final.Result["latency_ms_p50"] != 12.3 {
		t.Fatalf("result.latency_ms_p50 = %v, esperado 12.3", final.Result["latency_ms_p50"])
	}
}

func TestCreateCommand_UnsupportedType(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	body, _ := json.Marshal(map[string]any{"type": "traceroute"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/commands", bytes.NewReader(body))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 para tipo não suportado, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateCommand_ViewerSemPermissao(t *testing.T) {
	siteID := uuid.New()
	viewer := store.User{ID: uuid.New(), Email: "viewer@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleViewer}
	deps := newAgentTestServer(viewer)
	cookie := loginAndGetCookie(t, deps.server, "viewer@example.com", "senha12345")

	body, _ := json.Marshal(map[string]any{
		"type":   "ping",
		"params": map[string]string{"target": "1.1.1.1"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/commands", bytes.NewReader(body))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperava 403 para viewer, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestReportCommandResult_AgenteErradoNegado(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	agentID, agentSecret := enrollTestAgent(t, deps, siteID)
	_ = agentSecret

	createBody, _ := json.Marshal(map[string]any{
		"type":   "ping",
		"params": map[string]string{"target": "1.1.1.1"},
	})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/commands", bytes.NewReader(createBody))
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(createRec, createReq)
	var created struct {
		ID string `json:"id"`
	}
	json.Unmarshal(createRec.Body.Bytes(), &created)

	// Enrola um SEGUNDO agente (em outro site) e tenta reportar resultado,
	// autenticado como si mesmo (credencial válida, agentId da URL bate com
	// a própria credencial), do comando que pertence ao PRIMEIRO agente —
	// deve ser negado pelo handler (403 forbidden), não pela auth (401).
	otherSiteID := uuid.New()
	otherAgentID, otherSecret := enrollTestAgent(t, deps, otherSiteID)

	resultBody, _ := json.Marshal(map[string]any{"status": "completed", "result": map[string]any{}})
	resultReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+otherAgentID+"/commands/"+created.ID+"/result", bytes.NewReader(resultBody))
	resultReq.Header.Set("Authorization", "Bearer "+otherSecret)
	resultRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(resultRec, resultReq)
	if resultRec.Code != http.StatusForbidden {
		t.Fatalf("esperava 403 (comando pertence a outro agente), recebeu %d: %s", resultRec.Code, resultRec.Body.String())
	}

	// Sanity check: a credencial errada (agentId != dono da credencial)
	// continua barrada na autenticação (401), como nos outros endpoints de
	// agente já testados em handlers_agents_test.go.
	wrongAuthReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/commands/"+created.ID+"/result", bytes.NewReader(resultBody))
	wrongAuthReq.Header.Set("Authorization", "Bearer "+otherSecret)
	wrongAuthRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(wrongAuthRec, wrongAuthReq)
	if wrongAuthRec.Code != http.StatusUnauthorized {
		t.Fatalf("esperava 401 (credencial não bate com agentId da URL), recebeu %d: %s", wrongAuthRec.Code, wrongAuthRec.Body.String())
	}
}

func TestGetCommand_NaoEncontrado(t *testing.T) {
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/commands/"+uuid.New().String(), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperava 404, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}
