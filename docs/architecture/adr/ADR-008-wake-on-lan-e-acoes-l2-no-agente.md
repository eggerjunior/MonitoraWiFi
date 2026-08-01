# ADR-008 — Wake-on-LAN e ações de broadcast/L2 executadas pelo agente, nunca pelo app iOS

## Status
Aceito

## Contexto
Enviar um magic packet (Wake-on-LAN) requer UDP broadcast/multicast. Em iOS isso
exige o entitlement `com.apple.developer.networking.multicast` (aprovação da Apple
por app) e, mesmo com o entitlement concedido, há relatos ativos e não resolvidos de
falhas `EACCES` em versões recentes de iOS/iPadOS (Apple Developer Forums, threads
805690, 805719, 770023) — comportamento fora do nosso controle e não confiável para
uma funcionalidade que o usuário espera que simplesmente funcione.

## Decisão
Wake-on-LAN (Seção 5, "Funções do agente") e qualquer outra ação de broadcast/L2 são
executadas exclusivamente pelo **agente local**, que está dentro da mesma LAN do alvo
e não tem restrição de sandbox de app móvel. iOS e Web apenas enviam o comando ao
backend, que o roteia ao agente do site (mesmo canal de comando de
`03-fluxo-de-dados.md` §3.2).

## Consequências
- Comportamento consistente e testável (o agente roda em Linux/macOS sem as
  restrições documentadas do iOS).
- App iOS não precisa solicitar o entitlement Multicast Networking à Apple para essa
  funcionalidade — reduz superfície de aprovação/risco de rejeição na App Store.
- Wake-on-LAN só funciona em sites que tenham um agente ativo — aceitável, pois o
  agente já é infraestrutura obrigatória do produto (Seção 3), não uma dependência
  nova introduzida só para esta funcionalidade.

## Alternativas consideradas
- **Implementar WoL diretamente no app iOS com o entitlement Multicast**: rejeitado —
  dependeria de aprovação discricionária da Apple por app e tem bugs de plataforma
  documentados e não resolvidos até a data desta decisão.
