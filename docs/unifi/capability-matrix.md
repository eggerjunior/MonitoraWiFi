# Capability Matrix

Esta matriz é o mecanismo pelo qual o sistema **nunca assume** uma capacidade que a
plataforma (iOS, Web, Backend, Agente, UniFi, SNMP) não confirma ter. Cada linha tem
um status e, quando aplicável, a fonte que confirma esse status. O backend expõe essa
matriz via `GET /api/v1/unifi/capabilities` por site, calculada em runtime a partir da
versão detectada do UniFi OS/Network — a UI **consulta esta matriz antes de renderizar
qualquer card**, em vez de assumir que um endpoint existe.

Legenda de status: `confirmado` (documentação oficial ou teste direto),
`provável` (relatado pela comunidade, não documentado oficialmente — tratar com
adaptador legado opt-in), `indisponível` (confirmado que não existe), `a validar`
(depende da instalação real do cliente, ver `verificacoes-pendentes-instalacao.md`).

## 1. iOS/iPadOS (Network.framework, ARKit, NetworkExtension)

| Capacidade | Status | Fonte |
|---|---|---|
| Path status (Wi-Fi/celular/wired, expensive, constrained) | confirmado | `NWPathMonitor` (Network.framework), API pública |
| SSID/BSSID da rede atual | confirmado, condicional | `NEHotspotNetwork.fetchCurrent` + autorização de localização (ver limitações §1.2) |
| RSSI/sinal da rede atual (via iPhone) | indisponível | `signalStrength` de `NEHotspotNetwork` não confiável (Apple Developer Forums thread 733598/128844) |
| Lista de redes Wi-Fi próximas | indisponível | Sem CoreWLAN em iOS |
| Canal/largura/potência de qualquer rede | indisponível | Nenhuma API pública expõe isso no iOS |
| Ping ICMP simples | confirmado | Socket ICMP não privilegiado, sem entitlement |
| Traceroute (ICMP/UDP TTL) | provável, a prototipar | Tecnicamente viável, não documentado oficialmente como suportado; validar antes de expor na UI |
| Descoberta ARP/NDP | indisponível | Sem acesso a socket raw L2 em iOS |
| Descoberta mDNS/Bonjour | confirmado, com permissão | Exige `NSLocalNetworkUsageDescription` + `NSBonjourServices` |
| Port scan (TCP connect) para hosts específicos | confirmado, com permissão | `Network.framework`, dispara prompt de Rede Local |
| Wake-on-LAN (broadcast UDP) | não recomendado | Exige entitlement Multicast + relatos de `EACCES` em iOS/iPadOS recentes (forums 805690/805719/770023). Delegado ao agente. |
| Reconstrução de malha 3D (LiDAR) | confirmado, condicional a hardware | Só em iPhone Pro/Pro Max ≥12 Pro e iPad Pro ≥2020; ver limitações §1.6 |
| AR sem LiDAR (tracking por feature points) | confirmado | `ARWorldTrackingConfiguration` em qualquer device com ARKit |
| Execução de levantamento em background | indisponível | ARKit exige app em primeiro plano |
| Métricas do próprio app (crash/hang/energia) | confirmado | MetricKit |
| Notificação push com app fechado | confirmado, requer infra própria | APNs via provider do backend (não CloudKit) |
| Biometria para desbloqueio de ações sensíveis | confirmado | LocalAuthentication |

## 2. Web (navegador, PWA)

| Capacidade | Status | Fonte |
|---|---|---|
| SSID/BSSID/RSSI do navegador | indisponível | Nenhum navegador expõe isso por API web padrão |
| Latência ao backend (RTT aproximado) | confirmado | Medição client-side via `fetch`/WebSocket timing, rotulada como estimativa do navegador, não da rede local |
| Teste de velocidade a partir do navegador | confirmado, complementar | Serve como ponto de vista adicional "da perspectiva deste dispositivo", nunca substitui o speed test do agente |
| Notificações push (Web Push) | confirmado | Web Push API, canal independente do APNs |
| Visualização 3D/heatmap (WebGL) | confirmado | Renderização do que já foi processado pelo worker; web não faz LiDAR |
| Geolocalização do navegador | confirmado, com permissão | Não substitui Core Location/LiDAR do app iOS; uso limitado (ex.: fuso horário do site) |

## 3. Backend (Go)

| Capacidade | Status | Fonte |
|---|---|---|
| API REST versionada + OpenAPI 3.1 | confirmado | Escolha de stack própria |
| WebSocket/SSE em tempo real | confirmado | Escolha de stack própria |
| Persistência série-temporal | confirmado | PostgreSQL + TimescaleDB (ou particionamento nativo se Timescale indisponível no ambiente de deploy) |
| Execução de testes ativos contra a LAN do cliente | indisponível diretamente | Backend não está na LAN do cliente; sempre delega ao agente (ver ADR-001) |
| Chamada direta à Network API local do UniFi | indisponível diretamente | Mesma razão acima — feito via agente |
| Chamada à UniFi Site Manager API (cloud) | confirmado, opcional | `https://api.ui.com/v1/`, X-API-KEY, rate limit 10.000 req/min documentado |

## 4. Agente local (Go, dentro da LAN)

