# 07 — Critérios de Aceite da Fase 1 (Fundação)

A Fase 1 é considerada concluída quando **todos** os itens abaixo forem verdadeiros.
Nenhuma tela final de produto (dashboards completos, LiDAR, diagnósticos) faz parte
deste escopo — é fundação técnica.

> **Status em 2026-07-31**: itens marcados `[x]` foram implementados e
> **validados de verdade** nesta sessão (build real, teste real, requisição
> HTTP real contra Postgres real — não apenas leitura de código). Itens `[~]`
> foram implementados mas não puderam ser validados por limitação do ambiente
> de desenvolvimento (sem Xcode/macOS, sem GitHub remote). Itens `[ ]`
> continuam pendentes. Detalhe em `docs/development-handoff/RELEASE_LOG.md`.

## Monorepo e CI
- [x] Estrutura de diretórios de `03. ARQUITETURA GERAL` criada e populada com pelo
      menos um artefato real (não só README) em cada app/package aplicável à Fase 1.
- [~] CI (GitHub Actions) criado para `apps/api`, `apps/web` e `apps/ios`
      (`.github/workflows/api-ci.yml`, `web-ci.yml`, `ios-ci.yml`) — **nenhum
      rodou de verdade ainda**, pois o repositório não tem remote no GitHub
      nesta sessão. `apps/local-agent` não tem CI ainda porque não tem código
      (é escopo da Fase 2, não da Fase 1 — ajuste registrado aqui).
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
- [~] Código escrito para login, Keychain, navegação adaptativa e detecção de
      LiDAR — **mas não compilado nem executado nesta sessão**: este ambiente
      Linux não tem Xcode, `swift` nem `xcodegen` instalados. Ver aviso
      detalhado em `apps/ios/README.md`. Próximo passo real: abrir no Xcode ou
      rodar o workflow `iOS CI` (`macos-26`) e corrigir o que aparecer.
- [x] `TabView` (iPhone) e `NavigationSplitView` (iPad) implementados em
      `RootView.swift`, alternando por `horizontalSizeClass` — não validado em
      runtime (ver item acima).
- [x] Detecção de suporte a LiDAR implementada (`LiDARCapabilityChecker`,
      checagem pura de hardware, sem sessão de câmera) — não validado em
      runtime.
- [x] Keychain implementado (`KeychainStore`) com testes unitários
      (`Tests/KeychainStoreTests.swift`, Swift Testing) — testes escritos mas
      não executados nesta sessão pelo mesmo motivo acima.

## Design system
- [x] `packages/design-tokens/tokens.json` é a fonte única; `tokens.ts`,
      `tokens.css` e `DesignTokens.swift` são gerados por
      `scripts/generate.mjs` e efetivamente usados por `apps/web`
      (confirmado no build) e referenciados por `apps/ios/project.yml`
      (não confirmado em build real, ver item iOS acima).

## Segurança
- [x] `.env.example` presente na raiz, sem segredo real; `.gitignore` cobre
      `.env`, `*.p8`, `**/asc.env`. Scanner de segredos (`gitleaks`) configurado
      em `.github/workflows/security-scan.yml` — não rodou ainda (sem GitHub
      remote nesta sessão).
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
- Nenhuma publicação (TestFlight, deploy web, GitHub remote) foi feita —
  bloqueada por falta de credencial/ambiente, não por escolha; ver
  `apps/ios/README.md` e o README principal para os próximos passos exatos.
