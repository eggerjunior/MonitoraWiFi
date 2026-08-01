# ADR-009 — Notificações críticas via provider APNs próprio, não CloudKit

## Status
Aceito

## Contexto
A Seção 10 exige que alertas críticos ("Internet caiu", "AP offline") cheguem ao
usuário mesmo com o app fechado. Existe conhecimento acumulado (skill
`ildemar_skill-notificacoes-ios`) de que depender só de `CKQuerySubscription`
(CloudKit) não é confiável em ambiente de Produção da Apple para esse cenário —
há um bug documentado da Apple com CKQuerySubscription pública em Production que pode
simplesmente não entregar a notificação. Notificação local agendada/polling também não
serve, pois o evento (ex.: WAN caiu) não é previsível com antecedência.

## Decisão
O backend do produto atua como **provider APNs próprio**: gera JWT assinado com chave
`.p8`, envia via HTTP/2 diretamente ao Apple Push Notification service quando o worker
determina que um alerta deve virar notificação push (fluxo em
`03-fluxo-de-dados.md` §3.4). O app iOS registra seu device token no backend
(endpoint próprio, não CloudKit). CloudKit não é usado como mecanismo de entrega de
alertas críticos.

## Consequências
- Entrega de notificação com app fechado depende apenas da infraestrutura própria do
  backend e da APNs (caminho testado e documentado pela Apple para esse cenário),
  não de um comportamento de CloudKit conhecidamente instável em produção.
- Backend precisa gerenciar chave `.p8`, renovação de JWT e device tokens (registro,
  invalidação quando o token expira/muda) — trabalho adicional aceito em troca de
  confiabilidade.
- E-mail e webhook (Seção 10) continuam como canais paralelos, não substitutos, para
  usuários/integrações que preferem esses canais.

## Alternativas consideradas
- **CKQuerySubscription (CloudKit)**: rejeitado para alertas críticos — falha
  conhecida em Production já documentada em experiência prévia com apps iOS deste
  mesmo portfólio (ver skill de notificações).
- **Silent push + processamento local**: mantido como possibilidade complementar para
  sincronizar estado sem acordar o usuário, mas não como veículo do alerta em si, que
  precisa ser uma notificação visível/sonora imediata.
