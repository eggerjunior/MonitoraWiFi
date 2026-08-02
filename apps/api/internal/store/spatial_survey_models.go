package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SpatialSurveySample é uma amostra capturada durante um levantamento
// espacial (Fase 6) — posição real do ARKit (world tracking) + qualidade de
// rede medida no próprio ponto (RTT ao backend via Network.framework,
// Wi-Fi/celular/expensive/constrained via NWPathMonitor). Deliberadamente
// não inclui RSSI/canal/PHY rate: nenhum desses campos é obtido pelo iPhone
// (ver capability matrix) — nunca inventar dado que a plataforma não expõe.
type SpatialSurveySample struct {
	ID            uuid.UUID
	SurveyID      uuid.UUID
	PositionX     float64
	PositionY     float64
	PositionZ     float64
	SSID          *string
	BSSID         *string
	RTTMs         *float64
	IsExpensive   bool
	IsConstrained bool
	InterfaceType string
	CapturedAt    time.Time
}

// SpatialSurvey é um levantamento completo — o app iOS envia o levantamento
// inteiro (metadados + todas as amostras) de uma vez, ao final da captura
// guiada, não amostra por amostra.
type SpatialSurvey struct {
	ID          uuid.UUID
	SiteID      uuid.UUID
	CreatedBy   uuid.UUID
	Name        string
	DeviceModel string
	LiDARUsed   bool
	StartedAt   time.Time
	FinishedAt  time.Time
	CreatedAt   time.Time
	// SampleCount é sempre populado (list e detail), independente de Samples
	// estar carregado — ListBySite nunca carrega Samples (custaria caro pra
	// uma tela de lista), então nunca derivar a contagem de len(Samples).
	SampleCount int
	Samples     []SpatialSurveySample
}

type SpatialSurveyStore interface {
	// Create persiste o levantamento e todas as amostras numa única
	// transação — nunca fica um levantamento "pela metade" se a gravação
	// falhar no meio.
	Create(ctx context.Context, s SpatialSurvey) error
	Get(ctx context.Context, id uuid.UUID) (SpatialSurvey, error)
	// ListBySite não inclui as amostras (só metadados) — evita transferir
	// milhares de pontos pra uma tela de lista; GetById carrega as amostras.
	ListBySite(ctx context.Context, siteID uuid.UUID, page Page) ([]SpatialSurvey, int, error)
}
