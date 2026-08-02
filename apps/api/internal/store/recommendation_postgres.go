package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRecommendations struct{ Pool *pgxpool.Pool }

func (s *PostgresRecommendations) ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]Recommendation, int, error) {
	offset := (page.Page - 1) * page.PageSize

	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM recommendations WHERE site_id = $1`, siteID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT id, diagnosis_id, site_id, action, confidence, impact, risk, evidence, created_at
		 FROM recommendations WHERE site_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		siteID, page.PageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Recommendation
	for rows.Next() {
		var r Recommendation
		if err := rows.Scan(&r.ID, &r.DiagnosisID, &r.SiteID, &r.Action, &r.Confidence, &r.Impact, &r.Risk, &r.Evidence, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}
