// Package unifi implementa a integração UniFi (Fase 3, início) atrás da
// interface UniFiIntegrationProvider (ADR-007) — o resto do sistema nunca
// chama um endpoint específico da Network API diretamente; sempre passa por
// aqui. Só a Network API local está implementada nesta fase (NetworkAPIAdapter);
// Site Manager/SNMP/Syslog/adaptador legado ficam para quando houver decisão
// e necessidade real (ver docs/unifi/capability-matrix.md).
//
// Roda dentro do agente, nunca no backend (ADR-001: o backend não está na
// LAN do cliente).
package unifi

import (
	"context"
	"time"
)

// Site espelha só os campos confirmados na resposta real da Network API
// local (docs/unifi/verificacoes-pendentes-instalacao.md) — nunca inventa
// campos não observados.
type Site struct {
	ID                string
	InternalReference string
	Name              string
}

type Device struct {
	ID                string
	MACAddress        string
	IPAddress         string
	Name              string
	Model             string
	State             string // ONLINE | OFFLINE, conforme observado
	FirmwareVersion   string
	FirmwareUpdatable bool
	Features          []string // ex.: "switching", "accessPoint"
	Interfaces        []string // ex.: "ports", "radios"
}

type Client struct {
	ID             string
	Type           string // WIRED | WIRELESS
	Name           string
	IPAddress      string
	MACAddress     string
	ConnectedAt    time.Time
	UplinkDeviceID string
}

// UniFiIntegrationProvider é a única porta de entrada para dado UniFi no
// resto do agente — ADR-007. Um adaptador por fonte implementa esta
// interface; hoje só existe NetworkAPIAdapter.
type UniFiIntegrationProvider interface {
	ListSites(ctx context.Context) ([]Site, error)
	ListDevices(ctx context.Context, siteID string) ([]Device, error)
	ListClients(ctx context.Context, siteID string) ([]Client, error)
}
