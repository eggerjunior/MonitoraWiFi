# 00 — Resumo Executivo

## Egger Network Intelligence

Plataforma de observabilidade de rede residencial/empresarial que unifica dados de
infraestrutura UniFi, um agente local de monitoramento ativo e um app iOS/iPadOS com
mapeamento espacial via LiDAR, para dar visibilidade honesta e acionável sobre
Internet, LAN, Wi-Fi, segurança e capacidade — em uma ou várias instalações.

## O que o sistema é

Um conjunto de 5 componentes que compartilham um único modelo de dados e uma única
fonte de verdade (o backend), cada um responsável por uma fatia do problema:

1. **Backend central (Go + PostgreSQL/TimescaleDB + Redis)** — ingestão, persistência
   série-temporal, regras de alerta/anomalia, correlação, API REST/OpenAPI, WebSocket/SSE
   para tempo real, RBAC, auditoria.
2. **Agente local (Go, dentro da LAN)** — única entidade capaz de fazer testes ativos
   dentro da rede do cliente (ping, DNS, traceroute, port scan autorizado, iPerf3,
   descoberta ARP/mDNS/SNMP, Wake-on-LAN, syslog). Conexão **outbound-only** para o
   backend; nunca abre porta de entrada na residência.
3. **Integração UniFi (`UniFiIntegrationProvider`)** — camada de adaptadores
   (UniFi Local Network API, UniFi Site Manager API, SNMP, Syslog) que traduz o estado
   real dos equipamentos (gateway, APs, switches, clientes, VLANs, eventos) para o
   modelo de dados interno, respeitando a capability matrix por versão de UniFi OS.
4. **App iOS/iPadOS nativo (Swift/SwiftUI)** — dashboard, ferramentas de diagnóstico
   client-side (o que o `Network.framework` permite honestamente) e o módulo
   **Spatial WiFi Survey**: reconstrução 3D via LiDAR/ARKit correlacionada a métricas
   de rede reais vindas do UniFi e do agente — nunca RSSI "lido" por hack.
5. **Aplicação web responsiva (Next.js/React/TypeScript)** — mesma superfície de dados
   do app iOS, com foco em topologia, relatórios, administração e uso em desktop.

Todos os cinco consomem os mesmos contratos (`packages/contracts`, OpenAPI 3.1) e o
mesmo modelo de domínio (`docs/architecture/05-modelo-dados.md`), sincronizados em
tempo real via WebSocket/SSE.

## Princípio não negociável

**Precisão sobre completude.** Quando um dado não pode ser medido com uma fonte
identificável (UniFi API, agente, SNMP, ARKit, estimativa matemática declarada como tal,
ou input do usuário), a interface mostra "Indisponível" com o motivo e a permissão ou
integração necessária — nunca um valor inventado. Isso vale especialmente para RSSI,
canal, potência e throughput, que dependem inteiramente do que a infraestrutura UniFi
e o sistema operacional expõem.

## Escopo da instalação inicial vs. escopo do produto

A instalação de referência (Cloud Gateway Max, 4× U7 Pro, Switch Lite 16 PoE, VLANs
Principal/IoT/Convidados) é o ambiente de desenvolvimento e validação, não um limite
de arquitetura. O modelo de dados é multi-organização → multi-site desde o ADR-002,
e a capability matrix (`docs/unifi/capability-matrix.md`) existe justamente para que
o sistema funcione (com funcionalidade honestamente reduzida) em sites com hardware,
versões de UniFi OS ou permissões diferentes.

## Estado desta entrega (Fase 0)

Esta entrega **não contém código de produto**. Ela contém os artefatos de descoberta e
planejamento exigidos antes do início da Fase 1: arquitetura, limitações técnicas
confirmadas, threat model inicial, capability matrix, estratégia de LiDAR, modelo de
dados inicial, estrutura do monorepo, ADR-001 a ADR-010, roadmap, critérios de aceite
da Fase 1 e a lista de verificações que só podem ser feitas na instalação UniFi real.
