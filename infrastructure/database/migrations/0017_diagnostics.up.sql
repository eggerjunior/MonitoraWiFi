-- Fase 7 (continuação): motor de correlação/diagnóstico + recomendações +
-- relatórios. Constrói em cima de `anomalies` (migração 0007) — um
-- diagnóstico nunca é gravado sem pelo menos uma anomalia real como
-- evidência (mesmo princípio que já vale para anomalias: nunca reportar
-- sem histórico/evidência suficiente).
CREATE TABLE diagnoses (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id       uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    category      text NOT NULL CHECK (category IN ('internet_slow', 'wifi_slow')),
    summary       text NOT NULL,
    confidence    numeric NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    impact        text NOT NULL CHECK (impact IN ('low', 'medium', 'high')),
    risk          text NOT NULL CHECK (risk IN ('low', 'medium', 'high')),
    evidence      jsonb NOT NULL,
    window_start  timestamptz NOT NULL,
    window_end    timestamptz NOT NULL,
    detected_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_diagnoses_site_time ON diagnoses(site_id, window_end DESC);
-- window_end é o observed_at mais recente entre as anomalias usadas como
-- evidência (não o horário de execução do worker) — duas execuções que
-- olham para a mesma evidência recalculam o mesmo window_end e colidem
-- aqui de propósito (idempotência), igual ao padrão de idx_anomalies_dedup.
CREATE UNIQUE INDEX idx_diagnoses_dedup ON diagnoses(site_id, category, window_end);

CREATE TABLE recommendations (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    diagnosis_id  uuid NOT NULL REFERENCES diagnoses(id) ON DELETE CASCADE,
    site_id       uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    action        text NOT NULL,
    confidence    numeric NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    impact        text NOT NULL CHECK (impact IN ('low', 'medium', 'high')),
    risk          text NOT NULL CHECK (risk IN ('low', 'medium', 'high')),
    evidence      jsonb NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_recommendations_site_time ON recommendations(site_id, created_at DESC);
-- Uma recomendação por diagnóstico nesta fatia (MVP) — simplifica upsert e
-- evita recomendações duplicadas/redundantes para o mesmo diagnóstico.
CREATE UNIQUE INDEX idx_recommendations_one_per_diagnosis ON recommendations(diagnosis_id);

-- REPORT (docs/architecture/05-modelo-dados.md §7), gerado sob demanda pelo
-- backend (não pré-gerado/armazenado em blob storage — não há infraestrutura
-- de armazenamento de objetos neste projeto ainda; conteúdo vai inteiro em
-- `content`, formato JSON só nesta fatia).
CREATE TABLE reports (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id        uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    kind           text NOT NULL CHECK (kind IN ('diagnostics_summary')),
    period_start   timestamptz NOT NULL,
    period_end     timestamptz NOT NULL,
    content        jsonb NOT NULL,
    generated_by   uuid REFERENCES users(id) ON DELETE SET NULL,
    generated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_reports_site_time ON reports(site_id, generated_at DESC);
