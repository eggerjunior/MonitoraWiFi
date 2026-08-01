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

## 2026-08-01 — Git remoto criado, iOS CI validado, Bundle ID criado

- App/plataforma: repositório GitHub + apps/ios (Swift)
- Versão/build/commit: iOS `project.yml` MARKETING_VERSION=0.1.0
  CURRENT_PROJECT_VERSION=1 GIT_COMMIT=dev (nenhum archive de release feito
  ainda); commit `10f35d0` (HEAD no momento deste registro)
- Status:
  - Repositório privado criado: https://github.com/eggerjunior/MonitoraWiFi
    (`gh repo create` via script `ensure_private_github_repo.sh` da skill
    `ildemar_ios-native-testflight`). Commit inicial (148 arquivos) + 7 commits
    de correção enviados para `main`.
  - Secrets configurados no repositório (nunca impressos/logados):
    `IOS_ASC_KEY_ID`, `IOS_ASC_ISSUER_ID`, `IOS_ASC_KEY_P8_BASE64` (gerado a
    partir de `/config/.appstoreconnect/private_keys/AuthKey_37943WH8RQ.p8`).
  - **`iOS CI (build validation)` rodou de verdade em runner `macos-26`
    (Xcode 26.6) e está verde** — `xcodebuild build` para
    `generic/platform=iOS Simulator` compila sem erro. 5 problemas reais
    corrigidos ao longo de 7 iterações de CI (ver commits `6b29341` a
    `bc29a2c`): API `List(_:selection:rowContent:)` removida do SwiftUI,
    nome de simulador fixo (`iPhone 16`, não existe mais no lineup atual),
    `sed -E`/`\s` incompatível com BSD sed do macOS, ambiguidade de
    arch/UDID de simulador.
  - Execução de testes unitários (`xcodebuild test`/`build-for-testing`) via
    CI **não foi resolvida**: falha consistente com "Could not find test
    host for EggerNetworkIntelligenceTests" mesmo com dependência de target
    declarada, `-derivedDataPath` isolado, duas fases
    (`build-for-testing`+`test-without-building`) e destino por UDID único.
    Removido do `ios-ci.yml` (mantém só build) — mesmo padrão do template de
    referência da skill. Registrado como pendência real para investigação
    com Xcode de verdade.
  - Bundle ID `br.app.egger.network-intelligence` criado via API
    (`apps/ios/scripts/create_app.py`, `POST /v1/bundleIds`, id
    `44994AHNQU`). App record em App Store Connect **não existe** —
    confirmado bloqueio permanente da Apple (`POST /v1/apps` → 403
    `FORBIDDEN_ERROR` para qualquer chave), pendência manual do Ildemar.
- Comandos executados: `git init`/`add`/`commit`/`push`; `gh repo create`
  (via script); `gh secret set` (3x); `gh workflow run "iOS CI (build
  validation)"` (7x, iterando); `gh run watch`/`gh run view --log-failed`
  para diagnosticar cada falha; `python3 apps/ios/scripts/create_app.py`
  (JWT ES256 assinado com a chave real, nunca exibida).
- Pendências reais:
  1. **Ildemar**: criar app record em App Store Connect (Bundle ID já
     existe, aparecerá na lista) — só então `iOS TestFlight release` pode
     rodar.
  2. Investigar com Xcode real por que testes unitários não rodam sob
     `xcodebuild` headless.
  3. Ao rodar o primeiro archive de Release, observar o risco conhecido de
     esgotamento de cota de certificados (`Automatic signing`, ver
     `references/ildemar-ios-release.md` da skill) — aplicar a correção de
     Manual signing documentada lá se ocorrer, sem reinvestigar do zero.
  4. Versionamento formal (`ildemar_app-versioning`) ainda não aplicado a
     `apps/api`/`apps/web`.
