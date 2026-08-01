// Comando principal do agente local (Fase 2). Conexão outbound-only para o
// backend (ADR-001) — nunca escuta porta de entrada na rede do cliente.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"egger/local-agent/internal/agent"
	"egger/local-agent/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("agente encerrado com erro", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	a, err := agent.New(ctx, cfg, logger)
	if err != nil {
		return err
	}

	logger.Info("agente iniciado",
		slog.String("backend_url", cfg.BackendURL),
		slog.Int("targets", len(cfg.Targets)),
		slog.Duration("probe_interval", cfg.ProbeInterval),
		slog.Duration("heartbeat_interval", cfg.HeartbeatInterval))

	a.Run(ctx)

	logger.Info("agente encerrado")
	return nil
}
