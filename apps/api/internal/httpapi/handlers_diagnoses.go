// Diagnósticos (Fase 7, motor de correlação): calculados pelo worker
// (apps/worker/internal/diagnostics) a partir de anomalias reais, só lidos
// aqui.
package httpapi

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (s *Server) handleListDiagnoses(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	page := parsePage(r)
	diagnoses, total, err := s.diagnoses.ListBySite(r.Context(), siteID, page)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar diagnósticos")
		return
	}

	items := make([]map[string]any, 0, len(diagnoses))
	for _, d := range diagnoses {
		items = append(items, map[string]any{
			"id":           d.ID.String(),
			"category":     d.Category,
			"summary":      d.Summary,
			"confidence":   d.Confidence,
			"impact":       d.Impact,
			"risk":         d.Risk,
			"evidence":     d.Evidence,
			"window_start": d.WindowStart.Format(time.RFC3339),
			"window_end":   d.WindowEnd.Format(time.RFC3339),
			"detected_at":  d.DetectedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     total,
	})
}
