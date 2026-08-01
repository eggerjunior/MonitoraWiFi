package httpapi

import (
	"net/http"
	"time"

	"egger/api/internal/store"
)

type organizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	PlanTier  string `json:"plan_tier"`
	CreatedAt string `json:"created_at"`
}

func toOrganizationResponse(o store.Organization) organizationResponse {
	return organizationResponse{
		ID:        o.ID.String(),
		Name:      o.Name,
		PlanTier:  o.PlanTier,
		CreatedAt: o.CreatedAt.Format(time.RFC3339),
	}
}

// handleListOrganizations lista organizações. Na Fase 1, RBAC é avaliado por
// papel (Seção 18); restrição fina "só a própria organização do usuário" é
// aplicada nas fases seguintes quando houver mais de um tenant real navegando
// pela API — hoje o usuário autenticado só enxerga a própria organização por
// construção do login (ver ADR-002 sobre multi-tenancy desde o schema).
func (s *Server) handleListOrganizations(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())
	page := parsePage(r)

	orgs, total, err := s.orgs.List(r.Context(), page)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar organizações")
		return
	}

	items := make([]organizationResponse, 0, len(orgs))
	for _, o := range orgs {
		items = append(items, toOrganizationResponse(o))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     total,
	})
}
