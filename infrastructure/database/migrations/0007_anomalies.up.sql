-- Fase 7 (início): anomalias estatísticas explicáveis (worker/internal/baseline).
-- Cada linha é uma amostra que desviou do baseline (hora/dia da semana) do
-- próprio site além do threshold — nunca gerada sem baseline histórico
-- suficiente (worker/internal/baseline.MinBucketSamples).
CREATE TABLE anomalies (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id       uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    metric        text NOT NULL, -- ex.: "ping_latency_ms_p50", "speedtest_download_mbps"
    observed_at   timestamptz NOT NULL,
    value         numeric NOT NULL,
    bucket_mean   numeric NOT NULL,
    bucket_size   integer NOT NULL,
    z_score       numeric NOT NULL,
    detected_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_anomalies_site_time ON anomalies(site_id, observed_at DESC);
CREATE UNIQUE INDEX idx_anomalies_dedup ON anomalies(site_id, metric, observed_at);
