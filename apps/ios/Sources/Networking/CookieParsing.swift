import Foundation

/// Extração de valor de cookie a partir de um cabeçalho `Set-Cookie` —
/// isolado em um utilitário puro para ser testável sem rede nem Keychain.
enum CookieParsing {
    static func extractValue(named name: String, fromSetCookieHeader header: String) -> String? {
        for cookiePair in header.split(separator: ";") {
            let parts = cookiePair.split(separator: "=", maxSplits: 1)
            guard parts.count == 2 else { continue }
            let key = parts[0].trimmingCharacters(in: .whitespaces)
            if key == name {
                return String(parts[1])
            }
        }
        return nil
    }
}
