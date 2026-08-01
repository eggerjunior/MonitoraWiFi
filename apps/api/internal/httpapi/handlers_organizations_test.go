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

func loginAndGetCookie(t *testing.T, s *Server, email, password string) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login falhou: %d %s", rec.Code, rec.Body.String())
	}
	return rec.Result().Cookies()[0]
}

func TestListOrganizations_RequiresAuth(t *testing.T) {
	s := newTestServer(&fakePinger{}, newFakeUsers(), newFakeSessions(), &fakeOrgs{}, &fakeSites{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("esperava 401 sem sessão, recebeu %d", rec.Code)
	}
}

func TestListOrganizations_ViewerCanView(t *testing.T) {
	org := store.Organization{ID: uuid.New(), Name: "Egger", PlanTier: "standard", CreatedAt: time.Now().UTC()}
	user := store.User{ID: uuid.New(), OrganizationID: org.ID, Email: "viewer@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleViewer}
	s := newTestServer(&fakePinger{}, newFakeUsers(user), newFakeSessions(), &fakeOrgs{items: []store.Organization{org}}, &fakeSites{})

	cookie := loginAndGetCookie(t, s, "viewer@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200 para viewer (tem permissão view), recebeu %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("resposta inválida: %v", err)
	}
	if payload.Total != 1 {
		t.Fatalf("esperava total=1, recebeu %d", payload.Total)
	}
}

func TestListOrganizations_UnknownRoleIsForbidden(t *testing.T) {
	// Fail-closed: um papel não reconhecido pela matriz RBAC nunca tem
	// permissão, mesmo que a sessão seja válida (ver auth.HasPermission).
	user := store.User{ID: uuid.New(), Email: "raro@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.Role("papel-inexistente")}
	s := newTestServer(&fakePinger{}, newFakeUsers(user), newFakeSessions(), &fakeOrgs{}, &fakeSites{})

	cookie := loginAndGetCookie(t, s, "raro@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperava 403 para papel desconhecido (fail-closed), recebeu %d", rec.Code)
	}
}
