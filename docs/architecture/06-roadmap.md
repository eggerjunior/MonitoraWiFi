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
> (não simulado). **Atualização (2026-08-01)**: migrações `0002_agents`/
> `0003_speed_tests` confirmadas já aplicadas em produção
> (`monitorawifi-postgres`), e o pipeline de release do binário
> (`.github/workflows/local-agent-release.yml`) foi criado — nenhum release
> disparado ainda. **Atualização (2026-08-01, continuação)**: primeiro
> release publicado e validado ponta a ponta (`agent-v0.1.0`), speed test
> modo LAN (iPerf3) implementado e testado com servidor real. **Atualização
> (2026-08-01)**: primeiro agente real enrolado em produção — rodando como
> container Docker num mini PC (Home Assistant OS) dentro da LAN residencial,
> heartbeat confirmado (`last_seen_at` atualizando no banco). **Falta**:
> comparação entre resolvedores DNS. Ver `apps/local-agent/README.md` para o
> detalhamento completo.

Registro/enrolamento do agente (ADR-006), heartbeat, ping (ICMP/TCP/HTTP/DNS),
speed test (Internet e LAN), buffer offline com backoff exponencial, streaming de
métricas para o backend, primeiros dashboards mínimos consumindo dados reais do
agente (não mock).

## Fase 3 — UniFi

