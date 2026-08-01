# Project Context

Generated: 2026-07-31T22:37:06-03:00

## Snapshot

- Project: `MonitoraWiFi`
- Root: `/host/opt/projetos/MonitoraWiFi`
- Branch: `main`
- Commit: `unknown`
- Git status: dirty
- Version: `unknown`
- Build: `unknown`
- Bundle/package id: `unknown`
- Detected stack: Go 1.25 (apps/api), Next.js 16/React 19/TypeScript (apps/web), Swift/SwiftUI via XcodeGen (apps/ios, não compilado neste ambiente)
- Fase atual: **Fase 1 — Fundação** concluída (backend, web e iOS shell); Fase 0 (docs de descoberta) permanece válida em `docs/`
- Repositório git local inicializado nesta sessão (branch `main`, sem commits ainda) — sem remote GitHub

## Product Purpose

Egger Network Intelligence: plataforma de observabilidade completa (Internet, LAN,
Wi-Fi, UniFi, clientes, segurança, capacidade) para uma ou várias instalações
residenciais/empresariais baseadas em UniFi. Público-alvo inicial: o próprio
Ildemar, na residência com UniFi Cloud Gateway Max + 4x U7 Pro + Switch Lite 16 PoE.
Arquitetura multi-tenant desde o schema (Organization → Site, ADR-002).

## Current User-Facing Features

- **Login real** (backend Go com bcrypt + sessão opaca em cookie httpOnly) em
  Web e iOS, com RBAC (5 papéis) aplicado no backend.
- **Web** (`apps/web`): shell com sidebar recolhível, tema claro/escuro sem
  flash, Visão geral listando organizações/sites reais; demais módulos
  (Internet, Wi-Fi, Dispositivos, Clientes, Mapa, Diagnósticos, Alertas,
  Relatórios, Configurações) são placeholders honestos.
- **iOS** (`apps/ios`, código escrito mas **não compilado** neste ambiente Linux —
  ver `apps/ios/README.md`): login, sessão persistida no Keychain, navegação
  adaptativa TabView/NavigationSplitView, detecção de suporte a LiDAR, tela de
  versão/changelog.
- Nenhum dado de UniFi/rede real ainda — isso é Fase 3/4 em diante.

## Architecture

Ver `docs/architecture/02-arquitetura-proposta.md`. Resumo: backend Go
(`apps/api`) com PostgreSQL, sessão por cookie opaco (hash em banco), RBAC por
papel, OpenTelemetry (stdout exporters nesta fase); worker e agente local
ainda não implementados (Fase 2/3); web Next.js 16 App Router com padrão
"backend for frontend" (rotas `/api/auth/*` fazem proxy ao backend Go e
reemitem o cookie sob o domínio do próprio Next.js); iOS com `APIClient`
(URLSession/actor) + `SessionStore` (Observable) + Keychain.

## Important Files

- `README.md` — visão geral e ponto de entrada
- `docs/architecture/` (00 a 07) + `docs/architecture/adr/ADR-001..010`
- `docs/unifi/capability-matrix.md`, `docs/unifi/verificacoes-pendentes-instalacao.md`
- `packages/contracts/openapi.yaml` — contrato único da API (Fase 1: auth, organizations, sites)
- `packages/design-tokens/tokens.json` (+ gerados: `.ts`, `.css`, `.swift`)
- `apps/api/` (Go), `apps/web/` (Next.js), `apps/ios/` (Swift/XcodeGen)
- `infrastructure/database/migrations/0001_init.{up,down}.sql`
- `infrastructure/docker/docker-compose.dev.yml`

## Data Model And Storage

Fase 1 implementa um subconjunto do modelo completo
(`docs/architecture/05-modelo-dados.md`): `organizations`, `sites`, `users`,
`sessions`, `audit_log` (migração `0001_init`). PostgreSQL puro nesta fase
(TimescaleDB/particionamento entram quando as tabelas de série temporal forem
criadas, Fase 2+). UUIDs (`pgcrypto`), e-mail via `citext`, timestamps `timestamptz`.

