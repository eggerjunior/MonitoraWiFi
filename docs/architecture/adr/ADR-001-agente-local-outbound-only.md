# ADR-001 — Agente local como única ponte outbound-only para a LAN

## Status
Aceito

## Contexto
O sistema precisa executar testes ativos (ping, traceroute, port scan, iPerf3),
descoberta (ARP/mDNS/SNMP) e falar com a Network API local do UniFi, todos dentro da
LAN do cliente. O backend central não está fisicamente na LAN. A Seção 3 do
documento-fonte exige explicitamente que "o agente local não poderá exigir a
abertura de portas de entrada na residência".

## Decisão
Todo acesso à LAN do cliente passa por um **agente local** (Go), instalado dentro da
rede, que inicia sempre conexões **outbound** (TLS) para o backend. O backend nunca
inicia conexão para dentro da LAN do cliente. Comandos do backend para o agente
(ex.: "rode um speed test agora") são entregues via long-poll/assinatura que o
próprio agente inicia, nunca via socket aberto pelo agente para o mundo.

## Consequências
- Nenhuma porta de entrada precisa ser aberta no roteador/firewall do cliente —
  compatível com instalações residenciais sem conhecimento de NAT/port forwarding.
- O `UniFiIntegrationProvider` roda dentro do agente (não no backend), pois é o
  agente quem tem visibilidade de rede para a Network API local do console.
- Introduz uma dependência de disponibilidade: se o agente cair, perde-se
  visibilidade ativa daquele site até reconexão — mitigado por heartbeat + alerta de
  "agente offline" (Seção 10) e buffer local com fila offline (Seção 3).
- Todo teste ativo (Fases 2 e 5) é implementado uma vez no agente e reutilizado por
  todos os clientes (iOS e Web apenas disparam e consomem resultado via backend).

## Alternativas consideradas
- **Backend fala diretamente com o console UniFi via túnel/VPN permanente**: rejeitado
  — exige infraestrutura de rede adicional no cliente e viola o requisito de não
  abrir portas de entrada; também tornaria o backend dependente de conectividade
  ponto-a-ponto complexa para múltiplos sites com NAT/CGNAT.
- **App iOS/Web fala diretamente com o console UniFi quando na mesma rede**: possível
  como funcionalidade complementar futura (ex.: modo "estou na rede local"), mas não
  substitui o agente porque testes ativos e histórico contínuo exigem um processo
  sempre ativo, o que um app móvel não garante (ver ADR relacionado a background no
  iOS, limitações §1.7).