> **Status (2026-08-01)**: primeira fatia real implementada —
> `NetworkAPIAdapter` (`apps/local-agent/internal/unifi`) sincroniza
> inventário de dispositivos e clientes contra a Network API local,
> autenticado por API key gerada pela instalação real do usuário. **Atualização
> (2026-08-01)**: validado contra o console real de verdade
> (`192.168.110.1`) com o primeiro agente enrolado em produção — sincronizou
> 14 dispositivos e 80 clientes reais, gravados em `unifi_devices`/
> `unifi_clients` e confirmados via query direta no banco de produção. 13 dos
> 18 itens de `verificacoes-pendentes-instalacao.md` já confirmados com dados
> reais (versões, autenticação, VLANs ativas, firmwares, topologia
> cliente→dispositivo básica). **Faltam**: detecção automática de
> capability matrix por versão, detalhe de rádio/porta (campos ainda "a
> validar"), eventos/alarmes, topologia dispositivo→dispositivo, adaptadores
> SNMP/Syslog/Site Manager/legado (nenhum começado).

`UniFiIntegrationProvider` com adaptador Network API local funcional contra a
instalação real, detecção de versão + capability matrix populada automaticamente,
inventário de sites/gateway/APs/switches/clientes, topologia básica, eventos.
Bloqueado parcialmente até as verificações de `verificacoes-pendentes-instalacao.md`
serem respondidas.

## Fase 4 — Dashboards

> **Status (2026-08-01)**: Internet (Fase 2), Dispositivos, Wi-Fi e Clientes
> (Fase 3) implementados em web e iOS, consumindo dado real com proveniência
> declarada. **Faltam**: Switches (sem tela dedicada — hoje aparece junto de
> Dispositivos), Alertas, Histórico — nenhum começado.

Internet, Wi-Fi, Clientes, APs, Switches, Alertas, Histórico — todos consumindo dado
real (agente + UniFi), com badge de proveniência (`02-arquitetura-proposta.md` §2.6)
em cada card.

## Fase 5 — Diagnósticos

> **Status (2026-08-01)**: ping, DNS lookup e traceroute sob demanda
> implementados de ponta a ponta (usuário→backend→agente→backend→usuário,
> mesmo ciclo de comandos da Fase 5) e testados com sondas reais (traceroute
> com socket ICMP real contra loopback, DNS lookup real). Calculadora de
> sub-rede (cálculo puro, sem agente) também pronta em web e iOS.
> **Atualização (2026-08-01)**: ping em lote (`batch_ping`, até 20 alvos por
> execução) implementado ponta a ponta e testado com sondas reais (dois
> listeners TCP reais no teste do agente) — reaproveita a mesma fila de
> comando sob demanda. **Atualização (2026-08-01, continuação)**: as 6
> ferramentas restantes implementadas e testadas localmente, nesta ordem —
> SSL/TLS checker (handshake TLS real, valida cadeia contra raízes do
> sistema), RDAP/WHOIS (roda direto no backend via bootstrap real da IANA,
> sem agente, validado também contra a internet real: Verisign/APNIC), HTTP
> client sob demanda (requisição real, corpo/headers reais, até 64KB),
> LAN scanner (varredura concorrente por portas comuns, só aceita CIDR
> privado RFC 1918 até /22), Wake-on-LAN (magic packet UDP real via agente,
> ADR-008, só aceita destino privado/broadcast), e port scanner (só aceita
> IPv4 privado literal — nunca hostname — e no máximo 1024 portas,
> mitigação completa exigida pelo threat model §5 antes de expor a
> ferramenta). **Fase 5 completa em código e testes locais** — falta apenas
> o deploy em produção (migrações 0009-0013, releases do agente/API/web/
> iOS), decisão do usuário sobre quando/como.

Ping, batch ping, DNS lookup, traceroute, port scanner (com allowlist e auditoria),
SSL/TLS checker, RDAP/WHOIS, HTTP client, LAN scanner, subnet calculator,
Wake-on-LAN (via agente, ADR-008).

## Fase 6 — LiDAR

> **Status (2026-08-01)**: **não iniciada, deliberadamente** — decisão
> registrada aqui, não uma pendência esquecida. Esta fase exige ARKit rodando
> em hardware real com sensor LiDAR (iPhone/iPad Pro) para produzir qualquer
> dado real de verdade; o ambiente desta sessão de desenvolvimento não tem
> Xcode, simulador com câmera funcional, nem um dispositivo físico. Escrever
> a UI de captura AR/RealityKit sem conseguir validar contra uma sessão de
> câmera real violaria o princípio central do projeto ("nunca simular
> dado", Seção 2.1) na pior forma possível: código que parece funcionar mas
> nunca rodou de verdade. Pelo mesmo motivo, a modelagem de dados do lado
> backend (armazenar amostras espaciais/heatmap) não foi antecipada
> especulativamente — sem o fluxo de captura real para popular com
> coordenadas de verdade, seria infraestrutura sem consumidor real (design
> especulativo que o próprio projeto instrui a evitar).
>
> **Pré-requisito já satisfeito**: Fases 2 (agente) e 3 (UniFi, início)
> já entregam métricas de rede reais (latência/perda/jitter, inventário de
> dispositivos) que um levantamento espacial precisaria sincronizar — a
> dependência declarada abaixo ("Fase 6 depende da Fase 2 e 3") está
> tecnicamente desbloqueada.
>
> **Próximo passo real**: uma sessão com Xcode em um Mac de verdade e um
> iPhone/iPad Pro com LiDAR — implementar e testar a captura guiada
> (`ARWorldTrackingConfiguration` com reconstrução de cena), validar contra
> um espaço físico real, e só então desenhar o modelo de dados do lado
> backend a partir do que a captura realmente produz (não do que a Seção 6
> do documento-fonte imaginou antes de qualquer protótipo).

Fluxo completo de `Spatial WiFi Survey` (detecção de LiDAR, captura guiada, malha,
amostras, sincronização com métricas de rede, heatmap 2D/3D, modo AR, fallback sem
LiDAR, comparação entre levantamentos).

## Fase 7 — Inteligência

> **Status (2026-08-01)**: anomalias estatísticas explicáveis implementadas e
> testadas (`apps/worker/internal/baseline` — baseline por hora/dia da
> semana, z-score, nunca reporta sem histórico suficiente) e integradas
> (`cmd/worker` grava em `anomalies`, backend expõe `GET /sites/{id}/anomalies`).
> Validado ponta a ponta com Postgres real. **Atualização (2026-08-01)**:
> agendamento configurado em produção (cron a cada 6h, `egger-worker:latest`
> buildado direto do repo e rodado com `docker run --rm` na rede
> `monitorawifi_net`) e primeira execução real confirmada contra o site com
> agente de verdade — reportou corretamente "sem histórico suficiente ainda"
> (nenhuma anomalia falsa). **Faltam**: acumular histórico real suficiente
> pro baseline ter efeito prático, cobrir métricas de speed test (só ping
> por enquanto — infraestrutura já genérica o bastante), motor de
> correlação/diagnóstico, recomendações com evidência/confiança/impacto/risco,
> relatórios completos — nenhum começado.

Anomalias estatísticas explicáveis (baseline por hora/dia da semana), motor de
correlação/diagnóstico (Internet lenta, Wi-Fi lento, cliente desconectando),
recomendações com evidência/confiança/impacto/risco, relatórios completos.

## Fase 8 — Produção

> **Status (2026-08-01)**: dois itens reais resolvidos nesta sessão —
> **backup automatizado** (`infrastructure/scripts/backup-postgres.sh`,
> testado ponta a ponta incluindo restore, cron diário instalado em
> produção) e **rate limiting nos endpoints de comando sob demanda**
> (gap real encontrado revisando `docs/security/threat-model.md`, corrigido
> e testado). **Faltam** (nenhum começado): acessibilidade formal
> (WCAG/VoiceOver/Dynamic Type — só revisão de código pontual feita ao
> longo das fases, não uma auditoria dedicada), suíte de testes completa
> (cobertura por módulo nunca medida formalmente), agendamento do worker
> (Fase 7), allowlist de alvo para ferramentas de rede (ver threat-model.md
> §5), manual do usuário e runbooks formais.

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
