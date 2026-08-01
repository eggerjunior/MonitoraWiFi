package httpapi

import (
	"net/http"
	"time"

	"egger/api/internal/auth"
)

const sessionCookieName = "egger_session"

// requireAuth valida o cookie de sessão e injeta o usuário autenticado no
// contexto. Sessões expiradas ou revogadas são tratadas como não autenticadas
// (fail closed), nunca aceitas "por precaução".
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		correlationID := correlationIDFromContext(r.Context())

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, correlationID, http.StatusUnauthorized, "unauthorized", "sessão ausente")
			return
		}

		tokenHash := auth.HashSessionToken(cookie.Value)
		sess, err := s.sessions.GetByTokenHash(r.Context(), tokenHash)
		if err != nil {
			writeError(w, correlationID, http.StatusUnauthorized, "unauthorized", "sessão inválida")
			return
		}
		if sess.RevokedAt != nil || sess.ExpiresAt.Before(time.Now().UTC()) {
			writeError(w, correlationID, http.StatusUnauthorized, "unauthorized", "sessão expirada")
			return
		}

		user, err := s.users.Get(r.Context(), sess.UserID)
		if err != nil {
			writeError(w, correlationID, http.StatusUnauthorized, "unauthorized", "usuário da sessão não encontrado")
			return
		}

		next(w, r.WithContext(withUser(r.Context(), user)))
	}
}

// requirePermission assume que requireAuth já rodou antes na cadeia (o usuário
// já está no contexto). Retorna 403 (não 401) quando autenticado mas sem
// permissão, distinção exigida pelo contrato OpenAPI.
func (s *Server) requirePermission(perm auth.Permission, next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		correlationID := correlationIDFromContext(r.Context())
		user, ok := userFromContext(r.Context())
		if !ok {
			writeError(w, correlationID, http.StatusUnauthorized, "unauthorized", "sessão ausente")
			return
		}
		if !auth.HasPermission(user.Role, perm) {
			writeError(w, correlationID, http.StatusForbidden, "forbidden", "papel sem permissão para esta ação")
			return
		}
		next(w, r)
	})
}
