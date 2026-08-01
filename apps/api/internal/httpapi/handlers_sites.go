package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"egger/api/internal/store"
)

type siteResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Timezone       string `json:"timezone"`
	CreatedAt      string `json:"created_at"`
}

func toSiteResponse(site store.Site) siteResponse {
	return siteResponse{
		ID:             site.ID.String(),
		OrganizationID: site.OrganizationID.String(),
		Name:           site.Name,
		Timezone:       site.Timezone,
		CreatedAt:      site.CreatedAt.Format(time.RFC3339),
	}
}

func (s *Server) handleListSites(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	orgID, err := uuid.Parse(r.URL.Query().Get("organization_id"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_organization_id", "organization_id inválido ou ausente")
		return
	}

	page := parsePage(r)
	sites, total, err := s.sites.ListByOrganization(r.Context(), orgID, page)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar sites")
		return
	}

	items := make([]siteResponse, 0, len(sites))
	for _, site := range sites {
		items = append(items, toSiteResponse(site))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     total,
	})
}

func (s *Server) handleGetSite(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	site, err := s.sites.Get(r.Context(), siteID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, correlationID, http.StatusNotFound, "not_found", "site não encontrado")
			return
		}
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao buscar site")
		return
	}

	writeJSON(w, http.StatusOK, toSiteResponse(site))
}
