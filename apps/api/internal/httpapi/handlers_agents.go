package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"egger/api/internal/auth"
	"egger/api/internal/store"
)

// handleCreateAgentEnrollmentToken gera um token de uso único (Seção 3,
// ADR-006) que o instalador do agente ("curl ... | sh") troca por uma
// credencial de longa duração no primeiro contato. O token nunca é
// persistido em texto claro — só o hash.
func (s *Server) handleCreateAgentEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	user, _ := userFromContext(r.Context())

	token, tokenHash, err := auth.GenerateEnrollmentToken()
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao gerar token")
		return
	}

	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)
	record := store.AgentEnrollmentToken{
		ID:        newUUID(),
		SiteID:    siteID,
		TokenHash: tokenHash,
		CreatedBy: &user.ID,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	if err := s.agentEnrollTokens.Create(r.Context(), record); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao persistir token")
		return
	}

	// O token só é exibido nesta resposta — não há como recuperá-lo depois
	// (só o hash fica no banco). Se perdido, gerar um novo.
	writeJSON(w, http.StatusCreated, map[string]any{
		"enrollment_token": token,
		"site_id":          siteID.String(),
		"expires_at":       expiresAt.Format(time.RFC3339),
	})
}

type enrollAgentRequest struct {
	EnrollmentToken string `json:"enrollment_token"`
	Hostname        string `json:"hostname"`
	Platform        string `json:"platform"`
	Version         string `json:"version"`
}

func (s *Server) handleEnrollAgent(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	var req enrollAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "corpo da requisição inválido")
		return
	}
	if req.EnrollmentToken == "" || req.Hostname == "" || req.Platform == "" {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "enrollment_token, hostname e platform são obrigatórios")
		return
	}

	now := time.Now().UTC()
	tokenHash := auth.HashEnrollmentToken(req.EnrollmentToken)
	tokenRecord, err := s.agentEnrollTokens.GetValidByTokenHash(r.Context(), tokenHash, now)
	if err != nil {
		if errors.Is(err, store.ErrTokenExpiredOrUsed) {
			writeError(w, correlationID, http.StatusUnauthorized, "invalid_token", "token de enrolamento inválido, expirado ou já usado")
			return
		}
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao validar token")
		return
	}

	secret, secretHash, err := auth.GenerateAgentSecret()
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao gerar credencial do agente")
		return
	}

	version := req.Version
	if version == "" {
		version = "dev"
	}

	agent := store.Agent{
		ID:         newUUID(),
		SiteID:     tokenRecord.SiteID,
		Hostname:   req.Hostname,
		Version:    version,
		Platform:   req.Platform,
		AuthMethod: "rotating_credential",
		SecretHash: secretHash,
		EnrolledAt: now,
	}
	if err := s.agents.Create(r.Context(), agent); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao criar agente")
		return
	}
	if err := s.agentEnrollTokens.MarkUsed(r.Context(), tokenRecord.ID, agent.ID, now); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao marcar token como usado")
		return
	}

	// O segredo do agente só é exibido nesta resposta — o agente precisa
	// persisti-lo localmente (ex.: /etc/egger-agent/agent.json, 0600).
	writeJSON(w, http.StatusCreated, map[string]any{
		"agent_id":     agent.ID.String(),
		"agent_secret": secret,
		"site_id":      agent.SiteID.String(),
	})
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	page := parsePage(r)
	agents, total, err := s.agents.ListBySite(r.Context(), siteID, page)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar agentes")
		return
	}

	items := make([]map[string]any, 0, len(agents))
	for _, a := range agents {
		var lastSeen *string
		if a.LastSeenAt != nil {
			s := a.LastSeenAt.Format(time.RFC3339)
			lastSeen = &s
		}
		items = append(items, map[string]any{
			"id":           a.ID.String(),
			"site_id":      a.SiteID.String(),
			"hostname":     a.Hostname,
			"version":      a.Version,
			"platform":     a.Platform,
			"auth_method":  a.AuthMethod,
			"enrolled_at":  a.EnrolledAt.Format(time.RFC3339),
			"last_seen_at": lastSeen,
			"revoked":      a.RevokedAt != nil,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     total,
	})
}

