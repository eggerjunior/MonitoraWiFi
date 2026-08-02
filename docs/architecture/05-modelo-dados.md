# 05 — Modelo Inicial de Banco de Dados

Convenções globais (Seção 16 do documento-fonte):

- Chave primária: `UUID` (gerado com `gen_random_uuid()`, extensão `pgcrypto`).
- Todo timestamp armazenado em **UTC** (`timestamptz`); conversão para o fuso do site
  acontece na camada de apresentação, usando `Site.timezone`.
- Toda tabela de negócio carrega `organization_id` e, onde aplicável, `site_id` —
  nunca inferido implicitamente via join; sempre uma coluna própria, indexada, para
  permitir isolamento multi-tenant no nível de query (ver threat model §2.2).
- Séries temporais de alto volume (`measurements`, testes, heartbeats) são
  hypertables do **TimescaleDB** quando disponível no ambiente de deploy; caso não
  esteja (ex.: gerenciado sem extensão Timescale), particionamento nativo do
  PostgreSQL por `time` (partição mensal) como equivalente funcional — decisão em
  ADR-004.
- Retenção: política por tipo de série temporal, configurável por organização,
  aplicada por job do worker (drop/archive de partições antigas), nunca por exclusão
  linha a linha em tabela não particionada.

## 1. Identidade, organização e local físico

```mermaid
erDiagram
    ORGANIZATION ||--o{ SITE : possui
    ORGANIZATION ||--o{ USER : emprega
    SITE ||--o{ LOCATION : contem
    LOCATION ||--o{ FLOOR : contem
    FLOOR ||--o{ ROOM : contem
    USER }o--o{ SITE : "acesso via RBAC"

    ORGANIZATION {
        uuid id PK
        text name
        text plan_tier
        timestamptz created_at
    }
    SITE {
        uuid id PK
        uuid organization_id FK
        text name
        text timezone
        jsonb address
        timestamptz created_at
    }
    USER {
        uuid id PK
        uuid organization_id FK
        text email
        text role "Owner|Administrator|Operator|Viewer|Auditor"
        text auth_method "password|passkey"
        timestamptz mfa_enrolled_at
        timestamptz created_at
    }
    LOCATION {
        uuid id PK
        uuid site_id FK
        text name
        numeric area_sqm
        timestamptz created_at
    }
    FLOOR {
        uuid id PK
        uuid location_id FK
        int level_index
        text name
    }
    ROOM {
        uuid id PK
        uuid floor_id FK
        text name
        jsonb polygon_2d
    }
```

## 2. Inventário UniFi

```mermaid
erDiagram
    SITE ||--o{ UNIFI_CONSOLE : gerencia
    UNIFI_CONSOLE ||--o{ GATEWAY : possui
    UNIFI_CONSOLE ||--o{ ACCESS_POINT : possui
    UNIFI_CONSOLE ||--o{ SWITCH : possui
    GATEWAY ||--o{ WAN : possui
    ACCESS_POINT ||--o{ RADIO : possui
    SWITCH ||--o{ SWITCH_PORT : possui

    UNIFI_CONSOLE {
        uuid id PK
        uuid site_id FK
        text base_url
        text unifi_os_version
        text unifi_network_version
        jsonb capabilities
        text integration_mode "network_api|site_manager|snmp|legacy"
        timestamptz last_capability_check_at
    }
    GATEWAY {
        uuid id PK
        uuid unifi_console_id FK
        text model
        macaddr mac
        inet ip_address
        text firmware_version
        timestamptz uptime_since
        text state
    }
    WAN {
        uuid id PK
        uuid gateway_id FK
        text label "primaria|secundaria"
        inet public_ip
        boolean is_active
        timestamptz last_failover_at
        int failover_count_24h
    }
    ACCESS_POINT {
        uuid id PK
        uuid unifi_console_id FK
        text model
        macaddr mac
        inet ip_address
        text firmware_version
        text state
        uuid uplink_switch_port_id FK
    }
    RADIO {
        uuid id PK
        uuid access_point_id FK
        text band "2.4|5|6"
        int channel
        int channel_width_mhz
        numeric tx_power_dbm
        boolean dfs_enabled
    }
    SWITCH {
        uuid id PK
        uuid unifi_console_id FK
        text model
        macaddr mac
        text firmware_version
        text state
    }
    SWITCH_PORT {
        uuid id PK
        uuid switch_id FK
        int port_number
        text negotiated_speed
        boolean poe_enabled
        numeric poe_watts
        text vlan_native
    }
```

