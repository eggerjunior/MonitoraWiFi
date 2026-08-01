# ADR-011 — Fila de comando sob demanda via Postgres, não Redis

## Status
Aceito

## Contexto
`03-fluxo-de-dados.md` §3.2 ("Teste sob demanda disparado pelo usuário") descreve o
mecanismo de comando usuário→backend→agente usando **Redis** como fila/pub-sub. Redis
já está presente no `docker-compose.dev.yml` (Fase 1) exatamente para isso, mas nunca
foi implantado em produção (`docker-compose.prod.yml`/o deploy real em
`monitorawifi-postgres`/`monitorawifi-api` não incluem um serviço Redis) — a Fase 1/2
não chegaram a precisar dele de fato.

Ao implementar o primeiro comando sob demanda real (Fase 5, início — "ping agora"),
essa decisão precisou ser tomada de verdade: introduzir Redis em produção só para
isso, ou usar o Postgres que já está lá.

## Decisão
A fila de comando é implementada inteiramente em **Postgres** (tabela
`agent_commands`, migração `0004_agent_commands`), com o agente fazendo **polling**
periódico (`GET /agents/{id}/commands`, mesmo padrão de conexão outbound do
heartbeat) em vez de assinar um canal pub/sub. O "claim" de comandos pendentes usa
`UPDATE ... WHERE id IN (SELECT ... FOR UPDATE SKIP LOCKED)` — seguro contra o agente
reconectando/repetindo o poll concorrentemente, sem precisar de Redis para isso.

Redis **continua** no `docker-compose.dev.yml` e na arquitetura documentada como
opção futura — esta decisão não o remove do produto, só adia sua introdução em
produção até haver uma necessidade real (ver Consequências).

## Consequências
- Zero novas dependências de infraestrutura stateful em produção para este recurso —
  o backend continua com uma única dependência de dados (Postgres).
- Latência de "usuário disparou → agente percebeu" é limitada pelo intervalo de
  polling (`COMMAND_POLL_INTERVAL_SECONDS`, padrão 5s) em vez de near-instantâneo
  como pub/sub — aceitável para testes sob demanda (o usuário já espera alguns
  segundos para o resultado de um ping/speed test de qualquer forma).
- Não escala para múltiplas réplicas do backend competindo por polling
  distribuído/pub-sub em tempo real de forma tão natural quanto Redis — se/quando o
  backend precisar rodar com múltiplas réplicas atrás de um load balancer e a
  latência de polling deixar de ser aceitável, revisitar esta decisão e introduzir
  Redis (ou outro pub/sub) nesse momento, não antes.
- `03-fluxo-de-dados.md` §3.2 permanece como a descrição conceitual do fluxo
  (usuário→backend→agente→backend→usuário); este ADR documenta que a implementação
  real da "fila de comando" ali citada é Postgres, não Redis, até segunda ordem.

## Alternativas consideradas
- **Redis pub/sub, como o documento-fonte original descrevia**: mais próximo de
  tempo real, mas adiciona uma dependência stateful nova em produção (deploy,
  backup, monitoramento) para um ganho de latência que não é necessário na Fase 5.
- **WebSocket/SSE do agente para o backend**: não violaria ADR-001 em si (a conexão
  ainda seria iniciada pelo agente), mas manter uma conexão persistente de longa
  duração por agente tem complexidade operacional própria (reconexão, keep-alive,
  proxies intermediários) maior que polling simples — não justificada para o volume
  de sites desta fase.
