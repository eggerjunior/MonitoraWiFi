# apps/ios — Egger Network Intelligence (iOS/iPadOS)

Status: Fase 1 (shell). Swift, SwiftUI, Swift Concurrency, Observation, ARKit
(detecção de LiDAR), Keychain. Projeto gerado via **XcodeGen** a partir de
`project.yml` — não há `.xcodeproj` commitado (gerado, `.gitignore`).

## ⚠️ Aviso importante sobre esta entrega

Este código foi escrito em um ambiente Linux **sem Xcode, sem toolchain Swift e
sem simulador iOS disponível** — não há `swift`, `xcodebuild` nem `xcodegen`
instalados neste sandbox. Por isso, **este código Swift não foi compilado nem
executado nesta sessão** — diferente do backend Go e do app web, que foram de
fato compilados, testados e validados ponta a ponta aqui.

O que isso significa na prática:

- A sintaxe e a arquitetura seguem as convenções corretas de Swift 6 / SwiftUI
  / Observation / ARKit / Keychain (Security framework) até onde é possível
  garantir sem um compilador real.
- Erros de compilação (typos, imports faltando, assinatura de API errada)
  **podem existir** e só serão pegos ao rodar `xcodegen generate` + build no
  Xcode (localmente em um Mac) ou no CI (`ios-ci.yml`, runner `macos-26`).
- Antes de considerar este shell "pronto", rode o build no Xcode ou dispare
  o workflow `iOS CI (build validation)` e corrija o que aparecer — isso é
  esperado como próximo passo, não uma falha desta entrega.

## Estrutura

```text
project.yml                  # fonte única de verdade (XcodeGen): bundle id, versão, build
Sources/App/                 # @main App + alternância login/shell
Sources/Auth/                # KeychainStore, SessionStore (Observable)
Sources/Networking/          # APIClient (URLSession async/await), modelos, parsing de cookie
Sources/Views/                # LoginView, RootView (TabView/NavigationSplitView), Overview, Settings, Placeholder
Sources/Version/             # VersionManager, VersionHistory (skill ildemar_app-versioning)
Sources/LiDAR/                # Detecção de suporte a LiDAR (ARKit, sem sessão de câmera)
Tests/                        # Swift Testing (@Test) — CookieParsing, KeychainStore, VersionManager
scripts/                      # testflight.sh, ExportOptions.plist, asc.env.example (skill ios-native-testflight)
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

## Bloqueado nesta entrega (Fase 1) — decisão do usuário necessária

Por instrução do skill `ildemar_ios-native-testflight`, publicação/TestFlight
não é opcional a perguntar depois — mas aqui ela está genuinamente bloqueada
por falta de ambiente/credencial, o que a própria regra prevê como exceção:

1. **Este projeto ainda não é um repositório git** (`git init` pendente) nem
   tem remote no GitHub — `ios-testflight.yml` precisa de um repo real para
   existir como Actions workflow executável.
2. **Sem credenciais da App Store Connect** (`IOS_ASC_KEY_ID`,
   `IOS_ASC_ISSUER_ID`, `IOS_ASC_KEY_P8_BASE64`) configuradas como secrets —
   só o Ildemar tem o arquivo `.p8` real.
3. **Sem Bundle ID nem app record criados** em App Store Connect ainda
   (`br.app.egger.network-intelligence`).

Próxima ação exata quando o usuário quiser avançar para publicação (Fase 8,
ou antes se desejado): `git init` + `gh repo create eggerjunior/MonitoraWiFi
--private` (ou nome equivalente) + criar o Bundle ID + pedir para o Ildemar
criar o app record em App Store Connect + configurar os três secrets acima +
disparar `iOS CI` para validar o build antes de qualquer `iOS TestFlight
release`.