| Capacidade | Status | Fonte |
|---|---|---|
| ICMP/TCP/UDP/DNS/HTTP ping | confirmado | Sockets nativos Go, sem restrição de sandbox como no iOS |
| Traceroute ICMP/UDP/TCP | confirmado | Mesmo motivo |
| Port scan TCP connect | confirmado | Escopo restrito a alvos cadastrados (ver threat model) |
| Descoberta ARP/NDP | confirmado | Acesso a socket raw disponível em Linux/macOS para processo com privilégio adequado |
| Descoberta mDNS/SSDP/LLDP/SNMP | confirmado | Bibliotecas Go maduras existem para cada protocolo; LLDP depende do switch anunciar |
| Wake-on-LAN | confirmado | Broadcast UDP sem sandbox de app store |
| Syslog receiver | confirmado | Requer o dispositivo de origem ser configurado para enviar ao agente |
| iPerf3 (LAN) | confirmado | Requer binário/servidor iperf3 em outro nó da LAN |
| Conexão outbound-only ao backend | confirmado | Requisito de arquitetura, não de plataforma |

## 5. UniFi Network API (local, por console)

| Capacidade | Status | Fonte |
|---|---|---|
| Inventário de dispositivos (gateway/AP/switch) | confirmado, implementado (2026-08-01) | `NetworkAPIAdapter.ListDevices` (`apps/local-agent/internal/unifi`) — testado contra a instalação real do usuário e validado ponta a ponta |
| Lista de clientes conectados | confirmado, implementado (2026-08-01) | `NetworkAPIAdapter.ListClients`, com paginação real (confirmado: 80 clientes reais, resposta pagina em blocos de 25) |
| Canal/largura de canal/padrão Wi-Fi por rádio de AP | confirmado (2026-08-02) | `GET .../devices/{id}` real — `interfaces.radios[].{wlanStandard,frequencyGHz,channelWidthMHz,channel}`, testado contra um U7 Pro real (802.11be nos 3 rádios) |
| Potência de transmissão, utilização do canal, clientes/airtime/retries/PHY rate por rádio | indisponível | Confirmado ausente na resposta real de `GET .../devices/{id}` desta versão — não é "a validar", é confirmado que a API não expõe |
| Estado/velocidade negociada/PoE por porta de switch | confirmado (2026-08-02) | `GET .../devices/{id}` real — `interfaces.ports[].{idx,state,connector,maxSpeedMbps,speedMbps,poe}`, testado contra uma USW Lite 16 PoE real |
| Contadores RX/TX/erros/CRC/flaps/consumo PoE em watts por porta | indisponível | Confirmado ausente na resposta real desta versão |
| Eventos/alarmes em tempo real via polling da Network API local | indisponível | `GET .../alarms` e `GET .../events` retornam 404 explícito nesta versão — não existe esse endpoint na integration API local |
| Topologia dispositivo→dispositivo (uplink) | confirmado (2026-08-02) | `GET .../devices/{id}` real — campo `uplink.deviceId`, testado num AP (aponta pro switch) e num switch (aponta pro gateway) |
| Topologia cliente→dispositivo (uplink) | confirmado (2026-08-01) | `GET .../clients` real — campo `uplinkDeviceId` por cliente |
| DPI/categorização de aplicação por cliente | indisponível | Confirmado ausente em `GET .../clients` real (amostra de 5 clientes, nenhum campo de categoria/app) |

## 6. UniFi Site Manager API (cloud, oficial v1.0)

| Capacidade | Status | Fonte |
|---|---|---|
| Lista de sites da conta Ubiquiti | confirmado | `developer.ui.com/site-manager-api/list-sites` |
| Internet Health Metrics agregadas | confirmado | Documentação oficial do Site Manager |
| Detalhe fino de rádio/porta por dispositivo | indisponível/não é o propósito | API pensada para visão agregada multi-site, não telemetria fina — usar Network API local para isso |
| Uso sem depender da nuvem Ubiquiti | não aplicável | Por definição é uma API cloud; produto trata como opcional, não obrigatório |

## 7. SNMP

| Capacidade | Status | Fonte |
|---|---|---|
| Uptime/CPU/memória genéricos (quando MIB suportada) | provável | Depende do dispositivo expor essas OIDs — validar por modelo |
| Contadores de interface (RX/TX/erros) | provável | Suporte comum em MIB-II padrão |
| Detalhe de rádio Wi-Fi (canal, clientes por rádio) | indisponível, tipicamente | Dispositivos UniFi não expõem tipicamente esse nível via SNMP — tratado como fallback, não fonte primária |

## 8. Adaptador legado (API não oficial, opt-in, desativado por padrão)

| Capacidade | Status | Fonte |
|---|---|---|
| Telemetria profunda de rádio/porta/DPI | provável | Usado por ferramentas da comunidade (ex.: unpoller) contra endpoints não documentados `/api/s/{site}/...` |
| Estabilidade entre versões | indisponível como garantia | Sem contrato de estabilidade da Ubiquiti — cada upgrade de UniFi Network pode quebrar o adaptador |
| Uso em produção | desativado por padrão | Habilitação exige ação explícita do administrador, com aviso de risco na UI |

## Como esta matriz é usada em runtime

1. No cadastro de uma integração UniFi (Seção 4 do documento-fonte), o agente detecta
   versão do UniFi OS e do UniFi Network.
2. O agente testa (uma vez, não repetidamente) quais endpoints da Network API local
   respondem com sucesso e persiste isso como `capabilities` daquele `UniFiConsole`.
3. O backend expõe essas capabilities por site; a UI (iOS/Web) consulta antes de
   decidir se mostra um card com dado real ou um estado "recurso não suportado nesta
   versão do UniFi" — nunca inventa o dado nem esconde silenciosamente a limitação.
4. Feature flags por versão vivem no backend, não hardcoded no cliente, para permitir
   ajuste sem re-submissão à App Store quando uma nova versão de UniFi mudar algo.
