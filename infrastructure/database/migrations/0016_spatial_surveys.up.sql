-- Fase 6: levantamento espacial (Spatial WiFi Survey) — primeira fatia real.
-- Esquema deliberadamente menor que o ER especulativo original de
-- docs/architecture/05-modelo-dados.md §6 (bssid/rssi/canal/PHY rate por
-- amostra): esses campos não são obtidos pelo iPhone (ver capability
-- matrix — RSSI/canal indisponíveis via API pública do iOS). O que a
-- captura real produz é posição (ARKit world tracking) + qualidade de rede
-- medida do próprio ponto (RTT ao backend, via Network.framework) — este
-- esquema reflete exatamente isso, não o que foi imaginado antes de um
-- protótipo real.
CREATE TABLE spatial_surveys (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id       uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    created_by    uuid NOT NULL REFERENCES users(id),
    name          text NOT NULL,
    device_model  text NOT NULL,
    lidar_used    boolean NOT NULL,
    started_at    timestamptz NOT NULL,
    finished_at   timestamptz NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_spatial_surveys_site ON spatial_surveys(site_id);

CREATE TABLE spatial_survey_samples (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id       uuid NOT NULL REFERENCES spatial_surveys(id) ON DELETE CASCADE,
    position_x      double precision NOT NULL,
    position_y      double precision NOT NULL,
    position_z      double precision NOT NULL,
    ssid            text,
    bssid           text,
    rtt_ms          double precision,
    is_expensive    boolean NOT NULL,
    is_constrained  boolean NOT NULL,
    interface_type  text NOT NULL CHECK (interface_type IN ('wifi', 'cellular', 'wired', 'other')),
    captured_at     timestamptz NOT NULL
);
CREATE INDEX idx_spatial_survey_samples_survey ON spatial_survey_samples(survey_id);
