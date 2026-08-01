package httpapi

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"egger/api/internal/auth"
	"egger/api/internal/store"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	MFAEnrolled    bool   `json:"mfa_enrolled"`
	CreatedAt      string `json:"created_at"`
}

func toUserResponse(u store.User) userResponse {
	return userResponse{
		ID:             u.ID.String(),
		OrganizationID: u.OrganizationID.String(),
		Email:          u.Email,
		Role:           string(u.Role),
		MFAEnrolled:    u.MFAEnrolledAt != nil,
		CreatedAt:      u.CreatedAt.Format(time.RFC3339),
	}
}

// handleLogin autentica por e-mail/senha (Seção 18). Passkeys e MFA são
// evoluções deste mesmo endpoint em fases futuras, não um redesenho.
//
// Resposta é deliberadamente idêntica (401 genérico) para "usuário não existe"
// e "senha errada", para não permitir enumeração de e-mails cadastrados.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	clientIP := clientIPFromRequest(r)
	if !s.loginLimiter.Allow(clientIP) {
		writeError(w, correlationID, http.StatusTooManyRequests, "rate_limited", "muitas tentativas de login, tente novamente em instantes")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "corpo da requisição inválido")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.users.GetByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, correlationID, http.StatusUnauthorized, "invalid_credentials", "e-mail ou senha inválidos")
			return
		}
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao autenticar")
		return
	}

	if !auth.VerifyPassword(user.PasswordHash, req.Password) {
		writeError(w, correlationID, http.StatusUnauthorized, "invalid_credentials", "e-mail ou senha inválidos")
		return
	}

	token, tokenHash, err := auth.GenerateSessionToken()
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao criar sessão")
		return
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.sessionTTL)
	session := store.Session{
		ID:        newUUID(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	if err := s.sessions.Create(r.Context(), session); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao persistir sessão")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"user":       toUserResponse(user),
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		tokenHash := auth.HashSessionToken(cookie.Value)
		if sess, err := s.sessions.GetByTokenHash(r.Context(), tokenHash); err == nil {
			_ = s.sessions.Revoke(r.Context(), sess.ID)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, correlationIDFromContext(r.Context()), http.StatusUnauthorized, "unauthorized", "sessão ausente")
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func clientIPFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
