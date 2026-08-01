package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAgents struct{ Pool *pgxpool.Pool }
type PostgresAgentEnrollmentTokens struct{ Pool *pgxpool.Pool }
type PostgresAgentHeartbeats struct{ Pool *pgxpool.Pool }
type PostgresPingTests struct{ Pool *pgxpool.Pool }

func (s *PostgresAgents) Create(ctx context.Context, a Agent) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO agents (id, site_id, hostname, version, platform, auth_method, secret_hash, enrolled_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		a.ID, a.SiteID, a.Hostname, a.Version, a.Platform, a.AuthMethod, a.SecretHash, a.EnrolledAt)
	return err
}

func (s *PostgresAgents) Get(ctx context.Context, id uuid.UUID) (Agent, error) {
	var a Agent
	err := s.Pool.QueryRow(ctx,
		`SELECT id, site_id, hostname, version, platform, auth_method, secret_hash, enrolled_at, last_seen_at, revoked_at
		 FROM agents WHERE id = $1`, id,
	).Scan(&a.ID, &a.SiteID, &a.Hostname, &a.Version, &a.Platform, &a.AuthMethod, &a.SecretHash, &a.EnrolledAt, &a.LastSeenAt, &a.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Agent{}, ErrNotFound
	}
	return a, err
}

func (s *PostgresAgents) ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]Agent, int, error) {
	offset := (page.Page - 1) * page.PageSize

	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM agents WHERE site_id = $1`, siteID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT id, site_id, hostname, version, platform, auth_method, secret_hash, enrolled_at, last_seen_at, revoked_at
		 FROM agents WHERE site_id = $1 ORDER BY enrolled_at DESC LIMIT $2 OFFSET $3`,
		siteID, page.PageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.SiteID, &a.Hostname, &a.Version, &a.Platform, &a.AuthMethod, &a.SecretHash, &a.EnrolledAt, &a.LastSeenAt, &a.RevokedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (s *PostgresAgents) UpdateLastSeen(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := s.Pool.Exec(ctx, `UPDATE agents SET last_seen_at = $2 WHERE id = $1`, id, at)
	return err
}

func (s *PostgresAgentEnrollmentTokens) Create(ctx context.Context, t AgentEnrollmentToken) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO agent_enrollment_tokens (id, site_id, token_hash, created_by, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.SiteID, t.TokenHash, t.CreatedBy, t.CreatedAt, t.ExpiresAt)
	return err
}

func (s *PostgresAgentEnrollmentTokens) GetValidByTokenHash(ctx context.Context, tokenHash string, now time.Time) (AgentEnrollmentToken, error) {
	var t AgentEnrollmentToken
	err := s.Pool.QueryRow(ctx,
		`SELECT id, site_id, token_hash, created_by, created_at, expires_at, used_at, used_by_agent_id
		 FROM agent_enrollment_tokens
		 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2`,
		tokenHash, now,
	).Scan(&t.ID, &t.SiteID, &t.TokenHash, &t.CreatedBy, &t.CreatedAt, &t.ExpiresAt, &t.UsedAt, &t.UsedByAgentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AgentEnrollmentToken{}, ErrTokenExpiredOrUsed
	}
	return t, err
}

func (s *PostgresAgentEnrollmentTokens) MarkUsed(ctx context.Context, id uuid.UUID, agentID uuid.UUID, at time.Time) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE agent_enrollment_tokens SET used_at = $2, used_by_agent_id = $3 WHERE id = $1`,
		id, at, agentID)
	return err
}

func (s *PostgresAgentHeartbeats) Record(ctx context.Context, h AgentHeartbeat) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO agent_heartbeats (time, agent_id, status, queued_items, cpu_pct, mem_pct)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		h.Time, h.AgentID, h.Status, h.QueuedItems, h.CPUPct, h.MemPct)
	return err
}

func (s *PostgresPingTests) InsertBatch(ctx context.Context, tests []PingTest) error {
	if len(tests) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, t := range tests {
		batch.Queue(
			`INSERT INTO ping_tests (id, agent_id, target, protocol, latency_ms_p50, latency_ms_p95, latency_ms_p99, jitter_ms, packet_loss_pct, executed_at, idempotency_key)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			 ON CONFLICT (agent_id, idempotency_key) DO NOTHING`,
			t.ID, t.AgentID, t.Target, t.Protocol, t.LatencyMsP50, t.LatencyMsP95, t.LatencyMsP99, t.JitterMs, t.PacketLossPct, t.ExecutedAt, t.IdempotencyKey)
	}
	br := s.Pool.SendBatch(ctx, batch)
	defer br.Close()
	for range tests {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}
