# apps/local-agent — Agente local de monitoramento (LAN)

Status: Fase 2 (parcial). Go, binário único + imagem Docker. Conexão
**outbound-only** para o backend (ADR-001) — nunca escuta porta de entrada.

## O que existe nesta fase

- **Enrolamento** (`internal/agent/identity.go`): troca um `ENROLLMENT_TOKEN`
  de uso único (gerado via `POST /sites/{id}/agent-enrollment-tokens`) por
  uma credencial de longa duração, persistida em `STATE_FILE` (0600).
- **Heartbeat** periódico (`HEARTBEAT_INTERVAL_SECONDS`, padrão 30s).
- **Sondas ativas** (`internal/probes`): ICMP (melhor esforço — depende de
  permissão do SO, ver comentário em `icmp.go`), TCP, HTTP, DNS. Calculam
  p50/p95/p99 e jitter a partir de múltiplas amostras — nunca inventam
  latência quando todas as tentativas falham (100% de perda é reportado
  como tal).
- **Fila offline** (`internal/queue`): buffer em disco (JSON Lines,
  `QUEUE_FILE`) — sobrevive a reinício do processo. Item mais antigo é
  descartado (com aviso, não silenciosamente) quando `QUEUE_MAX_ITEMS` é
  excedido.
- **Reenvio idempotente com backoff exponencial**: cada resultado carrega
  uma `idempotency_key` estável; o backend ignora duplicatas. Falha de envio
  dobra o intervalo de retry (5s → até 10min), resetado no primeiro sucesso.
- **Speed test HTTP** (`internal/probes/speedtest.go`, Seção 5.3): download e
  upload contra URLs configuráveis (nunca um servidor de terceiros escolhido
  sozinho pelo agente), com bufferbloat medido pela diferença entre latência
  ociosa e latência sob carga durante a transferência. Desativado por padrão
  (`SPEEDTEST_ENABLED` implícito por `SPEEDTEST_DOWNLOAD_URL`/
  `SPEEDTEST_UPLOAD_URL` não vazios) — fila e reenvio próprios, mesmo padrão
  do buffer de ping.
- **Speed test LAN via iPerf3** (`internal/probes/iperf3.go`): download (`-R`)
  e upload contra um servidor iperf3 já em execução em outro nó da LAN —
  o agente nunca sobe seu próprio servidor nem escolhe um alvo por conta
  própria. Chama o binário `iperf3` via subprocesso e parseia a saída
  `-J` (reimplementar o protocolo em Go não valeria o risco frente ao
  binário oficial). Se o binário não existir no PATH, cada execução reporta
  erro explícito (nunca finge throughput) e isso é logado uma vez na
  inicialização do agente. Desativado por padrão
  (`SPEEDTEST_LAN_TARGET` vazio).

## Testado nesta sessão

29 testes automatizados, todos rodando operações reais (não mocks opacos):
sondas TCP contra um listener TCP real, HTTP contra um `httptest.Server`
real, DNS resolvendo nomes de verdade (`localhost`, domínio inexistente),
speed test de download/upload contra servidores de teste reais medindo
throughput de verdade, fila persistindo e sobrevivendo a "reinício"
simulado, cliente HTTP validado contra um servidor de teste real.
`go build`/`go vet`/`go test` verdes; Dockerfile de produção construído com
sucesso (2x, incluindo após adicionar o speed test).

## Não testado ainda / pendências reais

- **Enrolamento/heartbeat/telemetria não validados contra o backend em
  produção real** — testados com fakes (unitário) e com `httptest.Server`
  (contrato HTTP), não com uma chamada real fim a fim contra
  `monitorawifi-api`. Não criei dados de teste em produção para isso
  deliberadamente (evitar poluir o banco real): confirmado em 2026-08-01
  que a tabela `agents` em produção está vazia (0 linhas) — nenhum agente
  real foi enrolado ainda.
- **ICMP depende de capability do SO** (`CAP_NET_RAW` ou
  `net.ipv4.ping_group_range`) — dentro de containers sem essa capability,
  cai para 100% de perda reportada (nunca finge sucesso). Documentado em
  `internal/probes/icmp.go`.
- **Comparação entre resolvedores DNS** ainda não implementada.

## Resolvido nesta sessão (2026-08-01)

- **Migrações `0002_agents` e `0003_speed_tests` já estão aplicadas em
  produção** — confirmado inspecionando `monitorawifi-postgres` diretamente
  (`\d agents`, `\d speed_tests` batem exatamente com as migrações). A nota
  anterior aqui estava desatualizada.
