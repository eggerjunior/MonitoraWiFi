package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Diagnosis é uma conclusão do motor de correlação (Fase 7,
// apps/worker/internal/diagnostics) — só gravada pelo worker quando há
// evidência real (anomalias) suficiente; a API aqui só lê.
type Diagnosis struct {
	ID          uuid.UUID
	SiteID      uuid.UUID
	Category    string
	Summary     string
	Confidence  float64
	Impact      string
	Risk        string
	Evidence    json.RawMessage
	WindowStart time.Time
	WindowEnd   time.Time
	DetectedAt  time.Time
}

type DiagnosisStore interface {
	ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]Diagnosis, int, error)
}
