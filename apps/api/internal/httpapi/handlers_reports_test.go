package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"egger/api/internal/store"
)

func TestCreateReport_AgregaDadosReaisDoPeriodo(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	now := time.Now().UTC()
	diagnosisID := uuid.New()

	// Uma anomalia dentro da janela padrão (últimos 7 dias) e uma fora dela
	// (30 dias atrás) — só a primeira deve entrar no relatório.
	deps.anomalies.bySite[siteID] = []store.Anomaly{
		{ID: uuid.New(), SiteID: siteID, Metric: "ping_latency_ms_p50", ObservedAt: now.Add(-time.Hour), Value: 400, BucketMean: 20, BucketSize: 8, ZScore: 6, DetectedAt: now},
		{ID: uuid.New(), SiteID: siteID, Metric: "ping_latency_ms_p50", ObservedAt: now.Add(-30 * 24 * time.Hour), Value: 400, BucketMean: 20, BucketSize: 8, ZScore: 6, DetectedAt: now},
	}
	deps.diagnoses.bySite[siteID] = []store.Diagnosis{
		{ID: diagnosisID, SiteID: siteID, Category: "internet_slow", Summary: "Internet lenta.", Confidence: 0.5, Impact: "medium", Risk: "low",
			Evidence: json.RawMessage(`[]`), WindowStart: now.Add(-time.Hour), WindowEnd: now, DetectedAt: now},
	}
	deps.recommendations.bySite[siteID] = []store.Recommendation{
		{ID: uuid.New(), DiagnosisID: diagnosisID, SiteID: siteID, Action: "Verificar o provedor.", Confidence: 0.5, Impact: "medium", Risk: "low",
			Evidence: json.RawMessage(`[]`), CreatedAt: now},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/reports", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("esperava 201, recebeu %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID      string         `json:"id"`
		Kind    string         `json:"kind"`
		Content map[string]any `json:"content"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("resposta inválida: %v", err)
	}
	if resp.Kind != "diagnostics_summary" {
		t.Fatalf("kind inesperado: %s", resp.Kind)
	}
	if resp.Content["anomaly_count"] != float64(1) {
		t.Fatalf("esperava anomaly_count=1 (só a anomalia dentro da janela), obtive %+v", resp.Content["anomaly_count"])
	}
	diagnoses, _ := resp.Content["diagnoses"].([]any)
	if len(diagnoses) != 1 {
		t.Fatalf("esperava 1 diagnóstico no relatório, obtive %+v", resp.Content["diagnoses"])
	}
	recommendations, _ := resp.Content["recommendations"].([]any)
	if len(recommendations) != 1 {
		t.Fatalf("esperava 1 recomendação no relatório, obtive %+v", resp.Content["recommendations"])
	}
}

func TestCreateReport_ViewerSemPermissao(t *testing.T) {
	siteID := uuid.New()
	viewer := store.User{ID: uuid.New(), Email: "viewer@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleViewer}
	deps := newAgentTestServer(viewer)
	cookie := loginAndGetCookie(t, deps.server, "viewer@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/reports", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperava 403 (viewer sem PermExportData), recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateReport_PeriodoInvalido(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	body, _ := json.Marshal(map[string]string{
		"period_start": "2026-08-02T00:00:00Z",
		"period_end":   "2026-08-01T00:00:00Z",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/reports", bytes.NewReader(body))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 (period_start depois de period_end), recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListAndGetReport(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/reports", nil)
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("esperava 201 ao criar, recebeu %d: %s", createRec.Code, createRec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	json.Unmarshal(createRec.Body.Bytes(), &created)

	// Lista não deve incluir o conteúdo completo.
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/reports", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(listRec, listReq)
	var listResp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	json.Unmarshal(listRec.Body.Bytes(), &listResp)
	if listResp.Total != 1 {
		t.Fatalf("esperava 1 relatório na lista, obtive %+v", listResp)
	}
	if _, hasContent := listResp.Items[0]["content"]; hasContent {
		t.Fatalf("lista não deveria incluir content: %+v", listResp.Items[0])
	}

	// Get por ID deve incluir o conteúdo completo.
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/reports/"+created.ID, nil)
	getReq.AddCookie(cookie)
	getRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("esperava 200, recebeu %d: %s", getRec.Code, getRec.Body.String())
	}
	var getResp map[string]any
	json.Unmarshal(getRec.Body.Bytes(), &getResp)
	if _, hasContent := getResp["content"]; !hasContent {
		t.Fatalf("get deveria incluir content: %+v", getResp)
	}
}

func TestGetReport_NaoEncontrado(t *testing.T) {
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/"+uuid.New().String(), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperava 404, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}
