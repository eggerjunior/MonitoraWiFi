package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"egger/api/internal/auth"
	"egger/api/internal/store"
)

func mustHash(t *testing.T, plain string) string {
	t.Helper()
	h, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatalf("hash de senha: %v", err)
	}
	return h
}

func TestLogin_Success(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	user := store.User{
		ID:             userID,
		OrganizationID: orgID,
		Email:          "ildemar@example.com",
		PasswordHash:   mustHash(t, "senha-correta-123"),
		Role:           store.RoleAdministrator,
		CreatedAt:      time.Now().UTC(),
	}
	s := newTestServer(&fakePinger{}, newFakeUsers(user), newFakeSessions(), &fakeOrgs{}, &fakeSites{})

	body, _ := json.Marshal(map[string]string{"email": "ildemar@example.com", "password": "senha-correta-123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200, recebeu %d: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != sessionCookieName {
		t.Fatalf("esperava cookie de sessão %q, recebeu %+v", sessionCookieName, cookies)
	}
	if !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatalf("cookie de sessão deve ser HttpOnly e Secure")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	user := store.User{
		ID:           uuid.New(),
		Email:        "ildemar@example.com",
		PasswordHash: mustHash(t, "senha-correta-123"),
		Role:         store.RoleViewer,
	}
	s := newTestServer(&fakePinger{}, newFakeUsers(user), newFakeSessions(), &fakeOrgs{}, &fakeSites{})

	body, _ := json.Marshal(map[string]string{"email": "ildemar@example.com", "password": "senha-errada"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("esperava 401, recebeu %d", rec.Code)
	}
}

func TestLogin_UnknownUser_SameErrorAsWrongPassword(t *testing.T) {
	// Não deve ser possível enumerar e-mails cadastrados pela resposta.
	s := newTestServer(&fakePinger{}, newFakeUsers(), newFakeSessions(), &fakeOrgs{}, &fakeSites{})

	body, _ := json.Marshal(map[string]string{"email": "nao-existe@example.com", "password": "qualquer"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("esperava 401, recebeu %d", rec.Code)
	}

	var payload apiError
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("resposta não é JSON de erro válido: %v", err)
	}
	if payload.Error != "invalid_credentials" {
		t.Fatalf("esperava código invalid_credentials, recebeu %q", payload.Error)
	}
}

func TestAuthMe_RequiresSession(t *testing.T) {
	s := newTestServer(&fakePinger{}, newFakeUsers(), newFakeSessions(), &fakeOrgs{}, &fakeSites{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("esperava 401 sem cookie de sessão, recebeu %d", rec.Code)
	}
}

func TestAuthMe_WithValidSession(t *testing.T) {
	user := store.User{
		ID:           uuid.New(),
		Email:        "ildemar@example.com",
		PasswordHash: mustHash(t, "senha-correta-123"),
		Role:         store.RoleOwner,
		CreatedAt:    time.Now().UTC(),
	}
	s := newTestServer(&fakePinger{}, newFakeUsers(user), newFakeSessions(), &fakeOrgs{}, &fakeSites{})

	// login para obter cookie
	body, _ := json.Marshal(map[string]string{"email": "ildemar@example.com", "password": "senha-correta-123"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	loginRec := httptest.NewRecorder()
	s.Routes().ServeHTTP(loginRec, loginReq)
	cookie := loginRec.Result().Cookies()[0]

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.AddCookie(cookie)
	meRec := httptest.NewRecorder()
	s.Routes().ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusOK {
		t.Fatalf("esperava 200, recebeu %d: %s", meRec.Code, meRec.Body.String())
	}

	var resp userResponse
	if err := json.Unmarshal(meRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("resposta inválida: %v", err)
	}
	if resp.Email != "ildemar@example.com" {
		t.Fatalf("esperava email correto, recebeu %q", resp.Email)
	}
}

func TestLogin_RateLimited(t *testing.T) {
	s := newTestServer(&fakePinger{}, newFakeUsers(), newFakeSessions(), &fakeOrgs{}, &fakeSites{})

	body, _ := json.Marshal(map[string]string{"email": "x@example.com", "password": "qualquer"})

	var lastCode int
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.RemoteAddr = "203.0.113.9:12345"
		rec := httptest.NewRecorder()
		s.Routes().ServeHTTP(rec, req)
		lastCode = rec.Code
		if lastCode == http.StatusTooManyRequests {
			break
		}
	}
	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("esperava eventualmente 429 após rajada de tentativas, último código: %d", lastCode)
	}
}
