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

	agents            store.AgentStore
	agentEnrollTokens store.AgentEnrollmentTokenStore
	agentHeartbeats   store.AgentHeartbeatStore
	pingTests         store.PingTestStore
	speedTests        store.SpeedTestStore
	agentCommands     store.AgentCommandStore

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

	Agents            store.AgentStore
	AgentEnrollTokens store.AgentEnrollmentTokenStore
	AgentHeartbeats   store.AgentHeartbeatStore
	PingTests         store.PingTestStore
	SpeedTests        store.SpeedTestStore
	AgentCommands     store.AgentCommandStore
}

func NewServer(d Deps) *Server {
	return &Server{
		logger:            d.Logger,
		tracer:            d.Tracer,
		pool:              d.Pool,
		orgs:              d.Orgs,
		sites:             d.Sites,
		users:             d.Users,
		sessions:          d.Sessions,
		audit:             d.Audit,
		agents:            d.Agents,
		agentEnrollTokens: d.AgentEnrollTokens,
		agentHeartbeats:   d.AgentHeartbeats,
		pingTests:         d.PingTests,
		speedTests:        d.SpeedTests,
		agentCommands:     d.AgentCommands,
		sessionTTL:        d.SessionTTL,
		loginLimiter:      ratelimit.New(30, 10), // 30/min por IP, burst 10 — ajustável em produção
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

	mux.HandleFunc("POST /api/v1/sites/{siteId}/agent-enrollment-tokens", s.withObservability("agents.create-enrollment-token",
		s.requirePermission(auth.PermManageIntegrations, s.handleCreateAgentEnrollmentToken)))
	mux.HandleFunc("GET /api/v1/sites/{siteId}/agents", s.withObservability("agents.list",
		s.requirePermission(auth.PermView, s.handleListAgents)))

	mux.HandleFunc("GET /api/v1/sites/{siteId}/ping-tests", s.withObservability("ping-tests.list",
		s.requirePermission(auth.PermView, s.handleListPingTests)))
	mux.HandleFunc("GET /api/v1/sites/{siteId}/speed-tests", s.withObservability("speed-tests.list",
		s.requirePermission(auth.PermView, s.handleListSpeedTests)))

	mux.HandleFunc("POST /api/v1/sites/{siteId}/commands", s.withObservability("commands.create",
		s.requirePermission(auth.PermRunTests, s.handleCreateCommand)))
	mux.HandleFunc("GET /api/v1/commands/{commandId}", s.withObservability("commands.get",
		s.requirePermission(auth.PermView, s.handleGetCommand)))

	mux.HandleFunc("POST /api/v1/agents/enroll", s.withObservability("agents.enroll", s.handleEnrollAgent))
	mux.HandleFunc("POST /api/v1/agents/{agentId}/heartbeat", s.withObservability("agents.heartbeat",
		s.requireAgentAuth(s.handleAgentHeartbeat)))
	mux.HandleFunc("POST /api/v1/agents/{agentId}/telemetry", s.withObservability("agents.telemetry",
		s.requireAgentAuth(s.handleAgentTelemetry)))
	mux.HandleFunc("GET /api/v1/agents/{agentId}/commands", s.withObservability("agents.commands.claim",
		s.requireAgentAuth(s.handleClaimAgentCommands)))
	mux.HandleFunc("POST /api/v1/agents/{agentId}/commands/{commandId}/result", s.withObservability("agents.commands.result",
		s.requireAgentAuth(s.handleReportCommandResult)))

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
