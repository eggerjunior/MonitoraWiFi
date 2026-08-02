// Comando do worker (Fase 7): calcula baseline estatístico por site a
// partir do histórico de ping_tests e speed_tests (modo "internet") e
// detecta anomalias no período recente, uma métrica por vez
// (ping_latency_ms_p50, speedtest_download_mbps, speedtest_upload_mbps,
// speedtest_bufferbloat_ms). Execução única — agendado via cron do host
// a cada 6h em produção (Fase 8, ver docs/development-handoff/RELEASE_LOG.md).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"egger/worker/internal/baseline"
	"egger/worker/internal/buildinfo"
	"egger/worker/internal/store"
)

const (
	historicalWindow = 30 * 24 * time.Hour // baseline: últimos 30 dias
	recentWindow     = 24 * time.Hour      // período avaliado contra o baseline
	zScoreThreshold  = 3.0
)

// metricFetcher busca a série de uma métrica para um site — mesma
// assinatura pra ping (Fase 2) e speed test (Fase 4), permitindo tratar
// as duas fontes de forma uniforme no loop de detecção abaixo.
type metricFetcher func(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]store.MetricSample, error)

// metrics lista toda métrica coberta pelo baseline estatístico (Fase 7).
// Cobertura de speed test (download/upload/bufferbloat, sempre modo
// "internet") fecha o item do roadmap "Faltam... cobrir métricas de speed
// test (só ping por enquanto)".
var metrics = map[string]metricFetcher{
	"ping_latency_ms_p50": func(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]store.MetricSample, error) {
		samples, err := store.ListPingLatencies(ctx, pool, siteID, since)
		if err != nil {
			return nil, err
		}
		out := make([]store.MetricSample, len(samples))
		for i, s := range samples {
			out[i] = store.MetricSample{ExecutedAt: s.ExecutedAt, Value: s.LatencyMsP50}
		}
		return out, nil
	},
	"speedtest_download_mbps":  store.ListSpeedTestDownload,
	"speedtest_upload_mbps":    store.ListSpeedTestUpload,
	"speedtest_bufferbloat_ms": store.ListSpeedTestBufferbloat,
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("worker iniciado", slog.String("version", buildinfo.Version), slog.String("commit", buildinfo.GitCommit))

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
		for metricName, fetch := range metrics {
			count, err := processMetric(ctx, pool, logger, site.ID, metricName, fetch, now)
			if err != nil {
				logger.Error("erro ao processar métrica", slog.String("site_id", site.ID.String()), slog.String("metric", metricName), slog.Any("error", err))
				continue
			}
			totalAnomalies += count
		}
	}

	logger.Info("worker concluído", slog.Int("sites_processados", len(sites)), slog.Int("anomalias_totais", totalAnomalies))
	return nil
}

// processMetric aplica o mesmo algoritmo de baseline (Fase 7) a uma única
// métrica de um site — extraído do loop principal pra tratar
// ping/download/upload/bufferbloat de forma idêntica, sem duplicar a
// lógica de split histórico/recente + detecção + gravação.
func processMetric(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, siteID uuid.UUID, metricName string, fetch metricFetcher, now time.Time) (int, error) {
	samples, err := fetch(ctx, pool, siteID, now.Add(-historicalWindow))
	if err != nil {
		return 0, fmt.Errorf("ler série: %w", err)
	}

	var historical, recent []baseline.Sample
	recentStart := now.Add(-recentWindow)
	for _, s := range samples {
		sample := baseline.Sample{Time: s.ExecutedAt, Value: s.Value}
		if s.ExecutedAt.Before(recentStart) {
			historical = append(historical, sample)
		} else {
			recent = append(recent, sample)
		}
	}

	if len(historical) == 0 {
		logger.Info("sem histórico suficiente para baseline ainda",
			slog.String("site_id", siteID.String()), slog.String("metric", metricName))
		return 0, nil
	}

	b := baseline.Compute(historical)
	anomalies := baseline.Detect(recent, b, zScoreThreshold)

	records := make([]store.AnomalyRecord, 0, len(anomalies))
	for _, a := range anomalies {
		records = append(records, store.AnomalyRecord{
			SiteID:     siteID,
			Metric:     metricName,
			ObservedAt: a.Sample.Time,
			Value:      a.Sample.Value,
			BucketMean: a.BucketMean,
			BucketSize: a.BucketSize,
			ZScore:     a.ZScore,
		})
	}
	if err := store.UpsertAnomalies(ctx, pool, records); err != nil {
		return 0, fmt.Errorf("gravar anomalias: %w", err)
	}

	logger.Info("processamento de baseline concluído",
		slog.String("site_id", siteID.String()),
		slog.String("metric", metricName),
		slog.Int("historical_samples", len(historical)),
		slog.Int("recent_samples", len(recent)),
		slog.Int("anomalies_found", len(anomalies)))
	return len(anomalies), nil
}
