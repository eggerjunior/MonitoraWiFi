// Comando do worker (Fase 7, início): calcula baseline estatístico por
// site a partir do histórico de ping_tests e detecta anomalias no período
// recente. Execução única (não é um daemon/cron ainda — agendamento fica
// para quando houver decisão de infraestrutura, Fase 8) — pensado para
// rodar via `docker run --rm` periódico (cron do host) até então.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"egger/worker/internal/baseline"
	"egger/worker/internal/store"
)

const (
	historicalWindow = 30 * 24 * time.Hour // baseline: últimos 30 dias
	recentWindow     = 24 * time.Hour      // período avaliado contra o baseline
	zScoreThreshold  = 3.0
	metricName       = "ping_latency_ms_p50"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL não definido")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		logger.Error("erro ao conectar no Postgres", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	if err := run(ctx, pool, logger); err != nil {
		logger.Error("worker encerrado com erro", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	sites, err := store.ListSitesWithAgents(ctx, pool)
	if err != nil {
		return fmt.Errorf("listar sites com agente: %w", err)
	}
	if len(sites) == 0 {
		logger.Info("nenhum site com agente enrolado ainda — nada para processar")
		return nil
	}

	now := time.Now().UTC()
	totalAnomalies := 0

	for _, site := range sites {
		samples, err := store.ListPingLatencies(ctx, pool, site.ID, now.Add(-historicalWindow))
		if err != nil {
			logger.Error("erro ao ler ping_tests", slog.String("site_id", site.ID.String()), slog.Any("error", err))
			continue
		}

		var historical, recent []baseline.Sample
		recentStart := now.Add(-recentWindow)
		for _, s := range samples {
			sample := baseline.Sample{Time: s.ExecutedAt, Value: s.LatencyMsP50}
			if s.ExecutedAt.Before(recentStart) {
				historical = append(historical, sample)
			} else {
				recent = append(recent, sample)
			}
		}

		if len(historical) == 0 {
			logger.Info("sem histórico suficiente para baseline ainda",
				slog.String("site_id", site.ID.String()))
			continue
		}

		b := baseline.Compute(historical)
		anomalies := baseline.Detect(recent, b, zScoreThreshold)

		records := make([]store.AnomalyRecord, 0, len(anomalies))
		for _, a := range anomalies {
			records = append(records, store.AnomalyRecord{
				SiteID:     site.ID,
				Metric:     metricName,
				ObservedAt: a.Sample.Time,
				Value:      a.Sample.Value,
				BucketMean: a.BucketMean,
				BucketSize: a.BucketSize,
				ZScore:     a.ZScore,
			})
		}
		if err := store.UpsertAnomalies(ctx, pool, records); err != nil {
			logger.Error("erro ao gravar anomalias", slog.String("site_id", site.ID.String()), slog.Any("error", err))
			continue
		}

		logger.Info("processamento de baseline concluído",
			slog.String("site_id", site.ID.String()),
			slog.Int("historical_samples", len(historical)),
			slog.Int("recent_samples", len(recent)),
			slog.Int("anomalies_found", len(anomalies)))
		totalAnomalies += len(anomalies)
	}

	logger.Info("worker concluído", slog.Int("sites_processados", len(sites)), slog.Int("anomalias_totais", totalAnomalies))
	return nil
}
