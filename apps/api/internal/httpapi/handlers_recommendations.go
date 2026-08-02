// Recomendações (Fase 7, motor de correlação): geradas pelo worker
// (apps/worker/internal/diagnostics) a partir de um diagnóstico real, com
// evidência/confiança/impacto/risco — nunca uma recomendação solta sem
// diagnóstico por trás. Só lidas aqui.
package httpapi

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (s *Server) handleListRecommendations(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	page := parsePage(r)
	recommendations, total, err := s.recommendations.ListBySite(r.Context(), siteID, page)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar recomendações")
		return
	}

	items := make([]map[string]any, 0, len(recommendations))
	for _, rec := range recommendations {
		items = append(items, map[string]any{
			"id":           rec.ID.String(),
			"diagnosis_id": rec.DiagnosisID.String(),
			"action":       rec.Action,
			"confidence":   rec.Confidence,
			"impact":       rec.Impact,
			"risk":         rec.Risk,
			"evidence":     rec.Evidence,
			"created_at":   rec.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     total,
	})
}
