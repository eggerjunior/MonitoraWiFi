# 01 — Limitações Técnicas Confirmadas

Este documento existe para que nenhuma tela ou funcionalidade prometa algo que a
plataforma não pode honestamente entregar. Cada item abaixo foi verificado contra
documentação oficial (Apple Developer, Ubiquiti/`developer.ui.com`) em 2026-07-31.
Onde a informação depende da instalação real, isso está marcado e também listado em
`docs/unifi/verificacoes-pendentes-instalacao.md`.

## 1. iOS/iPadOS — o que é possível e o que não é

### 1.1 Não existe leitura de RSSI/canal de redes vizinhas

O iOS **não tem equivalente ao CoreWLAN do macOS**. Não há API pública para listar
redes Wi-Fi próximas, seus canais, RSSI ou segurança. Isso é definitivo e não muda
com entitlements — não vamos prometer isso em nenhuma tela.

### 1.2 SSID/BSSID da rede atual — condicional

`NEHotspotNetwork.fetchCurrent(completionHandler:)` (substituto do `CNCopyCurrentNetworkInfo`,
deprecado) só retorna SSID/BSSID se o app satisfizer **uma** destas condições:

- autorização de localização precisa (Core Location, "When In Use" ou "Always"); **ou**
- o app configurou a rede atual via `NEHotspotConfiguration`; **ou**
- o app tem uma configuração VPN ativa; **ou**
- o app tem uma configuração `NEDNSSettingsManager` ativa.

Além disso é obrigatório o entitlement **Access Wi-Fi Information**
(`com.apple.developer.networking.wifi-info`), habilitado no Xcode como capability.

Decisão de produto: usaremos a via de **autorização de localização** (é a mais simples
de justificar ao usuário — "para identificar o AP UniFi ao qual você está conectado").
A tela de permissões deve explicar isso explicitamente, não esconder atrás de um texto
genérico.

### 1.3 `signalStrength` do `NEHotspotNetwork` é conhecidamente não confiável

Documentado por múltiplos desenvolvedores nos fóruns da Apple (thread 128844, 733598):
o campo `signalStrength` frequentemente retorna `0` e não é populado de forma
consistente entre versões de iOS. **Não usaremos este campo como fonte de sinal.**
Onde houver sinal a mostrar no app, a fonte será sempre o **UniFi** (RSSI/SNR
reportado pela infraestrutura para aquele cliente), nunca uma leitura feita pelo
próprio iPhone. Isso é o princípio central da Seção 6 (LiDAR) do projeto: o LiDAR
nunca mede rádio.

RSSI verdadeiro por hardware (via `NEHotspotHelper`) exige um entitlement concedido
apenas a fabricantes/operadoras mediante processo de aprovação da Apple — não obtível
para este produto. Não vamos solicitar nem depender disso.

### 1.4 Descoberta de dispositivos na LAN a partir do iOS é limitada

- Não há acesso a socket raw de camada 2 (`AF_PACKET`/BPF) em iOS — **ARP e NDP não
  são acessíveis pelo app**. Isso é trabalho exclusivo do **agente local**.
- Bonjour/mDNS: possível via `NWBrowser`/`NetServiceBrowser`, mas exige
  `NSLocalNetworkUsageDescription` (prompt de permissão "Rede Local" desde iOS 14) e
  declarar `NSBonjourServices` no `Info.plist`. Enumerar *todos* os tipos de serviço
  Bonjour (em vez de tipos declarados) exige entitlement adicional solicitado à Apple.
- Varredura ativa de sub-rede (probing de TCP/UDP em múltiplos hosts) é tecnicamente
  possível via `Network.framework`, mas dispara o mesmo prompt de Rede Local e deve
  ser deliberadamente limitada em escopo/paralelismo para não parecer comportamento
  invasivo perante a revisão da App Store.
- **Decisão**: descoberta ampla (ARP/NDP/mDNS completo/SNMP discovery) é
  responsabilidade do **agente local**, que não tem essas restrições de sandbox. O app
  iOS consome os resultados via backend. O app iOS só faz probes pontuais e
  explicitamente autorizados (ex.: "verificar se este host específico está no ar").

### 1.5 Wake-on-LAN direto do iOS é frágil

Enviar um pacote UDP broadcast/multicast em iOS exige o entitlement
**Multicast Networking** (`com.apple.developer.networking.multicast`), que precisa ser
solicitado e aprovado pela Apple por app. Mesmo com o entitlement concedido, há
relatos ativos nos fóruns da Apple (threads 805690, 805719, 770023) de `EACCES` em
broadcast UDP em versões recentes de iOS/iPadOS — comportamento instável fora do
nosso controle.

**Decisão**: Wake-on-LAN é executado pelo **agente local** (que está na própria LAN e
não tem essa restrição). O app iOS e a web apenas solicitam ao backend, que repassa
ao agente do site correspondente. Não implementaremos WoL direto do iPhone.

### 1.6 ARKit e LiDAR — hardware elegível

LiDAR Scanner (necessário para `sceneReconstruction`/malha 3D de alta qualidade) só
existe em:

- iPhone: linha **Pro/Pro Max** a partir do 12 Pro (12 Pro, 13 Pro, 14 Pro, 15 Pro,
  16 Pro, 17 Pro e sucessores);
- iPad: **iPad Pro** 11" (2ª geração, 2020) e 12.9" (4ª geração, 2020) em diante.

iPhone/iPad "normal", Air, mini e SE **nunca** têm LiDAR. `ARWorldTrackingConfiguration`
funciona nesses aparelhos (tracking por feature points), mas sem reconstrução de malha
nem detecção de plano com a mesma qualidade. Por isso a Seção 6.4 do projeto (fallback
sem LiDAR) é obrigatória, não opcional, e deve ser tratada como um modo de primeira
classe, não um caso degradado.

