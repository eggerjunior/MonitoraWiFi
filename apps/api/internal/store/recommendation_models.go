package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Recommendation é a ação sugerida a partir de um Diagnosis (Fase 7) — uma
// por diagnóstico nesta fase (idx_recommendations_one_per_diagnosis),
// sempre amarrada a um DiagnosisID real.
type Recommendation struct {
	ID          uuid.UUID
	DiagnosisID uuid.UUID
	SiteID      uuid.UUID
	Action      string
	Confidence  float64
	Impact      string
	Risk        string
	Evidence    json.RawMessage
	CreatedAt   time.Time
}

type RecommendationStore interface {
	ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]Recommendation, int, error)
}
