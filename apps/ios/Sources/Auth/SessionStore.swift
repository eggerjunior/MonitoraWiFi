import Foundation
import Observation

/// Estado de autenticação observável pela UI (SwiftUI + framework Observation,
/// conforme stack preferida do projeto — Seção 3). Fonte única de verdade
/// sobre "há um usuário autenticado agora", usada por `RootView` para decidir
/// entre `LoginView` e o shell principal.
@MainActor
@Observable
public final class SessionStore {
    public enum State: Equatable {
        case checking
        case authenticated(User)
        case unauthenticated
    }

    public private(set) var state: State = .checking
    public private(set) var lastError: String?

    /// Não-privado: as telas autenticadas (ex.: `OverviewView`) precisam
    /// reusar esta mesma instância — ela carrega o token de sessão obtido no
    /// login. Uma `APIClient()` nova não teria o token e todo request
    /// autenticado falharia com 401.
    let client: APIClient
    private let keychain: KeychainStore

    public init(client: APIClient = APIClient(), keychain: KeychainStore = KeychainStore()) {
        self.client = client
        self.keychain = keychain
    }

    /// Chamado uma vez na inicialização do app: tenta restaurar uma sessão
    /// persistida no Keychain e validá-la contra o backend antes de assumir
    /// que ainda é válida (o backend pode tê-la revogado ou expirado).
    public func bootstrap() async {
        do {
            let token = try keychain.loadToken()
            guard let token else {
                state = .unauthenticated
                return
            }
            await client.restoreSessionToken(token)
            let user = try await client.me()
            state = .authenticated(user)
        } catch {
            state = .unauthenticated
        }
    }

    public func login(email: String, password: String) async {
        lastError = nil
        do {
            let (session, token) = try await client.login(email: email, password: password)
            try keychain.save(token: token)
            state = .authenticated(session.user)
        } catch let error as APIClient.ClientError {
            lastError = Self.message(for: error)
        } catch {
            lastError = "Não foi possível entrar. Verifique sua conexão."
        }
    }

    public func logout() async {
        try? await client.logout()
        try? keychain.clear()
        state = .unauthenticated
    }

    private static func message(for error: APIClient.ClientError) -> String {
        switch error {
        case .server(let payload):
            return payload.message
        case .invalidResponse, .decoding:
            return "Não foi possível falar com o servidor. Tente novamente."
        }
    }
}
