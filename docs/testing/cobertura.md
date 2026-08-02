# Cobertura de testes (Fase 8)

Medida pela primeira vez em 2026-08-02 (`go test ./... -cover` em cada
módulo). Números reais, não estimativa — reproduzível com o mesmo comando
em cada `apps/{api,local-agent,worker}`.

## apps/api

| Pacote | Cobertura |
|---|---|
| `internal/auth` | 89.7% |
| `internal/httpapi` | 69.2% |
| `internal/rdap` | 78.8% |
| `internal/store`, `internal/config`, `internal/db`, `internal/logging`, `internal/ratelimit`, `internal/telemetry`, `cmd/api`, `cmd/migrate` | sem testes automatizados |

## apps/local-agent

| Pacote | Cobertura |
|---|---|
| `internal/unifi` | 88.9% |
| `internal/probes` | 83.6% |
| `internal/apiclient` | 81.0% |
| `internal/queue` | 80.0% |
| `internal/state` | 72.2% |
| `internal/agent` | 44.6% |
| `cmd/agent`, `internal/config` | sem testes automatizados |

## apps/worker

| Pacote | Cobertura |
|---|---|
| `internal/baseline` | 96.7% |
| `internal/store`, `cmd/worker` | sem testes automatizados (validado manualmente contra Postgres real nesta sessão — ver `docs/development-handoff/RELEASE_LOG.md`, 2026-08-02) |

## Por que `internal/store` (api e worker) não tem teste automatizado

Decisão de arquitetura já existente no projeto, não uma lacuna desta
sessão: nenhum workflow de CI (`api-ci.yml`, `worker-ci.yml`) sobe um
serviço Postgres. Todo código que fala com Postgres real é validado
manualmente, com containers efêmeros de verdade, durante as próprias
sessões de desenvolvimento/deploy (ver `RELEASE_LOG.md` — cada migração
nova é testada up/down contra Postgres real antes de ir pra produção,
e o worker foi validado ponta a ponta contra dados reais ao estender pra
métricas de speed test em 2026-08-02). Mudar isso pra um pipeline de CI
com Postgres automatizado é uma decisão de arquitetura real (adicionar
`services:` nos workflows, gerenciar migração de schema no CI) — não foi
tomada aqui unilateralmente; fica registrada como pendência rastreável,
não como "cobertura zero por descuido".

## Por que `internal/agent` (local-agent) tem cobertura menor (44.6%)

As funções sem cobertura (`probeLoop`, `drainLoop`, `speedTestLoop`,
`heartbeatLoop`, `commandLoop`, `unifiLoop`, `Run`) são laços infinitos
(`for { select { ... } }`) que orquestram os outros componentes já
testados individualmente — testá-los exigiria mockar tickers/contextos de
cancelamento de um jeito que tende a testar a mecânica do teste, não o
comportamento real. Esses laços são validados de ponta a ponta com
containers reais (agente real rodando contra backend real), não com teste
unitário — mesmo padrão de decisão que `internal/store`.

## O que foi melhorado nesta sessão

- `apps/api/internal/auth`: 51.7% → 89.7% — `agent.go` (geração/hash de
  secret de agente e token de enrolamento) não tinha nenhum teste próprio,
  apesar de ser código de segurança (só era exercitado indiretamente via
  fluxo de enrolamento em `httpapi`). Testes novos verificam
  unicidade/imprevisibilidade dos tokens gerados e que hash nunca é igual
  ao valor em texto claro.
- `apps/local-agent/internal/agent`: 43.2% → 44.6% — `toSpeedTestPayload`
  (conversão pura, chave de idempotência) e `probeByProtocol` (dispatcher
  de protocolo, os 4 protocolos + caso desconhecido) ganharam teste direto
  contra sondas reais.
