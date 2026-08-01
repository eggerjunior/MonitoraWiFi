// Inventário UniFi (Fase 3, início — ADR-007). O backend nunca fala
// diretamente com o console UniFi (ADR-001) — só recebe o snapshot que o
// agente já coletou via NetworkAPIAdapter.
package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"egger/api/internal/store"
)

type unifiDevicePayload struct {
	ExternalID      string   `json:"external_id"`
	MACAddress      string   `json:"mac_address"`
	IPAddress       string   `json:"ip_address"`
	Name            string   `json:"name"`
	Model           string   `json:"model"`
	State           string   `json:"state"`
	FirmwareVersion string   `json:"firmware_version"`
	Features        []string `json:"features"`
	Interfaces      []string `json:"interfaces"`
}

type unifiClientPayload struct {
	ExternalID     string `json:"external_id"`
	Type           string `json:"type"`
	Name           string `json:"name"`
	IPAddress      string `json:"ip_address"`
	MACAddress     string `json:"mac_address"`
	ConnectedAt    string `json:"connected_at"`
	UplinkDeviceID string `json:"uplink_device_id"`
}

type unifiInventoryRequest struct {
	Devices []unifiDevicePayload `json:"devices"`
	Clients []unifiClientPayload `json:"clients"`
}

func (s *Server) handleUniFiInventory(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())
	agent, _ := agentFromContext(r.Context())

	var req unifiInventoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "corpo da requisição inválido")
		return
	}

	devices := make([]store.UniFiDevice, 0, len(req.Devices))
	for _, d := range req.Devices {
		devices = append(devices, store.UniFiDevice{
			SiteID:          agent.SiteID,
			ExternalID:      d.ExternalID,
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

	clients := make([]store.UniFiClient, 0, len(req.Clients))
	for _, c := range req.Clients {
		var connectedAt *time.Time
		if t, err := time.Parse(time.RFC3339, c.ConnectedAt); err == nil && !t.IsZero() {
			connectedAt = &t
		}
		clients = append(clients, store.UniFiClient{
			SiteID:         agent.SiteID,
			ExternalID:     c.ExternalID,
			Type:           c.Type,
			Name:           c.Name,
			IPAddress:      c.IPAddress,
			MACAddress:     c.MACAddress,
			ConnectedAt:    connectedAt,
			UplinkDeviceID: c.UplinkDeviceID,
		})
	}

	if err := s.unifiDevices.ReplaceBySite(r.Context(), agent.SiteID, devices); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao persistir dispositivos UniFi")
		return
	}
	if err := s.unifiClients.ReplaceBySite(r.Context(), agent.SiteID, clients); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao persistir clientes UniFi")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"devices": len(devices), "clients": len(clients)})
}

func (s *Server) handleListUniFiDevices(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	devices, err := s.unifiDevices.ListBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar dispositivos UniFi")
		return
	}

	items := make([]map[string]any, 0, len(devices))
	for _, d := range devices {
		items = append(items, map[string]any{
			"id":               d.ID.String(),
			"external_id":      d.ExternalID,
			"mac_address":      d.MACAddress,
			"ip_address":       d.IPAddress,
			"name":             d.Name,
			"model":            d.Model,
			"state":            d.State,
			"firmware_version": d.FirmwareVersion,
			"features":         d.Features,
			"interfaces":       d.Interfaces,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListUniFiClients(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	clients, err := s.unifiClients.ListBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar clientes UniFi")
		return
	}

	items := make([]map[string]any, 0, len(clients))
	for _, c := range clients {
		var connectedAt *string
		if c.ConnectedAt != nil {
			s := c.ConnectedAt.Format(time.RFC3339)
			connectedAt = &s
		}
		items = append(items, map[string]any{
			"id":               c.ID.String(),
			"external_id":      c.ExternalID,
			"type":             c.Type,
			"name":             c.Name,
			"ip_address":       c.IPAddress,
			"mac_address":      c.MACAddress,
			"connected_at":     connectedAt,
			"uplink_device_id": c.UplinkDeviceID,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
