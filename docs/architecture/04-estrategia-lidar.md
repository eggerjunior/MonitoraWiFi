# 04 — Estratégia do Spatial WiFi Survey (LiDAR)

## Princípio inegociável

**O LiDAR nunca mede rádio.** Ele mede geometria (posição 3D, planos, volumes). Todo
valor de qualidade de rede exibido sobre o mapa 3D vem do UniFi (via agente) ou do
próprio agente (ping/DNS/throughput), correlacionado por *timestamp + posição*, nunca
inferido a partir de dados de câmera/profundidade. Isso está confirmado como
tecnicamente necessário: `NEHotspotNetwork.signalStrength` não é confiável em iOS
(ver `01-limitacoes-tecnicas.md` §1.3) — mesmo que quiséssemos usar sinal "do
telefone", a API não entrega isso de forma utilizável.

## Duas fontes de dado, um único ponto de junção

1. **Fonte espacial** (dispositivo iOS): pose 3D (`simd_float4x4`) da câmera a cada
   amostra, malha do ambiente (`ARMeshAnchor` quando LiDAR disponível), plano de piso.
2. **Fonte de rede** (backend, agregando UniFi + agente): para o MAC do próprio
   iPhone conectado à rede, qual AP/BSSID/rádio/canal/RSSI/SNR/PHY rate o UniFi reporta
   *naquele momento*; e, em paralelo, latência ao gateway, jitter, perda, throughput
   local, medidos pelo agente da mesma forma que qualquer outro teste de rotina.

O ponto de junção é o **timestamp**, com uma janela de tolerância configurável
(padrão: 5 segundos). Cada `SpatialSample` grava explicitamente a diferença de tempo
entre a amostra espacial e a métrica de rede usada; se exceder a tolerância, o dado é
marcado com confiança reduzida e a UI mostra isso (não silencia a discrepância).

## Fluxo técnico (ver também `03-fluxo-de-dados.md` §3.3)

1. App abre sessão `ARSession` com `ARWorldTrackingConfiguration`
   (+ `sceneReconstruction = .meshWithClassification` se o device suportar LiDAR).
2. App verifica *runtime* se há LiDAR: `ARWorldTrackingConfiguration.supportsSceneReconstruction(.mesh)`.
   Se falso, entra no modo fallback (ver seção "Sem LiDAR" abaixo) — não é um erro,
   é um modo suportado de primeira classe.
3. App solicita, uma vez por sessão, o BSSID atual via `NEHotspotNetwork.fetchCurrent`
   (requer autorização de localização já concedida — ver limitações §1.2). Esse
   BSSID é o que permite ao backend identificar "a qual AP este device está associado
   agora", que por sua vez é cruzado com o inventário de APs do UniFi para nomear o AP
   no mapa ("Sala de estar — U7 Pro #2").
4. A cada intervalo configurável (padrão sugerido: 1 amostra/segundo durante
   caminhada ativa), o app grava um `SpatialSample` local com pose 3D + piso atual.
5. Em paralelo (não bloqueante), o app consulta o backend por um snapshot das métricas
   de rede mais recentes daquele site (essas já estão fluindo continuamente da
   telemetria de rotina do agente — Seção 3.1 do fluxo de dados). Não pedimos ao
   agente para gerar uma métrica nova por amostra; isso saturaria a rede e o AP com
   testes ativos disparados a 1Hz.
6. Ao final do levantamento (ou incrementalmente), as amostras + a malha simplificada
   são enviadas ao backend, que persiste e enfileira o processamento no worker.
7. O worker gera heatmaps por piso, detecta zonas críticas, e associa
   evidência+confiança+impacto a cada recomendação (nunca uma recomendação sem essas
   três informações — requisito explícito da Seção 6.5).

## Amostragem de rede: por que não "uma métrica de rede por passo"

Rodar um teste ativo de latência/throughput a cada passo do usuário criaria uma carga
de teste artificial não realista e poderia, ela mesma, degradar a rede que está sendo
medida. A estratégia é: a telemetria de rotina do agente já mede latência/jitter/perda
continuamente (Fase 2); o levantamento LiDAR *consome* essa série temporal existente,
apenas adicionando a dimensão espacial. Testes ativos adicionais (ex.: throughput
local pontual) só são disparados sob demanda explícita do usuário durante o
levantamento ("medir aqui agora"), não automaticamente a cada amostra.

## Sem LiDAR (fallback obrigatório, não degradado)

Dispositivos sem LiDAR (iPhone não-Pro, iPad não-Pro) usam:

- `ARWorldTrackingConfiguration` sem `sceneReconstruction` — tracking por feature
  points, suficiente para registrar uma trilha 2D aproximada do usuário.
- Marcação manual da planta: o usuário desenha/importa a planta (imagem ou PDF) e
  marca pontos de medição manualmente tocando na planta.
- Todas as demais capturas (BSSID, métricas UniFi, ping) funcionam idênticas —
  a diferença é exclusivamente na captura geométrica.
- O relatório de cobertura gerado a partir de dados manuais é rotulado como
  "planta manual" vs. "malha LiDAR", nunca apresentado com o mesmo selo de precisão.

## O que a Realidade Aumentada mostra (e não mostra) durante a caminhada

Mostra: posição já percorrida, densidade de amostragem, AP associado no momento,
métricas de rede da amostra mais recente (rotuladas com fonte), aviso de área não
percorrida, aviso de perda de tracking ARKit.

Não mostra e nunca mostrará: uma "visualização de ondas de rádio" realista ou
qualquer elemento visual que sugira medição física de RF pela câmera/LiDAR — o
overlay visual é sempre uma representação dos dados de rede vindos do UniFi/agente,
posicionados no espaço 3D reconstruído, não uma simulação de propagação de sinal.

## Processamento (worker, Fase 6/7)

- Interpolação espacial (ex.: IDW — inverse distance weighting, ou kriging simples)
  entre `SpatialSample`s para gerar o heatmap contínuo por piso — é uma
  **estimativa matemática**, e todo pixel do heatmap fora de uma amostra real deve
  poder mostrar, ao ser inspecionado, que é interpolado (fonte `estimated`) e qual a
  distância à amostra real mais próxima.
- Detecção de zonas críticas (Seção 6.5) usa regras explicáveis (limiares
  configuráveis sobre RSSI/latência/perda/throughput), não um modelo de ML opaco
  nesta fase — compatível com o requisito de "não atribuir causalidade sem evidência"
  (Seção 12) e "mostrar fórmula e dados utilizados" (Seção 7).

## Superfície de dados por `SpatialSample`

Ver modelo de dados (`05-modelo-dados.md`) para o schema completo. Campos mínimos:
`survey_id`, `floor_id`, `position (x,y,z)`, `orientation (quaternion opcional)`,
`captured_at`, `bssid`, `ssid`, `unifi_ap_id` (resolvido), `radio_band`, `channel`,
`rssi`, `snr`, `phy_rate`, `network_metrics_timestamp`, `network_metrics_source`,
`time_sync_delta_seconds`, `confidence`.
