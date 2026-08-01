# Egger Network Intelligence

Plataforma de observabilidade completa para Internet, LAN, Wi-Fi, equipamentos UniFi
e clientes conectados — com app nativo iOS/iPadOS (incluindo mapeamento espacial via
LiDAR), aplicação web, backend central e um agente local de monitoramento instalado
dentro da LAN.

Nome provisório do projeto. Instalação de referência: residência com UniFi Cloud
Gateway Max, 4× access points U7 Pro e UniFi Switch Lite 16 PoE — mas a arquitetura é
desenhada desde o início para múltiplas organizações, sites, gateways e modelos de
equipamento (ver ADR-002).

## Estado atual: Fase 1 (Fundação) concluída

A Fase 0 (documentos de descoberta) está completa em `docs/`. A Fase 1 entregou:
backend (`apps/api`, Go) com autenticação, RBAC e organizações/sites reais contra
PostgreSQL; web (`apps/web`, Next.js 16) com login e shell de navegação consumindo
o backend real; e o shell do app iOS (`apps/ios`, Swift/SwiftUI) — este último
**escrito mas não compilado nesta sessão**, por falta de Xcode/macOS no ambiente de
desenvolvimento (ver aviso em [`apps/ios/README.md`](apps/ios/README.md)). Backend
e web foram validados ponta a ponta com testes reais (não apenas leitura de código).

Ver [`docs/architecture/07-criterios-aceite-fase1.md`](docs/architecture/07-criterios-aceite-fase1.md)
para o detalhamento exato do que foi cumprido, adiado ou bloqueado.

Comece a leitura pela documentação de arquitetura (Fase 0), ainda válida:

1. [`docs/architecture/00-resumo-executivo.md`](docs/architecture/00-resumo-executivo.md)
2. [`docs/architecture/01-limitacoes-tecnicas.md`](docs/architecture/01-limitacoes-tecnicas.md)
3. [`docs/architecture/02-arquitetura-proposta.md`](docs/architecture/02-arquitetura-proposta.md)
4. [`docs/architecture/03-fluxo-de-dados.md`](docs/architecture/03-fluxo-de-dados.md)
5. [`docs/security/threat-model.md`](docs/security/threat-model.md)
6. [`docs/unifi/capability-matrix.md`](docs/unifi/capability-matrix.md)
7. [`docs/architecture/04-estrategia-lidar.md`](docs/architecture/04-estrategia-lidar.md)
8. [`docs/architecture/05-modelo-dados.md`](docs/architecture/05-modelo-dados.md)
9. [`docs/architecture/adr/`](docs/architecture/adr/) — ADR-001 a ADR-010
10. [`docs/architecture/06-roadmap.md`](docs/architecture/06-roadmap.md)
11. [`docs/architecture/07-criterios-aceite-fase1.md`](docs/architecture/07-criterios-aceite-fase1.md)
12. [`docs/unifi/verificacoes-pendentes-instalacao.md`](docs/unifi/verificacoes-pendentes-instalacao.md)

## Princípio central do produto

**Nunca simular resultado falso.** Toda métrica exibida indica sua origem (medição
direta do dispositivo, agente local, API UniFi local/cloud, SNMP, ARKit, estimativa
matemática ou informação declarada pelo usuário). Quando um dado não estiver
disponível, a interface mostra "Indisponível" com o motivo — nunca inventa RSSI,
canal, potência, velocidade, interferência ou latência.

## Estrutura do monorepo

```text
/apps
  /ios            App nativo iOS/iPadOS (Swift/SwiftUI/ARKit)
  /web            Aplicação web (Next.js/React/TypeScript)
  /api            Backend central (Go)
  /worker         Processamento assíncrono (Go)
  /local-agent    Agente local de monitoramento (Go, roda dentro da LAN)

/packages
  /contracts        OpenAPI 3.1 — fonte única de verdade dos contratos
  /design-tokens    Tokens de design compartilhados (iOS + Web)
  /network-models   Modelos de domínio de rede compartilhados
  /validation       Validação de host/porta/URL (anti-SSRF, anti-command-injection)
  /shared-docs      Textos/strings/glossário compartilhados

/infrastructure
  /docker /nginx /terraform /monitoring /database /scripts

/docs
  /architecture (+ /adr)  /api  /unifi  /security  /deployment  /testing  /user-guide
  /development-handoff    Documentação contínua de handoff (skill Ildemar)
```

## Por que existe um agente local

O backend central nunca acessa a LAN do cliente diretamente — não há porta de entrada
aberta na residência. O agente local roda dentro da LAN, fala com a Network API do
UniFi e com a rede local, e envia dados para o backend por conexão **outbound**
apenas. Ver [ADR-001](docs/architecture/adr/ADR-001-agente-local-outbound-only.md).

## Convenções deste projeto

- Toda a documentação e comunicação do projeto é em **português**.
- Versionamento, build e publicação seguem a skill `ildemar_app-versioning`.
- Publicação iOS/TestFlight segue a skill `ildemar_ios-native-testflight`.
- Continuidade de documentação/handoff segue a skill `ildemar_project-handoff-docs`
  (ver [`docs/development-handoff/`](docs/development-handoff/)).
- Nenhum dado de rede/UniFi é simulado — módulos ainda não implementados mostram
  estados vazios honestos, nunca dado inventado.

## Próximo passo

1. Validar `apps/ios` num Mac real (ou disparar o workflow `iOS CI`) — ver
   bloqueio documentado em [`apps/ios/README.md`](apps/ios/README.md).
2. Decidir sobre criar o repositório remoto no GitHub (há um repo git local
   nesta máquina, sem remote ainda) antes de qualquer publicação.
3. Iniciar a Fase 2 (Agente local) conforme
   [`docs/architecture/06-roadmap.md`](docs/architecture/06-roadmap.md).

Itens de [`docs/unifi/verificacoes-pendentes-instalacao.md`](docs/unifi/verificacoes-pendentes-instalacao.md)
devem ser respondidos com acesso à instalação real antes/durante a Fase 3.
