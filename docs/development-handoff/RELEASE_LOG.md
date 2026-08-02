# Release Log

Generated: 2026-07-31T21:50:53-03:00

Record every deploy, TestFlight/App Store upload, web publish and external processing status here.

## 2026-08-02 — Topologia dispositivo→dispositivo (Fase 3) em produção

- Commit `44d973e`. Confirmado contra a instalação real do usuário (14
  devices reais, script de verificação rodado pelo próprio usuário
  contra `192.168.110.1`): `GET .../devices/{id}` traz `uplink.deviceId`
  — ausente na lista, só no detalhe. Fecha o item 9 das verificações
  pendentes.
- Backup prévio de rotina. Migração `0014_unifi_device_uplink` aplicada
  (versão 13→14, confirmada `dirty=false`).
- API (`monitorawifi-api:44d973e6`) e web (`monitorawifi-web:44d973e6`)
  reconstruídos e reimplantados — **desta vez lembrando de
  `-p 127.0.0.1:8422:8080`** (ver incidente registrado acima). Saudáveis
  pela rede interna (`/healthz` → `version:"0.2.0"`) e pela rota pública
  real (`https://wifi.egger.app.br/api/v1/auth/me` → 401 correto,
  `/login` → 200, `/devices` → 307 auth redirect).
- Agente: `agent-v0.6.0` publicado (testes reais como gate, incluindo os
  dois novos casos de topologia). **Pendente do lado do usuário**: o
  agente real já enrolado (mini PC do usuário) continua rodando a imagem
  anterior até ele mesmo atualizar — fora do meu acesso, é hardware
  dele.
- iOS 0.5.0 (Build 12): `iOS CI` verde, depois `iOS TestFlight release`
  **concluído com sucesso** (2m8s,
  https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30730192951).
- **Fase 3 (topologia dispositivo→dispositivo) concluída em produção**
  nas 4 superfícies (backend, agente, web, iOS).
- **Nota operacional real**: o disco deste ambiente de desenvolvimento
  encheu durante o deploy (imagens/cache Docker acumulados nas várias
  validações contra Postgres real desta sessão) — `docker system prune
  -af --volumes` liberou 42.85GB. Não afeta a VPS de produção (ambientes
  Docker completamente separados).

## 2026-08-02 — Incidente real: agente em produção sem conectividade por ~1h30

- **Causa raiz**: `nginx.ssl.conf` (porta 443, a config ativa — não a de
  porta 80 revisada anteriormente) tem `location /api/v1/` roteando pra
  `127.0.0.1:8422`, além do `location /` genérico pro web em `:8421`. Essa
  porta 8422 é usada só pelo agente local e pelo app iOS (falam direto com
  o backend Go, sem passar pelo BFF do Next.js) — nunca foi checada nos
  deploys anteriores desta sessão porque toda verificação de saúde usou a
  rede interna Docker ou `/login` (servido pelo web, que fala com a API
  pela rede interna, não por essa porta).
- **O que aconteceu**: o primeiro redeploy do `monitorawifi-api` nesta
  sessão (Fase 5, imagem `77d4ab3c`, ~00:45 UTC) substituiu o container
  sem repetir `-p 127.0.0.1:8422:8080` — a porta simplesmente deixou de
  existir. O agente real do usuário (rodando no mini PC dele, fora da
  rede Docker) começou a receber 502 do nginx às 00:47:49 UTC e continuou
  recebendo por ~1h30, através de mais dois redeploys da API
  (`f3813137`, `7e916db5`) que repetiram o mesmo erro. Detectado pelo
  próprio usuário, olhando o log do agente (`docker logs egger-agent`) —
  nenhum monitoramento do lado do servidor pegou isso, porque `/healthz`
  interno e `/login` continuaram respondendo 200 o tempo todo.
- **Sem perda de dado real**: o agente mantém fila local com backoff
  (Fase 2) — toda telemetria que falhou durante a janela ficou
  "mantida na fila local para nova tentativa" nos próprios logs, não foi
  descartada. Deve reenviar sozinho assim que a conectividade voltar.
- **Corrigido**: `monitorawifi-api` redeployado com
  `-p 127.0.0.1:8422:8080`. Confirmado pela rota pública real
  (`https://wifi.egger.app.br/api/v1/auth/me` → 401, correto — antes
  dava 502) e pela interna (`/healthz` → 200).
- **Runbook corrigido** (`docs/deployment/runbook-producao.md`) — o
  comando de redeploy documentado tinha o mesmo erro (eu tinha
  documentado o comando errado que eu mesmo vinha rodando). Passo de
  verificação de saúde agora inclui checagem pela rota pública real como
  passo obrigatório, não só pela rede interna.

## 2026-08-02 — Endpoint de revogação de agente em produção

- Commit `7e916db`. `POST /sites/{siteId}/agents/{agentId}/revoke` — gap
  real encontrado escrevendo o runbook de produção (até então, revogar
  só era possível via UPDATE direto no Postgres). Sem migração nova
  (usa a coluna `revoked_at` já existente). Backup prévio de rotina.
- API reconstruída e reimplantada (`monitorawifi-api:7e916db5`),
  confirmada saudável (`/healthz` com version/commit corretos, `/readyz`
  200, `/login` 200). **Não testado contra o agente real de produção**
  (revogar o agente real derrubaria o monitoramento de verdade do
  usuário) — validado ponta a ponta só com testes automatizados
  (heartbeat real aceito antes da revogação, rejeitado com 401 logo
  depois, mesma credencial).

## 2026-08-02 — Fase 7 (cobertura de speed test) + Fase 8 (hardening) em produção

- Commit `f381313`. Sem migração nova — nenhuma mudança de schema nesta
  entrega. Backup prévio de rotina mesmo assim
  (`backup-20260801-225642.sql.gz`).
- API/web reconstruídos e reimplantados (`monitorawifi-api:f3813137`,
  `monitorawifi-web:f3813137`). Saudáveis: `GET /healthz` agora responde
  `{"status":"ok","version":"0.1.0","commit":"f3813137"}` (primeiro
  deploy com versionamento formal da API — confirma a injeção via
  ldflags funcionando de verdade em produção, não só em teste local).
  `/readyz` 200, `https://wifi.egger.app.br/login` 200.
