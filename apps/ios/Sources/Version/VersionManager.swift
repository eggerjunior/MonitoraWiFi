import Foundation

/// Lê versão/build/commit do `Info.plist` gerado pelo XcodeGen a partir de
/// `project.yml` (fonte única de verdade — ver skill `ildemar_app-versioning`).
/// `GIT_COMMIT` é injetado na linha de comando por `scripts/testflight.sh` no
/// momento do archive; builds locais/manuais mostram `dev`.
public enum VersionManager {
    public static var marketingVersion: String {
        Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "0.1.0"
    }

    public static var buildNumber: String {
        Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "1"
    }

    public static var gitCommit: String {
        Bundle.main.infoDictionary?["GitCommit"] as? String ?? "dev"
    }

    public static var currentVersionString: String {
        "\(marketingVersion) (Build \(buildNumber))"
    }

    /// `nil` quando o commit é `dev` (build local/manual) — nesse caso a UI
    /// deve mostrar o texto sem link, em vez de um link quebrado.
    public static var commitURL: URL? {
        guard gitCommit != "dev" else { return nil }
        return URL(string: "https://github.com/eggerjunior/MonitoraWiFi/commit/\(gitCommit)")
    }

    /// Data de modificação do executável dentro do bundle — mostra quando o
    /// binário foi de fato compilado, não um valor gravado em código.
    public static var buildDate: Date? {
        guard let executableURL = Bundle.main.executableURL,
              let attributes = try? FileManager.default.attributesOfItem(atPath: executableURL.path)
        else {
            return nil
        }
        return attributes[.modificationDate] as? Date
    }

    public static var buildDateString: String {
        guard let buildDate else { return "data de build desconhecida" }
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        formatter.timeStyle = .short
        formatter.locale = Locale(identifier: "pt_BR")
        return formatter.string(from: buildDate)
    }
}
