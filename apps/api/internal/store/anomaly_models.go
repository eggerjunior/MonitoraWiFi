package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Anomaly é uma amostra que desviou do baseline estatístico do site (Fase 7,
// worker/internal/baseline) — nunca gerada sem baseline histórico
// suficiente (ver worker.MinBucketSamples).
type Anomaly struct {
	ID         uuid.UUID
	SiteID     uuid.UUID
	Metric     string
	ObservedAt time.Time
	Value      float64
	BucketMean float64
	BucketSize int
	ZScore     float64
	DetectedAt time.Time
}

type AnomalyStore interface {
	ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]Anomaly, int, error)
}
