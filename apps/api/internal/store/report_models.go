package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Report é um relatório de diagnóstico gerado sob demanda (Fase 7,
// docs/architecture/05-modelo-dados.md §7) — conteúdo computado a partir de
// diagnoses/recommendations/anomalies reais do período e gravado inteiro em
// `content` (não há armazenamento de objetos configurado neste projeto
// ainda; nada aqui é pré-gerado/assíncrono).
type Report struct {
	ID          uuid.UUID
	SiteID      uuid.UUID
	Kind        string
	PeriodStart time.Time
	PeriodEnd   time.Time
	// Content só é populado por Create e Get — ListBySite devolve só os
	// metadados (mesmo padrão de SpatialSurvey.Samples: uma lista não
	// precisa carregar o corpo inteiro de cada item).
	Content     json.RawMessage
	GeneratedBy *uuid.UUID
	GeneratedAt time.Time
}

type ReportStore interface {
	Create(ctx context.Context, r Report) (Report, error)
	Get(ctx context.Context, id uuid.UUID) (Report, error)
	ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]Report, int, error)
}
