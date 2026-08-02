// Comandos sob demanda (Fase 5, início): o agente consulta o backend
// periodicamente por comandos pendentes endereçados a ele — nunca expõe
// porta de entrada, sempre consulta (ADR-001), na mesma conexão outbound do
// heartbeat (docs/architecture/03-fluxo-de-dados.md §3.2).
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"egger/local-agent/internal/apiclient"
	"egger/local-agent/internal/probes"
)

func (a *Agent) commandLoop(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.CommandPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollAndRunCommands(ctx)
		}
	}
}

func (a *Agent) pollAndRunCommands(ctx context.Context) {
	cmds, err := a.client.ClaimCommands(ctx, a.identity.AgentID, a.identity.AgentSecret)
	if err != nil {
		a.logger.Warn("erro ao consultar comandos pendentes", slog.Any("error", err))
		return
	}

	for _, cmd := range cmds {
		a.runCommand(ctx, cmd)
	}
}

func (a *Agent) runCommand(ctx context.Context, cmd apiclient.Command) {
	switch cmd.Type {
	case "ping":
		a.runPingCommand(ctx, cmd)
	case "dns_lookup":
		a.runDNSLookupCommand(ctx, cmd)
	case "traceroute":
		a.runTracerouteCommand(ctx, cmd)
	case "batch_ping":
		a.runBatchPingCommand(ctx, cmd)
	case "ssl_check":
		a.runSSLCheckCommand(ctx, cmd)
	case "http_request":
		a.runHTTPRequestCommand(ctx, cmd)
	default:
		// O backend já valida o tipo na criação (CHECK constraint +
		// validação no handler) — chegar aqui com um tipo desconhecido só
		// deveria acontecer se o agente estiver desatualizado em relação
		// ao backend. Reporta falha honesta em vez de travar o loop.
		a.reportCommandFailure(ctx, cmd.ID, fmt.Sprintf("tipo de comando não suportado por esta versão do agente: %s", cmd.Type))
	}
}

type pingCommandParams struct {
	Target   string `json:"target"`
	Protocol string `json:"protocol"`
}

// probeByProtocol despacha pro probe certo — reaproveitado pelo ping único
// e pelo ping em lote (batch_ping), que só dispara o mesmo probe várias
// vezes em sequência (nunca em paralelo, pra não virar uma ferramenta de
// flood contra a LAN — Seção 2.1 do threat model).
func probeByProtocol(ctx context.Context, target, protocol string, opts probes.Options) (probes.Result, bool) {
	switch protocol {
	case "icmp":
		return probes.ProbeICMP(ctx, target, opts), true
	case "tcp":
		return probes.ProbeTCP(ctx, target, opts), true
	case "http":
		return probes.ProbeHTTP(ctx, target, opts), true
	case "dns":
		return probes.ProbeDNS(ctx, target, "", opts), true
	default:
		return probes.Result{}, false
	}
}

func (a *Agent) runPingCommand(ctx context.Context, cmd apiclient.Command) {
	var params pingCommandParams
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		a.reportCommandFailure(ctx, cmd.ID, "params inválido: "+err.Error())
		return
	}

	opts := probes.DefaultOptions()
	result, ok := probeByProtocol(ctx, params.Target, params.Protocol, opts)
	if !ok {
		a.reportCommandFailure(ctx, cmd.ID, "protocolo não suportado: "+params.Protocol)
		return
	}

	payload := map[string]any{
		"target":          result.Target,
		"protocol":        result.Protocol,
		"latency_ms_p50":  result.LatencyMsP50,
		"latency_ms_p95":  result.LatencyMsP95,
		"latency_ms_p99":  result.LatencyMsP99,
		"jitter_ms":       result.JitterMs,
		"packet_loss_pct": result.PacketLossPct,
		"executed_at":     result.ExecutedAt.Format(time.RFC3339Nano),
	}

	if err := a.client.ReportCommandResult(ctx, a.identity.AgentID, a.identity.AgentSecret, cmd.ID, "completed", payload, ""); err != nil {
		a.logger.Error("erro ao reportar resultado do comando", slog.String("command_id", cmd.ID), slog.Any("error", err))
	}
}

type dnsLookupCommandParams struct {
	Hostname string `json:"hostname"`
}

