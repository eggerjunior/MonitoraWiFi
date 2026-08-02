// Package store dá ao worker acesso mínimo e direto ao Postgres — não
// reimplementa o store completo de apps/api (módulo Go separado, ADR-003);
// só o necessário para ler séries de ping_tests por site e gravar
// anomalias detectadas.
package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Site struct {
	ID uuid.UUID
}

// ListSitesWithAgents retorna os sites que têm pelo menos um agente
// enrolado — não há série de ping_tests para calcular baseline em sites
// sem agente.
func ListSitesWithAgents(ctx context.Context, pool *pgxpool.Pool) ([]Site, error) {
	rows, err := pool.Query(ctx, `SELECT DISTINCT s.id FROM sites s JOIN agents a ON a.site_id = s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Site
	for rows.Next() {
		var s Site
		if err := rows.Scan(&s.ID); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type PingLatencySample struct {
	ExecutedAt   time.Time
	LatencyMsP50 float64
}

// ListPingLatencies retorna amostras de latência p50 (nunca inventa uma
// amostra para um ping que teve 100% de perda — latency_ms_p50 NULL nesse
// caso é excluído na própria query).
func ListPingLatencies(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]PingLatencySample, error) {
	rows, err := pool.Query(ctx,
		`SELECT pt.executed_at, pt.latency_ms_p50
		 FROM ping_tests pt
		 JOIN agents a ON a.id = pt.agent_id
		 WHERE a.site_id = $1 AND pt.executed_at >= $2 AND pt.latency_ms_p50 IS NOT NULL
		 ORDER BY pt.executed_at`,
		siteID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PingLatencySample
	for rows.Next() {
		var s PingLatencySample
		if err := rows.Scan(&s.ExecutedAt, &s.LatencyMsP50); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// MetricSample é o formato genérico devolvido por qualquer consulta de
// série usada para baseline — mesmo shape de PingLatencySample, mas
// reaproveitável por speed test e futuras métricas.
type MetricSample struct {
	ExecutedAt time.Time
	Value      float64
}

// listSpeedTestColumn é o núcleo comum às três métricas de speed test
// abaixo — sempre filtra mode='internet' (Fase 7: correlacionar "Internet
// lenta" exige comparar o mesmo tipo de teste; misturar com modo LAN/HTTP
// corromperia a estatística, já que são contextos de rede diferentes).
func listSpeedTestColumn(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time, column string) ([]MetricSample, error) {
	query := `SELECT st.executed_at, st.` + column + `
		FROM speed_tests st
		JOIN agents a ON a.id = st.agent_id
		WHERE a.site_id = $1 AND st.executed_at >= $2 AND st.mode = 'internet' AND st.` + column + ` IS NOT NULL
		ORDER BY st.executed_at`
	rows, err := pool.Query(ctx, query, siteID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MetricSample
	for rows.Next() {
		var s MetricSample
		if err := rows.Scan(&s.ExecutedAt, &s.Value); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// As três funções abaixo passam um nome de coluna fixo (nunca vindo de
// input externo/dinâmico) — evita construir SQL a partir de uma variável
// arbitrária, mesmo sendo seguro hoje.
func ListSpeedTestDownload(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]MetricSample, error) {
	return listSpeedTestColumn(ctx, pool, siteID, since, "download_mbps")
}

func ListSpeedTestUpload(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]MetricSample, error) {
	return listSpeedTestColumn(ctx, pool, siteID, since, "upload_mbps")
}

func ListSpeedTestBufferbloat(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]MetricSample, error) {
	return listSpeedTestColumn(ctx, pool, siteID, since, "bufferbloat_ms")
}

type AnomalyRecord struct {
	SiteID     uuid.UUID
	Metric     string
	ObservedAt time.Time
	Value      float64
	BucketMean float64
	BucketSize int
	ZScore     float64
}

// UpsertAnomalies grava (ignorando duplicatas — mesma site_id+metric+observed_at
// já gravada antes, idempotente entre execuções repetidas do worker).
func UpsertAnomalies(ctx context.Context, pool *pgxpool.Pool, records []AnomalyRecord) error {
	for _, r := range records {
		_, err := pool.Exec(ctx,
			`INSERT INTO anomalies (site_id, metric, observed_at, value, bucket_mean, bucket_size, z_score)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (site_id, metric, observed_at) DO NOTHING`,
			r.SiteID, r.Metric, r.ObservedAt, r.Value, r.BucketMean, r.BucketSize, r.ZScore)
		if err != nil {
			return err
		}
	}
	return nil
}
