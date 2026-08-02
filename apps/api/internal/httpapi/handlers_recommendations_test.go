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

func TestListRecommendations_FullFlow(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	emptyReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/recommendations", nil)
	emptyReq.AddCookie(cookie)
	emptyRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(emptyRec, emptyReq)
	if emptyRec.Code != http.StatusOK {
		t.Fatalf("esperava 200, recebeu %d: %s", emptyRec.Code, emptyRec.Body.String())
	}

	diagnosisID := uuid.New()
	deps.recommendations.bySite[siteID] = []store.Recommendation{
		{
			ID: uuid.New(), DiagnosisID: diagnosisID, SiteID: siteID,
			Action:     "Verificar o link com o provedor de internet — latência de ping (internet) mostrou desvio real do padrão histórico deste site na janela analisada.",
			Confidence: 0.5, Impact: "medium", Risk: "low",
			Evidence:  json.RawMessage(`[]`),
			CreatedAt: time.Now().UTC(),
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/recommendations", nil)
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
		t.Fatalf("esperava 1 recomendação, obtive %+v", resp)
	}
	if resp.Items[0]["diagnosis_id"] != diagnosisID.String() {
		t.Fatalf("recomendação deveria referenciar o diagnosis_id real: %+v", resp.Items[0])
	}
}
