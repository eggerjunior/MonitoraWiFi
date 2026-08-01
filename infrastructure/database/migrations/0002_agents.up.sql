-- Fase 2: agente local — enrolamento, heartbeat e resultados de ping.
-- Ver docs/architecture/adr/ADR-001 (agente outbound-only) e ADR-006
-- (identidade do agente: credencial rotacionável nesta fase, mTLS futuro).

CREATE TABLE agents (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id       uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    hostname      text NOT NULL,
    version       text NOT NULL DEFAULT 'dev',
    platform      text NOT NULL,
    auth_method   text NOT NULL DEFAULT 'rotating_credential' CHECK (auth_method IN ('rotating_credential', 'mtls')),
    secret_hash   text NOT NULL,
    enrolled_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at  timestamptz,
    revoked_at    timestamptz
);
CREATE INDEX idx_agents_site_id ON agents(site_id);

CREATE TABLE agent_enrollment_tokens (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id           uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    token_hash        text NOT NULL,
    created_by        uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    expires_at        timestamptz NOT NULL,
    used_at           timestamptz,
    used_by_agent_id  uuid REFERENCES agents(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX idx_agent_enrollment_tokens_hash ON agent_enrollment_tokens(token_hash);

CREATE TABLE agent_heartbeats (
    time          timestamptz NOT NULL DEFAULT now(),
    agent_id      uuid NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    status        text NOT NULL,
    queued_items  int NOT NULL DEFAULT 0,
    cpu_pct       numeric,
    mem_pct       numeric
);
CREATE INDEX idx_agent_heartbeats_agent_time ON agent_heartbeats(agent_id, time DESC);

CREATE TABLE ping_tests (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id         uuid NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    target           text NOT NULL,
    protocol         text NOT NULL CHECK (protocol IN ('icmp', 'tcp', 'http', 'dns')),
    latency_ms_p50   numeric,
    latency_ms_p95   numeric,
    latency_ms_p99   numeric,
    jitter_ms        numeric,
    packet_loss_pct  numeric,
    executed_at      timestamptz NOT NULL,
    idempotency_key  text NOT NULL
);
-- Reenvio idempotente (Seção 3): o mesmo lote reenviado após reconexão não
-- duplica resultados.
CREATE UNIQUE INDEX idx_ping_tests_idempotency ON ping_tests(agent_id, idempotency_key);
CREATE INDEX idx_ping_tests_agent_time ON ping_tests(agent_id, executed_at DESC);
