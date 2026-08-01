-- Fase 3 (início): inventário de dispositivos/clientes UniFi sincronizado
-- pelo agente (ADR-007, NetworkAPIAdapter). Estado atual, não série
-- temporal — cada sincronização substitui o inventário anterior do site
-- (ver store.PostgresUniFiDevices/Clients.Replace).
CREATE TABLE unifi_devices (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id          uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    external_id      text NOT NULL,
    mac_address      text NOT NULL,
    ip_address       text,
    name             text NOT NULL,
    model            text NOT NULL,
    state            text NOT NULL,
    firmware_version text,
    features         jsonb NOT NULL DEFAULT '[]'::jsonb,
    interfaces       jsonb NOT NULL DEFAULT '[]'::jsonb,
    synced_at        timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_unifi_devices_site_external ON unifi_devices(site_id, external_id);

CREATE TABLE unifi_clients (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id          uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    external_id      text NOT NULL,
    type             text NOT NULL,
    name             text,
    ip_address       text,
    mac_address      text NOT NULL,
    connected_at     timestamptz,
    uplink_device_id text,
    synced_at        timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_unifi_clients_site_external ON unifi_clients(site_id, external_id);