- `egger-worker:latest` reconstruído (agora com cobertura de speed test)
  e **rodado manualmente uma vez contra produção** pra confirmar o
  comportamento honesto: as 4 métricas (ping + 3 novas de speed test)
  reportaram corretamente "sem histórico suficiente ainda" — nenhum
  falso positivo, consistente com produção ter só ~1 dia de dado real. O
  cron a cada 6h (já configurado) segue rodando essa mesma imagem.
- Web: correções de acessibilidade (contraste de cores, aria-label do
  menu recolhido, foco visível no login) e Sidebar/tokens regenerados —
  já no ar, sem necessidade de passo extra além do redeploy padrão.
- Runbook de produção e manual do usuário publicados (ver
  `docs/deployment/` e `docs/user-guide/`) — usados nesta própria sessão
  pra conduzir o deploy acima.

## 2026-08-02 — Fase 4 completa: Switches, Alertas, Histórico em produção

- Commit `2394189`. Três telas que faltavam (web + iOS): Switches (rota/seção
  dedicada), Alertas (dado real de `GET /sites/{id}/anomalies`, severidade
  derivada do z-score), Histórico (ping tests/speed tests/anomalias, sem
  lib de gráfico nova). Nenhuma mudança de backend — os 3 endpoints já
  existiam.
- Web 0.7.0: `docker build`/redeploy (`monitorawifi-web:2394189`),
  confirmado saudável (`/login` 200) e as 3 rotas novas registradas de
  verdade (`/switches`, `/history`, `/alerts` → 307 auth redirect, igual
  a `/devices`; comparado com uma rota inexistente → 404, prova que não é
  coincidência de configuração de proxy).
- iOS 0.4.0 (Build 11): `iOS CI (build validation)` verde (compilou as 6
  abas, incluindo a nova "Histórico") antes do `iOS TestFlight release`,
  que **terminou com sucesso**
  (https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30726961459).
- **Fase 4 concluída em produção** nas 4 superfícies (backend, agente —
  sem mudança nesta entrega —, web, iOS).
- Achado real registrado no roadmap: o iOS nunca teve tela de "Internet"
  apesar do status da Fase 4 dizer o contrário — corrigido no
  `06-roadmap.md`, e os modelos criados agora pro Histórico
  (`PingTestRecord`/`SpeedTestRecord`) deixam o terreno pronto pra uma
  tela dedicada futura.

## 2026-08-02 — Fase 5 completa: deploy das 6 ferramentas restantes em produção

- Commits: `8c265e2` (SSL/TLS checker), `48a319d` (RDAP/WHOIS), `a5afb3c`
  (HTTP client), `346b648` (LAN scanner), `4e44305` (Wake-on-LAN),
  `c9a1e73` (port scanner), `77d4ab3` (doc do roadmap). Implementados e
  testados localmente numa sessão anterior; deploy feito nesta sessão a
  pedido explícito do usuário.
- **Backup prévio**: `backup-postgres.sh` rodado antes de qualquer
  migração (`/opt/data/monitorawifi/backups/backup-20260801-214413.sql.gz`).
- **Migrações `0009` a `0013` aplicadas em produção** (ssl_check,
  http_request, lan_scan, wake_on_lan, port_scan adicionados ao CHECK
  constraint de `agent_commands.type`) — `migrate up` confirmado
  versão=13, dirty=false. Constraint final verificada via `\d
  agent_commands` no Postgres real.
- API e web reconstruídos (`docker build` a partir do checkout de
  produção, mesmo padrão de sempre) e reimplantados
  (`monitorawifi-api:77d4ab3c`, depois `monitorawifi-web:c01588a` após o
  bump de versão). **Achado real corrigido durante o deploy**: o redeploy
  inicial do container web usando `--env-file` do `.env` compartilhado
  (pensado pra API) deixou a variável `API_BASE_URL` de fora — o web
  ficaria tentando falar com `localhost:8080` de dentro do próprio
  container. Corrigido recriando o container com
  `-e API_BASE_URL=http://monitorawifi-api:8080/api/v1` explícito, mesmo
  valor do deploy anterior (confirmado via `docker inspect` do container
  antigo antes de substituí-lo).
- Saúde confirmada com serviços reais: `GET /healthz`/`GET /readyz` da
  API (200, via container `curlimages/curl` na mesma rede Docker — o
  container distroless da API não tem shell/curl) e `GET /login` do web
  local (200) e público (`https://wifi.egger.app.br/login`, 200 via
  `curl` externo).
- Agente: `VERSION` subido para `0.5.0`, release publicado via workflow
  `Local Agent release` — rodou a suíte de testes real (incluindo os
  testes novos com handshake TLS real, servidor HTTP/UDP real, listeners
  TCP reais) como gate antes de publicar; binários linux/darwin
  amd64/arm64 em
  https://github.com/eggerjunior/MonitoraWiFi/releases/tag/agent-v0.5.0.
  **Pendente**: o agente real já enrolado (mini PC do usuário, Home
  Assistant OS) não foi atualizado — é hardware do próprio usuário, fora
  do escopo de acesso desta sessão; ele decide quando atualizar.
- Web: versão `0.6.0` (changelog atualizado), commit `c01588a`, reimplantado
  em produção (ver acima).
- iOS: versão `0.3.0` (Build 10), commit `4935e3d`. `iOS CI (build
  validation)` rodou verde antes do release (mesmo padrão cauteloso de
  sempre — validar compilação antes de gastar um upload de TestFlight).
  `iOS TestFlight release` rodou e **terminou com sucesso**
  (`ARCHIVE`/`EXPORT`/upload verdes, run
  https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30726202399,
  2m22s) — build 10 enviado, processamento no App Store Connect segue
  assíncrono do lado da Apple.
- **Resumo**: as 6 ferramentas da Fase 5 estão em produção nas 4
  superfícies (backend, agente, web, iOS) no fim desta sessão.

## 2026-08-01 — Fase 5: ping em lote (batch_ping), deploy completo

- Commit `63d1dc92`. Novo tipo de comando sob demanda reaproveitando a fila
  já existente (ping/dns_lookup/traceroute) — até 20 alvos por execução,
  testado com sondas reais (dois listeners TCP reais no teste do agente).
- Migração `0008_command_type_batch_ping` aplicada em produção (backup
  prévio via `backup-postgres.sh`).
- API redeployada (`monitorawifi-api:63d1dc92`) e Web redeployada
  (`monitorawifi-web:63d1dc92`) — ambas confirmadas saudáveis
  (`/healthz`, `/readyz`, `/login` 200) logo após o deploy.
- Agente `agent-v0.4.0` publicado (release do binário, linux/darwin,
  amd64/arm64).
