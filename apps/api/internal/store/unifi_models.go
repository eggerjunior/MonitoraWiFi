package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UniFiDevice/UniFiClient espelham só os campos confirmados reais na
// Network API local (Fase 3, início — docs/unifi/verificacoes-pendentes-instalacao.md).
// Estado atual (inventário), não série temporal — sincronizado inteiro a
// cada ciclo do agente.
type UniFiDevice struct {
	ID              uuid.UUID
	SiteID          uuid.UUID
	ExternalID      string
	MACAddress      string
	IPAddress       string
	Name            string
	Model           string
	State           string
	FirmwareVersion string
	Features        []string
	Interfaces      []string
	// UplinkDeviceID é o external_id do dispositivo upstream (ex.: o
	// switch a que um AP está conectado) — confirmado em 2026-08-02 via
	// GET .../devices/{id} real. Vazio para o dispositivo raiz (gateway).
	UplinkDeviceID string
}

type UniFiClient struct {
	ID             uuid.UUID
	SiteID         uuid.UUID
	ExternalID     string
	Type           string
	Name           string
	IPAddress      string
	MACAddress     string
	ConnectedAt    *time.Time // nullable — cliente pode não reportar
	UplinkDeviceID string
}

type UniFiDeviceStore interface {
	// ReplaceBySite substitui todo o inventário de dispositivos daquele
	// site pelo snapshot recebido — nunca acumula histórico (é estado
	// atual, não série temporal).
	ReplaceBySite(ctx context.Context, siteID uuid.UUID, devices []UniFiDevice) error
	ListBySite(ctx context.Context, siteID uuid.UUID) ([]UniFiDevice, error)
}

type UniFiClientStore interface {
	ReplaceBySite(ctx context.Context, siteID uuid.UUID, clients []UniFiClient) error
	ListBySite(ctx context.Context, siteID uuid.UUID) ([]UniFiClient, error)
}
