-- Fase 5 (início): comandos sob demanda disparados pelo usuário e executados
-- pelo agente (docs/architecture/03-fluxo-de-dados.md §3.2 "Teste sob demanda").
--
-- O design original (§3.2) descreve Redis como fila de comando/pub-sub. Esta
-- migração usa uma fila baseada em Postgres em vez disso — decisão registrada
-- em ADR-011 (docs/architecture/adr/ADR-011-fila-de-comando-via-postgres.md).
CREATE TABLE agent_commands (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id       uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    agent_id      uuid NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    requested_by  uuid NOT NULL REFERENCES users(id),
    type          text NOT NULL CHECK (type IN ('ping')),
    params        jsonb NOT NULL DEFAULT '{}'::jsonb,
    status        text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'claimed', 'completed', 'failed')),
    result        jsonb,
    error         text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    claimed_at    timestamptz,
    completed_at  timestamptz
);
CREATE INDEX idx_agent_commands_agent_status ON agent_commands(agent_id, status, created_at);
CREATE INDEX idx_agent_commands_site_time ON agent_commands(site_id, created_at DESC);
