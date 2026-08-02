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

func validSpatialSurveyBody() map[string]any {
	now := time.Now().UTC()
	rtt := 24.5
	ssid := "Egger_Principal"
	return map[string]any{
		"name":         "Térreo",
		"device_model": "iPhone15,2",
		"lidar_used":   true,
		"started_at":   now.Add(-5 * time.Minute).Format(time.RFC3339Nano),
		"finished_at":  now.Format(time.RFC3339Nano),
		"samples": []map[string]any{
			{
				"position_x":     1.5,
				"position_y":     0.0,
				"position_z":     -2.3,
				"ssid":           ssid,
				"bssid":          nil,
				"rtt_ms":         rtt,
				"is_expensive":   false,
				"is_constrained": false,
				"interface_type": "wifi",
				"captured_at":    now.Format(time.RFC3339Nano),
			},
		},
	}
}

func TestCreateSpatialSurvey_FluxoCompleto(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	body, _ := json.Marshal(validSpatialSurveyBody())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/spatial-surveys", bytes.NewReader(body))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("esperava 201, recebeu %d: %s", rec.Code, rec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("resposta inválida: %v", err)
	}
	if created["sample_count"].(float64) != 1 {
		t.Fatalf("esperava sample_count=1, recebeu %v", created["sample_count"])
	}
	surveyID := created["id"].(string)

	// Listagem não deve incluir amostras.
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/spatial-surveys", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("esperava 200 na listagem, recebeu %d: %s", listRec.Code, listRec.Body.String())
	}
	var listBody map[string]any
	json.Unmarshal(listRec.Body.Bytes(), &listBody)
	items := listBody["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("esperava 1 levantamento na listagem, recebeu %d", len(items))
	}
	if _, hasSamples := items[0].(map[string]any)["samples"]; hasSamples {
		t.Fatal("listagem não deveria incluir amostras (só metadados)")
	}

	// Detalhe deve incluir amostras.
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/spatial-surveys/"+surveyID, nil)
	getReq.AddCookie(cookie)
	getRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("esperava 200 no detalhe, recebeu %d: %s", getRec.Code, getRec.Body.String())
	}
	var detail map[string]any
	json.Unmarshal(getRec.Body.Bytes(), &detail)
	samples, ok := detail["samples"].([]any)
	if !ok || len(samples) != 1 {
		t.Fatalf("esperava 1 amostra no detalhe, recebeu %v", detail["samples"])
	}
	sample := samples[0].(map[string]any)
	if sample["ssid"] != "Egger_Principal" {
		t.Fatalf("ssid inesperado: %v", sample["ssid"])
	}
}

func TestCreateSpatialSurvey_RequerAoMenosUmaAmostra(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	b := validSpatialSurveyBody()
	b["samples"] = []map[string]any{}
	body, _ := json.Marshal(b)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/spatial-surveys", bytes.NewReader(body))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 sem amostras, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateSpatialSurvey_InterfaceTypeInvalido(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	b := validSpatialSurveyBody()
	samples := b["samples"].([]map[string]any)
	samples[0]["interface_type"] = "bluetooth"
	body, _ := json.Marshal(b)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/spatial-surveys", bytes.NewReader(body))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 com interface_type inválido, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetSpatialSurvey_NaoEncontrado(t *testing.T) {
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial-surveys/"+uuid.NewString(), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperava 404, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateSpatialSurvey_ViewerSemPermissao(t *testing.T) {
	siteID := uuid.New()
	viewer := store.User{ID: uuid.New(), Email: "viewer@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleViewer}
	deps := newAgentTestServer(viewer)
	cookie := loginAndGetCookie(t, deps.server, "viewer@example.com", "senha12345")

	body, _ := json.Marshal(validSpatialSurveyBody())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/"+siteID.String()+"/spatial-surveys", bytes.NewReader(body))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperava 403 pra viewer, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}