## Integrations And External Services

Nenhuma integração externa real ainda (UniFi, SNMP, APNs) — tudo planejado em
`docs/unifi/` e `docs/architecture/adr/`. O `.env.example` na raiz documenta as
variáveis de ambiente necessárias para rodar `apps/api` e `apps/web` localmente.

## Versioning And Release Rules

Detected version/build fields:

```json
{"api": "sem versionamento de artefato ainda (Fase 1, não distribuído)", "web": "package.json version 0.1.0 (padrão create-next-app, versionamento formal ainda não aplicado)", "ios": "project.yml MARKETING_VERSION=0.1.0 CURRENT_PROJECT_VERSION=1 GIT_COMMIT=dev (skill ildemar_app-versioning aplicada)"}
```

iOS já segue o esquema completo da skill `ildemar_app-versioning`
(`VersionManager`/`VersionHistory`/changelog em `apps/ios/Sources/Version/`).
Web e API ainda não têm esse esquema formalizado — pendência a resolver antes
da primeira publicação real (Fase 8), não bloqueante para desenvolvimento local.

## Local Development

```bash
# Backend
cd apps/api && cp ../../.env.example ../../.env  # editar se preciso
docker compose -f infrastructure/docker/docker-compose.dev.yml up -d postgres redis
docker compose -f infrastructure/docker/docker-compose.dev.yml --profile tools run --rm migrate
go run ./cmd/api

# Web
cd apps/web && npm install && npm run dev

# iOS (requer macOS + Xcode + XcodeGen — não disponível neste ambiente)
cd apps/ios && xcodegen generate && open EggerNetworkIntelligence.xcodeproj
```

## Testing And Validation

- `apps/api`: `go build ./...`, `go vet ./...`, `go test ./...` — todos verdes
  nesta sessão. Migração `0001_init` validada up/down contra PostgreSQL 16 real
  (via `docker exec`+`psql`, não apenas revisão de SQL).
- `apps/web`: `npx tsc --noEmit`, `npm run lint`, `npm run build` — todos
  verdes. Fluxo completo (login real → cookie → dashboard com dados reais)
  validado ponta a ponta contra o backend Go + Postgres reais (containers
  compartilhando network namespace, ver histórico da sessão para o método).
- `apps/ios`: **não compilado nesta sessão** (sem Xcode/Swift no ambiente
  Linux) — próximo passo real é rodar `ios-ci.yml` ou abrir no Xcode.

## Recent Decisions

Ver `docs/architecture/adr/` (ADR-001 a ADR-010, Fase 0). Decisões novas da
Fase 1: sessão via cookie opaco com hash SHA-256 em banco (não JWT) para
permitir revogação server-side; rate limiting de login em memória (simples,
1 instância — revisitar se escalar horizontalmente); design tokens gerados a
partir de `tokens.json` para Web/iOS/CSS a partir de uma única fonte; CI usa
runner `macos-26` da GitHub para o iOS (skill `ildemar_ios-native-testflight`),
já que este ambiente de desenvolvimento não tem Xcode.

## Known Risks And Pending Work

- **iOS não compilado nesta sessão** — validar no Xcode/CI antes de confiar
  no shell como "pronto"; ver aviso em `apps/ios/README.md`.
- **Sem git remote / sem publicação** — `git init` local existe, mas não há
  repositório GitHub nem credenciais Apple configuradas; bloqueio documentado
  em `apps/ios/README.md` (seção "Bloqueado nesta entrega").
- Identidade do agente (Fase 2) ainda usará credencial rotacionável simples,
  não mTLS (débito técnico rastreado, ADR-006).
- 18 perguntas sobre a instalação UniFi real seguem pendentes
  (`docs/unifi/verificacoes-pendentes-instalacao.md`) — bloqueiam a Fase 3.
- Versionamento formal (skill `ildemar_app-versioning`) ainda não aplicado a
  `apps/api`/`apps/web`, só a `apps/ios`.

## Import Notes For Other Tools

Read `IMPORT_MANIFEST.json` first, then this file, then key files listed above. Never load secret files.
