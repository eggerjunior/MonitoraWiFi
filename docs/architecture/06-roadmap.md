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
> heartbeat confirmado (`last_seen_at` atualizando no banco). **Atualização
> (2026-08-02)**: comparação entre resolvedores DNS implementada — novo
> comando sob demanda `dns_resolver_compare`, resolve contra o resolvedor
> padrão da rede e três resolvedores públicos fixos (Cloudflare, Google,
> Quad9), disponível nas 4 superfícies (agente, backend, web, iOS). **Fase 2
> concluída** — nenhum item pendente restante. Ver `apps/local-agent/README.md`
> para o detalhamento completo.

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
> confirmados desabilitados nesta instalação, então esses adaptadores não
> têm dado real pra consumir ainda mesmo se implementados). **Decisão
> registrada em 2026-08-02**: usuário optou por **não habilitar a Site
> Manager API por ora** (sistema segue 100% local/self-hosted) — o
> benefício confirmado dela (lista de sites, métricas agregadas de
> internet) se sobrepõe ao que o agente já mede, e ela **não confirma**
> um endpoint de eventos/alarmes (correção de uma suposição anterior
> deste documento, que presumia isso sem checar a documentação real).
> Eventos/alarmes seguem **sem fonte real confirmada nenhuma** (nem
> local, nem cloud) — item genuinamente sem solução disponível agora,
> não uma tarefa pendente de implementação.

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
> **Atualização (2026-08-02)**: primeira fatia real implementada e enviada
> a produção — o usuário confirmou ter um iPhone com LiDAR disponível pra
> testar de verdade, o que muda a estratégia acima ("esperar Mac/iPhone
> Pro nesta sessão de dev"): o código roda em produção real (via
> `iOS TestFlight release`, que compila num runner macOS de verdade), e a
> validação da sessão AR/LiDAR em si acontece no aparelho físico do
> usuário, não neste ambiente de desenvolvimento (que segue sem
> Xcode/simulador — só não é mais o único lugar onde o código pode ser
> exercitado). **Escopo desta fatia, deliberadamente menor que o ER
> especulativo original** (`05-modelo-dados.md` §6, `04-estrategia-lidar.md`):
> - App iOS abre uma sessão `ARWorldTrackingConfiguration` com
>   `sceneReconstruction = .mesh` quando `LiDARCapabilityChecker` confirma
>   suporte (fallback sem malha, só tracking por feature points, em
>   aparelhos sem LiDAR — já era um requisito de primeira classe, não um
>   modo degradado).
> - Botão "Capturar aqui" grava a posição real da câmera
>   (`ARCamera.transform`) + SSID/BSSID atual (`NEHotspotNetwork.fetchCurrent`,
>   com permissão de localização) + RTT real medido do próprio ponto até o
>   backend (reaproveitando `/auth/me`, sem endpoint novo só pra isso) +
>   estado do `NWPathMonitor` (Wi-Fi/celular, expensive/constrained). Cada
>   ponto capturado aparece na cena AR como uma esfera colorida pela
>   qualidade do RTT — feedback visual real, não pré-visualização de dado
>   fictício.
> - **Corte deliberado em relação ao ER original**: os campos
>   `bssid`/`radio_band`/`channel`/`rssi_dbm`/`snr_db`/`phy_rate_mbps` por
>   amostra do desenho especulativo **não são obtidos por nenhuma fonte
>   real disponível hoje** — confirmado empiricamente nesta mesma sessão
>   (Fase 3/5): a Network API local do UniFi não expõe RSSI/canal por
>   cliente em `/clients` (só `type`, `id`, `name`, `connectedAt`,
>   `ipAddress`, `macAddress`, `uplinkDeviceId`, `access.type`), e o iOS não
>   expõe RSSI de forma confiável (`signalStrength` do `NEHotspotNetwork`,
>   ver `01-limitacoes-tecnicas.md` §1.3). Persistir esses campos seria
>   infraestrutura sem dado real pra popular — o mesmo princípio que já
>   bloqueava a fase inteira. O esquema implementado
>   (`spatial_surveys`/`spatial_survey_samples`, migração 0016) reflete só o
>   que é honestamente capturável: posição + SSID/BSSID reportado + RTT
>   medido + estado de rede do `NWPathMonitor`.
> - **Também fora desta fatia** (adiado, não esquecido): correlação por
>   timestamp com a telemetria contínua do agente/UniFi (arquitetura
>   completa em `04-estrategia-lidar.md`), `floor_id`/múltiplos andares,
>   malha 3D persistida (`MESH_ASSET`), modo sem LiDAR com planta manual.
>   A visualização web (`/map`) não é uma malha 3D.
> - Validado: backend testado contra Postgres real efêmero (transação +
>   batch insert + leitura, `docs/development-handoff/RELEASE_LOG.md`);
>   testes automatizados dos handlers HTTP; funções puras de posição/RTT
>   testáveis (`SpatialSurveyMathTests.swift`, Swift Testing — mesma
>   ressalva de sempre sobre `xcodebuild test` não rodar no CI headless).
>   A sessão AR/LiDAR em si (mesh reconstruction, captura real de
>   posição/SSID/RTT em campo) depende do teste do usuário no aparelho
>   físico — pendência real de validação de campo, não de implementação.
> - **Atualização (2026-08-02, mesmo dia)**: heatmap contínuo interpolado
>   (IDW — inverse distance weighting, `apps/web/src/lib/spatial-heatmap.ts`)
>   substituiu o scatter simples — cada célula da grade mostra sua
>   distância até a amostra real mais próxima (via `<title>`, opacidade
>   proporcional à confiança, nunca 100% opaca fora de medição real).
>   Matemática validada por script isolado (4 casos: ponto médio entre
>   duas amostras, célula colada numa amostra real, amostra com falha
>   sem quebrar a interpolação, amostra única). **Bug real encontrado e
>   corrigido nesta mesma verificação**: `sample_count` na listagem
>   sempre voltava `0` — `ListBySite` nunca carrega `Samples` (por
>   design, seria caro numa tela de lista), mas o serializador calculava
>   a contagem a partir de `len(Samples)`; corrigido com um campo
>   `SampleCount` próprio, populado por subquery `COUNT(*)` no
>   `ListBySite` e por `len(Samples)` no `Get`. O teste automatizado
>   original não pegou isso porque o fake de teste carregava `Samples`
>   inteiro na listagem (diferente do Postgres real) — corrigido também,
>   e um novo assert de `sample_count` foi adicionado para não repetir.
>   Só foi descoberto porque a verificação rodou contra uma stack real
>   containerizada (Postgres + api + web, login de verdade via
>   Playwright), não só testes unitários com fakes.

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
>
> **Atualização (2026-08-02)**: decisão revisitada a pedido do usuário —
> em vez de esperar semanas de histórico, o motor de correlação/diagnóstico
> foi implementado agora **com a mesma guarda defensiva já usada pelo
> detector de anomalias**: nunca diagnostica/recomenda sem evidência real
> (anomalias já detectadas) suficiente, então roda honestamente "sem nada a
> diagnosticar" enquanto o histórico for pequeno, e produz diagnóstico real
> assim que houver anomalia real — sem esperar um volume artificial mínimo.
> **Implementado** (`apps/worker/internal/diagnostics`, motor baseado em
> regras, não ML): duas categorias com evidência real disponível hoje —
> **internet_slow** (anomalias em `ping_latency_ms_p50`/
> `speedtest_download_mbps`/`speedtest_upload_mbps`/`speedtest_bufferbloat_ms`,
> todas medidas contra a internet real) e **wifi_slow** (evidência nova:
> baseline estendido pra cobrir `speed_tests` modo `lan`, que já era
> coletado pelo agente mas não alimentava nenhum baseline até agora — sem
> essa extensão não haveria nenhuma fonte real pra distinguir "Wi-Fi lento"
> de "Internet lenta"). Cada diagnóstico carrega confiança (cresce com o
> número de métricas distintas que corroboram), impacto e risco (dos
> mesmos limiares de z-score já usados em Alertas), e a evidência bruta
> (anomalias reais, com ID rastreável). Recomendações são geradas 1:1 por
> diagnóstico, com texto que nunca afirma algo não sustentado pelos dados
> (ex.: a recomendação de wifi_slow só diz "a internet está normal" quando
> internet_slow de fato não foi diagnosticada na mesma janela).
> **Categoria "cliente desconectando" ficou de fora** desta fatia — corte
> deliberado, não esquecido: `unifi_clients` é um snapshot substituído a
> cada sincronização (migração 0005), não uma série histórica, então não
> existe hoje nenhuma fonte real que prove reconexões repetidas de um
> cliente ao longo do tempo (o mesmo tipo de corte já aplicado a RSSI/DPI
> em outras fases). **Relatórios** (`reports`, migração 0017): gerados sob
> demanda (`POST /sites/{id}/reports`, sem período informado usa os
> últimos 7 dias) agregando anomalias/diagnósticos/recomendações reais do
> período — sem armazenamento de objetos externo (não há essa
> infraestrutura neste projeto), conteúdo inteiro em `reports.content`
> (jsonb). Web: `/reports` deixa de ser placeholder — lista diagnósticos,
> recomendações e relatórios já gerados, com botão para gerar um novo.
> **Validado contra Postgres real** (mesma técnica de sessões anteriores:
> containers efêmeros de Postgres+worker+api+web, dados reais semeados) —
> encontrado e corrigido um bug real nessa validação: o resumo do
> diagnóstico dizia "anomalias **reals**" (plural incorreto de "real"; o
> correto é "reais") por concatenar "s" cegamente num plural irregular.
> Web 0.12.0, API 0.6.0, worker 0.2.0.

Anomalias estatísticas explicáveis (baseline por hora/dia da semana), motor de
correlação/diagnóstico (Internet lenta, Wi-Fi lento — cliente desconectando
segue sem fonte de dado real disponível, ver atualização acima),
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
> testados nesta e em sessões anteriores, não especulativos. **Atualização
> (2026-08-02)**: auditoria WCAG automatizada real rodada com
> `@axe-core/playwright` — `/login` em produção real (0 violations, dois
> temas) e as 12 rotas autenticadas do dashboard contra uma stack local
> inteiramente containerizada e efêmera (Postgres com as 14 migrações
> reais + api/web buildados dos Dockerfiles reais + dado de exemplo +
> login real via formulário), 0 violations em todas as 24 combinações
> (12 rotas × 2 temas) — confirma que a revisão manual anterior já tinha
> corrigido os problemas reais existentes. Infraestrutura de teste
> (containers/rede `a11y-*`) já removida, nada ficou em produção. **Faltam**
> (deliberadamente adiado, não esquecido): teste com leitor de tela real
> (NVDA/JAWS/VoiceOver macOS) e verificação de VoiceOver/Dynamic Type do
> iOS em hardware real — ambiente sem dispositivo físico/simulador.

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
