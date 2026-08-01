// RDAP/WHOIS (Fase 5): consulta pública sobre domínio/IP, resolvida via
// bootstrap real da IANA (RFC 7484) — não depende do agente local porque a
// informação é pública na internet, não da LAN do usuário.
package httpapi

import (
	"errors"
	"net/http"

	"egger/api/internal/rdap"
)

func (s *Server) handleRDAPLookup(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	user, _ := userFromContext(r.Context())
	if !s.rdapLimiter.Allow(user.ID.String()) {
		writeError(w, correlationID, http.StatusTooManyRequests, "rate_limited", "muitas consultas RDAP em pouco tempo, tente novamente em instantes")
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_query", "parâmetro query é obrigatório")
		return
	}

	result, err := s.rdapClient.Lookup(r.Context(), query)
	if err != nil {
		if errors.Is(err, rdap.ErrNoServer) {
			writeError(w, correlationID, http.StatusNotFound, "no_rdap_server", err.Error())
			return
		}
		writeError(w, correlationID, http.StatusBadGateway, "rdap_lookup_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
