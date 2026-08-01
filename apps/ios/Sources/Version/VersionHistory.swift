import Foundation

/// Changelog em código (skill `ildemar_app-versioning`). Nova entrada sempre
/// no topo com `isCurrent: true`; a anterior vira `false`. Manter em sincronia
/// com `MARKETING_VERSION`/`CURRENT_PROJECT_VERSION` em `project.yml` a cada
/// release.
public struct VersionEntry: Identifiable, Sendable {
    public let id = UUID()
    public let version: String
    public let build: String
    public let date: String
    public let changes: [String]
    public let isCurrent: Bool
}

public enum VersionHistory {
    /// Fallbacks defensivos — usados apenas se o Info.plist não tiver os
    /// valores (não deveria acontecer em um build gerado pelo XcodeGen).
    public static let fallbackVersionString = "0.1.3 (Build 4)"
    public static let fallbackCommit = "dev"

    public static let entries: [VersionEntry] = [
        VersionEntry(
            version: "0.1.3",
            build: "4",
            date: "2026-08-01",
            changes: [
                "Corrige a causa real de 'Não foi possível falar com o servidor': caminhos com \"/\" inicial (ex.: \"/auth/login\") eram resolvidos como absolutos, descartando \"/api/v1\" do endereço do servidor e batendo na URL errada (o site, não a API)",
            ],
            isCurrent: true
        ),
        VersionEntry(
            version: "0.1.2",
            build: "3",
            date: "2026-08-01",
            changes: [
                "Corrige interceptação do cabeçalho Set-Cookie pela URLSession (httpShouldSetCookies desativado) — necessário mas não suficiente; a causa raiz era outra (ver 0.1.3)",
            ],
            isCurrent: false
        ),
        VersionEntry(
            version: "0.1.1",
            build: "2",
            date: "2026-08-01",
            changes: [
                "Corrige o app apontando para localhost — agora usa a URL de produção (https://wifi.egger.app.br/api/v1) por padrão, configurável via Info.plist sem alterar código",
            ],
            isCurrent: false
        ),
        VersionEntry(
            version: "0.1.0",
            build: "1",
            date: "2026-07-31",
            changes: [
                "Shell inicial: login, navegação adaptativa (TabView/NavigationSplitView), tema claro/escuro",
                "Detecção de suporte a LiDAR (sem sessão AR ainda — chega na Fase 6)",
                "Sessão persistida com segurança no Keychain",
            ],
            isCurrent: false
        ),
    ]

    public static var current: VersionEntry? {
        entries.first(where: \.isCurrent)
    }
}
