package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresOrganizations struct{ Pool *pgxpool.Pool }
type PostgresSites struct{ Pool *pgxpool.Pool }
type PostgresUsers struct{ Pool *pgxpool.Pool }
type PostgresSessions struct{ Pool *pgxpool.Pool }
type PostgresAudit struct{ Pool *pgxpool.Pool }

func (s *PostgresOrganizations) List(ctx context.Context, page Page) ([]Organization, int, error) {
	offset := (page.Page - 1) * page.PageSize

	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM organizations`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT id, name, plan_tier, created_at FROM organizations ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		page.PageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.PlanTier, &o.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (s *PostgresSites) ListByOrganization(ctx context.Context, orgID uuid.UUID, page Page) ([]Site, int, error) {
	offset := (page.Page - 1) * page.PageSize

	var total int
	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM sites WHERE organization_id = $1`, orgID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT id, organization_id, name, timezone, created_at FROM sites
		 WHERE organization_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		orgID, page.PageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Site
	for rows.Next() {
		var site Site
		if err := rows.Scan(&site.ID, &site.OrganizationID, &site.Name, &site.Timezone, &site.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, site)
	}
	return out, total, rows.Err()
}

func (s *PostgresSites) Get(ctx context.Context, id uuid.UUID) (Site, error) {
	var site Site
	err := s.Pool.QueryRow(ctx,
		`SELECT id, organization_id, name, timezone, created_at FROM sites WHERE id = $1`, id,
	).Scan(&site.ID, &site.OrganizationID, &site.Name, &site.Timezone, &site.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	return site, err
}

func (s *PostgresUsers) GetByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.Pool.QueryRow(ctx,
		`SELECT id, organization_id, email, password_hash, role, mfa_enrolled_at, created_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.OrganizationID, &u.Email, &u.PasswordHash, &u.Role, &u.MFAEnrolledAt, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *PostgresUsers) Get(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := s.Pool.QueryRow(ctx,
		`SELECT id, organization_id, email, password_hash, role, mfa_enrolled_at, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.OrganizationID, &u.Email, &u.PasswordHash, &u.Role, &u.MFAEnrolledAt, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *PostgresSessions) Create(ctx context.Context, sess Session) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, token_hash, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		sess.ID, sess.UserID, sess.TokenHash, sess.CreatedAt, sess.ExpiresAt)
	return err
}

func (s *PostgresSessions) GetByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	var sess Session
	err := s.Pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, created_at, expires_at, revoked_at
		 FROM sessions WHERE token_hash = $1`, tokenHash,
	).Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &sess.CreatedAt, &sess.ExpiresAt, &sess.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	return sess, err
}

func (s *PostgresSessions) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `UPDATE sessions SET revoked_at = $2 WHERE id = $1`, id, time.Now().UTC())
	return err
}

func (s *PostgresAudit) Record(ctx context.Context, e AuditEntry) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO audit_log (id, organization_id, actor_user_id, action, resource_type, resource_id, diff, actor_ip, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		e.ID, e.OrganizationID, e.ActorUserID, e.Action, e.ResourceType, e.ResourceID, e.Diff, e.ActorIP, e.CreatedAt)
	return err
}
