// Anomalias estatísticas (Fase 7, início) — calculadas pelo worker
// (apps/worker), só lidas aqui.
package httpapi

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (s *Server) handleListAnomalies(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	page := parsePage(r)
	anomalies, total, err := s.anomalies.ListBySite(r.Context(), siteID, page)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar anomalias")
		return
	}

	items := make([]map[string]any, 0, len(anomalies))
	for _, a := range anomalies {
		items = append(items, map[string]any{
			"id":          a.ID.String(),
			"metric":      a.Metric,
			"observed_at": a.ObservedAt.Format(time.RFC3339),
			"value":       a.Value,
			"bucket_mean": a.BucketMean,
			"bucket_size": a.BucketSize,
			"z_score":     a.ZScore,
			"detected_at": a.DetectedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     total,
	})
}