- iOS 0.2.0 (Build 9) publicado no TestFlight.
- Threat model atualizado: limite de 20 alvos documentado como mitigação
  (não é primitiva de varredura em massa).

## 2026-08-01 — Fase 7: agendamento do worker de anomalias em produção

- Imagem `egger-worker:latest` buildada direto do checkout de produção
  (`apps/worker/Dockerfile`).
- Rodado manualmente uma vez contra o Postgres real antes de agendar —
  reportou corretamente "sem histórico suficiente ainda" pro único site com
  agente (esperado, agente rodando há minutos; 0 anomalias falsas).
- Cron instalado (`0 */6 * * *`), reaproveitando `/opt/apps/monitorawifi/.env`
  e a rede `monitorawifi_net`, log em `/var/log/egger-worker.log`. Fecha o
  pendente de agendamento que estava adiado desde a introdução do worker.

## 2026-08-01 — Primeiro agente real enrolado em produção + UniFi validado contra o console real

- Instalação real feita pelo usuário: container Docker (`apps/local-agent/Dockerfile`,
  buildado direto do repo com `docker build ... https://github.com/eggerjunior/MonitoraWiFi.git#main`)
  rodando num mini PC com Home Assistant OS 16.2, dentro da LAN residencial
  (`192.168.110.85`) — não numa VPS/cloud, então alcança de verdade o console
  UniFi local (`192.168.110.1`).
- Tentativa inicial na VPS de produção (`painel`) foi revogada (`agents.revoked_at`)
  sem nenhum dado de telemetria gravado — identificada a tempo antes de poluir
  métricas reais.
- Agente `37f8283b-5c29-4b5a-98ec-2ceff5b152e2` enrolado com sucesso via token
  de uso único; heartbeat confirmado (`last_seen_at` atualizando).
- Sincronização UniFi (`NetworkAPIAdapter`) validada contra o console real pela
  primeira vez (antes só testado contra um console fake): **14 dispositivos e
  80 clientes** sincronizados e gravados em `unifi_devices`/`unifi_clients`,
  confirmados via query direta no banco de produção.
- Pendente que fechou: "enrolar um agente real em produção" (Fase 2) e "testar
  UniFi contra o console real" (Fase 3) — ambos atualizados em
  `docs/architecture/06-roadmap.md`.
- Próximo: Fase 7 (worker de anomalias) agora tem histórico real começando a
  se acumular; ainda precisa de volume/tempo antes do baseline fazer sentido,
  e o agendamento do worker continua pendente (Fase 8).

## 2026-08-01 — Fase 7 (início): worker de anomalias + Fase 6 documentada como não iniciada

- Commit `90f1e2b`. `apps/worker` ganha código real pela primeira vez
  (era só docs desde a Fase 0): `internal/baseline` (algoritmo de
  detecção por z-score, testado com estatística conhecida) +
  `cmd/worker` (execução única, testado ponta a ponta com Postgres real).
- Migração `0007_anomalies` aplicada em produção (backup prévio). API
  reimplantada com o endpoint `GET /sites/{id}/anomalies`.
- **Worker não foi rodado contra produção** — não há histórico real
  ainda (nenhum agente enrolado no site real), rodar agora só reportaria
  "sem site com agente" honestamente. Também não há agendamento
  (cron/systemd timer) configurado — decisão de infraestrutura adiada
  para a Fase 8.
- **Fase 6 (LiDAR) deliberadamente não iniciada** — decisão e
  justificativa registradas em `docs/architecture/06-roadmap.md`: exige
  hardware real (LiDAR) que este ambiente não tem; escrever código de
  captura AR sem validar contra câmera real seria pior que não escrever.

## 2026-08-01 — Fase 5: DNS lookup + traceroute + calculadora de sub-rede (deploy completo)
## 2026-08-01 — Fase 5: DNS lookup + traceroute + calculadora de sub-rede (deploy completo)

- Commit `351afa8` (fix real de compilação encontrado pela CI real do iOS —
  `Command` não podia ser `Codable` com `CommandResult` só `Decodable`;
  `Result<Info, String>` exigia `String: Error`). CI iOS verde só na
  segunda tentativa — primeira falhou de verdade, não foi simulada.
- API/migração 0006 aplicada em produção (backup prévio).
- Web v0.4.0 publicado em produção.
- iOS 0.1.7 (Build 8) publicado no TestFlight.
- Agente `agent-v0.3.0` publicado (release do binário).
- Todos os 4 componentes confirmados saudáveis em produção após o deploy.

## 2026-08-01 — Fase 3 (início): integração UniFi + deploy em produção
## 2026-08-01 — Fase 4 (início): dashboards Dispositivos/Wi-Fi/Clientes (web 0.3.0 + iOS 0.1.6)

- Web v0.3.0 publicado em produção (commit `9d3963e`) — Dispositivos, Wi-Fi
  e Clientes deixam de ser placeholder, consomem inventário UniFi real.
- iOS 0.1.6 (Build 7) publicado no TestFlight (commit `b595555`) — tela
  Rede com o mesmo escopo (paridade mantida no mesmo ciclo).
- Validado ponta a ponta com containers efêmeros antes do deploy web.

## 2026-08-01 — Fase 3 (início): integração UniFi + deploy em produção
## 2026-08-01 — Fase 3 (início): integração UniFi + deploy em produção

- Commit `7d6fd33` (API/agente), `f2d0232` (bump versão do agente).
- `NetworkAPIAdapter` real (ADR-007) sincroniza inventário de
  dispositivos/clientes via Network API local, autenticado por API key
  gerada pelo próprio usuário na instalação real. Migração `0005` aplicada
  em produção (backup prévio em `/opt/data/monitorawifi/backups/`).
