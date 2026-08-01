-- Fase 2: resultados de speed test do agente (Seção 5.3 — modo HTTP).
CREATE TABLE speed_tests (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id              uuid NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    mode                  text NOT NULL CHECK (mode IN ('internet', 'lan', 'http')),
    download_mbps         numeric,
    upload_mbps           numeric,
    idle_latency_ms       numeric,
    loaded_latency_ms      numeric,
    bufferbloat_ms        numeric,
    jitter_ms             numeric,
    executed_at           timestamptz NOT NULL,
    idempotency_key       text NOT NULL
);
CREATE UNIQUE INDEX idx_speed_tests_idempotency ON speed_tests(agent_id, idempotency_key);
CREATE INDEX idx_speed_tests_agent_time ON speed_tests(agent_id, executed_at DESC);
