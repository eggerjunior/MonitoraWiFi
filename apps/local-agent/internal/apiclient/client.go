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

type SpeedTestPayload struct {
	Mode            string   `json:"mode"`
	DownloadMbps    *float64 `json:"download_mbps,omitempty"`
	UploadMbps      *float64 `json:"upload_mbps,omitempty"`
	IdleLatencyMs   *float64 `json:"idle_latency_ms,omitempty"`
	LoadedLatencyMs *float64 `json:"loaded_latency_ms,omitempty"`
	BufferbloatMs   *float64 `json:"bufferbloat_ms,omitempty"`
	JitterMs        *float64 `json:"jitter_ms,omitempty"`
	ExecutedAt      string   `json:"executed_at"`
	IdempotencyKey  string   `json:"idempotency_key"`
}

func (c *Client) SendTelemetry(ctx context.Context, agentID, agentSecret string, pingTests []PingTestPayload) error {
	return c.sendTelemetryBatch(ctx, agentID, agentSecret, pingTests, nil)
}

func (c *Client) SendSpeedTests(ctx context.Context, agentID, agentSecret string, speedTests []SpeedTestPayload) error {
	return c.sendTelemetryBatch(ctx, agentID, agentSecret, nil, speedTests)
}

func (c *Client) sendTelemetryBatch(ctx context.Context, agentID, agentSecret string, pingTests []PingTestPayload, speedTests []SpeedTestPayload) error {
	path := fmt.Sprintf("/agents/%s/telemetry", agentID)
	body := map[string]any{}
	if len(pingTests) > 0 {
		body["ping_tests"] = pingTests
	}
	if len(speedTests) > 0 {
		body["speed_tests"] = speedTests
	}
	return c.doJSON(ctx, http.MethodPost, path, agentSecret, body, nil)
}

// UniFiDevicePayload/UniFiClientPayload espelham só os campos confirmados
// reais na Network API local (Fase 3, início — ADR-007). O agente envia o
// snapshot completo a cada sincronização; o backend substitui o inventário
// anterior daquele site (nunca acumula histórico de inventário — isso é
// estado atual, não série temporal).
type UniFiDevicePayload struct {
	ExternalID      string   `json:"external_id"`
	MACAddress      string   `json:"mac_address"`
	IPAddress       string   `json:"ip_address"`
	Name            string   `json:"name"`
	Model           string   `json:"model"`
	State           string   `json:"state"`
	FirmwareVersion string   `json:"firmware_version"`
	Features        []string `json:"features"`
	Interfaces      []string `json:"interfaces"`
}

type UniFiClientPayload struct {
	ExternalID     string `json:"external_id"`
	Type           string `json:"type"`
	Name           string `json:"name"`
	IPAddress      string `json:"ip_address"`
	MACAddress     string `json:"mac_address"`
	ConnectedAt    string `json:"connected_at"`
	UplinkDeviceID string `json:"uplink_device_id"`
}

func (c *Client) SendUniFiInventory(ctx context.Context, agentID, agentSecret string, devices []UniFiDevicePayload, clients []UniFiClientPayload) error {
	path := fmt.Sprintf("/agents/%s/unifi-inventory", agentID)
	body := map[string]any{
		"devices": devices,
		"clients": clients,
	}
	return c.doJSON(ctx, http.MethodPost, path, agentSecret, body, nil)
}

// Command é um teste sob demanda disparado pelo usuário (Fase 5, início —
// docs/architecture/03-fluxo-de-dados.md §3.2). Params fica cru (json.RawMessage)
// porque cada tipo de comando define seu próprio formato.
type Command struct {
	ID     string          `json:"id"`
	SiteID string          `json:"site_id"`
	Type   string          `json:"type"`
	Params json.RawMessage `json:"params"`
	Status string          `json:"status"`
}

// ClaimCommands consulta o backend por comandos pendentes endereçados a este
// agente — o agente nunca expõe porta de entrada, sempre consulta (ADR-001),
// na mesma conexão outbound usada pelo heartbeat/telemetria.
func (c *Client) ClaimCommands(ctx context.Context, agentID, agentSecret string) ([]Command, error) {
	path := fmt.Sprintf("/agents/%s/commands", agentID)
	var resp struct {
		Items []Command `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, agentSecret, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// ReportCommandResult envia o resultado (ou falha) de um comando previamente
// reivindicado via ClaimCommands.
func (c *Client) ReportCommandResult(ctx context.Context, agentID, agentSecret, commandID, status string, result any, errMsg string) error {
	path := fmt.Sprintf("/agents/%s/commands/%s/result", agentID, commandID)
	body := map[string]any{"status": status}
	if result != nil {
		body["result"] = result
	}
	if errMsg != "" {
		body["error"] = errMsg
	}
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