- API de produção reimplantada com o commit `7d6fd339`.
- Release do binário do agente `agent-v0.2.0` publicado
  (https://github.com/eggerjunior/MonitoraWiFi/releases).
- Pendência real: nenhum agente rodando na rede real do usuário ainda —
  a integração está pronta e validada com um console fake em containers
  efêmeros, mas não testada contra o console de verdade (192.168.110.1).
  Próximo passo natural: usuário rodar o agente na rede real com
  `UNIFI_BASE_URL=https://192.168.110.1`, `UNIFI_API_KEY` (a key
  "MonitoraWiFi" já gerada), `UNIFI_SITE_ID=88f7af54-98f8-306a-a1c7-c9349722b1f6`.

## 2026-08-01 — iOS 0.1.5 (6): "ping sob demanda" em Ferramentas (paridade com web)

- Commit `6e17f15`. `DiagnosticsView` substitui o placeholder de
  "Ferramentas" — mesma funcionalidade do web (`PingTool.tsx`): dispara
  `POST /sites/{id}/commands`, faz polling de `GET /commands/{id}` até
  completed/failed, mostra latência p50/perda/jitter reais.
- CI de build verde
  (https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30710781241) e
  release ao TestFlight verde (`ARCHIVE SUCCEEDED`, `EXPORT SUCCEEDED`,
  https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30710832688).
- Paridade iOS+Web mantida no mesmo ciclo de trabalho, conforme a skill
  `ildemar_app-versioning`.

## 2026-08-01 — Deploy em produção: comandos sob demanda (API + web), commit 5ad38eb

- Web v0.2.0: página **Diagnósticos** deixou de ser placeholder — ferramenta
  de ping sob demanda real (dispara comando, faz polling, mostra
  latência/perda/jitter reais quando o agente responde).
- **Achado real durante o deploy**: o banco de produção nunca teve a tabela
  de controle do `golang-migrate` (`schema_migrations`) — migrações
  0001-0003 foram aplicadas em algum momento anterior por outro processo,
  sem essa tabela existir corretamente (encontrada com `version=1,
  dirty=true`, resquício de uma tentativa de `migrate up` que falhou nesta
  sessão ao tentar recriar tabelas já existentes). Corrigido com backup
  prévio (`pg_dump` em `/opt/data/monitorawifi/backups/`) +
  `UPDATE schema_migrations SET version = 3, dirty = false` (autorizado
  explicitamente pelo usuário) — depois disso `migrate up` aplicou a
  0004 (`agent_commands`) normalmente. Registrar isso como pendência para
  as próximas migrações: confirmar `schema_migrations` antes de assumir
  que `migrate up` vai funcionar sem intervenção.
- API e web reconstruídos e reimplantados com o commit `5ad38eb`
  (`docker build`/`docker run` manuais via SSH, mesmo padrão anterior).
- Confirmado em produção: login via `https://wifi.egger.app.br/api/v1/auth/login`
  respondendo corretamente (401 com credenciais erradas, formato esperado).

## 2026-08-01 — Backend/agente: comandos sob demanda (Fase 5, início)
## 2026-08-01 — Backend/agente: comandos sob demanda (Fase 5, início)

- Commit `dd63c34`. Primeiro recurso de "teste sob demanda" (ping agora):
  usuário dispara via `POST /sites/{id}/commands`, backend enfileira em
  Postgres (`agent_commands`, migração `0004`), agente reivindica via
  polling (`GET /agents/{id}/commands`, `COMMAND_POLL_INTERVAL_SECONDS`
  padrão 5s) e reporta o resultado (`POST /agents/{id}/commands/{id}/result`).
  Usuário consulta status/resultado via `GET /commands/{id}`.
- Decisão de arquitetura registrada em ADR-011: fila via Postgres, não
  Redis (documento-fonte original previa Redis; nunca foi implantado em
  produção, decisão de não introduzi-lo ainda).
- Achado real corrigido: `apps/local-agent/Dockerfile` (distroless
  nonroot) não conseguia escrever em `/data` — nenhum agente real via
  Docker teria funcionado até esta correção.
- Testado em todas as camadas (fakes/handlers, apiclient, commandLoop do
  agente) e validado ponta a ponta com containers efêmeros reais —
  comando de ping real executado por um agente real contra outro
  container, resultado real (latência/jitter/perda) confirmado via API.
- CI (`API CI (Go)`, `Local Agent CI (Go)`) verde. OpenAPI atualizado.
- Implantado em produção no mesmo ciclo — ver entrada de deploy acima
  (commit `5ad38eb`) — junto com o botão "ping agora" exposto no web.

## 2026-08-01 — Web: versionamento (paridade com iOS) + redeploy em produção
## 2026-08-01 — Web: versionamento (paridade com iOS) + redeploy em produção

- App: web (Next.js), commit `2e54344`
- Motivo: usuário notou que o web não mostra nenhuma informação de
  versão/build, ao contrário do iOS (que já tem `VersionManager`/
  `VersionHistory` desde a Fase 1) — quebra da regra de paridade da skill
  `ildemar_app-versioning`.
- Implementado: `src/lib/version.ts` (versão do `package.json`, GIT_COMMIT/
  BUILD_DATE injetados via `--build-arg` no Dockerfile), `version-history.ts`
  (changelog em código), e a página Configurações (antes um placeholder
  vazio) agora mostra versão/build/commit (link pro GitHub) + histórico.
- Validado ponta a ponta com containers efêmeros reais (Postgres + API Go +
  web) antes do deploy: login real, cookie de sessão real, HTML
  confirmando "v0.1.0 — 1 de ago. de 2026, 12:00" e o commit de teste
  linkado corretamente.
- **Publicado em produção**: imagem `monitorawifi-web:2e54344e` (build com
  `GIT_COMMIT=2e54344e`, `BUILD_DATE` real do momento do build), container
  `monitorawifi-web` substituído mantendo a mesma rede/porta
  (`127.0.0.1:8421`). Confirmado servindo via `curl` contra
  `https://wifi.egger.app.br`.
- Pendência registrada (não resolvida agora): não há pipeline de release
  automatizado para o web (deploys continuam manuais via SSH — `docker
  build`/`docker run` — diferente do iOS, que já usa GitHub Actions
  workflow_dispatch). Considerar um workflow dedicado numa sessão futura.

## 2026-08-01 — Local agent: pipeline de release + repositório tornado público

- Criado `.github/workflows/local-agent-release.yml` (`workflow_dispatch`
  manual): lê `apps/local-agent/VERSION` (fonte única), testa, cross-compila
  `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
  (`CGO_ENABLED=0`, versão+commit injetados via `-ldflags -X`), publica
  GitHub Release `--latest` com os binários no nome que
  `scripts/install.sh` espera.
- Duas correções reais encontradas rodando o workflow de verdade (não só
  lendo o YAML): (1) erro de sintaxe YAML — `:` sem aspas dentro do `name`
  de um step quebrava o parser, GitHub registrava o workflow sem nome/sem
  trigger reconhecido; (2) path duplicado no upload dos assets
  (`apps/local-agent/dist/...` de dentro de um job cujo
  `working-directory` já é `apps/local-agent` — `gh release create`
  recebia `apps/local-agent/apps/local-agent/dist/...` e não encontrava
  nada). Ambas corrigidas e revalidadas com uma execução real verde.
- Primeiro release publicado: `agent-v0.1.0`
  (https://github.com/eggerjunior/MonitoraWiFi/releases/tag/agent-v0.1.0).
- **Achado durante a validação ponta a ponta**: `curl` sem autenticação
  contra o link público do asset (exatamente como `install.sh` faz)
  retornava 404 — o repositório era privado, e o GitHub não serve assets
  de release de repositórios privados sem autenticação. Perguntei ao
  usuário como preferia resolver (tornar público / hospedar em
  `wifi.egger.app.br` / manter privado usando `gh` autenticado por
  enquanto) — escolheu **tornar o repositório público**.
- Antes de mudar a visibilidade: auditei toda a história do git (`git log
  --all -p`) em busca de chaves privadas, certificados, `.env` real —
  nada encontrado; só `.env.example` com placeholders de desenvolvimento;
  `.gitignore` já excluía `*.p8`/`*.pem`/`*.key`/`.env*` desde o commit
  inicial.
- Repositório tornado público (`gh repo edit --visibility public`).
  Download público não-autenticado revalidado funcionando de verdade
  (`curl -fsSL https://github.com/.../releases/latest/download/egger-agent-linux-amd64`
  baixou o binário correto, versão `0.1.0+dd95c8a7` embutida confirmada
  via `strings`).
