package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"egger/api/internal/rdap"
	"egger/api/internal/store"
)

func TestRDAPLookup_Sucesso(t *testing.T) {
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")
	deps.rdapClient.result = rdap.Result{
		Server:          "https://rdap.verisign.com/com/v1/",
		ObjectClassName: "domain",
		Handle:          "EXAMPLE-HANDLE",
		Name:            "EXAMPLE.COM",
		Status:          []string{"active"},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rdap/lookup?query=example.com", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200, recebeu %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "EXAMPLE.COM") {
		t.Fatalf("esperava nome no corpo da resposta, recebi: %s", rec.Body.String())
	}
}

func TestRDAPLookup_QueryObrigatoria(t *testing.T) {
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rdap/lookup", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 sem query, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRDAPLookup_NenhumServidorEncontrado(t *testing.T) {
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")
	deps.rdapClient.err = rdap.ErrNoServer

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rdap/lookup?query=nada", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperava 404 quando nenhum servidor RDAP é encontrado, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRDAPLookup_ViewerSemPermissao(t *testing.T) {
	viewer := store.User{ID: uuid.New(), Email: "viewer@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleViewer}
	deps := newAgentTestServer(viewer)
	cookie := loginAndGetCookie(t, deps.server, "viewer@example.com", "senha12345")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rdap/lookup?query=example.com", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperava 403 (viewer sem PermRunTests), recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRDAPLookup_RateLimit(t *testing.T) {
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	sawRateLimit := false
	for i := 0; i < 40; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/rdap/lookup?query=example.com", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		deps.server.Routes().ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			sawRateLimit = true
			break
		}
	}
	if !sawRateLimit {
		t.Fatal("esperava 429 em algum momento das 40 requisições")
	}
}
