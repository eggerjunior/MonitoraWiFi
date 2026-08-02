package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDiagnoses struct{ Pool *pgxpool.Pool }

func (s *PostgresDiagnoses) ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]Diagnosis, int, error) {
	offset := (page.Page - 1) * page.PageSize

	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM diagnoses WHERE site_id = $1`, siteID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT id, site_id, category, summary, confidence, impact, risk, evidence, window_start, window_end, detected_at
		 FROM diagnoses WHERE site_id = $1 ORDER BY window_end DESC LIMIT $2 OFFSET $3`,
		siteID, page.PageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Diagnosis
	for rows.Next() {
		var d Diagnosis
		if err := rows.Scan(&d.ID, &d.SiteID, &d.Category, &d.Summary, &d.Confidence, &d.Impact, &d.Risk, &d.Evidence, &d.WindowStart, &d.WindowEnd, &d.DetectedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}