- Documentação atualizada em `README.md` (raiz), `apps/ios/README.md`,
  `apps/local-agent/README.md` e `docs/development-handoff/PROJECT_CONTEXT.md`
  para refletir a mudança de visibilidade.
- Pendências: nenhum agente real enrolado em produção ainda (tabela
  `agents` vazia); speed test modo LAN (iPerf3) não implementado.

## 2026-08-01 — iOS 0.1.4 (5): corrige "Não foi possível carregar organizações/sites"

- App: Egger Network Intelligence, bundle id `br.app.egger.network-intelligence`
- Versão/build: 0.1.4 (5), commit `f6f7c74`
- Motivo: usuário confirmou (print) que o login já funcionou no build 4
  (0.1.3) — a tela "Visão geral" abriu, mostrou versão/commit/LiDAR
  corretamente, mas exibiu "Não foi possível carregar organizações/sites.
  Verifique a conexão com o backend."
- Causa raiz: `OverviewViewModel` (usado por `OverviewView`) criava sua
  própria instância de `APIClient()` via parâmetro default do
  inicializador, em vez de reusar o client autenticado guardado pela
  `SessionStore` (que recebeu o token de sessão no login). Todo request a
  `/organizations`/`/sites` saía sem o cookie de sessão e voltava 401,
  caindo no `catch` genérico do ViewModel.
- Correção: `SessionStore.client` deixou de ser `private` (acessível no
  módulo); `RootView` passa `session.client` para `OverviewView`, que
  agora tem um `init(client:)` explícito em vez de depender do default
  parameterless (`apps/ios/Sources/Auth/SessionStore.swift`,
  `Views/OverviewView.swift`, `Views/RootView.swift`). Revisadas também
  `SettingsView`/`LoginView` — ambas já usavam `session.client`
  corretamente, não tinham o mesmo bug.
- Status: **enviado com sucesso ao TestFlight** — CI de build (run
  https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30684308024) e
  release (run
  https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30684345801)
  verdes.
- Pendências: aguardando confirmação do usuário no dispositivo real após
  atualizar para o build 5; ícone do app ainda é o placeholder gerado
  programaticamente.

## 2026-08-01 — iOS 0.1.3 (4): corrige causa real do erro de login (URL absoluta)

- App: Egger Network Intelligence, bundle id `br.app.egger.network-intelligence`
- Versão/build: 0.1.3 (4), commit `4631f67`
- Motivo: usuário confirmou que, mesmo já no build 3 (0.1.2), continuava
  recebendo "Não foi possível falar com o servidor" — ou seja, o fix
  anterior (Set-Cookie) não resolveu o problema de verdade.
- Causa raiz real: `APIClient.makeRequest` montava a URL com
  `URL(string: path, relativeTo: configuration.baseURL)`, onde `path`
  sempre começa com `/` (ex.: `/auth/login`). Por RFC 3986, uma string
  começando com `/` é resolvida como **caminho absoluto**, descartando
  qualquer componente de path do `baseURL` — então
  `https://wifi.egger.app.br/api/v1` + `/auth/login` virava
  `https://wifi.egger.app.br/auth/login` (sem `/api/v1`), que bate no
  domínio raiz servido pelo **Next.js** (web), não na API. Confirmado
  batendo direto em produção: `POST /auth/login` (sem prefixo) retorna
  404 em HTML do site; `POST /api/v1/auth/login` retorna 200 em JSON com
  `Set-Cookie` correto. O app tentava decodificar HTML como JSON e lançava
  `ClientError.invalidResponse` — exatamente a mensagem reportada,
  independente de usuário/senha.
- Correção: `makeRequest` agora monta a URL por concatenação de string
  (garantindo `/` entre base e path, sem duplicar), preservando o
  `/api/v1` do `baseURL` (`apps/ios/Sources/Networking/APIClient.swift`).
- Web: nenhuma mudança necessária — o cliente web usa Route Handlers
  (BFF) do Next.js com caminho relativo já correto; o bug era específico
  da resolução de URL do `APIClient` nativo iOS.
- Status: **enviado com sucesso ao TestFlight** — CI de build (run
  https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30684037809) e
  release (`ARCHIVE SUCCEEDED`, `EXPORT SUCCEEDED`, run
  https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30684070931)
  verdes.
- Pendências: aguardando confirmação do usuário testando login no
  dispositivo real após atualizar para o build 4; ícone do app ainda é o
  placeholder gerado programaticamente.

## 2026-08-01 — iOS 0.1.2 (3): corrige login travado por Set-Cookie interceptado

- App: Egger Network Intelligence, bundle id `br.app.egger.network-intelligence`
- Versão/build: 0.1.2 (3), commit `0355239`
- Motivo: após a correção da URL de produção (0.1.1), o usuário reportou um
  novo erro no app real: "Não foi possível falar com o servidor. Tente
  novamente." — mensagem diferente da anterior, indicando que a requisição
  chegava ao servidor mas a resposta não era processada corretamente.
