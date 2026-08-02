package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNoActiveAgent é retornado quando um comando sob demanda é solicitado
// para um site sem nenhum agente ativo (nunca revogado) — não há para quem
// endereçar o comando.
var ErrNoActiveAgent = errors.New("nenhum agente ativo neste site")

// AgentCommand é um teste sob demanda disparado pelo usuário e executado
// pelo agente (docs/architecture/03-fluxo-de-dados.md §3.2). A fila é
// implementada em Postgres (polling), não Redis — ver comentário na
// migração 0004_agent_commands.
type AgentCommand struct {
	ID          uuid.UUID
	SiteID      uuid.UUID
	AgentID     uuid.UUID
	RequestedBy uuid.UUID
	Type        string
	Params      []byte // JSON bruto — cada tipo de comando define seu próprio formato de params
	Status      string // pending | claimed | completed | failed
	Result      []byte // JSON bruto, nil até completed
	Error       *string
	CreatedAt   time.Time
	ClaimedAt   *time.Time
	CompletedAt *time.Time
}

const (
	AgentCommandStatusPending   = "pending"
	AgentCommandStatusClaimed   = "claimed"
	AgentCommandStatusCompleted = "completed"
	AgentCommandStatusFailed    = "failed"
)

const (
	AgentCommandTypePing               = "ping"
	AgentCommandTypeDNSLookup          = "dns_lookup"
	AgentCommandTypeTraceroute         = "traceroute"
	AgentCommandTypeBatchPing          = "batch_ping"
	AgentCommandTypeSSLCheck           = "ssl_check"
	AgentCommandTypeHTTPRequest        = "http_request"
	AgentCommandTypeLANScan            = "lan_scan"
	AgentCommandTypeWakeOnLAN          = "wake_on_lan"
	AgentCommandTypePortScan           = "port_scan"
	AgentCommandTypeDNSResolverCompare = "dns_resolver_compare"
)

type AgentCommandStore interface {
	// Create resolve o agente ativo do site (mais recente por last_seen_at,
	// nunca revogado) e insere o comando como "pending". Retorna
	// ErrNoActiveAgent se o site não tiver nenhum agente ativo.
	Create(ctx context.Context, siteID uuid.UUID, requestedBy uuid.UUID, cmdType string, params []byte, now time.Time) (AgentCommand, error)
	Get(ctx context.Context, id uuid.UUID) (AgentCommand, error)
	// ClaimPending marca até `limit` comandos pendentes do agente como
	// "claimed" atomicamente (SKIP LOCKED — seguro contra o agente
	// reconectando/repetindo o poll concorrentemente) e os retorna.
	ClaimPending(ctx context.Context, agentID uuid.UUID, limit int, now time.Time) ([]AgentCommand, error)
	Complete(ctx context.Context, id uuid.UUID, result []byte, at time.Time) error
	Fail(ctx context.Context, id uuid.UUID, errMsg string, at time.Time) error
	ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]AgentCommand, int, error)
}
