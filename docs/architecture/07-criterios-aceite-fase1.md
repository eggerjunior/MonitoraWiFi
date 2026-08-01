# 07 — Critérios de Aceite da Fase 1 (Fundação)

A Fase 1 é considerada concluída quando **todos** os itens abaixo forem verdadeiros.
Nenhuma tela final de produto (dashboards completos, LiDAR, diagnósticos) faz parte
deste escopo — é fundação técnica.

> **Status em 2026-07-31/08-01**: itens marcados `[x]` foram implementados e
> **validados de verdade** (build real, teste real, requisição HTTP real
> contra Postgres real, ou execução real em CI — não apenas leitura de
> código). Itens `[~]` foram implementados mas ficaram parcialmente validados
> ou com uma limitação real documentada. Itens `[ ]` continuam pendentes.
> Detalhe em `docs/development-handoff/RELEASE_LOG.md`.
>
> Atualização de 2026-08-01: repositório GitHub privado criado
> (`eggerjunior/MonitoraWiFi`, ver `docs/development-handoff/RELEASE_LOG.md`),
> código commitado e enviado, CI do iOS rodou de verdade em runner `macos-26`
> (ver seção "iOS" abaixo).

## Monorepo e CI
- [x] Estrutura de diretórios de `03. ARQUITETURA GERAL` criada e populada com pelo
      menos um artefato real (não só README) em cada app/package aplicável à Fase 1.
- [x] CI (GitHub Actions) criado para `apps/api`, `apps/web` e `apps/ios`
      (`.github/workflows/api-ci.yml`, `web-ci.yml`, `ios-ci.yml`) — **iOS CI
      rodou de verdade e está verde** (após 5 correções reais, ver seção
      "iOS"). `api-ci.yml`/`web-ci.yml` ainda não dispararam nesta sessão
      (nenhuma mudança nova em `apps/api`/`apps/web` desde o push inicial).
      `apps/local-agent` não tem CI ainda porque não tem código (é escopo da
      Fase 2, não da Fase 1 — ajuste registrado aqui).
- [x] Pipelines não usam `continue-on-error` para os passos obrigatórios; os
      passos informativos (`govulncheck`, `npm audit`) são explicitamente
      rotulados como não bloqueantes nesta fase, não escondidos.

## Backend
- [x] Serviço Go sobe localmente via Docker Compose de desenvolvimento
      (`infrastructure/docker/docker-compose.dev.yml`) — Dockerfile multi-stage
      construído e testado com sucesso.
- [x] Health check e readiness respondem corretamente (testado via HTTP real:
      `/healthz` → 200, `/readyz` → 200 com Postgres up, 503 quando indisponível).
- [x] Autenticação por e-mail/senha funcional (testada via HTTP real, incluindo
      rate limiting). Passkey/MFA: colunas de schema previstas
      (`mfa_enrolled_at`), implementação de fato adiada — registrado como
      pendência, não escondido.
- [x] RBAC com os 5 papéis aplicado em `/organizations` e `/sites`, testado por
      papel (testes automatizados + fail-closed para papel desconhecido).
- [x] Migração `0001_init` validada up/down contra PostgreSQL 16 real (via
      `docker exec`+`psql`), sem erro, revertível.
- [x] OpenAPI 3.1 em `packages/contracts/openapi.yaml` cobre exatamente os
      endpoints implementados nesta fase (auth, organizations, sites, health).
      Contract test automatizado (comparação schema↔handler) ainda não
      existe — validação foi manual nesta fase; registrado como débito.
- [x] Logs estruturados em JSON (`log/slog`); trace e métrica exportados via
      OpenTelemetry (exporters stdout) — confirmado rodando sem erro após
      corrigir conflito de Schema URL entre `resource.Default()` e o semconv
      pinado (ver `internal/telemetry/telemetry.go`).

## Web
- [x] Aplicação Next.js sobe, login funcional contra o backend real (validado
      ponta a ponta: BFF → API Go → Postgres, com cookie de sessão).
- [x] Shell de navegação (sidebar recolhível) presente; módulos internos são
      placeholders "em construção" honestos.
