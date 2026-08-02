# Manual do usuário — Egger Network Intelligence

Primeira versão (Fase 8, 2026-08-02). Cobre só o que existe de verdade em
produção hoje — nada aqui descreve uma funcionalidade planejada e ainda
não implementada (ver `docs/architecture/06-roadmap.md` pro que falta).

## Acesso

Web: `https://wifi.egger.app.br`. iOS: TestFlight (convite necessário —
peça ao administrador da organização). Login com e-mail e senha
cadastrados por um administrador; não há autoatendimento de cadastro
ainda.

## Papéis (RBAC)

Cinco papéis, cada um com um conjunto fixo de permissões: `owner`,
`administrator`, `operator`, `viewer`, `auditor`. Só quem tem a permissão
`manage_integrations` pode gerar token de enrolamento de agente; só quem
tem `run_tests` pode disparar as ferramentas de diagnóstico. Se um botão
não aparece ou retorna "papel sem permissão", é o RBAC, não um bug.

## Pré-requisito: um agente local enrolado

A maioria das telas (Internet, Wi-Fi, Dispositivos, Switches, Clientes,
Diagnósticos) depende de um **agente local** rodando dentro da mesma LAN
da instalação UniFi — sem ele, essas telas mostram um estado vazio
honesto, nunca um dado inventado. Peça ao administrador pra confirmar que
há um agente enrolado no seu site (`docs/deployment/runbook-producao.md`
tem o passo a passo técnico).

## Telas disponíveis

- **Visão geral**: organizações e sites cadastrados.
- **Internet**: histórico de speed test e ping do agente.
- **Wi-Fi**: access points sincronizados do UniFi, com contagem de
  clientes sem fio por AP.
- **Dispositivos**: inventário completo de dispositivos UniFi (APs,
  switches, gateway).
- **Switches**: mesma fonte de Dispositivos, filtrada só pros switches,
  com contagem de clientes cabeados.
- **Clientes**: dispositivos conectados à rede (cabeados e sem fio).
- **Diagnósticos**: ferramentas sob demanda — ping, ping em lote, DNS
  lookup, traceroute, SSL/TLS checker, RDAP/WHOIS, HTTP client, LAN
  scanner, Wake-on-LAN, port scanner, calculadora de sub-rede. Cada uma
  dispara um comando real executado pelo agente (ou, no caso de
  RDAP/WHOIS, direto pelo backend) — o resultado só aparece depois que a
  execução real termina, nunca antes.
  - **LAN scanner** e **port scanner** só aceitam alvo dentro da sua
    própria rede local (RFC 1918) — não servem pra varrer redes de
    terceiros.
  - **Wake-on-LAN** precisa do endereço MAC do dispositivo a ligar.
- **Alertas**: anomalias estatísticas reais detectadas pelo worker de
  baseline (compara o valor atual contra o padrão histórico do mesmo
  horário/dia da semana). Só aparece depois de haver histórico
  suficiente — "nenhuma anomalia" pode significar tanto "tudo normal"
  quanto "ainda sem dado suficiente pra comparar".
- **Histórico**: lista dos testes de ping/speed test e anomalias mais
  recentes.
- **Mapa**: ainda não implementado (Fase 6, depende de hardware LiDAR
  real — iPhone/iPad Pro).
- **Relatórios**: ainda não implementado (Fase 7).

## Coisas que ainda não existem (não é bug, é roadmap)

- Não há exportação de relatório (PDF/CSV) ainda.
- Não há detalhe de rádio (canal, potência) nem estatística por porta de
  switch — só o que já foi confirmado contra a instalação real.
- Não há tela de mapa/levantamento espacial (Fase 6).
- Revogar um agente comprometido exige contato com quem administra o
  backend (não é uma ação de usuário final ainda).
