# apps/ios — Egger Network Intelligence (iOS/iPadOS)

Status: Fase 1 (shell), **build validado em CI real** (`macos-26`, Xcode 26.6).
Swift, SwiftUI, Swift Concurrency, Observation, ARKit (detecção de LiDAR),
Keychain. Projeto gerado via **XcodeGen** a partir de `project.yml` — não há
`.xcodeproj` commitado (gerado, `.gitignore`).

Repositório: https://github.com/eggerjunior/MonitoraWiFi (privado) — branch `main`.

## Este código foi escrito sem Xcode local, mas já foi compilado de verdade

Este ambiente de desenvolvimento (Linux) não tem Xcode/Swift/XcodeGen — o
código foi escrito sem poder ser compilado localmente. Mas depois do primeiro
push, o workflow `iOS CI (build validation)` (`.github/workflows/ios-ci.yml`,
runner `macos-26` da GitHub) **rodou de verdade** e encontrou (e corrigiu) 5
problemas reais, todos já resolvidos e no histórico de commits:

1. `List(_:selection:rowContent:)` com `Binding` não-opcional — API removida
   do SwiftUI (`RootView.swift`).
2. Nome de simulador fixo (`iPhone 16`) não existe mais no lineup atual do
   runner (`iPhone 16e`/`17`/`17 Pro`) — corrigido para seleção em runtime.
3. Trim de string incompatível com o `sed` BSD do macOS (`\s` não existe fora
   do GNU sed) — trocado por `awk`.
4. Ambiguidade de destino arm64/x86_64 no mesmo simulador — mitigada
   selecionando por UDID.
5. **Execução de testes unitários no CI não foi resolvida** — ver seção
   "Pendência real" abaixo.

O build (`xcodebuild ... build` para `generic/platform=iOS Simulator`) está
**verde**: https://github.com/eggerjunior/MonitoraWiFi/actions/workflows/ios-ci.yml

## Pendência real: testes unitários não rodam no CI

`xcodebuild test`/`build-for-testing` falha consistentemente com:

```
Could not find test host for EggerNetworkIntelligenceTests: TEST_HOST evaluates to
".../EggerNetworkIntelligence.app/EggerNetworkIntelligence"
```

mesmo com a dependência do target do app declarada corretamente em
`project.yml`, com `-derivedDataPath` isolado, com `build-for-testing` +
`test-without-building` em duas fases, e com destino de simulador por UDID
único (5 tentativas registradas no histórico de commits deste workflow). O
próprio template de referência da skill `ildemar_ios-native-testflight`
(`references/ios-ci.yml`) também **não** roda testes no CI — só build — o que
sugere que essa fragilidade de "hosted unit test" sob `xcodebuild` headless é
conhecida o suficiente para ter sido evitada ali também.

`Tests/` continua com os 3 arquivos de teste (Swift Testing) escritos e
corretos por inspeção — só não validados por execução ainda. Próximo passo
real: investigar em uma sessão com Xcode de verdade (abrir o projeto gerado,
rodar `⌘U`, ver se o problema se reproduz na GUI ou é específico de
`xcodebuild` headless) antes de tentar mais correções às cegas.

## Estrutura

```text
project.yml                  # fonte única de verdade (XcodeGen): bundle id, versão, build
Sources/App/                 # @main App + alternância login/shell
Sources/Auth/                # KeychainStore, SessionStore (Observable)
Sources/Networking/          # APIClient (URLSession async/await), modelos, parsing de cookie
Sources/Views/                # LoginView, RootView (TabView/NavigationSplitView), Overview, Settings, Placeholder
Sources/Version/             # VersionManager, VersionHistory (skill ildemar_app-versioning)
Sources/LiDAR/                # Detecção de suporte a LiDAR (ARKit, sem sessão de câmera)
Tests/                        # Swift Testing (@Test) — escritos, não validados por execução (ver acima)
scripts/                      # testflight.sh, ExportOptions.plist, asc.env.example, create_app.py
```

`../../packages/design-tokens/DesignTokens.swift` é referenciado diretamente
em `project.yml` (fonte compartilhada com o resto do monorepo) — não duplicar
cores/tipografia aqui.

## Como abrir (em um Mac com Xcode)

```bash
brew install xcodegen   # se ainda não tiver
cd apps/ios
xcodegen generate
open EggerNetworkIntelligence.xcodeproj
```

## O que existe nesta fase

- Login contra o backend real (`APIClient`), com o token de sessão persistido
  no **Keychain** (`KeychainStore`) — nunca a senha, só o token opaco do
  cookie `egger_session`.
- `SessionStore` (Observable) restaura a sessão do Keychain no boot e valida
  contra `/auth/me` antes de assumir que ainda é válida.
- Navegação adaptativa: `TabView` no iPhone, `NavigationSplitView` no iPad,
  ambos com as 5 seções da Seção 15 (Visão geral, Rede, Mapa, Ferramentas,
  Alertas) — só "Visão geral" tem conteúdo real (org/sites do backend); as
  demais são `ContentUnavailableView` honestos apontando a fase do roadmap.
- Detecção de suporte a LiDAR (`ARWorldTrackingConfiguration.supportsSceneReconstruction`)
  exibida na Visão geral — checagem de hardware, sem iniciar câmera/AR.
- Versionamento completo (skill `ildemar_app-versioning`): `MARKETING_VERSION`/
  `CURRENT_PROJECT_VERSION`/`GIT_COMMIT` em `project.yml`, `VersionManager`,
  `VersionHistory` com changelog, exibidos em Configurações.

## Status de publicação (TestFlight)

- ✅ Repositório GitHub privado criado e código enviado.
- ✅ Secrets `IOS_ASC_KEY_ID`, `IOS_ASC_ISSUER_ID`, `IOS_ASC_KEY_P8_BASE64`
  configurados no repositório.
- ✅ Bundle ID `br.app.egger.network-intelligence` criado no App Store Connect
  via API (`scripts/create_app.py`).
- ❌ **App record ainda não existe em App Store Connect** — a Apple não
  permite criar isso via API para nenhuma chave (`POST /v1/apps` retorna 403
  `FORBIDDEN_ERROR`, restrição permanente, não um erro pontual). **Esta etapa
  é sempre manual, do Ildemar**:
  1. App Store Connect → Apps → "+" → New App
  2. Bundle ID: `br.app.egger.network-intelligence` (já aparece na lista)
  3. SKU: `br.app.egger.network-intelligence`
  4. Primary Language: pt-BR
  5. Avisar quando terminar (~1 minuto)
- Depois disso: rodar `python3 apps/ios/scripts/create_app.py` de novo para
  confirmar, e então `gh workflow run "iOS TestFlight release" --repo
  eggerjunior/MonitoraWiFi`.
- ⚠️ Risco conhecido a observar no primeiro archive de Release: histórico do
  projeto MonitoraVPS (mesma conta Apple) mostra que `CODE_SIGN_STYLE:
  Automatic` sob `xcodebuild` headless esgota a cota de certificados de
  Development da conta após poucas execuções (ver
  `references/ildemar-ios-release.md` da skill `ildemar_ios-native-testflight`,
  seção "Certificados de assinatura esgotados"). Se o primeiro
  `iOS TestFlight release` falhar com erro de certificado/perfil, aplicar a
  correção documentada lá (Manual signing na config Release) antes de tentar
  de novo — não é preciso reinvestigar do zero.