## 3. Redes lógicas, clientes e sessões

```mermaid
erDiagram
    UNIFI_CONSOLE ||--o{ NETWORK_VLAN : define
    NETWORK_VLAN ||--o{ SSID : expõe
    SSID ||--o{ CLIENT_SESSION : atende
    CLIENT ||--o{ CLIENT_SESSION : gera
    CLIENT ||--o{ DEVICE_IDENTITY : identificado_por
    ACCESS_POINT ||--o{ CLIENT_SESSION : serve

    NETWORK_VLAN {
        uuid id PK
        uuid unifi_console_id FK
        text name
        int vlan_id
        cidr subnet
        boolean guest_network
        boolean isolation_enabled
    }
    SSID {
        uuid id PK
        uuid network_vlan_id FK
        text name
        text security "wpa2|wpa3|open"
        boolean hidden
        boolean band_steering_enabled
        boolean fast_roaming_enabled
    }
    CLIENT {
        uuid id PK
        uuid site_id FK
        macaddr mac
        text hostname
        text oui_vendor
        text device_type_guess
        boolean is_fixed_ip
        timestamptz first_seen_at
        timestamptz last_seen_at
    }
    DEVICE_IDENTITY {
        uuid id PK
        uuid client_id FK
        text source "unifi|user_declared"
        text label
    }
    CLIENT_SESSION {
        uuid id PK
        uuid client_id FK
        uuid access_point_id FK
        uuid ssid_id FK
        text radio_band
        int channel
        text wifi_standard
        int rssi_dbm
        numeric snr_db
        numeric phy_rate_mbps
        timestamptz connected_at
        timestamptz disconnected_at
        text source "unifi_local_api|unifi_site_manager"
    }
```

## 4. Séries temporais e testes ativos (hypertables)

```mermaid
erDiagram
    SITE ||--o{ TIME_SERIES_METRIC : produz
    AGENT ||--o{ PING_TEST : executa
    AGENT ||--o{ SPEED_TEST : executa
    AGENT ||--o{ DNS_QUERY : executa
    AGENT ||--o{ TRACEROUTE : executa
    TRACEROUTE ||--o{ TRACEROUTE_HOP : contem
    AGENT ||--o{ PORT_SCAN : executa
    AGENT ||--o{ CERTIFICATE_CHECK : executa
    AGENT ||--o{ HTTP_REQUEST_TEST : executa

    TIME_SERIES_METRIC {
        timestamptz time
        uuid site_id FK
        uuid entity_id "AP/switch/client/gateway"
        text entity_type
        text metric_name
        double value
        text source
    }
    MEASUREMENT {
        uuid id PK
        uuid site_id FK
        text kind
        timestamptz captured_at
        jsonb payload
        text source
    }
    PING_TEST {
        uuid id PK
        uuid agent_id FK
        text target
        text protocol "icmp|tcp|http|dns"
        numeric latency_ms_p50
        numeric latency_ms_p95
        numeric latency_ms_p99
        numeric jitter_ms
        numeric packet_loss_pct
        timestamptz executed_at
    }
    SPEED_TEST {
        uuid id PK
        uuid agent_id FK
        text mode "internet|lan|http"
        numeric download_mbps
        numeric upload_mbps
        numeric latency_ms
        numeric jitter_ms
        numeric loaded_latency_ms
        numeric bufferbloat_score
        timestamptz executed_at
    }
    DNS_QUERY {
        uuid id PK
        uuid agent_id FK
        text query_name
        text record_type
        text resolver
        numeric resolution_ms
        jsonb answers
        timestamptz executed_at
    }
    TRACEROUTE {
        uuid id PK
        uuid agent_id FK
        text target
        text protocol
        timestamptz executed_at
    }
    TRACEROUTE_HOP {
        uuid id PK
        uuid traceroute_id FK
        int hop_index
        inet hop_ip
        text asn
        text reverse_dns
        numeric latency_ms
        numeric loss_pct
    }
    PORT_SCAN {
        uuid id PK
        uuid agent_id FK
        text target
        int4range port_range
        jsonb open_ports
        timestamptz executed_at
        text authorized_by
    }
    CERTIFICATE_CHECK {
        uuid id PK
        uuid agent_id FK
        text hostname
        text common_name
        jsonb sans
        text issuer
        timestamptz not_after
        int days_remaining
    }
    HTTP_REQUEST_TEST {
        uuid id PK
        uuid agent_id FK
        text method
        text url
        int status_code
        numeric dns_ms
        numeric connect_ms
        numeric tls_ms
        numeric ttfb_ms
        numeric total_ms
    }
```

