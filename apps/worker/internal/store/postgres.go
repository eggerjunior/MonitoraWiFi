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

// listSpeedTestColumn é o núcleo comum às métricas de speed test abaixo —
// sempre filtra por um único `mode` (Fase 7: correlacionar "Internet lenta"
// vs. "Wi-Fi lento" exige comparar o mesmo tipo de teste; misturar modos
// diferentes na mesma série corromperia a estatística, já que são
// contextos de rede diferentes). `column` e `mode` nunca vêm de input
// externo/dinâmico — evita construir SQL a partir de uma variável
// arbitrária, mesmo sendo seguro hoje.
func listSpeedTestColumn(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time, mode, column string) ([]MetricSample, error) {
	query := `SELECT st.executed_at, st.` + column + `
		FROM speed_tests st
		JOIN agents a ON a.id = st.agent_id
		WHERE a.site_id = $1 AND st.executed_at >= $2 AND st.mode = '` + mode + `' AND st.` + column + ` IS NOT NULL
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

func ListSpeedTestDownload(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]MetricSample, error) {
	return listSpeedTestColumn(ctx, pool, siteID, since, "internet", "download_mbps")
}

func ListSpeedTestUpload(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]MetricSample, error) {
	return listSpeedTestColumn(ctx, pool, siteID, since, "internet", "upload_mbps")
}

func ListSpeedTestBufferbloat(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]MetricSample, error) {
	return listSpeedTestColumn(ctx, pool, siteID, since, "internet", "bufferbloat_ms")
}

// ListSpeedTestLANDownload/Upload (Fase 7, motor de correlação): cobertura
// do modo 'lan' que faltava — sem baseline de throughput local, o
// diagnóstico "Wi-Fi lento" não teria nenhuma evidência real distinta de
// "Internet lenta" (ver apps/worker/internal/diagnostics).
func ListSpeedTestLANDownload(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]MetricSample, error) {
	return listSpeedTestColumn(ctx, pool, siteID, since, "lan", "download_mbps")
}

func ListSpeedTestLANUpload(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]MetricSample, error) {
	return listSpeedTestColumn(ctx, pool, siteID, since, "lan", "upload_mbps")
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

// AnomalyEvidence é a anomalia já gravada, no formato que o motor de
// correlação (internal/diagnostics) consome como entrada — inclui o ID
// real da anomalia pra que diagnósticos referenciem evidência rastreável,
// não um resumo solto.
type AnomalyEvidence struct {
	ID         uuid.UUID
	Metric     string
	ObservedAt time.Time
	Value      float64
	BucketMean float64
	ZScore     float64
}

// ListRecentAnomalies lê (não junto com a gravação acima) todas as
// anomalias já persistidas de um site numa janela — inclui as gravadas em
// execuções anteriores do worker, não só as desta execução, já que o
// motor de correlação precisa enxergar o quadro completo da janela.
func ListRecentAnomalies(ctx context.Context, pool *pgxpool.Pool, siteID uuid.UUID, since time.Time) ([]AnomalyEvidence, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, metric, observed_at, value, bucket_mean, z_score
		 FROM anomalies WHERE site_id = $1 AND observed_at >= $2 ORDER BY observed_at`,
		siteID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AnomalyEvidence
	for rows.Next() {
		var a AnomalyEvidence
		if err := rows.Scan(&a.ID, &a.Metric, &a.ObservedAt, &a.Value, &a.BucketMean, &a.ZScore); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DiagnosisRecord é o resultado do motor de correlação pronto para
// persistência — Evidence já serializado (json.Marshal de
// []diagnostics.AnomalyEvidenceRef feito pelo chamador, ver cmd/worker).
type DiagnosisRecord struct {
	SiteID      uuid.UUID
	Category    string
	Summary     string
	Confidence  float64
	Impact      string
	Risk        string
	EvidenceRaw []byte
	WindowStart time.Time
	WindowEnd   time.Time
}

// UpsertDiagnosis grava um diagnóstico e devolve seu ID mesmo em caso de
// conflito (mesma site_id+category+window_end já gravada antes) — usa
// `DO UPDATE` sem alterar nada de fato (write idempotente) só para poder
// usar `RETURNING id`, já que `DO NOTHING` não devolve linha no conflito e
// a recomendação associada precisa do ID do diagnóstico de qualquer forma.
func UpsertDiagnosis(ctx context.Context, pool *pgxpool.Pool, r DiagnosisRecord) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO diagnoses (site_id, category, summary, confidence, impact, risk, evidence, window_start, window_end)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (site_id, category, window_end) DO UPDATE SET summary = diagnoses.summary
		 RETURNING id`,
		r.SiteID, r.Category, r.Summary, r.Confidence, r.Impact, r.Risk, r.EvidenceRaw, r.WindowStart, r.WindowEnd).Scan(&id)
	return id, err
}

// RecommendationRecord é a recomendação gerada a partir de um diagnóstico
// já persistido — sempre amarrada a um DiagnosisID real (nunca uma
// recomendação solta sem diagnóstico/evidência por trás, ver
// docs/architecture/06-roadmap.md, Fase 7).
type RecommendationRecord struct {
	DiagnosisID uuid.UUID
	SiteID      uuid.UUID
	Action      string
	Confidence  float64
	Impact      string
	Risk        string
	EvidenceRaw []byte
}

// UpsertRecommendation grava (uma recomendação por diagnóstico nesta fase —
// idx_recommendations_one_per_diagnosis — reexecuções do worker sobre o
// mesmo diagnóstico não duplicam).
func UpsertRecommendation(ctx context.Context, pool *pgxpool.Pool, r RecommendationRecord) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO recommendations (diagnosis_id, site_id, action, confidence, impact, risk, evidence)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (diagnosis_id) DO NOTHING`,
		r.DiagnosisID, r.SiteID, r.Action, r.Confidence, r.Impact, r.Risk, r.EvidenceRaw)
	return err
}