func (a *Agent) runDNSLookupCommand(ctx context.Context, cmd apiclient.Command) {
	var params dnsLookupCommandParams
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		a.reportCommandFailure(ctx, cmd.ID, "params inválido: "+err.Error())
		return
	}

	start := time.Now()
	resolver := net.DefaultResolver
	addrs, err := resolver.LookupHost(ctx, params.Hostname)
	durationMs := float64(time.Since(start).Microseconds()) / 1000

	if err != nil {
		a.reportCommandFailure(ctx, cmd.ID, "falha ao resolver "+params.Hostname+": "+err.Error())
		return
	}

	payload := map[string]any{
		"hostname":    params.Hostname,
		"addresses":   addrs,
		"duration_ms": durationMs,
		"executed_at": time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := a.client.ReportCommandResult(ctx, a.identity.AgentID, a.identity.AgentSecret, cmd.ID, "completed", payload, ""); err != nil {
		a.logger.Error("erro ao reportar resultado do comando", slog.String("command_id", cmd.ID), slog.Any("error", err))
	}
}

type tracerouteCommandParams struct {
	Target string `json:"target"`
}

func (a *Agent) runTracerouteCommand(ctx context.Context, cmd apiclient.Command) {
	var params tracerouteCommandParams
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		a.reportCommandFailure(ctx, cmd.ID, "params inválido: "+err.Error())
		return
	}

	result := probes.Traceroute(ctx, params.Target, 30, 2*time.Second)
	if result.Error != "" {
		a.reportCommandFailure(ctx, cmd.ID, result.Error)
		return
	}

	hops := make([]map[string]any, 0, len(result.Hops))
	for _, h := range result.Hops {
		hops = append(hops, map[string]any{
			"hop":     h.Hop,
			"address": h.Address,
			"rtt_ms":  h.RTTMs,
		})
	}

	payload := map[string]any{
		"target":      result.Target,
		"reached":     result.Reached,
		"hops":        hops,
		"executed_at": result.ExecutedAt.Format(time.RFC3339Nano),
	}
	if err := a.client.ReportCommandResult(ctx, a.identity.AgentID, a.identity.AgentSecret, cmd.ID, "completed", payload, ""); err != nil {
		a.logger.Error("erro ao reportar resultado do comando", slog.String("command_id", cmd.ID), slog.Any("error", err))
	}
}

type batchPingCommandParams struct {
	Targets  []string `json:"targets"`
	Protocol string   `json:"protocol"`
}

func (a *Agent) runBatchPingCommand(ctx context.Context, cmd apiclient.Command) {
	var params batchPingCommandParams
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		a.reportCommandFailure(ctx, cmd.ID, "params inválido: "+err.Error())
		return
	}
	if len(params.Targets) == 0 {
		a.reportCommandFailure(ctx, cmd.ID, "params.targets vazio")
		return
	}

	protocol := params.Protocol
	if protocol == "" {
		protocol = "icmp"
	}

	opts := probes.DefaultOptions()
	results := make([]map[string]any, 0, len(params.Targets))
	for _, target := range params.Targets {
		result, ok := probeByProtocol(ctx, target, protocol, opts)
		if !ok {
			a.reportCommandFailure(ctx, cmd.ID, "protocolo não suportado: "+protocol)
			return
		}
		results = append(results, map[string]any{
			"target":          result.Target,
			"protocol":        result.Protocol,
			"latency_ms_p50":  result.LatencyMsP50,
			"latency_ms_p95":  result.LatencyMsP95,
			"latency_ms_p99":  result.LatencyMsP99,
			"jitter_ms":       result.JitterMs,
			"packet_loss_pct": result.PacketLossPct,
			"executed_at":     result.ExecutedAt.Format(time.RFC3339Nano),
		})
	}

	payload := map[string]any{
		"protocol":    protocol,
		"results":     results,
		"executed_at": time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := a.client.ReportCommandResult(ctx, a.identity.AgentID, a.identity.AgentSecret, cmd.ID, "completed", payload, ""); err != nil {
		a.logger.Error("erro ao reportar resultado do comando", slog.String("command_id", cmd.ID), slog.Any("error", err))
	}
}

