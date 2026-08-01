import Testing
@testable import EggerNetworkIntelligence

@Suite("KeychainStore")
struct KeychainStoreTests {
    // Serviço próprio para não colidir com o Keychain real do app durante
    // testes rodados no simulador/dispositivo de CI.
    private let store = KeychainStore(service: "br.app.egger.network-intelligence.tests")

    @Test("salva e lê de volta o mesmo token")
    func saveAndLoadRoundTrip() throws {
        try store.clear()
        try store.save(token: "token-de-teste-123")
        let loaded = try store.loadToken()
        #expect(loaded == "token-de-teste-123")
        try store.clear()
    }

    @Test("retorna nil quando não há token salvo")
    func loadReturnsNilWhenEmpty() throws {
        try store.clear()
        let loaded = try store.loadToken()
        #expect(loaded == nil)
    }

    @Test("salvar um novo token substitui o anterior, não duplica")
    func savingTwiceReplaces() throws {
        try store.clear()
        try store.save(token: "primeiro")
        try store.save(token: "segundo")
        let loaded = try store.loadToken()
        #expect(loaded == "segundo")
        try store.clear()
    }
}
