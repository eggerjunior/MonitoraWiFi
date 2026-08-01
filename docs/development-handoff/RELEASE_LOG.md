# Release Log

Generated: 2026-07-31T21:50:53-03:00

Record every deploy, TestFlight/App Store upload, web publish and external processing status here.

## 2026-07-31 — Fase 0 (Descoberta) concluída

- App/plataforma: N/A (nenhum código de produto ainda; entrega documental)
- Versão/build/commit: N/A — repositório ainda não é um git repo
- Status: documentos de descoberta completos e revisados (resumo executivo,
  limitações técnicas, arquitetura, fluxo de dados, threat model, capability
  matrix, estratégia LiDAR, modelo de dados, ADR-001 a ADR-010, roadmap, critérios
  de aceite da Fase 1, verificações pendentes na instalação UniFi real)
- Comandos executados: criação do esqueleto do monorepo (`mkdir -p apps/... packages/...
  infrastructure/... docs/...`), pesquisa web para confirmar comportamento real de
  `NEHotspotNetwork`, ARKit/LiDAR e APIs oficiais UniFi (`developer.ui.com`) antes de
  escrever a capability matrix
- Artefatos gerados: ver `docs/architecture/`, `docs/security/threat-model.md`,
  `docs/unifi/`
- Pendências: iniciar Fase 1 (Fundação); responder as 18 perguntas de
  `docs/unifi/verificacoes-pendentes-instalacao.md` com acesso à instalação real

## 2026-07-31 — Fase 1 (Fundação) — backend e web validados; iOS não compilado

- App/plataforma: apps/api (Go), apps/web (Next.js 16), apps/ios (Swift, não compilado)
- Versão/build/commit: sem tag de release ainda; iOS em `project.yml` MARKETING_VERSION=0.1.0 CURRENT_PROJECT_VERSION=1 GIT_COMMIT=dev (build local, sem commit real ainda)
- Status:
  - apps/api: `go build`/`go vet`/`go test` verdes; migração `0001_init`
    validada up/down contra PostgreSQL 16 real; login/RBAC/health/readiness
    validados via HTTP real contra o backend rodando em container.
  - apps/web: `tsc --noEmit`/`lint`/`build` verdes; fluxo completo login → cookie
    → dashboard com dados reais validado ponta a ponta contra apps/api + Postgres reais.
  - apps/ios: código escrito (login, Keychain, navegação adaptativa, detecção
    de LiDAR, versionamento) mas **não compilado nesta sessão** — sem Xcode/Swift
    disponíveis neste ambiente Linux. Bloqueador real, não apenas formalidade.
  - CI: `.github/workflows/api-ci.yml`, `web-ci.yml`, `ios-ci.yml`,
    `ios-testflight.yml` (manual), `security-scan.yml` criados; nenhum rodou
    ainda de verdade (sem GitHub remote nesta sessão).
- Comandos executados: `go mod tidy`, `go build/vet/test`, `docker build`
  (Dockerfile multi-stage do backend), containers efêmeros de Postgres/API/Web
  compartilhando network namespace para validação ponta a ponta, `npx
  create-next-app`, `npm run build/lint`, `node scripts/generate.mjs`
  (design tokens).
- Artefatos gerados: ver `apps/api/`, `apps/web/`, `apps/ios/`,
  `packages/design-tokens/`, `infrastructure/database/migrations/0001_init.*`,
  `infrastructure/docker/docker-compose.dev.yml`, `.github/workflows/`.
- Pendências: compilar/testar `apps/ios` num Mac real ou via `ios-ci.yml`;
  `git init` já feito localmente nesta sessão (branch `main`, sem commits) —
  falta decisão do usuário sobre criar o remote GitHub e fazer o primeiro
  commit; aplicar versionamento formal a `apps/api`/`apps/web`; responder as
  18 perguntas de `docs/unifi/verificacoes-pendentes-instalacao.md` antes da
  Fase 3.
