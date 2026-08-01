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

## Testado nesta sessão

24 testes automatizados, todos rodando operações reais (não mocks opacos):
sondas TCP contra um listener TCP real, HTTP contra um `httptest.Server`
real, DNS resolvendo nomes de verdade (`localhost`, domínio inexistente),
fila persistindo e sobrevivendo a "reinício" simulado, cliente HTTP validado
contra um servidor de teste real. `go build`/`go vet`/`go test` verdes;
Dockerfile de produção construído com sucesso.

## Não testado ainda / pendências reais

- **Sem pipeline de release do binário** — `scripts/install.sh` tenta
  baixar um binário pré-compilado de um release do GitHub que ainda não
  existe. Até isso ser criado (Fase 2, próximo passo), instalar via
  `go build` a partir do checkout ou via `apps/local-agent/Dockerfile`.
- **Enrolamento/heartbeat/telemetria não validados contra o backend em
  produção real** — testados com fakes (unitário) e com `httptest.Server`
  (contrato HTTP), não com uma chamada real fim a fim contra
  `monitorawifi-api`. Não criei dados de teste em produção para isso
  deliberadamente (evitar poluir o banco real).
- **ICMP depende de capability do SO** (`CAP_NET_RAW` ou
  `net.ipv4.ping_group_range`) — dentro de containers sem essa capability,
  cai para 100% de perda reportada (nunca finge sucesso). Documentado em
  `internal/probes/icmp.go`.
- **Speed test (download/upload/bufferbloat) ainda não implementado** —
  escopo da Seção 5 "Speed test", não coberto nesta rodada da Fase 2.
- Migração `0002_agents` **ainda não aplicada no banco de produção**
  (`monitorawifi-postgres`) — só testada localmente contra Postgres
  descartável. Aplicar exige decisão explícita antes de subir a nova versão
  do `apps/api` em produção (DEPLOYMENT_STANDARD.md: "migrações de estrutura
  serão feitas uma aplicação por vez, com backup e autorização específica").

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

Requer um release publicado (pendência acima) — enquanto isso, usar o
Dockerfile ou compilar localmente.
