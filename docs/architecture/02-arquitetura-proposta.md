# 02 — Arquitetura Proposta

## 2.1 Visão em componentes

```mermaid
graph TB
    subgraph Cliente["Clientes"]
        iOS["App iOS/iPadOS<br/>Swift/SwiftUI/ARKit"]
        Web["Web App<br/>Next.js/React/TS"]
    end

    subgraph Cloud["Backend Central (cloud ou on-prem do cliente)"]
        API["API REST + WS/SSE<br/>Go · OpenAPI 3.1"]
        Worker["Worker assíncrono<br/>Go"]
        DB[("PostgreSQL +<br/>TimescaleDB")]
        Redis[("Redis<br/>cache/filas/coordenação")]
        Otel["OpenTelemetry<br/>collector"]
    end

    subgraph LAN["LAN do cliente (por site)"]
        Agent["Agente local<br/>Go · outbound-only"]
        UniFiProvider["UniFiIntegrationProvider"]
        Gateway["UniFi Cloud Gateway Max"]
        APs["4x U7 Pro"]
        Switch["Switch Lite 16 PoE"]
        Clients["Clientes de rede"]
    end

    subgraph UniFiCloud["Serviços Ubiquiti (opcional)"]
        SiteManager["UniFi Site Manager API"]
    end

    iOS <-->|"HTTPS + WS/SSE<br/>OpenAPI contracts"| API
    Web <-->|"HTTPS + WS/SSE<br/>OpenAPI contracts"| API
    API --> DB
    API --> Redis
    Worker --> DB
    Worker --> Redis
    API -->|"eventos/jobs"| Worker
    API -. traces/metrics/logs .-> Otel
    Worker -. traces/metrics/logs .-> Otel

    Agent -->|"TLS outbound<br/>mTLS ou credencial rotacionável"| API
    Agent --> UniFiProvider
    UniFiProvider -->|"Network API local"| Gateway
    UniFiProvider -.->|"SNMP (fallback)"| Gateway
    UniFiProvider -.->|"Syslog"| Gateway
    Gateway --- APs
    Gateway --- Switch
    Switch --- Clients
    APs --- Clients

    API -.->|"opcional, multi-site"| SiteManager
```

Pontos de leitura obrigatória deste diagrama:

- O **agente local é a única ponte** entre a LAN do cliente e o backend. A conexão é
  sempre iniciada de dentro para fora (outbound), nunca o contrário — cumpre a
  exigência de "não exigir abertura de portas de entrada" (Seção 3).
- O `UniFiIntegrationProvider` roda **dentro do agente**, não no backend central,
  porque é o agente que está na mesma LAN do console UniFi e pode falar com a Network
  API local sem expor essa API à internet.
- A UniFi Site Manager API (cloud) é **opcional** e usada apenas quando o usuário quer
  visão agregada multi-site via Ubiquiti; não é dependência obrigatória do produto.
- iOS e Web são clientes simétricos da mesma API — nenhuma lógica de negócio "mora"
  exclusivamente em um dos dois; o que muda é a superfície de interação (LiDAR só
  existe no iOS, por depender de câmera+LiDAR do aparelho).

## 2.2 Por que o agente local existe (e por que não dá para eliminá-lo)

Alternativa descartada: o backend central falar diretamente com o console UniFi do
cliente. Isso exigiria abrir porta de entrada na rede residencial ou VPN permanente
site-to-cloud — rejeitado explicitamente pela Seção 3 ("O agente local não poderá
exigir a abertura de portas de entrada"). Também exigiria que testes ativos (ping,
traceroute, iPerf3, port scan, ARP) partissem de fora da LAN, o que muda
completamente o significado da medição (latência à borda da operadora, não latência
real da LAN/Wi-Fi do cliente). Por isso o agente roda dentro da rede e "empurra"
dados/resultados para fora.

## 2.3 Camadas do backend

```mermaid
graph LR
    subgraph API["apps/api (Go)"]
        REST["Handlers REST<br/>versionados /api/v1"]
        RT["WebSocket/SSE<br/>hub de tempo real"]
        AuthZ["AuthN/RBAC"]
        Ingest["Ingestão de<br/>telemetria do agente"]
    end
    subgraph Worker["apps/worker (Go)"]
        Anomaly["Motor de anomalias"]
        Correl["Motor de correlação/diagnóstico"]
        Reports["Geração de relatórios"]
        Spatial["Processamento de<br/>levantamentos LiDAR"]
        Notify["Disparo de alertas<br/>(push/e-mail/webhook)"]
    end
    DB[("PostgreSQL/TimescaleDB")]
    Redis[("Redis")]

    Ingest --> DB
    Ingest --> Redis
    Redis --> Anomaly
    Redis --> Correl
    Anomaly --> DB
    Correl --> DB
    Reports --> DB
    Spatial --> DB
    Notify --> DB
    RT --> Redis
```

O worker é deliberadamente separado do processo de API: ingestão de alta frequência
(métricas do agente, heartbeats) não pode competir por CPU/latência com processamento
pesado (recalcular baseline de anomalias, processar malha LiDAR, gerar PDF de
relatório). Comunicação entre API e worker via fila no Redis (jobs leves) — sem
introduzir um broker de mensagens dedicado nesta fase (YAGNI até haver evidência de
necessidade; revisitar se o volume de eventos exigir).

## 2.4 Multi-tenancy (organização → site)

```mermaid
graph TB
    Org["Organization<br/>(conta/cliente)"] --> Site1["Site: Residência Egger"]
    Org --> SiteN["Site: ... (futuro)"]
    Site1 --> Console1["UniFiConsole"]
    Site1 --> AgentInst1["Agent (instância)"]
    Console1 --> GW1["Gateway"]
    Console1 --> AP1["AccessPoint x4"]
    Console1 --> SW1["Switch"]
    GW1 --> WAN1["WAN primária"]
    GW1 --> WAN2["WAN secundária"]
```

Toda entidade de negócio (métrica, evento, cliente, AP...) pertence a exatamente um
`Site`, que pertence a exatamente uma `Organization`. RBAC (Seção 18) é avaliado no
nível de organização e, quando necessário, restrito por site. Isso é o que permite
"uma ou várias instalações... vários sites UniFi... diferentes gateways" (Seção 1)
sem redesenho futuro — ver ADR-002.

## 2.5 Tempo real

WebSocket é o transporte primário para o dashboard (bidirecional, permite comandos
como "rodar teste agora"); SSE é aceito como fallback em redes/proxies corporativos
que bloqueiam WebSocket, servindo o mesmo canal de eventos em modo somente-leitura.
O hub de tempo real vive no processo de API e usa Redis Pub/Sub para escalar
horizontalmente (múltiplas instâncias de API assinam o mesmo canal por site).

## 2.6 Design system honesto (provenance badge)

Todo componente de UI (iOS e Web) que exibe uma métrica de rede deve aceitar um campo
`source` obrigatório do contrato (`packages/contracts`), com um enum fechado:
`unifi_local_api | unifi_site_manager | agent_icmp | agent_tcp | agent_udp |
agent_dns | agent_http | snmp | arkit | estimated | user_declared | unavailable`.
Quando `unavailable`, o componente renderiza o estado "Indisponível" com motivo e ação
sugerida — nunca esconde o card nem preenche com traço. Este contrato é o mecanismo
técnico que impõe o princípio da Seção 2.1 do documento original em toda a stack,
em vez de depender de disciplina manual de cada desenvolvedor.
