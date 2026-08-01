package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAgentCommands struct{ Pool *pgxpool.Pool }

func (s *PostgresAgentCommands) Create(ctx context.Context, siteID uuid.UUID, requestedBy uuid.UUID, cmdType string, params []byte, now time.Time) (AgentCommand, error) {
	var agentID uuid.UUID
	err := s.Pool.QueryRow(ctx,
		`SELECT id FROM agents
		 WHERE site_id = $1 AND revoked_at IS NULL
		 ORDER BY last_seen_at DESC NULLS LAST, enrolled_at DESC
		 LIMIT 1`,
		siteID,
	).Scan(&agentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AgentCommand{}, ErrNoActiveAgent
	}
	if err != nil {
		return AgentCommand{}, err
	}

	cmd := AgentCommand{
		ID:          uuid.New(),
		SiteID:      siteID,
		AgentID:     agentID,
		RequestedBy: requestedBy,
		Type:        cmdType,
		Params:      params,
		Status:      AgentCommandStatusPending,
		CreatedAt:   now,
	}

	_, err = s.Pool.Exec(ctx,
		`INSERT INTO agent_commands (id, site_id, agent_id, requested_by, type, params, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		cmd.ID, cmd.SiteID, cmd.AgentID, cmd.RequestedBy, cmd.Type, cmd.Params, cmd.Status, cmd.CreatedAt)
	if err != nil {
		return AgentCommand{}, err
	}
	return cmd, nil
}

func (s *PostgresAgentCommands) Get(ctx context.Context, id uuid.UUID) (AgentCommand, error) {
	var c AgentCommand
	err := s.Pool.QueryRow(ctx,
		`SELECT id, site_id, agent_id, requested_by, type, params, status, result, error, created_at, claimed_at, completed_at
		 FROM agent_commands WHERE id = $1`, id,
	).Scan(&c.ID, &c.SiteID, &c.AgentID, &c.RequestedBy, &c.Type, &c.Params, &c.Status, &c.Result, &c.Error, &c.CreatedAt, &c.ClaimedAt, &c.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AgentCommand{}, ErrNotFound
	}
	return c, err
}

func (s *PostgresAgentCommands) ClaimPending(ctx context.Context, agentID uuid.UUID, limit int, now time.Time) ([]AgentCommand, error) {
	rows, err := s.Pool.Query(ctx,
		`UPDATE agent_commands
		 SET status = 'claimed', claimed_at = $3
		 WHERE id IN (
		     SELECT id FROM agent_commands
		     WHERE agent_id = $1 AND status = 'pending'
		     ORDER BY created_at
		     LIMIT $2
		     FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id, site_id, agent_id, requested_by, type, params, status, result, error, created_at, claimed_at, completed_at`,
		agentID, limit, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AgentCommand
	for rows.Next() {
		var c AgentCommand
		if err := rows.Scan(&c.ID, &c.SiteID, &c.AgentID, &c.RequestedBy, &c.Type, &c.Params, &c.Status, &c.Result, &c.Error, &c.CreatedAt, &c.ClaimedAt, &c.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *PostgresAgentCommands) Complete(ctx context.Context, id uuid.UUID, result []byte, at time.Time) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE agent_commands SET status = 'completed', result = $2, completed_at = $3 WHERE id = $1`,
		id, result, at)
	return err
}

func (s *PostgresAgentCommands) Fail(ctx context.Context, id uuid.UUID, errMsg string, at time.Time) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE agent_commands SET status = 'failed', error = $2, completed_at = $3 WHERE id = $1`,
		id, errMsg, at)
	return err
}

func (s *PostgresAgentCommands) ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]AgentCommand, int, error) {
	offset := (page.Page - 1) * page.PageSize

	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM agent_commands WHERE site_id = $1`, siteID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT id, site_id, agent_id, requested_by, type, params, status, result, error, created_at, claimed_at, completed_at
		 FROM agent_commands WHERE site_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		siteID, page.PageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []AgentCommand
	for rows.Next() {
		var c AgentCommand
		if err := rows.Scan(&c.ID, &c.SiteID, &c.AgentID, &c.RequestedBy, &c.Type, &c.Params, &c.Status, &c.Result, &c.Error, &c.CreatedAt, &c.ClaimedAt, &c.CompletedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}
