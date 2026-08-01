# 03 — Fluxo de Dados

## 3.1 Telemetria de rotina (heartbeat, métricas UniFi, testes programados)

```mermaid
sequenceDiagram
    participant GW as UniFi Gateway/APs/Switch
    participant UP as UniFiIntegrationProvider (no agente)
    participant Agent as Agente local
    participant Q as Fila local (buffer offline)
    participant API as Backend API
    participant DB as PostgreSQL/TimescaleDB
    participant Redis as Redis Pub/Sub
    participant Web as Web/iOS (assinantes)

    loop a cada intervalo configurado
        UP->>GW: Consulta Network API local (dispositivos, clientes, eventos)
        GW-->>UP: Estado atual
        UP->>Agent: Normaliza para modelo interno
        Agent->>Agent: Executa testes ativos (ping/DNS/HTTP) conforme agenda
        Agent->>Q: Enfileira métricas/resultados (idempotency key)
    end
    Agent->>API: POST /api/v1/agents/{id}/telemetry (TLS, comprimido, em lote)
    alt Internet indisponível
        API--xAgent: falha de conexão
        Agent->>Q: mantém no buffer local (limite de armazenamento)
        Agent->>Agent: backoff exponencial e retry
    else Sucesso
        API->>DB: Persiste série temporal + eventos
        API->>Redis: Publica no canal do site
        Redis->>Web: Push via WebSocket/SSE
        API-->>Agent: 200 OK (ack idempotente, drena buffer)
    end
```

Pontos de projeto:

- **Idempotência**: cada lote enviado carrega uma `idempotency key` derivada de
  `(agent_id, sequence_number)`. Reenvio após reconexão não duplica métricas —
  condição obrigatória da Seção 3 ("reenvio idempotente").
- **Buffer com limite**: o agente descarta as amostras mais antigas primeiro quando o
  buffer local atinge o teto configurado, e registra isso como evento de
  "perda de buffer local" — nunca falha silenciosamente.
- **Nenhum dado bruto de tráfego** sai da LAN (Seção 19): o que trafega para o backend
  são métricas agregadas/estruturadas (latência, contagem, estado), nunca payload de
  pacotes.

## 3.2 Teste sob demanda disparado pelo usuário (ex.: "rodar speed test agora")

```mermaid
sequenceDiagram
    participant User as Usuário (iOS/Web)
    participant API as Backend API
    participant Redis as Redis (fila de comando)
    participant Agent as Agente local
    participant DB as PostgreSQL

    User->>API: POST /api/v1/tests {type: speedtest, site_id}
    API->>DB: Cria registro de teste (status: pending)
    API->>Redis: Publica comando no canal do agente do site
    Redis->>Agent: Comando recebido (long-poll ou push do agente)
    Agent->>Agent: Executa teste (download/upload/latência/jitter)
    Agent->>API: POST resultado (correlacionado ao test_id)
    API->>DB: Atualiza registro (status: completed)
    API->>Redis: Publica resultado no canal do site
    Redis->>User: Atualização em tempo real (WebSocket/SSE)
```

O agente não expõe um servidor de comandos (isso violaria "sem porta de entrada");
ele consulta/assina o backend para saber se há comandos pendentes, usando a mesma
conexão outbound de telemetria.

## 3.3 Levantamento LiDAR (Spatial WiFi Survey)

```mermaid
sequenceDiagram
    participant App as App iOS (ARKit/RealityKit)
    participant Loc as Core Location / NEHotspotNetwork
    participant API as Backend API
    participant Agent as Agente local (site)
    participant UniFi as UniFi (via agente)
    participant Worker as Worker (processamento espacial)
    participant DB as PostgreSQL

    App->>App: Sessão ARKit ativa, usuário caminha pelo ambiente
    loop a cada amostra (intervalo configurável)
        App->>Loc: Lê BSSID/SSID da rede atual (se autorizado)
        App->>API: GET métricas atuais do site (latência gateway, jitter, perda, DNS)
        API->>Agent: (já coletado via telemetria de rotina, servido do cache)
        Agent->>UniFi: RSSI/SNR/canal/PHY do cliente atual (por MAC do iPhone)
        UniFi-->>API: dados do cliente (via ingestão de rotina)
        API-->>App: snapshot de métricas correlacionáveis por timestamp
        App->>App: Cria SpatialSample {posição 3D, piso, timestamp, métricas, fonte}
    end
    App->>API: POST /api/v1/surveys/{id}/samples (lote ao final ou incremental)
    API->>DB: Persiste SpatialSample + malha exportada (mesh anchors simplificados)
    API->>Worker: Job "processar levantamento"
    Worker->>DB: Gera heatmap por piso, detecta zonas de baixa cobertura
    Worker->>DB: Persiste Heatmap + recomendações (com evidência/confiança)
```

Ponto crítico: a métrica de rádio (RSSI/SNR/canal) usada em cada amostra **não é lida
pelo iPhone** — é o valor que o UniFi reporta para o cliente (o próprio iPhone,
identificado pelo MAC) no momento mais próximo do timestamp da amostra. O app apenas
identifica *a qual AP/BSSID está associado* (via `NEHotspotNetwork`, ver limitações) e
correlaciona posição 3D + timestamp com o que o UniFi já estava reportando. Isso é
verificado com uma tolerância de sincronismo (ex.: ±5s); fora dessa janela, a amostra é
marcada com `source: estimated` e confiança reduzida, nunca apresentada como medição
direta.

## 3.4 Alertas e notificação push

```mermaid
sequenceDiagram
    participant Worker as Worker (regras/anomalias)
    participant DB as PostgreSQL
    participant APNsProvider as Provider APNs (backend próprio)
    participant APNs as Apple Push Notification service
    participant iOS as App iOS
    participant Web as Web (in-app + e-mail/webhook)

    Worker->>DB: Detecta condição de alerta (regra ou anomalia)
    Worker->>DB: Cria Alert (severidade, deduplicação, agrupamento)
    Worker->>APNsProvider: Solicita push (device tokens do usuário/site)
    APNsProvider->>APNs: Envia notificação (HTTP/2, JWT com chave .p8)
    APNs-->>iOS: Notificação entregue (app em qualquer estado)
    Worker->>Web: Publica no canal em tempo real (in-app)
    Worker->>Worker: Dispara e-mail/webhook conforme configuração do canal
```

Decisão registrada em ADR-009: notificações críticas usam **APNs via provider próprio
do backend** (chave `.p8`, JWT, biblioteca HTTP/2 nativa), não CloudKit
`CKQuerySubscription` — essa via é conhecida por não ser confiável em produção quando o
app está fechado. Local notifications/polling não substituem push real para alertas
como "Internet caiu" ou "AP offline".