- [~] Tema claro/escuro funcional (sem flash, técnica documentada oficial do
      Next.js 16 aplicada) — contraste WCAG não foi verificado com ferramenta
      de auditoria automatizada nesta sessão (sem navegador com display
      disponível); paleta em `packages/design-tokens/tokens.json` foi
      desenhada visando contraste adequado, mas isso é uma afirmação de design,
      não uma medição confirmada.

## iOS
- [x] **Build validado em CI real** (`iOS CI (build validation)`, runner
      `macos-26`, Xcode 26.6): `xcodebuild build` para
      `generic/platform=iOS Simulator` está verde. Escrito sem Xcode local,
      mas 5 problemas reais foram encontrados e corrigidos rodando de
      verdade (API de `List` removida do SwiftUI, nome de simulador fixo,
      `sed` BSD sem `\s`, ambiguidade de arch) — ver histórico de commits e
      `apps/ios/README.md`.
- [x] `TabView` (iPhone) e `NavigationSplitView` (iPad) implementados em
      `RootView.swift`, alternando por `horizontalSizeClass` — compila; não
      validado visualmente em simulador rodando (sem GUI neste ambiente).
- [x] Detecção de suporte a LiDAR implementada (`LiDARCapabilityChecker`,
      checagem pura de hardware, sem sessão de câmera) — compila.
- [~] Keychain implementado (`KeychainStore`) com testes unitários
      (`Tests/KeychainStoreTests.swift`, Swift Testing) — código compila,
      mas **a execução dos testes no CI não foi resolvida**: `xcodebuild
      test`/`build-for-testing` falha com "Could not find test host" mesmo
      após 5 tentativas de correção distintas (ver `apps/ios/README.md`,
      seção "Pendência real"). O próprio template de referência da skill
      `ildemar_ios-native-testflight` também não roda testes no CI, só
      build — investigar com Xcode real antes de insistir às cegas.

## Design system
- [x] `packages/design-tokens/tokens.json` é a fonte única; `tokens.ts`,
      `tokens.css` e `DesignTokens.swift` são gerados por
      `scripts/generate.mjs` e efetivamente usados por `apps/web`
      (confirmado no build) e referenciados por `apps/ios/project.yml`
      (não confirmado em build real, ver item iOS acima).

## Segurança
- [x] `.env.example` presente na raiz, sem segredo real; `.gitignore` cobre
      `.env`, `*.p8`, `**/asc.env`. Scanner de segredos (`gitleaks`) configurado
      em `.github/workflows/security-scan.yml`. Segredos reais (`IOS_ASC_KEY_ID`,
      `IOS_ASC_ISSUER_ID`, `IOS_ASC_KEY_P8_BASE64`) configurados como GitHub
      Secrets via `gh secret set` — nunca impressos/logados nesta sessão.
- [x] SAST/dependency scanning informativos configurados: `govulncheck` (Go) e
      `npm audit` (web) no CI, explicitamente não bloqueantes nesta fase.

## Documentação
- [x] `docs/development-handoff/` atualizado refletindo o estado real da
      Fase 1 (`PROJECT_CONTEXT.md`, `RELEASE_LOG.md`, `SCREENSHOTS.md` com a
      ausência de screenshots justificada).
- [~] Changelog da Fase 1: aplicado integralmente em `apps/ios` (`VersionHistory.swift`,
      skill `ildemar_app-versioning`); **ainda não aplicado** a `apps/api`/`apps/web`
      — pendência explícita antes de qualquer publicação real.

## Não-objetivos explícitos da Fase 1 (para não gerar expectativa errada)
- Nenhum dado real de UniFi ainda (isso é Fase 3) — Fase 1 usa apenas dados
  reais de organização/site cadastrados manualmente no banco para teste, nunca
  fixtures apresentadas como se fossem produção.
- Nenhuma funcionalidade de LiDAR, diagnóstico ativo ou alerta ainda.
- TestFlight/App Store: Bundle ID criado, mas o **app record em App Store
  Connect ainda depende de ação manual do Ildemar** (restrição permanente da
  Apple, não escolha de escopo) — ver `apps/ios/README.md`, seção "Status de
  publicação".
