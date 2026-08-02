package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresReports struct{ Pool *pgxpool.Pool }

func (s *PostgresReports) Create(ctx context.Context, r Report) (Report, error) {
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO reports (site_id, kind, period_start, period_end, content, generated_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, generated_at`,
		r.SiteID, r.Kind, r.PeriodStart, r.PeriodEnd, r.Content, r.GeneratedBy,
	).Scan(&r.ID, &r.GeneratedAt)
	if err != nil {
		return Report{}, err
	}
	return r, nil
}

func (s *PostgresReports) Get(ctx context.Context, id uuid.UUID) (Report, error) {
	var r Report
	err := s.Pool.QueryRow(ctx,
		`SELECT id, site_id, kind, period_start, period_end, content, generated_by, generated_at
		 FROM reports WHERE id = $1`, id,
	).Scan(&r.ID, &r.SiteID, &r.Kind, &r.PeriodStart, &r.PeriodEnd, &r.Content, &r.GeneratedBy, &r.GeneratedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Report{}, ErrNotFound
	}
	if err != nil {
		return Report{}, err
	}
	return r, nil
}

func (s *PostgresReports) ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]Report, int, error) {
	offset := (page.Page - 1) * page.PageSize

	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM reports WHERE site_id = $1`, siteID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT id, site_id, kind, period_start, period_end, generated_by, generated_at
		 FROM reports WHERE site_id = $1 ORDER BY generated_at DESC LIMIT $2 OFFSET $3`,
		siteID, page.PageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Report
	for rows.Next() {
		var r Report
		if err := rows.Scan(&r.ID, &r.SiteID, &r.Kind, &r.PeriodStart, &r.PeriodEnd, &r.GeneratedBy, &r.GeneratedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}
