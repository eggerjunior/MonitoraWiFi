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
    public static let fallbackVersionString = "0.1.0 (Build 1)"
    public static let fallbackCommit = "dev"

    public static let entries: [VersionEntry] = [
        VersionEntry(
            version: "0.1.0",
            build: "1",
            date: "2026-07-31",
            changes: [
                "Shell inicial: login, navegação adaptativa (TabView/NavigationSplitView), tema claro/escuro",
                "Detecção de suporte a LiDAR (sem sessão AR ainda — chega na Fase 6)",
                "Sessão persistida com segurança no Keychain",
            ],
            isCurrent: true
        ),
    ]

    public static var current: VersionEntry? {
        entries.first(where: \.isCurrent)
    }
}
