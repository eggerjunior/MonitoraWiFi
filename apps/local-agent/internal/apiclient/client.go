// Package apiclient fala com o backend central (Seção 3: conexão outbound
// segura, TLS, credencial rotacionável). O agente nunca abre porta de
// entrada — toda comunicação é iniciada por ele.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type ApiError struct {
	Status  int
	Code    string `json:"error"`
	Message string `json:"message"`
}

func (e *ApiError) Error() string {
	return fmt.Sprintf("erro da API (status %d): %s", e.Status, e.Message)
}

type EnrollResult struct {
	AgentID     string `json:"agent_id"`
	AgentSecret string `json:"agent_secret"`
	SiteID      string `json:"site_id"`
}

func (c *Client) Enroll(ctx context.Context, enrollmentToken, hostname, platform, version string) (EnrollResult, error) {
	body := map[string]string{
		"enrollment_token": enrollmentToken,
		"hostname":         hostname,
		"platform":         platform,
		"version":          version,
	}

	var result EnrollResult
	err := c.doJSON(ctx, http.MethodPost, "/agents/enroll", "", body, &result)
	return result, err
}

type HeartbeatPayload struct {
	Status      string   `json:"status"`
	QueuedItems int      `json:"queued_items"`
	CPUPct      *float64 `json:"cpu_pct,omitempty"`
	MemPct      *float64 `json:"mem_pct,omitempty"`
}

func (c *Client) Heartbeat(ctx context.Context, agentID, agentSecret string, payload HeartbeatPayload) error {
	path := fmt.Sprintf("/agents/%s/heartbeat", agentID)
	return c.doJSON(ctx, http.MethodPost, path, agentSecret, payload, nil)
}

type PingTestPayload struct {
	Target         string   `json:"target"`
	Protocol       string   `json:"protocol"`
	LatencyMsP50   *float64 `json:"latency_ms_p50,omitempty"`
	LatencyMsP95   *float64 `json:"latency_ms_p95,omitempty"`
	LatencyMsP99   *float64 `json:"latency_ms_p99,omitempty"`
	JitterMs       *float64 `json:"jitter_ms,omitempty"`
	PacketLossPct  *float64 `json:"packet_loss_pct,omitempty"`
	ExecutedAt     string   `json:"executed_at"`
	IdempotencyKey string   `json:"idempotency_key"`
}

func (c *Client) SendTelemetry(ctx context.Context, agentID, agentSecret string, pingTests []PingTestPayload) error {
	path := fmt.Sprintf("/agents/%s/telemetry", agentID)
	body := map[string]any{"ping_tests": pingTests}
	return c.doJSON(ctx, http.MethodPost, path, agentSecret, body, nil)
}

func (c *Client) doJSON(ctx context.Context, method, path, bearerToken string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("serializar corpo da requisição: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("criar requisição: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("requisição ao backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var apiErr ApiError
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		apiErr.Status = resp.StatusCode
		return &apiErr
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decodificar resposta: %w", err)
		}
	}
	return nil
}
