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
> cliente→dispositivo básica). **Atualização (2026-08-02)**: **18 de 18
> itens de `verificacoes-pendentes-instalacao.md` confirmados** — SNMP
> confirmado desabilitado (print real de `Settings → Monitoramento
> SNMP`, nem v1/2C nem v3 marcados), fechando a lista. Detalhe de
> rádio/porta e eventos/alarmes **confirmados indisponíveis** nesta
> versão da Network API local (não "a validar" — testado e não vem: só
> canal/largura/PoE-básico por rádio/porta, sem potência/utilização/
> contadores/PoE-watts; `/alarms` e `/events` retornam 404 explícito).
> **Topologia dispositivo→dispositivo implementada e em produção**
> (`uplink.deviceId`, confirmado via `GET .../devices/{id}` real) — web
> mostra árvore gateway→switch→AP, iOS mostra "Conectado a". **Faltam**:
> detecção automática de capability matrix por versão, adaptadores
> SNMP/Syslog/Site Manager/legado (nenhum começado — SNMP e syslog
> confirmados desabilitados nesta instalação, então esses adaptadores
> não têm dado real pra consumir ainda mesmo se implementados; eventos/
> alarmes ficam dependentes da Site Manager API — cloud, item 5, ainda
> não decidida — já que a API local não os expõe).

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
>
> **Atualização (2026-08-02)**: Switches (seção dedicada, filtra
> `features.includes("switching")`, mesmo padrão já usado pra Wi-Fi com
> `"accessPoint"`), Alertas (anomalias reais do worker de baseline —
> Fase 7 — com severidade derivada do z-score na própria UI, já que ainda
> não existe schema de alerta com severidade/status próprio) e Histórico
> (ping tests, speed tests e anomalias recentes, sem biblioteca de
> gráfico — nenhuma existe no projeto, segue o padrão de lista já usado no
> resto do produto) implementados em web e iOS. **Achado real durante a
> implementação**: apesar deste status dizer "Internet... implementado em
> web e iOS", o iOS nunca teve nenhuma tela de Internet (nenhuma
> referência a ping/speed test em todo `apps/ios/Sources` antes desta
> sessão) — os modelos `PingTestRecord`/`SpeedTestRecord` e os métodos
> `APIClient.pingTests`/`speedTests` foram criados agora para alimentar o
> Histórico, e incidentalmente já deixam o terreno pronto para uma tela
> "Internet" dedicada no iOS no futuro, que segue sem existir. **Fase 4
> completa** nas 4 telas descritas no escopo original (Internet, Wi-Fi,
> Clientes/Dispositivos/Switches, Alertas, Histórico). **Atualização
> (2026-08-02)**: deploy completo em produção — web 0.7.0 reimplantado e
> confirmado saudável, iOS 0.4.0 (Build 11) enviado ao TestFlight com
> sucesso. Nenhuma mudança de backend/agente foi necessária (os 3
> endpoints já existiam desde as Fases 2/3/7).

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
> ferramenta). **Atualização (2026-08-02)**: deploy completo em produção —
> migrações 0009-0013 aplicadas (backup prévio), API/web reimplantados,
> agente `agent-v0.5.0` publicado, web 0.6.0 e iOS 0.3.0 (Build 10) no
> TestFlight. **Fase 5 concluída** nas 4 superfícies (backend, agente,
> web, iOS). Ver `docs/development-handoff/RELEASE_LOG.md` (2026-08-02)
> para os detalhes do deploy, incluindo um achado real corrigido na hora
> (variável de ambiente do web faltando no primeiro redeploy).

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
> (nenhuma anomalia falsa). **Atualização (2026-08-02)**: cobertura de
> speed test implementada — `download_mbps`/`upload_mbps`/`bufferbloat_ms`
> (sempre modo "internet", nunca misturado com LAN/HTTP, senão corromperia
> a estatística) passam pelo mesmo algoritmo de baseline do ping.
> Validado contra Postgres real com um cenário controlado (queda real de
> 100→10 Mbps corretamente detectada como anomalia em download, enquanto
> upload/bufferbloat — que não caíram — corretamente não geraram falso
> positivo). **Faltam** (deliberadamente adiado, não esquecido): motor de
> correlação/diagnóstico, recomendações com evidência/confiança/impacto/
> risco, relatórios completos. Produção ainda tem pouquíssimo histórico
> real (~1 dia, um agente) — implementar correlação contra esse volume
> violaria o mesmo princípio que já bloqueia isso aqui há duas atualizações
> ("não é útil implementar anomalias contra dados sintéticos de poucas
> horas"). Retomar quando houver semanas de histórico real acumulado.

Anomalias estatísticas explicáveis (baseline por hora/dia da semana), motor de
correlação/diagnóstico (Internet lenta, Wi-Fi lento, cliente desconectando),
recomendações com evidência/confiança/impacto/risco, relatórios completos.

## Fase 8 — Produção

> **Status (2026-08-01)**: dois itens reais resolvidos nesta sessão —
> **backup automatizado** (`infrastructure/scripts/backup-postgres.sh`,
> testado ponta a ponta incluindo restore, cron diário instalado em
> produção) e **rate limiting nos endpoints de comando sob demanda**
> (gap real encontrado revisando `docs/security/threat-model.md`, corrigido
> e testado). **Atualização (2026-08-02)**: mais itens fechados —
> **allowlist de alvo pras ferramentas de rede** (RFC 1918 obrigatório
> pra `lan_scan`/`wake_on_lan`/`port_scan`, Fase 5); **cobertura de testes
> medida pela primeira vez** por módulo (`docs/testing/cobertura.md` —
> alguns exemplos reais: `api/internal/auth` 51.7%→89.7% depois de cobrir
> geração/hash de credencial de agente, que não tinha teste próprio apesar
> de ser código de segurança; `internal/store` de api/worker seguem sem
> teste automatizado, decisão de arquitetura já existente — nenhum CI
> sobe Postgres, validação sempre manual contra container real);
> **versionamento formal aplicado a api e worker** (mesmo esquema de
> web/iOS/local-agent — `VERSION` + commit injetado via ldflags, exposto
> em `GET /healthz` pra api e log de boot pro worker); **auditoria de
> acessibilidade** (`docs/testing/acessibilidade.md` — 4 cores do design
> system abaixo do mínimo WCAG AA corrigidas, um bug real de
> `aria-label` faltando no menu recolhido do web, um `outline-none` sem
> substituto na tela de login; iOS revisado por código, sem dispositivo
> físico disponível); **runbook de produção**
> (`docs/deployment/runbook-producao.md`) e **manual do usuário**
> (`docs/user-guide/manual-do-usuario.md`), ambos com passos reais já
> testados nesta e em sessões anteriores, não especulativos. **Faltam**:
> agendamento do worker (Fase 7) — na verdade já resolvido, cron a cada
> 6h em produção desde 2026-08-01, este item estava desatualizado aqui;
> nenhuma auditoria WCAG com ferramenta dedicada (axe-core) nem teste com
> leitor de tela real; nenhuma verificação de VoiceOver em hardware real.

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
