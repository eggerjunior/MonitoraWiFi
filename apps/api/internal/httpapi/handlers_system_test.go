package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	s := newTestServer(&fakePinger{}, newFakeUsers(), newFakeSessions(), &fakeOrgs{}, &fakeSites{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200, recebeu %d", rec.Code)
	}
}

func TestReadyz_DatabaseOK(t *testing.T) {
	s := newTestServer(&fakePinger{}, newFakeUsers(), newFakeSessions(), &fakeOrgs{}, &fakeSites{})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200, recebeu %d: %s", rec.Code, rec.Body.String())
	}
}

func TestReadyz_DatabaseDown(t *testing.T) {
	s := newTestServer(&fakePinger{err: errors.New("connection refused")}, newFakeUsers(), newFakeSessions(), &fakeOrgs{}, &fakeSites{})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503 quando o banco está indisponível, recebeu %d", rec.Code)
	}
}
