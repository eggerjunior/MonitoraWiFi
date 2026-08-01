// Comandos sob demanda (Fase 5, início): o agente consulta o backend
// periodicamente por comandos pendentes endereçados a ele — nunca expõe
// porta de entrada, sempre consulta (ADR-001), na mesma conexão outbound do
// heartbeat (docs/architecture/03-fluxo-de-dados.md §3.2).
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
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

func (a *Agent) runPingCommand(ctx context.Context, cmd apiclient.Command) {
	var params pingCommandParams
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		a.reportCommandFailure(ctx, cmd.ID, "params inválido: "+err.Error())
		return
	}

	opts := probes.DefaultOptions()
	var result probes.Result
	switch params.Protocol {
	case "icmp":
		result = probes.ProbeICMP(ctx, params.Target, opts)
	case "tcp":
		result = probes.ProbeTCP(ctx, params.Target, opts)
	case "http":
		result = probes.ProbeHTTP(ctx, params.Target, opts)
	case "dns":
		result = probes.ProbeDNS(ctx, params.Target, "", opts)
	default:
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

func (a *Agent) reportCommandFailure(ctx context.Context, commandID, reason string) {
	if err := a.client.ReportCommandResult(ctx, a.identity.AgentID, a.identity.AgentSecret, commandID, "failed", nil, reason); err != nil {
		a.logger.Error("erro ao reportar falha do comando", slog.String("command_id", commandID), slog.Any("error", err))
	}
}
