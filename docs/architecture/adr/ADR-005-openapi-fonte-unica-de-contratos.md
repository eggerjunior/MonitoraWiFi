# ADR-005 — OpenAPI 3.1 como fonte única de verdade dos contratos

## Status
Aceito

## Contexto
iOS (Swift), Web (TypeScript) e o próprio backend (Go) precisam concordar
exatamente sobre formato de request/response, incluindo o campo `source`/proveniência
descrito em `02-arquitetura-proposta.md` §2.6. Divergência entre clientes e servidor
sobre esse contrato é o tipo de bug mais caro de rastrear em um sistema com três
plataformas cliente.

## Decisão
`packages/contracts` contém a especificação OpenAPI 3.1 do backend como fonte única
de verdade. SDKs para Swift e TypeScript são gerados ou fortemente derivados dessa
especificação (Seção 17) — nunca escritos manualmente em paralelo. Mudança de contrato
= mudança no OpenAPI primeiro, depois regeneração dos SDKs, depois implementação.

## Consequências
- Elimina uma classe inteira de bugs de "campo renomeado só de um lado".
- Exige disciplina de CI: geração de SDK roda automaticamente e falha o build se o
  SDK gerado divergir do commitado (evita "esqueci de regenerar").
- Contract tests (Seção 21) validam que o backend realmente implementa o que o
  OpenAPI declara, fechando o ciclo de confiança.

## Alternativas consideradas
- **Cada plataforma define seus próprios tipos e sincroniza manualmente**: rejeitado
  — é exatamente o padrão de erro que este ADR existe para evitar, e não escala com
  3 plataformas cliente mais o próprio backend.
- **GraphQL como camada de contrato**: não adotado nesta fase — o documento-fonte já
  pede REST + OpenAPI 3.1 explicitamente (Seção 17), e GraphQL adicionaria complexidade
  de gateway sem resolver um problema que REST versionado não resolve aqui.