- Causa raiz: `APIClient.login()` extrai o token de sessão manualmente do
  header `Set-Cookie` da resposta (para poder persistir no Keychain). Mesmo
  com `URLSessionConfiguration.ephemeral`, a `URLSession` intercepta e
  consome esse header para popular seu próprio `HTTPCookieStorage` interno,
  fazendo com que ele deixe de aparecer via
  `HTTPURLResponse.value(forHTTPHeaderField: "Set-Cookie")` — o app recebia
  200 OK mas `ClientError.invalidResponse` era lançado ao tentar ler o
  cookie.
- Correção: `sessionConfig.httpShouldSetCookies = false` e
  `httpCookieAcceptPolicy = .never` na `URLSessionConfiguration` usada pelo
  `APIClient`, impedindo a URLSession de consumir o header antes do parsing
  manual (`apps/ios/Sources/Networking/APIClient.swift`).
- Web: nenhuma mudança necessária — o BFF do Next.js já usa o padrão de
  proxy de cookie de sessão sem esse problema (paridade mantida; a correção
  é específica do cliente iOS).
- Status: **enviado com sucesso ao TestFlight** — CI de build (`iOS CI
  (build validation)`, run
  https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30683676042) e
  release (`ARCHIVE SUCCEEDED`, `EXPORT SUCCEEDED`, run
  https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30683703105)
  verdes.
- Pendências: build em processamento no App Store Connect no momento deste
  registro; ícone do app ainda é o placeholder gerado programaticamente;
  ainda não confirmado por teste manual no dispositivo do usuário (próximo
  passo real: pedir que ele atualize o TestFlight e tente login de novo).

## 2026-08-01 — iOS 0.1.1 (2): corrige URL de produção + assinatura manual

- App: Egger Network Intelligence, bundle id `br.app.egger.network-intelligence`
- Versão/build: 0.1.1 (2), commit `119db27`
- Motivo: usuário reportou "Não foi possível entrar. Verifique sua conexão"
  no app real — o app apontava para `http://localhost:8080/api/v1`
  (`APIClient.Configuration.developmentDefault`), que no iPhone real não
  aponta para nada. Corrigido para ler `APIBaseURL` do Info.plist
  (`https://wifi.egger.app.br/api/v1` por padrão via `project.yml`).
- Infra necessária para o app funcionar: `apps/api` não tinha rota pública
  (só o web tinha). Adicionado `location /api/v1/` em
  `/home/eggerjunior/conf/web/wifi.egger.app.br/nginx.ssl.conf` (proxy para
  `127.0.0.1:8422`, nova porta publicada pelo container `monitorawifi-api`),
  nginx recarregado (autorizado pelo usuário). Login confirmado via curl
  contra `https://wifi.egger.app.br/api/v1/auth/login` antes do rebuild do app.
- **Achado e resolvido durante o release**: primeira tentativa de archive
  falhou com "Choose a certificate to revoke" / "No profiles for ... iOS App
  Development" — cota de certificados de Development esgotada por Automatic
  signing (mesmo problema documentado em `references/ildemar-ios-release.md`
  da skill `ildemar_ios-native-testflight`, já resolvido antes no
  MonitoraVPS). Aplicada a mesma correção: `scripts/create_dist_cert.py`
  gerou um certificado de distribuição + perfil próprios (secrets
  `IOS_DIST_CERT_P12_BASE64`/`_PASSWORD`, `IOS_DIST_PROFILE_BASE64`, nunca
  impressos); Release passou a usar Manual signing; `ios-testflight.yml`
  importa o certificado num keychain temporário antes do archive.
  Revogado 1 certificado de distribuição órfão (criado nesta sessão) antes
  de criar o novo — mantido intacto o certificado mais antigo da conta
  (provavelmente em uso por outro projeto, MonitoraVPS), pergunta explícita
  ao usuário antes de revogar (`AskUserQuestion`), autorizado por ele.
- Status: **enviado com sucesso** — `ARCHIVE SUCCEEDED`, `EXPORT SUCCEEDED`,
  run https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30683212916
- Pendências: build em processamento no App Store Connect no momento deste
  registro; ícone do app ainda é o placeholder gerado programaticamente.

## 2026-08-01 — Deploy em produção: commit b004a4a (Fase 2 completa)

- App/plataforma: apps/api + apps/web, produção (`wifi.egger.app.br`, host `2.25.189.37`)
- Repositório/commit: `eggerjunior/MonitoraWiFi` @ `b004a4a`
- Comandos executados:
  1. Backup do banco: `pg_dump` → `/opt/backups/monitorawifi/pre-migration-20260801T032807Z.sql`
  2. Migrações `0002_agents.up.sql` e `0003_speed_tests.up.sql` aplicadas
     via `psql` dentro do container `monitorawifi-postgres`
  3. `docker build` de `monitorawifi-api:b004a4a` e `monitorawifi-web:b004a4a`
  4. Substituição dos containers `monitorawifi-api` e `monitorawifi-web`
     (imagens anteriores `:7e22acd5` preservadas para rollback)
- Health check: `/healthz` e `/readyz` da API → 200; `/login` do web → 200
  (interno, via rede Docker, e externo via `https://wifi.egger.app.br`)
- Dados seed criados em produção (não existiam antes): organização "Egger",
  site "Residência Egger", usuário owner `ildemar.junior@egger.com.br` —
  login real confirmado (200) contra produção.
- Rollback: `docker run` das imagens `monitorawifi-api:7e22acd5` /
  `monitorawifi-web:7e22acd5` (mesmos parâmetros); schema novo
  (`agents`, `speed_tests` etc.) é aditivo — não quebra o código anterior,
  não precisa de rollback de schema junto.
- Pendência conhecida: sem tela de troca/recuperação de senha (Fase 1 não
  implementou isso) — senha do usuário seed só pode ser alterada
  diretamente no banco por enquanto.

## 2026-08-01 — Fase 2 concluída: speed test + página Internet com dados reais

- App/plataforma: apps/local-agent, apps/api, apps/web
- Status: speed test HTTP (download/upload/latência ociosa/sob carga/
  bufferbloat/jitter) implementado no agente com fila e reenvio próprios;
  migração `0003_speed_tests`; endpoint de telemetria estendido; novos
  `GET /sites/{id}/ping-tests` e `GET /sites/{id}/speed-tests`; página
  Internet do web consumindo esses dados com badge de proveniência.
