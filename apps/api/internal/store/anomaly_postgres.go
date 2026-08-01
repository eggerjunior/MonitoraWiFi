package store

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"context"

	"github.com/google/uuid"
)

type PostgresAnomalies struct{ Pool *pgxpool.Pool }

func (s *PostgresAnomalies) ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]Anomaly, int, error) {
	offset := (page.Page - 1) * page.PageSize

	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM anomalies WHERE site_id = $1`, siteID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT id, site_id, metric, observed_at, value, bucket_mean, bucket_size, z_score, detected_at
		 FROM anomalies WHERE site_id = $1 ORDER BY observed_at DESC LIMIT $2 OFFSET $3`,
		siteID, page.PageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Anomaly
	for rows.Next() {
		var a Anomaly
		if err := rows.Scan(&a.ID, &a.SiteID, &a.Metric, &a.ObservedAt, &a.Value, &a.BucketMean, &a.BucketSize, &a.ZScore, &a.DetectedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}
