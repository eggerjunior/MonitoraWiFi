// Package httpapi implementa os handlers REST da Fase 1 (Seção 17): health
// checks, autenticação, organizações e sites. Endpoints das fases seguintes
// (telemetria, UniFi, testes ativos, LiDAR, alertas) são adicionados aqui sem
// alterar o que já existe.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"

	"egger/api/internal/auth"
	"egger/api/internal/ratelimit"
	"egger/api/internal/store"
)

// Pinger é satisfeito por *pgxpool.Pool em produção; em teste, um fake simples
// permite exercitar /readyz sem um PostgreSQL real (Seção 21).
type Pinger interface {
	Ping(ctx context.Context) error
}

type Server struct {
	logger   *slog.Logger
	tracer   trace.Tracer
	pool     Pinger // usado apenas pelo readiness probe
	orgs     store.OrganizationStore
	sites    store.SiteStore
	users    store.UserStore
	sessions store.SessionStore
	audit    store.AuditStore

	sessionTTL   time.Duration
	loginLimiter *ratelimit.Limiter
}

type Deps struct {
	Logger     *slog.Logger
	Tracer     trace.Tracer
	Pool       Pinger
	Orgs       store.OrganizationStore
	Sites      store.SiteStore
	Users      store.UserStore
	Sessions   store.SessionStore
	Audit      store.AuditStore
	SessionTTL time.Duration
}

func NewServer(d Deps) *Server {
	return &Server{
		logger:       d.Logger,
		tracer:       d.Tracer,
		pool:         d.Pool,
		orgs:         d.Orgs,
		sites:        d.Sites,
		users:        d.Users,
		sessions:     d.Sessions,
		audit:        d.Audit,
		sessionTTL:   d.SessionTTL,
		loginLimiter: ratelimit.New(30, 10), // 30/min por IP, burst 10 — ajustável em produção
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.withObservability("healthz", s.handleHealthz))
	mux.HandleFunc("GET /readyz", s.withObservability("readyz", s.handleReadyz))

	mux.HandleFunc("POST /api/v1/auth/login", s.withObservability("auth.login", s.handleLogin))
	mux.HandleFunc("POST /api/v1/auth/logout", s.withObservability("auth.logout", s.requireAuth(s.handleLogout)))
	mux.HandleFunc("GET /api/v1/auth/me", s.withObservability("auth.me", s.requireAuth(s.handleMe)))

	mux.HandleFunc("GET /api/v1/organizations", s.withObservability("organizations.list",
		s.requirePermission(auth.PermView, s.handleListOrganizations)))

	mux.HandleFunc("GET /api/v1/sites", s.withObservability("sites.list",
		s.requirePermission(auth.PermView, s.handleListSites)))
	mux.HandleFunc("GET /api/v1/sites/{siteId}", s.withObservability("sites.get",
		s.requirePermission(auth.PermView, s.handleGetSite)))

	return mux
}

func (s *Server) checkReadiness(ctx context.Context) map[string]string {
	deps := map[string]string{}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := s.pool.Ping(pingCtx); err != nil {
		deps["database"] = "unavailable"
	} else {
		deps["database"] = "ok"
	}

	return deps
}