- **Validação ponta a ponta completa e real** (containers efêmeros, já
  removidos): usuário logado via API real → token de enrolamento criado via
  API real → agente enrolado via `POST /agents/enroll` real → telemetria
  (ping + speed test) enviada via `POST /agents/{id}/telemetry` real e
  aceita (202) → página `/internet` do web renderizando os valores exatos
  enviados (312.7 Mbps download, 48.3 Mbps upload, 8.4ms latência p50 ICMP,
  13.5ms bufferbloat) com as fontes corretas ("Agente — HTTP",
  "Agente — ICMP"). Nenhum dado simulado nessa cadeia.
- CI real (GitHub Actions) verde nos três: `api-ci.yml`,
  `local-agent-ci.yml`, `web-ci.yml`.
- Pendências reais: speed test modo LAN (iPerf3) não implementado; sem
  pipeline de release do binário do agente; migrações `0002`/`0003` não
  aplicadas no banco de produção (`monitorawifi-postgres`) — decisão
  deliberadamente não tomada nesta sessão, exige autorização explícita
  (DEPLOYMENT_STANDARD.md).

## 2026-08-01 — Fase 2 (Agente local, parcial): enrolamento, heartbeat, sondas, fila offline

- App/plataforma: apps/local-agent (Go, novo) + apps/api (endpoints de agente)
- Status: 24 testes automatizados verdes (probes TCP/HTTP/DNS contra
  servidores reais de teste, fila persistente com "reinício" simulado,
  enrolamento com fake injetável, backoff exponencial, conversão de payload
  com chave de idempotência estável); `go build`/`go vet` limpos;
  `apps/local-agent/Dockerfile` construído com sucesso; migração
  `0002_agents` validada up/down contra PostgreSQL 16 real (mesma técnica de
  `docker exec`+`psql` das fases anteriores).
- Endpoints novos em apps/api: `POST /sites/{id}/agent-enrollment-tokens`,
  `POST /agents/enroll`, `POST /agents/{id}/heartbeat`,
  `POST /agents/{id}/telemetry`, `GET /sites/{id}/agents` — autenticação de
  agente via `Authorization: Bearer <secret>` (hash SHA-256, comparação em
  tempo constante), distinta da sessão de usuário.
- Pendências reais (não escondidas, ver `apps/local-agent/README.md`):
  speed test ainda não implementado; sem pipeline de release do binário do
  agente (`scripts/install.sh` depende de um GitHub Release inexistente);
  fluxo completo enrolamento→heartbeat→telemetria não validado contra
  `monitorawifi-api` em produção (só contra fakes/servidores de teste —
  decisão deliberada de não criar dados de teste no banco de produção);
  migração `0002_agents` **não aplicada em produção ainda**.
- CI: `.github/workflows/local-agent-ci.yml` criado (mesmo padrão de
  `api-ci.yml`).

## 2026-08-01 — iOS enviado ao TestFlight; web publicado em wifi.egger.app.br

### iOS TestFlight

- App: Egger Network Intelligence, bundle id `br.app.egger.network-intelligence`
- Versão/build: 0.1.0 (1), commit `7e22acd5`
- Status: **enviado com sucesso** — `ARCHIVE SUCCEEDED`, `EXPORT SUCCEEDED`,
  "Uploaded package is processing", run
  https://github.com/eggerjunior/MonitoraWiFi/actions/runs/30679778025
- Pendências: build ainda em processamento no App Store Connect no momento
  deste registro (checar TestFlight); DEVELOPMENT_TEAM=E743636TCJ e
  AppIcon (1024x1024, placeholder gerado localmente) adicionados nesta
  sessão para viabilizar o archive — trocar o ícone placeholder por um
  definitivo antes de convidar testadores externos.

### Web — wifi.egger.app.br

- Repositório/commit: `eggerjunior/MonitoraWiFi` @ `7e22acd5` (imagens
  `monitorawifi-api:7e22acd5`, `monitorawifi-web:7e22acd5`)
- Topologia: 3 containers Docker na rede `monitorawifi_net` —
  `monitorawifi-postgres` (volume `/opt/data/monitorawifi/postgres`),
  `monitorawifi-api` (sem porta publicada, só acessível via rede interna),
  `monitorawifi-web` (publicado em `127.0.0.1:8421`)
- Configuração: `/opt/apps/monitorawifi/.env` (não commitado; senha de banco
  gerada aleatoriamente nesta sessão)
- Migração `0001_init` aplicada com sucesso no banco de produção
- Health check interno (via container na mesma rede): `GET /login` (web) →
  200; `GET /readyz` (api) → 200
- nginx do Hestia para `wifi.egger.app.br` **reescrito** em
  `/home/eggerjunior/conf/web/wifi.egger.app.br/nginx.conf` e `nginx.ssl.conf`
  (mesmo padrão já usado em `ged.egger.app.br`/`monitor.egger.app.br`):
  `proxy_pass http://127.0.0.1:8421`
- **Reload do nginx feito** (autorizado explicitamente pelo usuário): via SSH
  (`root@2.25.189.37`, chave `/config/.ssh/monitoravps_deploy`), `nginx -t`
  validado e `systemctl reload nginx` executado com sucesso.
  **`https://wifi.egger.app.br` está no ar**, servindo `/login` real (HTTP
  200, confirmado por `curl` externo).
- Rollback: parar/remover os 3 containers `monitorawifi-*` e reverter os dois
  arquivos de nginx para a versão anterior (backup não foi feito
  explicitamente — o conteúdo original é o template padrão do Hestia,
  regenerável via "Rebuild Web Domain" no painel), depois `systemctl reload nginx`.

## 2026-07-31 — Fase 0 (Descoberta) concluída

- App/plataforma: N/A (nenhum código de produto ainda; entrega documental)
- Versão/build/commit: N/A — repositório ainda não é um git repo
- Status: documentos de descoberta completos e revisados (resumo executivo,
  limitações técnicas, arquitetura, fluxo de dados, threat model, capability
  matrix, estratégia LiDAR, modelo de dados, ADR-001 a ADR-010, roadmap, critérios
  de aceite da Fase 1, verificações pendentes na instalação UniFi real)
- Comandos executados: criação do esqueleto do monorepo (`mkdir -p apps/... packages/...
  infrastructure/... docs/...`), pesquisa web para confirmar comportamento real de
  `NEHotspotNetwork`, ARKit/LiDAR e APIs oficiais UniFi (`developer.ui.com`) antes de
  escrever a capability matrix
