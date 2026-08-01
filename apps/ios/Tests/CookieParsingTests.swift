import Testing
@testable import EggerNetworkIntelligence

@Suite("CookieParsing")
struct CookieParsingTests {
    @Test("extrai o valor do cookie de sessão de um Set-Cookie completo")
    func extractsSessionToken() {
        let header = "egger_session=abc123XYZ; Path=/; Expires=Sat, 08 Aug 2026 01:23:18 GMT; HttpOnly; Secure; SameSite=Strict"
        let value = CookieParsing.extractValue(named: "egger_session", fromSetCookieHeader: header)
        #expect(value == "abc123XYZ")
    }

    @Test("retorna nil quando o cookie procurado não está presente")
    func returnsNilWhenMissing() {
        let header = "outro_cookie=valor; Path=/"
        let value = CookieParsing.extractValue(named: "egger_session", fromSetCookieHeader: header)
        #expect(value == nil)
    }

    @Test("não confunde um cookie cujo nome é prefixo de outro")
    func doesNotMatchPartialName() {
        let header = "egger_session_extra=valor_errado; egger_session=valor_certo; Path=/"
        let value = CookieParsing.extractValue(named: "egger_session", fromSetCookieHeader: header)
        #expect(value == "valor_certo")
    }
}
