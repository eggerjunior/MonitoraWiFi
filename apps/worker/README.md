# apps/worker — Processamento assíncrono

Status: Fase 7 (início). Go, execução única (não é daemon/cron ainda).

## O que existe nesta fase

- **`internal/baseline`**: detecção de anomalias estatísticas explicáveis
  (roadmap Fase 7 — "baseline por hora/dia da semana"). Agrupa amostras por
  bucket (hora + dia da semana), calcula média/desvio-padrão, e reporta
  como anomalia toda amostra recente cujo z-score exceda o threshold
  (padrão 3.0). **Nunca reporta anomalia sem baseline histórico
  suficiente** (`MinBucketSamples = 5`) — "sem dado suficiente" é sempre
  preferível a um falso positivo.
- **`cmd/worker`**: conecta no mesmo Postgres do backend, lista sites com
  pelo menos um agente enrolado, lê `ping_tests` dos últimos 30 dias,
  separa em histórico (mais de 24h atrás) vs. recente (últimas 24h),
  calcula o baseline e grava anomalias detectadas na tabela `anomalies`
  (migração `0007_anomalies`, idempotente — reexecutar não duplica).
- Backend expõe `GET /sites/{id}/anomalies` (só leitura) para consumo
  futuro por web/iOS.

## Testado nesta sessão

- `internal/baseline`: testes reais com estatística conhecida de antemão
  (média/desvio-padrão calculados à mão e comparados), outlier
  corretamente detectado, valor normal corretamente NÃO sinalizado, e dois
  casos de "nunca finge certeza sem dado suficiente" (bucket com poucas
  amostras, bucket sem nenhuma amostra histórica).
- `cmd/worker` validado ponta a ponta com containers efêmeros reais
  (Postgres real, dados sintéticos inseridos diretamente via SQL — 8
  amostras históricas em torno de 20ms + 1 amostra recente de 500ms):
  detectou exatamente a anomalia esperada, com `bucket_mean`/`z_score`
  corretos: gravou 1 anomalia, reexecução não duplicou (idempotência
  confirmada), e um site sem agente foi corretamente ignorado (não
  processado, não gerou erro).

## Pendências reais

- **Não há agendamento** — hoje só roda via `docker run` manual/pontual.
  Decisão de infraestrutura (cron do host, systemd timer, ou dispatch
  agendado) fica para a Fase 8.
- **Produção não tem histórico real ainda** — nenhum agente foi enrolado
  contra o site real do usuário (ver `apps/local-agent/README.md`); rodar
  o worker contra produção hoje reportaria corretamente "sem histórico
  suficiente" para todo site, o que é o comportamento certo, não um bug.
- Só a métrica `ping_latency_ms_p50` está coberta; speed test
  (download/upload/bufferbloat) usa a mesma infraestrutura (`internal/baseline`
  já é genérico o bastante — só falta ligar a leitura de `speed_tests` no
  `cmd/worker`).
- Motor de correlação/diagnóstico e recomendações com
  evidência/confiança/impacto/risco (resto do roadmap da Fase 7) não
  começaram.

## Rodar localmente

```bash
export DATABASE_URL=postgres://egger:devpass@localhost:5432/egger?sslmode=disable
go run ./cmd/worker
```

Requer as migrações da Fase 1-3 já aplicadas (mesmo banco do
`apps/api`) e pelo menos um agente enrolado com histórico de
`ping_tests` — caso contrário, reporta corretamente "nenhum site com
agente enrolado ainda" e termina sem erro.
