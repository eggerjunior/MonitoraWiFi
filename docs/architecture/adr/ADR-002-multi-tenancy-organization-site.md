# ADR-002 — Multi-tenancy Organization → Site desde o primeiro schema

## Status
Aceito

## Contexto
A instalação inicial é uma única residência, mas a Seção 1 do documento-fonte exige
suporte a "uma ou várias instalações; vários sites UniFi; diferentes gateways" desde
o princípio, não como evolução futura. Adicionar multi-tenancy depois do fato costuma
exigir migração de dados dolorosa e reescrita de queries.

## Decisão
Toda entidade de negócio pertence a um `Site`, que pertence a uma `Organization`,
desde o modelo de dados inicial (`05-modelo-dados.md`). RBAC (Seção 18) é avaliado no
nível de `Organization` e pode ser restrito por `Site`. Toda query de repositório no
backend filtra explicitamente por `organization_id`/`site_id` — nunca é opcional ou
inferido via join implícito.

## Consequências
- Uma única instalação (o caso de uso inicial) é modelada como `Organization` com um
  `Site`, sem tratamento especial — não há "modo single-tenant" separado para manter.
- Testes de isolamento entre tenants entram no pipeline de CI desde a Fase 1 (ver
  threat model §2.2), porque a superfície de risco já existe desde o primeiro commit.
- Pequeno overhead de modelagem (uma FK a mais em quase toda tabela) pago uma única
  vez, evitando reescrita estrutural quando um segundo site for cadastrado.

## Alternativas consideradas
- **Modelar só para um site e generalizar depois**: rejeitado — contradiz
  explicitamente a Seção 1 ("não poderá ficar limitada a essa instalação") e o custo
  de retrofit de multi-tenancy em um sistema com séries temporais já é
  desproporcionalmente mais alto do que modelar corretamente desde o início.
