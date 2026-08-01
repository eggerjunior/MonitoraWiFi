// Integração UniFi (Fase 3, início — ADR-007): sincroniza o inventário de
// dispositivos/clientes do site periodicamente. Não implementa detalhe de
// rádio/porta, eventos/alarmes, DPI nem topologia dispositivo→dispositivo
// ainda — esses campos continuam "a validar" em
// docs/unifi/capability-matrix.md até serem confirmados contra a instalação
// real (nunca inventados).
package agent

import (
	"context"
	"log/slog"
	"time"

	"egger/local-agent/internal/apiclient"
)

func (a *Agent) unifiLoop(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.UniFiSyncInterval)
	defer ticker.Stop()

	a.syncUniFiOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.syncUniFiOnce(ctx)
		}
	}
}

func (a *Agent) syncUniFiOnce(ctx context.Context) {
	devices, err := a.unifiProvider.ListDevices(ctx, a.cfg.UniFiSiteID)
	if err != nil {
		a.logger.Warn("erro ao listar dispositivos UniFi", slog.Any("error", err))
		return
	}
	clients, err := a.unifiProvider.ListClients(ctx, a.cfg.UniFiSiteID)
	if err != nil {
		a.logger.Warn("erro ao listar clientes UniFi", slog.Any("error", err))
		return
	}

	devicePayloads := make([]apiclient.UniFiDevicePayload, 0, len(devices))
	for _, d := range devices {
		devicePayloads = append(devicePayloads, apiclient.UniFiDevicePayload{
			ExternalID:      d.ID,
			MACAddress:      d.MACAddress,
			IPAddress:       d.IPAddress,
			Name:            d.Name,
			Model:           d.Model,
			State:           d.State,
			FirmwareVersion: d.FirmwareVersion,
			Features:        d.Features,
			Interfaces:      d.Interfaces,
		})
	}

	clientPayloads := make([]apiclient.UniFiClientPayload, 0, len(clients))
	for _, c := range clients {
		clientPayloads = append(clientPayloads, apiclient.UniFiClientPayload{
			ExternalID:     c.ID,
			Type:           c.Type,
			Name:           c.Name,
			IPAddress:      c.IPAddress,
			MACAddress:     c.MACAddress,
			ConnectedAt:    c.ConnectedAt.Format(time.RFC3339),
			UplinkDeviceID: c.UplinkDeviceID,
		})
	}

	if err := a.client.SendUniFiInventory(ctx, a.identity.AgentID, a.identity.AgentSecret, devicePayloads, clientPayloads); err != nil {
		a.logger.Error("erro ao enviar inventário UniFi ao backend", slog.Any("error", err))
		return
	}
	a.logger.Info("inventário UniFi sincronizado",
		slog.Int("devices", len(devicePayloads)),
		slog.Int("clients", len(clientPayloads)))
}
