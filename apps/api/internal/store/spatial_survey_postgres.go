package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSpatialSurveys struct{ Pool *pgxpool.Pool }

// Create grava o levantamento e todas as amostras numa transação — reverte
// tudo se qualquer amostra falhar, nunca deixa um levantamento truncado.
func (s *PostgresSpatialSurveys) Create(ctx context.Context, survey SpatialSurvey) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO spatial_surveys (id, site_id, created_by, name, device_model, lidar_used, started_at, finished_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		survey.ID, survey.SiteID, survey.CreatedBy, survey.Name, survey.DeviceModel, survey.LiDARUsed, survey.StartedAt, survey.FinishedAt)
	if err != nil {
		return err
	}

	batch := &pgx.Batch{}
	for _, sample := range survey.Samples {
		batch.Queue(
			`INSERT INTO spatial_survey_samples (id, survey_id, position_x, position_y, position_z, ssid, bssid, rtt_ms, is_expensive, is_constrained, interface_type, captured_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			sample.ID, survey.ID, sample.PositionX, sample.PositionY, sample.PositionZ, sample.SSID, sample.BSSID, sample.RTTMs, sample.IsExpensive, sample.IsConstrained, sample.InterfaceType, sample.CapturedAt)
	}
	if len(survey.Samples) > 0 {
		br := tx.SendBatch(ctx, batch)
		for range survey.Samples {
			if _, err := br.Exec(); err != nil {
				br.Close()
				return err
			}
		}
		if err := br.Close(); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *PostgresSpatialSurveys) Get(ctx context.Context, id uuid.UUID) (SpatialSurvey, error) {
	var survey SpatialSurvey
	err := s.Pool.QueryRow(ctx,
		`SELECT id, site_id, created_by, name, device_model, lidar_used, started_at, finished_at, created_at
		 FROM spatial_surveys WHERE id = $1`, id,
	).Scan(&survey.ID, &survey.SiteID, &survey.CreatedBy, &survey.Name, &survey.DeviceModel, &survey.LiDARUsed, &survey.StartedAt, &survey.FinishedAt, &survey.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SpatialSurvey{}, ErrNotFound
	}
	if err != nil {
		return SpatialSurvey{}, err
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT id, position_x, position_y, position_z, ssid, bssid, rtt_ms, is_expensive, is_constrained, interface_type, captured_at
		 FROM spatial_survey_samples WHERE survey_id = $1 ORDER BY captured_at ASC`, id)
	if err != nil {
		return SpatialSurvey{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var sample SpatialSurveySample
		sample.SurveyID = id
		if err := rows.Scan(&sample.ID, &sample.PositionX, &sample.PositionY, &sample.PositionZ, &sample.SSID, &sample.BSSID, &sample.RTTMs, &sample.IsExpensive, &sample.IsConstrained, &sample.InterfaceType, &sample.CapturedAt); err != nil {
			return SpatialSurvey{}, err
		}
		survey.Samples = append(survey.Samples, sample)
	}
	if err := rows.Err(); err != nil {
		return SpatialSurvey{}, err
	}

	return survey, nil
}

func (s *PostgresSpatialSurveys) ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]SpatialSurvey, int, error) {
	offset := (page.Page - 1) * page.PageSize

	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM spatial_surveys WHERE site_id = $1`, siteID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT id, site_id, created_by, name, device_model, lidar_used, started_at, finished_at, created_at
		 FROM spatial_surveys WHERE site_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		siteID, page.PageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []SpatialSurvey
	for rows.Next() {
		var s SpatialSurvey
		if err := rows.Scan(&s.ID, &s.SiteID, &s.CreatedBy, &s.Name, &s.DeviceModel, &s.LiDARUsed, &s.StartedAt, &s.FinishedAt, &s.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}