type sslCheckCommandParams struct {
	Target string `json:"target"`
	Port   int    `json:"port"`
}

func (a *Agent) runSSLCheckCommand(ctx context.Context, cmd apiclient.Command) {
	var params sslCheckCommandParams
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		a.reportCommandFailure(ctx, cmd.ID, "params inválido: "+err.Error())
		return
	}
	if params.Port == 0 {
		params.Port = 443
	}

	result := probes.CheckTLS(ctx, params.Target, params.Port, 5*time.Second)
	if !result.Reached {
		a.reportCommandFailure(ctx, cmd.ID, result.Error)
		return
	}

	payload := map[string]any{
		"target":            result.Target,
		"port":              result.Port,
		"valid_now":         result.ValidNow,
		"verify_error":      result.VerifyError,
		"not_before":        result.NotBefore.Format(time.RFC3339),
		"not_after":         result.NotAfter.Format(time.RFC3339),
		"days_until_expiry": result.DaysUntilExpiry,
		"issuer":            result.Issuer,
		"subject":           result.Subject,
		"dns_names":         result.DNSNames,
		"executed_at":       result.ExecutedAt.Format(time.RFC3339Nano),
	}
	if err := a.client.ReportCommandResult(ctx, a.identity.AgentID, a.identity.AgentSecret, cmd.ID, "completed", payload, ""); err != nil {
		a.logger.Error("erro ao reportar resultado do comando", slog.String("command_id", cmd.ID), slog.Any("error", err))
	}
}

type httpRequestCommandParams struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// maxHTTPResponseBodySnippet limita quanto do corpo da resposta é
// devolvido ao usuário — o objetivo é depurar um serviço (status, headers,
// tempo), não baixar arquivos grandes através do agente.
const maxHTTPResponseBodySnippet = 65536

func (a *Agent) runHTTPRequestCommand(ctx context.Context, cmd apiclient.Command) {
	var params httpRequestCommandParams
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		a.reportCommandFailure(ctx, cmd.ID, "params inválido: "+err.Error())
		return
	}
	method := strings.ToUpper(params.Method)
	if method == "" {
		method = http.MethodGet
	}

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var bodyReader io.Reader
	if params.Body != "" {
		bodyReader = strings.NewReader(params.Body)
	}
	req, err := http.NewRequestWithContext(reqCtx, method, params.URL, bodyReader)
	if err != nil {
		a.reportCommandFailure(ctx, cmd.ID, "requisição inválida: "+err.Error())
		return
	}
	for k, v := range params.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	durationMs := float64(time.Since(start).Microseconds()) / 1000
	if err != nil {
		a.reportCommandFailure(ctx, cmd.ID, "erro ao executar requisição: "+err.Error())
		return
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxHTTPResponseBodySnippet+1)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		a.reportCommandFailure(ctx, cmd.ID, "erro ao ler corpo da resposta: "+err.Error())
		return
	}
	truncated := len(bodyBytes) > maxHTTPResponseBodySnippet
	if truncated {
		bodyBytes = bodyBytes[:maxHTTPResponseBodySnippet]
	}

	headers := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		headers[k] = strings.Join(v, ", ")
	}

	payload := map[string]any{
		"url":            params.URL,
		"method":         method,
		"status_code":    resp.StatusCode,
		"status_text":    resp.Status,
		"headers":        headers,
		"body_snippet":   string(bytes.ToValidUTF8(bodyBytes, []byte("�"))),
		"body_truncated": truncated,
		"content_length": resp.ContentLength,
		"duration_ms":    durationMs,
		"executed_at":    time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := a.client.ReportCommandResult(ctx, a.identity.AgentID, a.identity.AgentSecret, cmd.ID, "completed", payload, ""); err != nil {
		a.logger.Error("erro ao reportar resultado do comando", slog.String("command_id", cmd.ID), slog.Any("error", err))
	}
}

func (a *Agent) reportCommandFailure(ctx context.Context, commandID, reason string) {
	if err := a.client.ReportCommandResult(ctx, a.identity.AgentID, a.identity.AgentSecret, commandID, "failed", nil, reason); err != nil {
		a.logger.Error("erro ao reportar falha do comando", slog.String("command_id", commandID), slog.Any("error", err))
	}
}
