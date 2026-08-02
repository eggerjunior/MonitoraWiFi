-- Fase 3: topologia dispositivo→dispositivo (uplink), confirmada em
-- 2026-08-02 contra a instalação real (GET .../devices/{id}, campo
-- `uplink.deviceId`). Nullable: o dispositivo raiz (gateway) não tem
-- uplink.
ALTER TABLE unifi_devices ADD COLUMN uplink_device_id text;
