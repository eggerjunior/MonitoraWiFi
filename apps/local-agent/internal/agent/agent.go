// Package agent orquestra o agente local: identidade, heartbeat, sondas
// ativas, fila offline e reenvio (Seção 5 do documento-fonte, Fase 2 do
// roadmap). O agente nunca escuta porta de entrada — todo laço aqui inicia
// conexões para fora (ADR-001).
package agent

import (
	"context"
	"log/slog"
	"time"

	"egger/local-agent/internal/apiclient"
	"egger/local-agent/internal/config"
	"egger/local-agent/internal/probes"
	"egger/local-agent/internal/queue"
	"egger/local-agent/internal/state"
)

type Agent struct {
	cfg      config.Config
	client   *apiclient.Client
	identity state.Identity
	queue    *queue.FileQueue[apiclient.PingTestPayload]
	logger   *slog.Logger
}

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*Agent, error) {
	client := apiclient.New(cfg.BackendURL)

	identity, err := EnsureIdentity(ctx, cfg, client)
	if err != nil {
		return nil, err
	}

	return &Agent{
		cfg:      cfg,
		client:   client,
		identity: identity,
		queue:    queue.NewFileQueue[apiclient.PingTestPayload](cfg.QueueFilePath, cfg.QueueMaxItems),
		logger:   logger,
	}, nil
}

// Run bloqueia até o contexto ser cancelado, rodando os três laços
// concorrentemente: sondas, drenagem da fila e heartbeat.
func (a *Agent) Run(ctx context.Context) {
	done := make(chan struct{}, 3)

	go func() { a.probeLoop(ctx); done <- struct{}{} }()
	go func() { a.drainLoop(ctx); done <- struct{}{} }()
	go func() { a.heartbeatLoop(ctx); done <- struct{}{} }()

	for i := 0; i < 3; i++ {
		<-done
	}
}

func (a *Agent) probeLoop(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.ProbeInterval)
	defer ticker.Stop()

	a.runProbesOnce(ctx) // primeira rodada imediata, não espera o primeiro tick
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.runProbesOnce(ctx)
		}
	}
}

func (a *Agent) runProbesOnce(ctx context.Context) {
	opts := probes.DefaultOptions()
	for _, target := range a.cfg.Targets {
		var result probes.Result
		switch target.Protocol {
		case "icmp":
			result = probes.ProbeICMP(ctx, target.Name, opts)
		case "tcp":
			result = probes.ProbeTCP(ctx, target.Name, opts)
		case "http":
			result = probes.ProbeHTTP(ctx, target.Name, opts)
		case "dns":
			result = probes.ProbeDNS(ctx, target.Name, "", opts)
		default:
			a.logger.Warn("protocolo de sonda desconhecido, ignorando alvo", slog.String("target", target.Name), slog.String("protocol", target.Protocol))
			continue
		}

		payload := toPingTestPayload(result)
		dropped, err := a.queue.Enqueue(payload)
		if err != nil {
			a.logger.Error("erro ao enfileirar resultado de sonda", slog.Any("error", err))
			continue
		}
		if dropped {
			a.logger.Warn("fila local atingiu o limite — amostra mais antiga descartada",
				slog.String("target", target.Name))
		}
		a.logger.Debug("sonda executada",
			slog.String("target", target.Name),
			slog.String("protocol", target.Protocol),
			slog.Float64("packet_loss_pct", result.PacketLossPct))
	}
}

func (a *Agent) drainLoop(ctx context.Context) {
	backoff := NewBackoff(5*time.Second, 10*time.Minute)
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			err := a.queue.Drain(func(items []apiclient.PingTestPayload) error {
				sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				return a.client.SendTelemetry(sendCtx, a.identity.AgentID, a.identity.AgentSecret, items)
			})

			var next time.Duration
			if err != nil {
				a.logger.Warn("falha ao enviar telemetria — mantida na fila local para nova tentativa", slog.Any("error", err))
				next = backoff.Next()
			} else {
				backoff.Reset()
				next = a.cfg.ProbeInterval
			}
			timer.Reset(next)
		}
	}
}

func (a *Agent) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.HeartbeatInterval)
	defer ticker.Stop()

	a.sendHeartbeat(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.sendHeartbeat(ctx)
		}
	}
}

func (a *Agent) sendHeartbeat(ctx context.Context) {
	queued, err := a.queue.Len()
	if err != nil {
		a.logger.Error("erro ao medir fila local para heartbeat", slog.Any("error", err))
		queued = -1
	}

	hbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = a.client.Heartbeat(hbCtx, a.identity.AgentID, a.identity.AgentSecret, apiclient.HeartbeatPayload{
		Status:      "ok",
		QueuedItems: queued,
	})
	if err != nil {
		a.logger.Warn("falha ao enviar heartbeat (agente segue rodando, tentará de novo no próximo ciclo)", slog.Any("error", err))
	}
}