## 5. Agente, eventos, alertas, incidentes, automações

```mermaid
erDiagram
    SITE ||--o{ AGENT : hospeda
    AGENT ||--o{ AGENT_HEARTBEAT : envia
    SITE ||--o{ EVENT : gera
    EVENT ||--o{ ALERT : dispara
    ALERT ||--o{ INCIDENT : agrupa
    SITE ||--o{ AUTOMATION : configura
    AUTOMATION ||--o{ EVENT : "gatilho"

    AGENT {
        uuid id PK
        uuid site_id FK
        text hostname
        text version
        text platform "linux_amd64|linux_arm64|macos"
        text auth_method "mtls|rotating_credential"
        timestamptz enrolled_at
        timestamptz last_seen_at
    }
    AGENT_HEARTBEAT {
        timestamptz time
        uuid agent_id FK
        text status
        int queued_items
        numeric cpu_pct
        numeric mem_pct
    }
    EVENT {
        uuid id PK
        uuid site_id FK
        text category
        text severity
        text summary
        jsonb evidence
        timestamptz occurred_at
        text source
    }
    ALERT {
        uuid id PK
        uuid event_id FK
        text rule_name
        text severity
        text status "open|acknowledged|resolved|silenced"
        uuid assigned_to FK
        timestamptz created_at
        timestamptz resolved_at
    }
    INCIDENT {
        uuid id PK
        uuid site_id FK
        text title
        text status
        timestamptz opened_at
        timestamptz closed_at
    }
    AUTOMATION {
        uuid id PK
        uuid site_id FK
        text trigger_type
        jsonb conditions
        jsonb actions
        boolean enabled
        boolean requires_admin_approval
    }
```

## 6. Spatial WiFi Survey (LiDAR)

> **Nota (2026-08-02)**: o ER abaixo é o desenho especulativo original
> (Fase 0), tratado desde então como hipótese de partida, não contrato —
> ver `06-roadmap.md` Fase 6. O esquema **realmente implementado**
> (migração `0016_spatial_surveys`) é deliberadamente mais simples:
> `spatial_surveys` (id, site_id, created_by, name, device_model,
> lidar_used, started_at, finished_at) e `spatial_survey_samples` (id,
> survey_id, position_x/y/z, ssid, bssid, rtt_ms, is_expensive,
> is_constrained, interface_type, captured_at) — sem `bssid`/`radio_band`/
> `channel`/`rssi_dbm`/`snr_db`/`phy_rate_mbps`/`confidence` por amostra,
> sem `FLOOR`/`MESH_ASSET`/`HEATMAP`. Motivo: nenhuma fonte real disponível
> hoje expõe RSSI/canal/PHY rate por cliente (nem a Network API local do
> UniFi, nem o iOS) — persistir esses campos seria coluna sem dado real
> pra popular. Revisitar este ER se uma dessas fontes passar a expor esses
> campos (nova versão da Network API do UniFi, ou adaptador SNMP/legado
> habilitado).

