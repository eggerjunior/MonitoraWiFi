# ADR-006 — Identidade do agente: credencial rotacionável já, mTLS como alvo

## Status
Aceito (com trabalho de segurança pendente rastreado)

## Contexto
O agente precisa se autenticar perante o backend de forma que (a) não exija
intervenção manual complexa na instalação ("curl | sh" simples, Seção 22), (b) seja
rotacionável sem reinstalar o agente, e (c) minimize o risco de um agente comprometido
se passar por outro site (threat model §2.1, Spoofing).

## Decisão
Fase 2 implementa autenticação do agente via **credencial rotacionável** (token de
enrolamento de uso único que troca por um token de longa duração vinculado
unicamente àquele `Agent.id`, renovável via endpoint próprio, revogável pelo backend).
**mTLS mútuo** é o alvo de médio prazo (rastreado como débito técnico desde já, não
como "nice to have" esquecível) para instalações que exigirem garantia mais forte de
identidade — a ser adotado quando o volume de sites justificar a complexidade
operacional de emissão/rotação de certificados por agente.

## Consequências
- Instalação inicial simples (Seção 22: "instalação simples: curl ... | sh") não fica
  bloqueada por complexidade de PKI desde o dia 1.
- Rotação de credencial (Seção 2.2) é possível sem reinstalar o agente.
- Registrado como item de segurança pendente em `threat-model.md` §5, para não ser
  esquecido silenciosamente quando a Fase 8 (produção) chegar.

## Alternativas consideradas
- **Somente mTLS desde o início**: mais forte, mas eleva a barreira de instalação
  (emissão e distribuição de certificado por site) de forma desproporcional ao risco
  real da Fase 2 (poucos sites, ambiente controlado de desenvolvimento).
- **API key estática sem rotação**: rejeitado — viola "rotação de chaves" (Seção 2.2)
  como requisito explícito de segurança.
