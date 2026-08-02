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
	"egger/api/internal/rdap"
	"egger/api/internal/store"
)

// Pinger é satisfeito por *pgxpool.Pool em produção; em teste, um fake simples
// permite exercitar /readyz sem um PostgreSQL real (Seção 21).
type Pinger interface {
	Ping(ctx context.Context) error
}

// RDAPLookuper é satisfeito por *rdap.Client em produção; em teste, um fake
// evita depender de bootstrap/servidores RDAP reais na internet.
type RDAPLookuper interface {
	Lookup(ctx context.Context, query string) (rdap.Result, error)
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
	unifiDevices      store.UniFiDeviceStore
	unifiClients      store.UniFiClientStore
	anomalies         store.AnomalyStore
	spatialSurveys    store.SpatialSurveyStore
	diagnoses         store.DiagnosisStore
	recommendations   store.RecommendationStore
	reports           store.ReportStore

	rdapClient RDAPLookuper

	sessionTTL     time.Duration
	loginLimiter   *ratelimit.Limiter
	commandLimiter *ratelimit.Limiter
	rdapLimiter    *ratelimit.Limiter
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
	UniFiDevices      store.UniFiDeviceStore
	UniFiClients      store.UniFiClientStore
	Anomalies         store.AnomalyStore
	SpatialSurveys    store.SpatialSurveyStore
	Diagnoses         store.DiagnosisStore
	Recommendations   store.RecommendationStore
	Reports           store.ReportStore

	RDAPClient RDAPLookuper
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
		unifiDevices:      d.UniFiDevices,
		unifiClients:      d.UniFiClients,
		anomalies:         d.Anomalies,
		spatialSurveys:    d.SpatialSurveys,
		diagnoses:         d.Diagnoses,
		recommendations:   d.Recommendations,
		reports:           d.Reports,
		rdapClient:        d.RDAPClient,
		sessionTTL:        d.SessionTTL,
		loginLimiter:      ratelimit.New(30, 10), // 30/min por IP, burst 10 — ajustável em produção
		// Threat model §5 ("Especificar rate limiting... antes de abrir
		// qualquer endpoint de teste ativo"): comandos sob demanda (ping/
		// dns_lookup/traceroute) rodam de verdade na LAN do usuário — sem
		// limite, uma conta comprometida poderia usar o agente como
		// ferramenta de flood contra um único alvo. Chave por usuário
		// (não IP): a ameaça aqui é abuso de uma conta autenticada, não
		// tentativa de login anônima.
		commandLimiter: ratelimit.New(20, 5), // 20/min por usuário, burst 5
		// RDAP consulta serviços de terceiros (IANA + RIRs/registries) —
		// limite prudente para não transformar o backend num proxy de
		// abuso contra esses serviços públicos.
		rdapLimiter: ratelimit.New(20, 5),
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
	mux.HandleFunc("POST /api/v1/sites/{siteId}/agents/{agentId}/revoke", s.withObservability("agents.revoke",
		s.requirePermission(auth.PermManageIntegrations, s.handleRevokeAgent)))

	mux.HandleFunc("GET /api/v1/sites/{siteId}/ping-tests", s.withObservability("ping-tests.list",
		s.requirePermission(auth.PermView, s.handleListPingTests)))
	mux.HandleFunc("GET /api/v1/sites/{siteId}/speed-tests", s.withObservability("speed-tests.list",
		s.requirePermission(auth.PermView, s.handleListSpeedTests)))

	mux.HandleFunc("POST /api/v1/sites/{siteId}/commands", s.withObservability("commands.create",
		s.requirePermission(auth.PermRunTests, s.handleCreateCommand)))
	mux.HandleFunc("GET /api/v1/commands/{commandId}", s.withObservability("commands.get",
		s.requirePermission(auth.PermView, s.handleGetCommand)))

	mux.HandleFunc("GET /api/v1/sites/{siteId}/unifi/devices", s.withObservability("unifi.devices.list",
		s.requirePermission(auth.PermView, s.handleListUniFiDevices)))
	mux.HandleFunc("GET /api/v1/sites/{siteId}/unifi/clients", s.withObservability("unifi.clients.list",
		s.requirePermission(auth.PermView, s.handleListUniFiClients)))

	mux.HandleFunc("GET /api/v1/sites/{siteId}/anomalies", s.withObservability("anomalies.list",
		s.requirePermission(auth.PermView, s.handleListAnomalies)))

	mux.HandleFunc("GET /api/v1/sites/{siteId}/diagnoses", s.withObservability("diagnoses.list",
		s.requirePermission(auth.PermView, s.handleListDiagnoses)))
	mux.HandleFunc("GET /api/v1/sites/{siteId}/recommendations", s.withObservability("recommendations.list",
		s.requirePermission(auth.PermView, s.handleListRecommendations)))

	mux.HandleFunc("POST /api/v1/sites/{siteId}/reports", s.withObservability("reports.create",
		s.requirePermission(auth.PermExportData, s.handleCreateReport)))
	mux.HandleFunc("GET /api/v1/sites/{siteId}/reports", s.withObservability("reports.list",
		s.requirePermission(auth.PermView, s.handleListReports)))
	mux.HandleFunc("GET /api/v1/reports/{reportId}", s.withObservability("reports.get",
		s.requirePermission(auth.PermView, s.handleGetReport)))

	mux.HandleFunc("GET /api/v1/rdap/lookup", s.withObservability("rdap.lookup",
		s.requirePermission(auth.PermRunTests, s.handleRDAPLookup)))

	mux.HandleFunc("POST /api/v1/sites/{siteId}/spatial-surveys", s.withObservability("spatial-surveys.create",
		s.requirePermission(auth.PermRunTests, s.handleCreateSpatialSurvey)))
	mux.HandleFunc("GET /api/v1/sites/{siteId}/spatial-surveys", s.withObservability("spatial-surveys.list",
		s.requirePermission(auth.PermView, s.handleListSpatialSurveys)))
	mux.HandleFunc("GET /api/v1/spatial-surveys/{surveyId}", s.withObservability("spatial-surveys.get",
		s.requirePermission(auth.PermView, s.handleGetSpatialSurvey)))

	mux.HandleFunc("POST /api/v1/agents/enroll", s.withObservability("agents.enroll", s.handleEnrollAgent))
	mux.HandleFunc("POST /api/v1/agents/{agentId}/heartbeat", s.withObservability("agents.heartbeat",
		s.requireAgentAuth(s.handleAgentHeartbeat)))
	mux.HandleFunc("POST /api/v1/agents/{agentId}/telemetry", s.withObservability("agents.telemetry",
		s.requireAgentAuth(s.handleAgentTelemetry)))
	mux.HandleFunc("GET /api/v1/agents/{agentId}/commands", s.withObservability("agents.commands.claim",
		s.requireAgentAuth(s.handleClaimAgentCommands)))
	mux.HandleFunc("POST /api/v1/agents/{agentId}/commands/{commandId}/result", s.withObservability("agents.commands.result",
		s.requireAgentAuth(s.handleReportCommandResult)))
	mux.HandleFunc("POST /api/v1/agents/{agentId}/unifi-inventory", s.withObservability("agents.unifi-inventory",
		s.requireAgentAuth(s.handleUniFiInventory)))

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
