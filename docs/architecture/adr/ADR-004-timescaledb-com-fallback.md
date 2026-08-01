# ADR-004 — TimescaleDB quando disponível, particionamento nativo como fallback

## Status
Aceito

## Contexto
O volume dominante de dados do sistema é série temporal (métricas UniFi, resultados
de teste, heartbeats, amostras espaciais). O documento-fonte pede explicitamente
"TimescaleDB, caso disponível, ou particionamento temporal no PostgreSQL" (Seção 3) —
já reconhecendo que nem todo ambiente de deploy (ex.: um Postgres gerenciado sem a
extensão) terá Timescale disponível.

## Decisão
O schema usa hypertables do TimescaleDB como caminho principal quando a extensão está
disponível no ambiente. A camada de acesso a dados é escrita de forma a não depender
de sintaxe exclusiva do Timescale nas queries de leitura mais comuns, permitindo que o
mesmo schema opere sobre particionamento nativo (`PARTITION BY RANGE (time)`) quando a
extensão não estiver presente. A decisão de qual modo está ativo é feita na migração
inicial do ambiente, não em runtime por query.

## Consequências
- Portabilidade para ambientes de deploy variados (self-hosted do cliente, cloud
  gerenciada com ou sem Timescale) sem reescrever o modelo de dados.
- Continuous aggregates e compressão nativa do Timescale (quando presentes) são
  tratadas como otimização, não como dependência funcional — funcionalidades que as
  usam devem ter um caminho equivalente (ainda que menos eficiente) sem elas.
- Testes de banco (Seção 21) rodam contra ambos os modos antes de uma migração ser
  considerada pronta para produção, para não introduzir SQL exclusivo de um dos dois
  sem perceber.

## Alternativas consideradas
- **Depender exclusivamente de TimescaleDB**: rejeitado — o próprio documento-fonte
  pede o fallback, e exigir a extensão fecharia portas de deploy legítimas
  (ex.: alguns provedores de Postgres gerenciado não oferecem Timescale).
- **Não particionar e confiar em índices**: rejeitado para tabelas de métrica de alta
  cardinalidade temporal — não escala para retenção de meses/anos exigida pela
  Seção 16 ("aplicar políticas de retenção").
