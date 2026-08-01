package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"egger/api/internal/auth"
)

// requireAgentAuth autentica o agente local via
// "Authorization: Bearer <agent_secret>" — um esquema distinto do cookie de
// sessão do usuário (Seção 2.2). O segredo nunca é comparado em texto claro:
// o handler compara o hash do valor recebido contra o hash armazenado.
func (s *Server) requireAgentAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		correlationID := correlationIDFromContext(r.Context())

		agentID, err := uuid.Parse(r.PathValue("agentId"))
		if err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_agent_id", "agentId inválido")
			return
		}

		secret, ok := bearerToken(r)
		if !ok {
			writeError(w, correlationID, http.StatusUnauthorized, "unauthorized", "credencial do agente ausente")
			return
		}

		agent, err := s.agents.Get(r.Context(), agentID)
		if err != nil {
			writeError(w, correlationID, http.StatusUnauthorized, "unauthorized", "agente não encontrado")
			return
		}
		if agent.RevokedAt != nil {
			writeError(w, correlationID, http.StatusUnauthorized, "unauthorized", "credencial do agente revogada")
			return
		}
		if subtle.ConstantTimeCompare([]byte(auth.HashAgentSecret(secret)), []byte(agent.SecretHash)) != 1 {
			writeError(w, correlationID, http.StatusUnauthorized, "unauthorized", "credencial do agente inválida")
			return
		}

		next(w, r.WithContext(withAgent(r.Context(), agent)))
	}
}

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}