- Artefatos gerados: ver `docs/architecture/`, `docs/security/threat-model.md`,
  `docs/unifi/`
- Pendências: iniciar Fase 1 (Fundação); responder as 18 perguntas de
  `docs/unifi/verificacoes-pendentes-instalacao.md` com acesso à instalação real

## 2026-07-31 — Fase 1 (Fundação) — backend e web validados; iOS não compilado

- App/plataforma: apps/api (Go), apps/web (Next.js 16), apps/ios (Swift, não compilado)
- Versão/build/commit: sem tag de release ainda; iOS em `project.yml` MARKETING_VERSION=0.1.0 CURRENT_PROJECT_VERSION=1 GIT_COMMIT=dev (build local, sem commit real ainda)
- Status:
  - apps/api: `go build`/`go vet`/`go test` verdes; migração `0001_init`
    validada up/down contra PostgreSQL 16 real; login/RBAC/health/readiness
    validados via HTTP real contra o backend rodando em container.
  - apps/web: `tsc --noEmit`/`lint`/`build` verdes; fluxo completo login → cookie
    → dashboard com dados reais validado ponta a ponta contra apps/api + Postgres reais.
  - apps/ios: código escrito (login, Keychain, navegação adaptativa, detecção
    de LiDAR, versionamento) mas **não compilado nesta sessão** — sem Xcode/Swift
    disponíveis neste ambiente Linux. Bloqueador real, não apenas formalidade.
  - CI: `.github/workflows/api-ci.yml`, `web-ci.yml`, `ios-ci.yml`,
    `ios-testflight.yml` (manual), `security-scan.yml` criados; nenhum rodou
    ainda de verdade (sem GitHub remote nesta sessão).
- Comandos executados: `go mod tidy`, `go build/vet/test`, `docker build`
  (Dockerfile multi-stage do backend), containers efêmeros de Postgres/API/Web
  compartilhando network namespace para validação ponta a ponta, `npx
  create-next-app`, `npm run build/lint`, `node scripts/generate.mjs`
  (design tokens).
- Artefatos gerados: ver `apps/api/`, `apps/web/`, `apps/ios/`,
  `packages/design-tokens/`, `infrastructure/database/migrations/0001_init.*`,
  `infrastructure/docker/docker-compose.dev.yml`, `.github/workflows/`.
- Pendências: compilar/testar `apps/ios` num Mac real ou via `ios-ci.yml`;
  `git init` já feito localmente nesta sessão (branch `main`, sem commits) —
  falta decisão do usuário sobre criar o remote GitHub e fazer o primeiro
  commit; aplicar versionamento formal a `apps/api`/`apps/web`; responder as
  18 perguntas de `docs/unifi/verificacoes-pendentes-instalacao.md` antes da
  Fase 3.

## 2026-08-01 — Git remoto criado, iOS CI validado, Bundle ID criado

- App/plataforma: repositório GitHub + apps/ios (Swift)
- Versão/build/commit: iOS `project.yml` MARKETING_VERSION=0.1.0
  CURRENT_PROJECT_VERSION=1 GIT_COMMIT=dev (nenhum archive de release feito
  ainda); commit `10f35d0` (HEAD no momento deste registro)
- Status:
  - Repositório privado criado: https://github.com/eggerjunior/MonitoraWiFi
    (`gh repo create` via script `ensure_private_github_repo.sh` da skill
    `ildemar_ios-native-testflight`). Commit inicial (148 arquivos) + 7 commits
    de correção enviados para `main`.
  - Secrets configurados no repositório (nunca impressos/logados):
    `IOS_ASC_KEY_ID`, `IOS_ASC_ISSUER_ID`, `IOS_ASC_KEY_P8_BASE64` (gerado a
    partir de `/config/.appstoreconnect/private_keys/AuthKey_37943WH8RQ.p8`).
  - **`iOS CI (build validation)` rodou de verdade em runner `macos-26`
    (Xcode 26.6) e está verde** — `xcodebuild build` para
    `generic/platform=iOS Simulator` compila sem erro. 5 problemas reais
    corrigidos ao longo de 7 iterações de CI (ver commits `6b29341` a
    `bc29a2c`): API `List(_:selection:rowContent:)` removida do SwiftUI,
    nome de simulador fixo (`iPhone 16`, não existe mais no lineup atual),
    `sed -E`/`\s` incompatível com BSD sed do macOS, ambiguidade de
    arch/UDID de simulador.
  - Execução de testes unitários (`xcodebuild test`/`build-for-testing`) via
    CI **não foi resolvida**: falha consistente com "Could not find test
    host for EggerNetworkIntelligenceTests" mesmo com dependência de target
    declarada, `-derivedDataPath` isolado, duas fases
    (`build-for-testing`+`test-without-building`) e destino por UDID único.
    Removido do `ios-ci.yml` (mantém só build) — mesmo padrão do template de
    referência da skill. Registrado como pendência real para investigação
    com Xcode de verdade.
  - Bundle ID `br.app.egger.network-intelligence` criado via API
    (`apps/ios/scripts/create_app.py`, `POST /v1/bundleIds`, id
    `44994AHNQU`). App record em App Store Connect **não existe** —
    confirmado bloqueio permanente da Apple (`POST /v1/apps` → 403
    `FORBIDDEN_ERROR` para qualquer chave), pendência manual do Ildemar.
- Comandos executados: `git init`/`add`/`commit`/`push`; `gh repo create`
  (via script); `gh secret set` (3x); `gh workflow run "iOS CI (build
  validation)"` (7x, iterando); `gh run watch`/`gh run view --log-failed`
  para diagnosticar cada falha; `python3 apps/ios/scripts/create_app.py`
  (JWT ES256 assinado com a chave real, nunca exibida).
- Pendências reais:
  1. **Ildemar**: criar app record em App Store Connect (Bundle ID já
     existe, aparecerá na lista) — só então `iOS TestFlight release` pode
     rodar.
  2. Investigar com Xcode real por que testes unitários não rodam sob
     `xcodebuild` headless.
  3. Ao rodar o primeiro archive de Release, observar o risco conhecido de
     esgotamento de cota de certificados (`Automatic signing`, ver
     `references/ildemar-ios-release.md` da skill) — aplicar a correção de
     Manual signing documentada lá se ocorrer, sem reinvestigar do zero.
  4. Versionamento formal (`ildemar_app-versioning`) ainda não aplicado a
     `apps/api`/`apps/web`.
