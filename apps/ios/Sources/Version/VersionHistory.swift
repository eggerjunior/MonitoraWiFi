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
    public static let fallbackVersionString = "0.1.6 (Build 7)"
    public static let fallbackCommit = "dev"

    public static let entries: [VersionEntry] = [
        VersionEntry(
            version: "0.1.6",
            build: "7",
            date: "2026-08-01",
            changes: [
                "Rede: inventário UniFi real (dispositivos, Wi-Fi/APs, clientes) — deixa de ser placeholder (Fase 3/4, início; paridade com o web)",
            ],
            isCurrent: true
        ),
        VersionEntry(
            version: "0.1.5",
            build: "6",
            date: "2026-08-01",
            changes: [
                "Ferramentas: ping sob demanda — dispara um comando real executado pelo agente do site e acompanha o resultado (Fase 5, início; paridade com o web)",
            ],
            isCurrent: false
        ),
        VersionEntry(
            version: "0.1.4",
            build: "5",
            date: "2026-08-01",
            changes: [
                "Corrige 'Não foi possível carregar organizações/sites': a tela Visão geral criava um APIClient próprio (sem o token de sessão do login), então todo request voltava 401 — agora reusa o client autenticado da SessionStore",
            ],
            isCurrent: false
        ),
        VersionEntry(
            version: "0.1.3",
            build: "4",
            date: "2026-08-01",
            changes: [
                "Corrige a causa real de 'Não foi possível falar com o servidor': caminhos com \"/\" inicial (ex.: \"/auth/login\") eram resolvidos como absolutos, descartando \"/api/v1\" do endereço do servidor e batendo na URL errada (o site, não a API)",
            ],
            isCurrent: false
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
