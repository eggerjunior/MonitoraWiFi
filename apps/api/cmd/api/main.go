// Comando principal do backend central (Fase 1). Sobe o servidor HTTP com
// health checks, autenticação e os endpoints de organizações/sites.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"egger/api/internal/config"
	"egger/api/internal/db"
	"egger/api/internal/httpapi"
	"egger/api/internal/logging"
	"egger/api/internal/store"
	"egger/api/internal/telemetry"
)

func main() {
	if err := run(); err != nil {
		slog.Error("encerrado com erro", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.Environment)
	slog.SetDefault(logger)

	providers, shutdownTelemetry, err := telemetry.Setup(ctx, cfg.OTelServiceName)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			logger.Error("erro ao encerrar telemetria", slog.Any("error", err))
		}
	}()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	server := httpapi.NewServer(httpapi.Deps{
		Logger:     logger,
		Tracer:     providers.Tracer,
		Pool:       pool,
		Orgs:       &store.PostgresOrganizations{Pool: pool},
		Sites:      &store.PostgresSites{Pool: pool},
		Users:      &store.PostgresUsers{Pool: pool},
		Sessions:   &store.PostgresSessions{Pool: pool},
		Audit:      &store.PostgresAudit{Pool: pool},
		SessionTTL: cfg.SessionTTL,

		Agents:            &store.PostgresAgents{Pool: pool},
		AgentEnrollTokens: &store.PostgresAgentEnrollmentTokens{Pool: pool},
		AgentHeartbeats:   &store.PostgresAgentHeartbeats{Pool: pool},
		PingTests:         &store.PostgresPingTests{Pool: pool},
		SpeedTests:        &store.PostgresSpeedTests{Pool: pool},
		AgentCommands:     &store.PostgresAgentCommands{Pool: pool},
		UniFiDevices:      &store.PostgresUniFiDevices{Pool: pool},
		UniFiClients:      &store.PostgresUniFiClients{Pool: pool},
	})

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("servidor HTTP iniciado", slog.String("addr", cfg.HTTPAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("sinal de encerramento recebido, desligando graciosamente")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}
