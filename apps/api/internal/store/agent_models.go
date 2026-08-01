package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrTokenExpiredOrUsed = errors.New("token de enrolamento inválido, expirado ou já usado")

type Agent struct {
	ID         uuid.UUID
	SiteID     uuid.UUID
	Hostname   string
	Version    string
	Platform   string
	AuthMethod string
	SecretHash string
	EnrolledAt time.Time
	LastSeenAt *time.Time
	RevokedAt  *time.Time
}

type AgentEnrollmentToken struct {
	ID            uuid.UUID
	SiteID        uuid.UUID
	TokenHash     string
	CreatedBy     *uuid.UUID
	CreatedAt     time.Time
	ExpiresAt     time.Time
	UsedAt        *time.Time
	UsedByAgentID *uuid.UUID
}

type AgentHeartbeat struct {
	Time        time.Time
	AgentID     uuid.UUID
	Status      string
	QueuedItems int
	CPUPct      *float64
	MemPct      *float64
}

type PingTest struct {
	ID             uuid.UUID
	AgentID        uuid.UUID
	Target         string
	Protocol       string
	LatencyMsP50   *float64
	LatencyMsP95   *float64
	LatencyMsP99   *float64
	JitterMs       *float64
	PacketLossPct  *float64
	ExecutedAt     time.Time
	IdempotencyKey string
}

type AgentStore interface {
	Create(ctx context.Context, a Agent) error
	Get(ctx context.Context, id uuid.UUID) (Agent, error)
	ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]Agent, int, error)
	UpdateLastSeen(ctx context.Context, id uuid.UUID, at time.Time) error
}

type AgentEnrollmentTokenStore interface {
	Create(ctx context.Context, t AgentEnrollmentToken) error
	GetValidByTokenHash(ctx context.Context, tokenHash string, now time.Time) (AgentEnrollmentToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID, agentID uuid.UUID, at time.Time) error
}

type AgentHeartbeatStore interface {
	Record(ctx context.Context, h AgentHeartbeat) error
}

type PingTestStore interface {
	// InsertBatch é idempotente: registros cuja (agent_id, idempotency_key)
	// já existir são ignorados silenciosamente (reenvio seguro após
	// reconexão do agente — Seção 3).
	InsertBatch(ctx context.Context, tests []PingTest) error
	// ListBySite retorna os mais recentes primeiro, entre todos os agentes
	// do site (join implícito via agents.site_id).
	ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]PingTest, int, error)
}

type SpeedTest struct {
	ID              uuid.UUID
	AgentID         uuid.UUID
	Mode            string
	DownloadMbps    *float64
	UploadMbps      *float64
	IdleLatencyMs   *float64
	LoadedLatencyMs *float64
	BufferbloatMs   *float64
	JitterMs        *float64
	ExecutedAt      time.Time
	IdempotencyKey  string
}

type SpeedTestStore interface {
	InsertBatch(ctx context.Context, tests []SpeedTest) error
	ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]SpeedTest, int, error)
}