```mermaid
erDiagram
    FLOOR ||--o{ SPATIAL_SURVEY : recebe
    SPATIAL_SURVEY ||--o{ SPATIAL_SAMPLE : contem
    SPATIAL_SURVEY ||--o{ MESH_ASSET : contem
    SPATIAL_SURVEY ||--o{ HEATMAP : gera

    SPATIAL_SURVEY {
        uuid id PK
        uuid floor_id FK
        text device_model
        boolean lidar_used
        timestamptz started_at
        timestamptz finished_at
        text status
    }
    SPATIAL_SAMPLE {
        uuid id PK
        uuid survey_id FK
        point3d position
        timestamptz captured_at
        macaddr bssid
        uuid unifi_ap_id FK
        text radio_band
        int channel
        int rssi_dbm
        numeric snr_db
        numeric phy_rate_mbps
        timestamptz network_metrics_at
        text network_metrics_source
        numeric time_sync_delta_seconds
        numeric confidence
    }
    MESH_ASSET {
        uuid id PK
        uuid survey_id FK
        text asset_type "mesh|plane|plant_import"
        text storage_uri
        bigint size_bytes
    }
    HEATMAP {
        uuid id PK
        uuid survey_id FK
        text metric_name
        text floor_id FK
        text storage_uri
        text interpolation_method
        timestamptz generated_at
    }
```

## 7. Relatórios, integrações e auditoria

```mermaid
erDiagram
    SITE ||--o{ REPORT : gera
    ORGANIZATION ||--o{ INTEGRATION : configura
    INTEGRATION ||--o{ CREDENTIAL_REFERENCE : referencia
    ORGANIZATION ||--o{ AUDIT_LOG : registra

    REPORT {
        uuid id PK
        uuid site_id FK
        text kind
        text format "pdf|csv|json"
        text storage_uri
        timestamptz generated_at
        uuid generated_by FK
    }
    INTEGRATION {
        uuid id PK
        uuid organization_id FK
        text kind "unifi_network_api|unifi_site_manager|snmp|syslog|legacy"
        boolean enabled
        jsonb config_non_secret
        timestamptz created_at
    }
    CREDENTIAL_REFERENCE {
        uuid id PK
        uuid integration_id FK
        text secret_store "vault|kms|env"
        text secret_key_name
        timestamptz rotated_at
        timestamptz expires_at
    }
    AUDIT_LOG {
        uuid id PK
        uuid organization_id FK
        uuid actor_user_id FK
        text action
        text resource_type
        uuid resource_id
        jsonb diff
        inet actor_ip
        timestamptz created_at
    }
```

## Nota sobre `CredentialReference`

Nenhuma tabela armazena segredo em texto claro. `CredentialReference` guarda apenas
o **nome/localização** do segredo em um cofre externo (Vault, KMS ou variável de
ambiente do agente/backend) — cumpre "segredos fora do código-fonte" e "armazenamento
seguro" (Seção 2.2) desde o desenho do schema, não como reforço posterior.

## Índices e particionamento (a formalizar nas migrações da Fase 1)

- `time_series_metric`, `agent_heartbeat`: hypertable por `time`, chunk padrão de 1 a 7
  dias conforme volume real observado.
- `client_session`, `event`, `audit_log`: índice composto `(site_id, created_at desc)`
  ou equivalente, para os padrões de consulta mais comuns do dashboard (janelas de
  tempo por site).
- `ping_test`, `speed_test`, `traceroute`: índice em `(agent_id, executed_at desc)`.

Este documento é o modelo **inicial** — o schema definitivo com tipos exatos,
constraints e migrações versionadas (`infrastructure/database`) é produzido na
Fase 1, após confirmação dos campos reais disponíveis na API UniFi da instalação
(ver `docs/unifi/verificacoes-pendentes-instalacao.md`), que pode adicionar ou
remover colunas de `RADIO`, `SWITCH_PORT` e `CLIENT_SESSION`.
