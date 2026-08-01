# ADR-010 — LiDAR nunca mede rádio; junção por timestamp entre geometria e rede

## Status
Aceito

## Contexto
A Seção 6.1 do documento-fonte é explícita: "Nunca afirmar que o LiDAR mediu sinal de
rádio". Isso é reforçado tecnicamente pelo fato confirmado de que
`NEHotspotNetwork.signalStrength` (a única API de "sinal" acessível no iOS) é
conhecidamente não populada de forma confiável (limitações §1.3) — ou seja, mesmo que
o produto quisesse usar "o sinal que o iPhone vê", essa API não entrega isso de forma
utilizável.

## Decisão
O módulo Spatial WiFi Survey trata geometria (ARKit/LiDAR) e qualidade de rede
(UniFi + agente) como duas séries de dados independentes, unidas exclusivamente por
`timestamp` (com janela de tolerância explícita e registrada por amostra) e por
`BSSID` (para saber a qual AP o dispositivo estava associado). Nenhum valor de
RSSI/canal/potência exibido no heatmap é derivado de sensor de câmera/profundidade —
sempre é o valor que o UniFi reportou para aquele cliente naquele momento, ou uma
interpolação matemática explicitamente rotulada como `estimated`.

## Consequências
- Toda `SpatialSample` carrega `network_metrics_source` e `time_sync_delta_seconds`,
  tornando a proveniência auditável por amostra, não apenas por levantamento inteiro.
- Elimina o risco de o produto "prometer" uma capacidade de hardware que o iPhone não
  tem (nenhum iPhone mede RF de terceiros com o LiDAR).
- Zonas do heatmap sem amostra real próxima o suficiente (fora da janela de
  interpolação aceitável) são mostradas como "sem dado" em vez de extrapoladas sem
  aviso — consistente com a Seção 2.1 ("Indisponível... motivo").

## Alternativas consideradas
- **Usar `signalStrength` do `NEHotspotNetwork` como proxy de RSSI no ponto medido**:
  rejeitado — o próprio campo é documentadamente não confiável (frequentemente `0`)
  em múltiplas versões de iOS, tornando qualquer heatmap baseado nele enganoso.
- **Não mostrar nenhuma métrica de rede durante a caminhada, só ao final**: rejeitado
  — contradiz a Seção 6.4 (feedback em tempo real durante a AR), e é tecnicamente
  desnecessário já que a telemetria UniFi de rotina já flui continuamente e pode ser
  consumida ao vivo sem gerar carga extra na rede.