// handleRevokeAgent revoga a credencial de um agente — a partir daí,
// toda requisição autenticada com esse agente (heartbeat, telemetria,
// claim de comando) é rejeitada com 401 (requireAgentAuth já checa
// RevokedAt). Não existe "unrevoke": um agente revogado precisa ser
// enrolado de novo com um token novo. Antes desta rota, a única forma de
// revogar era `UPDATE agents SET revoked_at = now()` direto no Postgres
// (ver docs/deployment/runbook-producao.md) — gap real fechado aqui.
func (s *Server) handleRevokeAgent(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}
	agentID, err := uuid.Parse(r.PathValue("agentId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_agent_id", "agentId inválido")
		return
	}

	agent, err := s.agents.Get(r.Context(), agentID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, correlationID, http.StatusNotFound, "not_found", "agente não encontrado")
			return
		}
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao buscar agente")
		return
	}
	if agent.SiteID != siteID {
		writeError(w, correlationID, http.StatusNotFound, "not_found", "agente não encontrado neste site")
		return
	}

	if err := s.agents.Revoke(r.Context(), agentID, time.Now().UTC()); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao revogar agente")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

type agentHeartbeatRequest struct {
	Status      string   `json:"status"`
	QueuedItems int      `json:"queued_items"`
	CPUPct      *float64 `json:"cpu_pct"`
	MemPct      *float64 `json:"mem_pct"`
}

func (s *Server) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())
	agent, _ := agentFromContext(r.Context())

	var req agentHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "corpo da requisição inválido")
		return
	}
	if req.Status == "" {
		req.Status = "ok"
	}

	now := time.Now().UTC()
	hb := store.AgentHeartbeat{
		Time:        now,
		AgentID:     agent.ID,
		Status:      req.Status,
		QueuedItems: req.QueuedItems,
		CPUPct:      req.CPUPct,
		MemPct:      req.MemPct,
	}
	if err := s.agentHeartbeats.Record(r.Context(), hb); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao registrar heartbeat")
		return
	}
	if err := s.agents.UpdateLastSeen(r.Context(), agent.ID, now); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao atualizar last_seen_at")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type pingTestPayload struct {
	Target         string   `json:"target"`
	Protocol       string   `json:"protocol"`
	LatencyMsP50   *float64 `json:"latency_ms_p50"`
	LatencyMsP95   *float64 `json:"latency_ms_p95"`
	LatencyMsP99   *float64 `json:"latency_ms_p99"`
	JitterMs       *float64 `json:"jitter_ms"`
	PacketLossPct  *float64 `json:"packet_loss_pct"`
	ExecutedAt     string   `json:"executed_at"`
	IdempotencyKey string   `json:"idempotency_key"`
}

type speedTestPayload struct {
	Mode            string   `json:"mode"`
	DownloadMbps    *float64 `json:"download_mbps"`
	UploadMbps      *float64 `json:"upload_mbps"`
	IdleLatencyMs   *float64 `json:"idle_latency_ms"`
	LoadedLatencyMs *float64 `json:"loaded_latency_ms"`
	BufferbloatMs   *float64 `json:"bufferbloat_ms"`
	JitterMs        *float64 `json:"jitter_ms"`
	ExecutedAt      string   `json:"executed_at"`
	IdempotencyKey  string   `json:"idempotency_key"`
}

type agentTelemetryRequest struct {
	PingTests  []pingTestPayload  `json:"ping_tests"`
	SpeedTests []speedTestPayload `json:"speed_tests"`
}