- **Pipeline de release do binário criado e validado ponta a ponta**
  (`.github/workflows/local-agent-release.yml`, `workflow_dispatch` manual —
  mesmo padrão do TestFlight: publicar um binário é decisão, não efeito
  colateral de push). Lê a versão de `apps/local-agent/VERSION` (fonte
  única), roda os testes como gate, cross-compila
  `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` com
  `CGO_ENABLED=0` e a versão+commit injetados via `-ldflags -X`, e publica
  um GitHub Release (`--latest`) com os binários nomeados exatamente como
  `scripts/install.sh` espera (`egger-agent-<os>-<arch>`) mais um
  `SHA256SUMS.txt`. Primeiro release publicado: `agent-v0.1.0`
  (https://github.com/eggerjunior/MonitoraWiFi/releases/tag/agent-v0.1.0).
- **Achado e resolvido durante a validação**: `curl` sem autenticação no
  link público do asset retornava 404 — o repositório era **privado**, e o
  GitHub não serve assets de release de repos privados sem autenticação
  (`gh release download` funcionava, `curl` puro não). Pergunta explícita
  ao usuário (`AskUserQuestion`: tornar público / hospedar em
  `wifi.egger.app.br` / manter privado usando `gh` por enquanto) — escolheu
  tornar o repositório público. Antes de mudar a visibilidade, auditei toda
  a história do git em busca de segredos reais (chaves privadas, `.env`
  real, certificados) — não havia nenhum, só `.env.example` com
  placeholders de desenvolvimento; `.gitignore` já excluía `*.p8`, `*.pem`,
  `*.key`, `.env*` desde o início. Repositório tornado público
  (`gh repo edit --visibility public`) e o download não-autenticado
  confirmado funcionando de verdade (`curl -fsSL .../releases/latest/download/egger-agent-linux-amd64`
  baixou o binário certo, com a versão `0.1.0+dd95c8a7` embutida).
- **Speed test modo LAN (iPerf3) implementado** (`internal/probes/iperf3.go`)
  — testado com um servidor `iperf3` real (subprocesso) rodando dentro do
  teste automatizado, não um mock do protocolo: 3 testes cobrindo
  download+upload reais, servidor indisponível (erro honesto, sem
  throughput inventado) e alvo inválido. Suite completa continua verde
  (`go build`/`go vet`/`go test ./...`).

## Variáveis de ambiente do speed test

```bash
SPEEDTEST_DOWNLOAD_URL=https://sua-api/testfile-10mb   # arquivo controlado, não um CDN de terceiros arbitrário
SPEEDTEST_UPLOAD_URL=https://sua-api/upload-sink
SPEEDTEST_UPLOAD_SIZE_BYTES=4194304                     # 4 MiB, padrão
SPEEDTEST_LATENCY_TARGET=1.1.1.1:443                    # host:porta para medir latência ociosa/sob carga
SPEEDTEST_INTERVAL_MINUTES=30

SPEEDTEST_LAN_TARGET=192.168.1.50:5201  # host:porta de um servidor iperf3 já em execução na LAN — vazio desativa o modo LAN
SPEEDTEST_LAN_DURATION_SECONDS=5        # duração de cada direção (download/upload), padrão 5s
```

Requer o binário `iperf3` instalado no host onde o agente roda (não incluído
no `Dockerfile` por padrão — ver `apps/local-agent/Dockerfile` se for rodar
em container).

Sem `SPEEDTEST_DOWNLOAD_URL`/`SPEEDTEST_UPLOAD_URL`, o speed test fica
desativado — o agente nunca escolhe um servidor de terceiros por conta
própria (Seção 5.3).

## Uso local (desenvolvimento)

```bash
export BACKEND_URL=http://localhost:8080/api/v1
export ENROLLMENT_TOKEN=<gerado via POST /sites/{id}/agent-enrollment-tokens>
export STATE_FILE=./tmp/agent.json
export QUEUE_FILE=./tmp/queue.jsonl
go run ./cmd/agent
```

## Instalação em produção (systemd)

```bash
curl -fsSL https://raw.githubusercontent.com/eggerjunior/MonitoraWiFi/main/apps/local-agent/scripts/install.sh \
  | BACKEND_URL=https://sua-api/api/v1 ENROLLMENT_TOKEN=<token> sudo sh
```

Requer um release publicado via `.github/workflows/local-agent-release.yml`
(`gh workflow run "Local Agent release"`) — enquanto não houver um release
disparado, usar o Dockerfile ou compilar localmente.
