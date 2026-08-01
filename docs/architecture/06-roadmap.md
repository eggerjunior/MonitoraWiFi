# 06 — Roadmap por Fases

Cada fase só começa com a anterior testada e documentada (regra da Seção 23:
"não iniciar a fase seguinte com testes quebrados"). Este roadmap é o de alto nível;
o detalhamento de tarefas de cada fase é refinado no início dela, não antecipado aqui
em excesso de detalhe que provavelmente mudaria.

## Fase 0 — Descoberta (esta entrega)
Arquitetura, limitações confirmadas, threat model, capability matrix, estratégia
LiDAR, modelo de dados inicial, ADR-001 a ADR-010, estrutura do monorepo, roadmap,
critérios de aceite da Fase 1, lista de verificações pendentes na instalação real.
**Entregável**: os documentos em `docs/`, sem código de produto.

## Fase 1 — Fundação
Monorepo funcional, backend mínimo (auth, RBAC, health checks), banco com migrações
versionadas, web mínima (login + shell de navegação), app iOS mínimo (login + shell
TabView/NavigationSplitView), design system inicial (`packages/design-tokens`),
OpenAPI 3.1 publicada, CI rodando lint+build+test para todos os apps, Docker Compose
de desenvolvimento. Critérios de aceite detalhados em `07-criterios-aceite-fase1.md`.

## Fase 2 — Agente local

> **Status (2026-08-01)**: registro/enrolamento, heartbeat, ping
> (ICMP/TCP/HTTP/DNS), speed test HTTP (download/upload/bufferbloat), buffer
> offline com backoff exponencial e streaming de métricas para o backend
> estão implementados e testados (29 testes automatizados em
> `apps/local-agent` + endpoints correspondentes em `apps/api`, incluindo
> `GET /sites/{id}/ping-tests` e `GET /sites/{id}/speed-tests`). A página
> **Internet** do web (`apps/web`) já consome esses dados reais — validado
> ponta a ponta com um agente enrolado de verdade via containers efêmeros
> (não simulado). **Faltam**: speed test modo LAN (iPerf3), pipeline de
> release do binário (`scripts/install.sh` depende de um release do GitHub
> que ainda não existe), e aplicar as migrações `0002_agents`/
> `0003_speed_tests` no banco de produção (`monitorawifi-postgres`) — decisão
> deliberadamente não tomada nesta sessão. Ver `apps/local-agent/README.md`
> para o detalhamento completo.

Registro/enrolamento do agente (ADR-006), heartbeat, ping (ICMP/TCP/HTTP/DNS),
speed test (Internet e LAN), buffer offline com backoff exponencial, streaming de
métricas para o backend, primeiros dashboards mínimos consumindo dados reais do
agente (não mock).

## Fase 3 — UniFi
`UniFiIntegrationProvider` com adaptador Network API local funcional contra a
instalação real, detecção de versão + capability matrix populada automaticamente,
inventário de sites/gateway/APs/switches/clientes, topologia básica, eventos.
Bloqueado parcialmente até as verificações de `verificacoes-pendentes-instalacao.md`
serem respondidas.

## Fase 4 — Dashboards
Internet, Wi-Fi, Clientes, APs, Switches, Alertas, Histórico — todos consumindo dado
real (agente + UniFi), com badge de proveniência (`02-arquitetura-proposta.md` §2.6)
em cada card.

## Fase 5 — Diagnósticos
Ping, batch ping, DNS lookup, traceroute, port scanner (com allowlist e auditoria),
SSL/TLS checker, RDAP/WHOIS, HTTP client, LAN scanner, subnet calculator,
Wake-on-LAN (via agente, ADR-008).

## Fase 6 — LiDAR
Fluxo completo de `Spatial WiFi Survey` (detecção de LiDAR, captura guiada, malha,
amostras, sincronização com métricas de rede, heatmap 2D/3D, modo AR, fallback sem
LiDAR, comparação entre levantamentos).

## Fase 7 — Inteligência
Anomalias estatísticas explicáveis (baseline por hora/dia da semana), motor de
correlação/diagnóstico (Internet lenta, Wi-Fi lento, cliente desconectando),
recomendações com evidência/confiança/impacto/risco, relatórios completos.

## Fase 8 — Produção
Hardening de segurança, performance, acessibilidade (WCAG, VoiceOver, Dynamic Type),
suíte de testes completa, preparação App Store/TestFlight
(`ildemar_ios-native-testflight`, `ildemar_app-versioning`), observabilidade completa
do próprio sistema (Seção 20), política de backup testada, documentação final
(manual do usuário, runbooks).

## Dependências entre fases (o que bloqueia o quê)

- Fase 3 (UniFi) depende de respostas da Fase 0 sobre a instalação real — pode
  começar em paralelo com Fase 2, mas sua conclusão depende de acesso à instalação.
- Fase 6 (LiDAR) depende da Fase 2 (agente entregando métricas de rotina) e da
  Fase 3 (UniFi entregando RSSI/canal/AP por cliente) — não pode ser adiantada sem
  essas duas fontes de dado reais.
- Fase 7 (Inteligência) depende de volume histórico real das Fases 2-4 para que
  baseline estatístico faça sentido — não é útil implementar anomalias contra dados
  sintéticos de poucas horas.