Detecção em runtime: `ARWorldTrackingConfiguration.supportsSceneReconstruction(.mesh)`.

### 1.7 ARKit não roda em background

Sessão AR exige câmera ativa e app em primeiro plano. `BackgroundTasks`
(`BGAppRefreshTask`/`BGProcessingTask`) não pode manter um levantamento LiDAR rodando
com o app minimizado, e o iOS não garante frequência de execução em background —
é oportunístico. Consequência: o levantamento espacial é sempre uma sessão em
primeiro plano, guiada; monitoramento contínuo em segundo plano é feito pelo agente
local + backend, não pelo app.

### 1.8 Ping ICMP é possível sem entitlement especial

Sockets ICMP não privilegiados (`SOCK_DGRAM`/`IPPROTO_ICMP`) funcionam em iOS sem
entitlement adicional (mesmo mecanismo usado por apps já publicados como utilitários
de rede). Usaremos essa técnica no app para ping simples. Traceroute via ICMP/UDP com
TTL incremental é tecnicamente viável pelo mesmo caminho, mas mais sensível a
variações entre versões de iOS — será prototipado e validado antes de prometido na UI;
até validação, traceroute "pesado" fica preferencialmente no agente local.

### 1.9 MetricKit é sobre o próprio app, não sobre a rede

`MetricKit` fornece métricas agregadas de diagnóstico do nosso app (crashes, hangs,
consumo de energia, disco) para observabilidade do produto — não é uma fonte de dados
de rede de terceiros. Será usado na Seção 20 (observabilidade do próprio sistema),
não como fonte de métricas de rede exibidas ao usuário.

### 1.10 Sem APIs privadas, sem packet sniffing

Não usaremos `NEPacketTunnelProvider` para captura de tráfego, não usaremos APIs
privadas para RSSI, não usaremos entitlements reservados a operadoras. Onde o produto
pedir algo que dependeria disso (captura de payload, lista de redes vizinhas, RSSI
por hardware), a interface mostrará "Indisponível — restrição de plataforma iOS" com
link para este documento.

## 2. UniFi — o que está confirmado publicamente (a validar na instalação real)

Fonte: Ubiquiti Help Center, `developer.ui.com` (Network API e Site Manager API),
consultados em 2026-07-31.

### 2.1 Existem duas APIs oficiais distintas, com propósitos diferentes

1. **UniFi Network API (local, por console)** — documentada em
   `developer.ui.com/network/<versão>` (ex.: v9.1.120, v10.1.68, v10.1.84+), roda
   localmente no console UniFi OS (ex.: Cloud Gateway Max). Endpoints do tipo
   "List Connected Clients", inventário de dispositivos, etc. **A versão exata do
   UniFi OS/Network instalada define quais endpoints existem** — não vamos assumir
   paridade entre versões.
2. **UniFi Site Manager API (cloud, oficial v1.0)** — base
   `https://api.ui.com/v1/`, autenticação via `X-API-KEY`, rate limit documentado de
   10.000 req/min, acessível via conta `unifi.ui.com`. Dá visão agregada
   multi-site (Internet Health Metrics, status de dispositivos) — pensada para gestão
   em escala, não para telemetria fina de rádio por AP.

### 2.2 API legada (não oficial) ainda é amplamente usada pela comunidade

Ferramentas como `unpoller`/`unifi-poller` usam endpoints não documentados oficialmente
no padrão `/api/s/{site}/...` (controlador standalone) ou
`/proxy/network/api/s/{site}/...` (atrás do UniFi OS), autenticando com credenciais de
admin local ou API key local dependendo da versão. Esses endpoints expõem telemetria
mais profunda (rádio, DPI, portas) mas **não têm garantia de estabilidade contratual**
da Ubiquiti — podem mudar sem aviso entre versões.

**Decisão de arquitetura**: isso corresponde exatamente ao "adaptador legado" citado na
Seção 4 do projeto — implementado isolado, **desativado por padrão**, só habilitável
explicitamente pelo usuário/admin quando a API oficial não cobrir o dado necessário
para aquela versão de UniFi.

### 2.3 SNMP em dispositivos UniFi tem cobertura limitada

Quando habilitado no console, SNMP expõe principalmente contadores de interface e
informações genéricas de sistema (uptime, CPU/memória quando suportado pela MIB do
dispositivo). Detalhes finos de rádio Wi-Fi (canal, potência, clientes por rádio,
airtime) **não são tipicamente expostos via SNMP** nos dispositivos UniFi — SNMP é
tratado como fallback para monitoramento genérico de disponibilidade/interface, não
como substituto da API UniFi para dados de Wi-Fi.

### 2.4 O que ainda não sabemos (bloqueado até acesso à instalação real)

A lista completa está em `docs/unifi/verificacoes-pendentes-instalacao.md`. Resumo:
versão exata de UniFi OS / UniFi Network no Cloud Gateway Max do cliente, se a
Network API local já vem habilitada por padrão ou exige ativação manual, quais campos
de rádio por AP U7 Pro estão realmente presentes na resposta da API local nessa
versão, e se o modelo de licenciamento da Site Manager API atende ao caso de uso
totalmente self-hosted (sem depender da nuvem Ubiquiti).

## 3. Regra de produto derivada destas limitações

Toda tela que exibir um dado de rádio (RSSI, canal, potência, SNR, PHY rate) deve
anotar a fonte como "UniFi — API local" ou "UniFi — Site Manager" ou
"Indisponível nesta versão", nunca como medição do próprio dispositivo do usuário.
Isso é reforçado no design system (badge de proveniência) especificado em
`docs/architecture/02-arquitetura-proposta.md`.
