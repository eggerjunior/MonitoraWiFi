package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"egger/api/internal/store"
)

func TestListDiagnoses_FullFlow(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	emptyReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/diagnoses", nil)
	emptyReq.AddCookie(cookie)
	emptyRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(emptyRec, emptyReq)
	if emptyRec.Code != http.StatusOK {
		t.Fatalf("esperava 200, recebeu %d: %s", emptyRec.Code, emptyRec.Body.String())
	}
	var emptyResp struct {
		Items []any `json:"items"`
		Total int   `json:"total"`
	}
	json.Unmarshal(emptyRec.Body.Bytes(), &emptyResp)
	if len(emptyResp.Items) != 0 || emptyResp.Total != 0 {
		t.Fatalf("esperava lista vazia (sem evidência ainda), obtive %+v", emptyResp)
	}

	// Simula o que o worker (apps/worker/internal/diagnostics) gravaria.
	deps.diagnoses.bySite[siteID] = []store.Diagnosis{
		{
			ID: uuid.New(), SiteID: siteID, Category: "internet_slow",
			Summary:    "Internet lenta: 1 anomalia real em latência de ping (internet).",
			Confidence: 0.5, Impact: "medium", Risk: "low",
			Evidence:    json.RawMessage(`[{"anomaly_id":"` + uuid.New().String() + `","metric":"ping_latency_ms_p50"}]`),
			WindowStart: time.Now().UTC().Add(-time.Hour), WindowEnd: time.Now().UTC(),
			DetectedAt: time.Now().UTC(),
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/diagnoses", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200, recebeu %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("esperava 1 diagnóstico, obtive %+v", resp)
	}
	if resp.Items[0]["category"] != "internet_slow" || resp.Items[0]["confidence"] != 0.5 {
		t.Fatalf("diagnóstico inesperado: %+v", resp.Items[0])
	}
	evidence, ok := resp.Items[0]["evidence"].([]any)
	if !ok || len(evidence) != 1 {
		t.Fatalf("evidência deveria vir junto no payload: %+v", resp.Items[0]["evidence"])
	}
}

func TestListDiagnoses_ViewerPodeVer(t *testing.T) {
	siteID := uuid.New()
	viewer := store.User{ID: uuid.New(), Email: "viewer@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleViewer}
	deps := newAgentTestServer(viewer)
	cookie := loginAndGetCookie(t, deps.server, "viewer@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/diagnoses", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200 (viewer tem PermView), recebeu %d: %s", rec.Code, rec.Body.String())
	}
}
