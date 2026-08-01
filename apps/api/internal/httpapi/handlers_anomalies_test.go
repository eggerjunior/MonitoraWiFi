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

func TestListAnomalies_FullFlow(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	// Sem anomalias ainda — resposta honesta vazia, nunca inventada.
	emptyReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/anomalies", nil)
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
		t.Fatalf("esperava lista vazia, obtive %+v", emptyResp)
	}

	// Simula o que o worker (apps/worker) gravaria.
	deps.anomalies.bySite[siteID] = []store.Anomaly{
		{
			ID: uuid.New(), SiteID: siteID, Metric: "ping_latency_ms_p50",
			ObservedAt: time.Now().UTC(), Value: 500, BucketMean: 20, BucketSize: 8, ZScore: 391.9,
			DetectedAt: time.Now().UTC(),
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/anomalies", nil)
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
		t.Fatalf("esperava 1 anomalia, obtive %+v", resp)
	}
	if resp.Items[0]["metric"] != "ping_latency_ms_p50" || resp.Items[0]["value"] != 500.0 {
		t.Fatalf("anomalia inesperada: %+v", resp.Items[0])
	}
}

func TestListAnomalies_ViewerPodeVer(t *testing.T) {
	siteID := uuid.New()
	viewer := store.User{ID: uuid.New(), Email: "viewer@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleViewer}
	deps := newAgentTestServer(viewer)
	cookie := loginAndGetCookie(t, deps.server, "viewer@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/anomalies", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200 (viewer tem PermView), recebeu %d: %s", rec.Code, rec.Body.String())
	}
}
