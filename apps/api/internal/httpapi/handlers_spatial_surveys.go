// Levantamento espacial (Fase 6, "Spatial WiFi Survey"): o app iOS captura
// posição real (ARKit world tracking, só em hardware com LiDAR) e qualidade
// de rede medida no próprio ponto (RTT ao backend, Wi-Fi/celular via
// NWPathMonitor) durante uma caminhada guiada, e envia o levantamento
// completo de uma vez ao final — não há sessão incremental no backend.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"egger/api/internal/store"
)

var supportedInterfaceTypes = map[string]bool{
	"wifi": true, "cellular": true, "wired": true, "other": true,
}

type createSpatialSurveySampleRequest struct {
	PositionX     float64  `json:"position_x"`
	PositionY     float64  `json:"position_y"`
	PositionZ     float64  `json:"position_z"`
	SSID          *string  `json:"ssid"`
	BSSID         *string  `json:"bssid"`
	RTTMs         *float64 `json:"rtt_ms"`
	IsExpensive   bool     `json:"is_expensive"`
	IsConstrained bool     `json:"is_constrained"`
	InterfaceType string   `json:"interface_type"`
	CapturedAt    string   `json:"captured_at"`
}

type createSpatialSurveyRequest struct {
	Name        string                             `json:"name"`
	DeviceModel string                             `json:"device_model"`
	LiDARUsed   bool                               `json:"lidar_used"`
	StartedAt   string                             `json:"started_at"`
	FinishedAt  string                             `json:"finished_at"`
	Samples     []createSpatialSurveySampleRequest `json:"samples"`
}

// handleCreateSpatialSurvey valida e persiste um levantamento completo — o
// mesmo cuidado de nunca aceitar dado que o produto não sabe representar
// (Seção 2.1) já aplicado aos comandos sob demanda (handlers_commands.go).
func (s *Server) handleCreateSpatialSurvey(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())
	user, _ := userFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	var req createSpatialSurveyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "corpo da requisição inválido")
		return
	}
	if req.Name == "" {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "name é obrigatório")
		return
	}
	if req.DeviceModel == "" {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "device_model é obrigatório")
		return
	}
	startedAt, err := time.Parse(time.RFC3339Nano, req.StartedAt)
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "started_at precisa ser RFC3339")
		return
	}
	finishedAt, err := time.Parse(time.RFC3339Nano, req.FinishedAt)
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "finished_at precisa ser RFC3339")
		return
	}
	if len(req.Samples) == 0 {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "samples precisa ter ao menos 1 amostra")
		return
	}

	samples := make([]store.SpatialSurveySample, 0, len(req.Samples))
	for _, sr := range req.Samples {
		if !supportedInterfaceTypes[sr.InterfaceType] {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "samples[].interface_type inválido (wifi, cellular, wired ou other)")
			return
		}
		capturedAt, err := time.Parse(time.RFC3339Nano, sr.CapturedAt)
		if err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "samples[].captured_at precisa ser RFC3339")
			return
		}
		samples = append(samples, store.SpatialSurveySample{
			ID:            uuid.New(),
			PositionX:     sr.PositionX,
			PositionY:     sr.PositionY,
			PositionZ:     sr.PositionZ,
			SSID:          sr.SSID,
			BSSID:         sr.BSSID,
			RTTMs:         sr.RTTMs,
			IsExpensive:   sr.IsExpensive,
			IsConstrained: sr.IsConstrained,
			InterfaceType: sr.InterfaceType,
			CapturedAt:    capturedAt,
		})
	}

	survey := store.SpatialSurvey{
		ID:          uuid.New(),
		SiteID:      siteID,
		CreatedBy:   user.ID,
		Name:        req.Name,
		DeviceModel: req.DeviceModel,
		LiDARUsed:   req.LiDARUsed,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		Samples:     samples,
		SampleCount: len(samples),
	}

	if err := s.spatialSurveys.Create(r.Context(), survey); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao gravar levantamento")
		return
	}

	writeJSON(w, http.StatusCreated, spatialSurveyToJSON(survey, false))
}

func (s *Server) handleListSpatialSurveys(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	page := parsePage(r)
	surveys, total, err := s.spatialSurveys.ListBySite(r.Context(), siteID, page)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar levantamentos")
		return
	}

	items := make([]map[string]any, 0, len(surveys))
	for _, sv := range surveys {
		items = append(items, spatialSurveyToJSON(sv, false))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     total,
	})
}

func (s *Server) handleGetSpatialSurvey(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	id, err := uuid.Parse(r.PathValue("surveyId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_survey_id", "surveyId inválido")
		return
	}

	survey, err := s.spatialSurveys.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, correlationID, http.StatusNotFound, "not_found", "levantamento não encontrado")
			return
		}
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao buscar levantamento")
		return
	}

	writeJSON(w, http.StatusOK, spatialSurveyToJSON(survey, true))
}

func spatialSurveyToJSON(sv store.SpatialSurvey, includeSamples bool) map[string]any {
	out := map[string]any{
		"id":           sv.ID.String(),
		"site_id":      sv.SiteID.String(),
		"created_by":   sv.CreatedBy.String(),
		"name":         sv.Name,
		"device_model": sv.DeviceModel,
		"lidar_used":   sv.LiDARUsed,
		"started_at":   sv.StartedAt.Format(time.RFC3339),
		"finished_at":  sv.FinishedAt.Format(time.RFC3339),
		"sample_count": sv.SampleCount,
	}
	if !sv.CreatedAt.IsZero() {
		out["created_at"] = sv.CreatedAt.Format(time.RFC3339)
	}
	if includeSamples {
		samples := make([]map[string]any, 0, len(sv.Samples))
		for _, sample := range sv.Samples {
			samples = append(samples, map[string]any{
				"id":             sample.ID.String(),
				"position_x":     sample.PositionX,
				"position_y":     sample.PositionY,
				"position_z":     sample.PositionZ,
				"ssid":           sample.SSID,
				"bssid":          sample.BSSID,
				"rtt_ms":         sample.RTTMs,
				"is_expensive":   sample.IsExpensive,
				"is_constrained": sample.IsConstrained,
				"interface_type": sample.InterfaceType,
				"captured_at":    sample.CapturedAt.Format(time.RFC3339),
			})
		}
		out["samples"] = samples
	}
	return out
}