// handleAgentTelemetry recebe um lote de resultados de teste. Idempotente
// por (agent_id, idempotency_key) — reenvio após reconexão não duplica
// (Seção 3).
func (s *Server) handleAgentTelemetry(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())
	agent, _ := agentFromContext(r.Context())

	var req agentTelemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "corpo da requisição inválido")
		return
	}

	tests := make([]store.PingTest, 0, len(req.PingTests))
	for _, p := range req.PingTests {
		if p.IdempotencyKey == "" {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "idempotency_key é obrigatório em cada ping_test")
			return
		}
		executedAt, err := time.Parse(time.RFC3339, p.ExecutedAt)
		if err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "executed_at inválido (esperado RFC3339)")
			return
		}
		tests = append(tests, store.PingTest{
			ID:             newUUID(),
			AgentID:        agent.ID,
			Target:         p.Target,
			Protocol:       p.Protocol,
			LatencyMsP50:   p.LatencyMsP50,
			LatencyMsP95:   p.LatencyMsP95,
			LatencyMsP99:   p.LatencyMsP99,
			JitterMs:       p.JitterMs,
			PacketLossPct:  p.PacketLossPct,
			ExecutedAt:     executedAt,
			IdempotencyKey: p.IdempotencyKey,
		})
	}

	speedTests := make([]store.SpeedTest, 0, len(req.SpeedTests))
	for _, sp := range req.SpeedTests {
		if sp.IdempotencyKey == "" {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "idempotency_key é obrigatório em cada speed_test")
			return
		}
		executedAt, err := time.Parse(time.RFC3339, sp.ExecutedAt)
		if err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "executed_at inválido (esperado RFC3339)")
			return
		}
		speedTests = append(speedTests, store.SpeedTest{
			ID:              newUUID(),
			AgentID:         agent.ID,
			Mode:            sp.Mode,
			DownloadMbps:    sp.DownloadMbps,
			UploadMbps:      sp.UploadMbps,
			IdleLatencyMs:   sp.IdleLatencyMs,
			LoadedLatencyMs: sp.LoadedLatencyMs,
			BufferbloatMs:   sp.BufferbloatMs,
			JitterMs:        sp.JitterMs,
			ExecutedAt:      executedAt,
			IdempotencyKey:  sp.IdempotencyKey,
		})
	}

	if err := s.pingTests.InsertBatch(r.Context(), tests); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao persistir telemetria de ping")
		return
	}
	if err := s.speedTests.InsertBatch(r.Context(), speedTests); err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao persistir telemetria de speed test")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": len(tests) + len(speedTests)})
}

func (s *Server) handleListPingTests(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	page := parsePage(r)
	tests, total, err := s.pingTests.ListBySite(r.Context(), siteID, page)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar ping tests")
		return
	}

	items := make([]map[string]any, 0, len(tests))
	for _, t := range tests {
		items = append(items, map[string]any{
			"id":              t.ID.String(),
			"agent_id":        t.AgentID.String(),
			"target":          t.Target,
			"protocol":        t.Protocol,
			"latency_ms_p50":  t.LatencyMsP50,
			"latency_ms_p95":  t.LatencyMsP95,
			"latency_ms_p99":  t.LatencyMsP99,
			"jitter_ms":       t.JitterMs,
			"packet_loss_pct": t.PacketLossPct,
			"executed_at":     t.ExecutedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     total,
	})
}

func (s *Server) handleListSpeedTests(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	page := parsePage(r)
	tests, total, err := s.speedTests.ListBySite(r.Context(), siteID, page)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar speed tests")
		return
	}

	items := make([]map[string]any, 0, len(tests))
	for _, t := range tests {
		items = append(items, map[string]any{
			"id":                t.ID.String(),
			"agent_id":          t.AgentID.String(),
			"mode":              t.Mode,
			"download_mbps":     t.DownloadMbps,
			"upload_mbps":       t.UploadMbps,
			"idle_latency_ms":   t.IdleLatencyMs,
			"loaded_latency_ms": t.LoadedLatencyMs,
			"bufferbloat_ms":    t.BufferbloatMs,
			"jitter_ms":         t.JitterMs,
			"executed_at":       t.ExecutedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     total,
	})
}
