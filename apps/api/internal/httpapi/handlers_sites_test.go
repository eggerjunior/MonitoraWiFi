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

func TestGetSite_NotFound(t *testing.T) {
	user := store.User{ID: uuid.New(), Email: "op@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleOperator}
	s := newTestServer(&fakePinger{}, newFakeUsers(user), newFakeSessions(), &fakeOrgs{}, &fakeSites{})
	cookie := loginAndGetCookie(t, s, "op@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+uuid.NewString(), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperava 404, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListSites_FiltersByOrganization(t *testing.T) {
	orgA := uuid.New()
	orgB := uuid.New()
	siteA := store.Site{ID: uuid.New(), OrganizationID: orgA, Name: "Residência Egger", Timezone: "America/Sao_Paulo", CreatedAt: time.Now().UTC()}
	siteB := store.Site{ID: uuid.New(), OrganizationID: orgB, Name: "Outro site", Timezone: "UTC", CreatedAt: time.Now().UTC()}

	user := store.User{ID: uuid.New(), OrganizationID: orgA, Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	s := newTestServer(&fakePinger{}, newFakeUsers(user), newFakeSessions(), &fakeOrgs{}, &fakeSites{items: []store.Site{siteA, siteB}})
	cookie := loginAndGetCookie(t, s, "admin@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites?organization_id="+orgA.String(), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200, recebeu %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Items []siteResponse `json:"items"`
		Total int            `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("resposta inválida: %v", err)
	}
	if payload.Total != 1 || payload.Items[0].Name != "Residência Egger" {
		t.Fatalf("esperava apenas o site da organização A, recebeu %+v", payload)
	}
}
