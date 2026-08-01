import Foundation

/// Cliente HTTP para o backend central.
///
/// O token de sessão é mantido explicitamente (não apenas via cookie jar
/// implícito da `URLSession`) para que possa ser persistido no Keychain
/// (Seção 2.2) pelo `SessionStore` e restaurado entre lançamentos do app.
public actor APIClient {
    public struct Configuration: Sendable {
        public var baseURL: URL

        /// No simulador, `localhost` aponta para o próprio Mac rodando o
        /// Docker Compose de desenvolvimento (infrastructure/docker). Em um
        /// dispositivo físico na mesma LAN, isso precisa ser trocado pelo IP
        /// do backend — não há um valor único que sirva para os dois casos.
        public static let developmentDefault = Configuration(
            baseURL: URL(string: "http://localhost:8080/api/v1")!
        )
    }

    private static let sessionCookieName = "egger_session"

    private let configuration: Configuration
    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder
    private var sessionToken: String?

    public init(configuration: Configuration = .developmentDefault) {
        self.configuration = configuration
        self.session = URLSession(configuration: .ephemeral)

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        self.decoder = decoder

        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        self.encoder = encoder
    }

    public enum ClientError: Error, Sendable {
        case invalidResponse
        case server(ApiErrorPayload)
        case decoding(Error)
    }

    /// Restaura um token de sessão persistido (ex.: lido do Keychain no início
    /// do app), para reusar a sessão sem pedir login novamente.
    public func restoreSessionToken(_ token: String?) {
        self.sessionToken = token
    }

    public func login(email: String, password: String) async throws -> (session: AuthSession, token: String) {
        struct LoginBody: Encodable {
            let email: String
            let password: String
        }
        let request = try makeRequest(path: "/auth/login", method: "POST")
        var mutableRequest = request
        mutableRequest.httpBody = try encoder.encode(LoginBody(email: email, password: password))

        let (data, response) = try await session.data(for: mutableRequest)
        let http = try validate(response: response, data: data)

        guard let setCookie = http.value(forHTTPHeaderField: "Set-Cookie"),
              let token = CookieParsing.extractValue(named: Self.sessionCookieName, fromSetCookieHeader: setCookie)
        else {
            throw ClientError.invalidResponse
        }

        let authSession = try decode(AuthSession.self, from: data)
        self.sessionToken = token
        return (authSession, token)
    }

    public func logout() async throws {
        _ = try? await postNoBody("/auth/logout") as EmptyResponse
        sessionToken = nil
    }

    public func me() async throws -> User {
        try await get("/auth/me")
    }

    public func organizations(page: Int = 1, pageSize: Int = 20) async throws -> Page<Organization> {
        try await get("/organizations?page=\(page)&page_size=\(pageSize)")
    }

    public func sites(organizationId: String, page: Int = 1, pageSize: Int = 50) async throws -> Page<Site> {
        try await get("/sites?organization_id=\(organizationId)&page=\(page)&page_size=\(pageSize)")
    }

    // MARK: - Núcleo HTTP

    private struct EmptyResponse: Decodable {}

    private func get<T: Decodable>(_ path: String) async throws -> T {
        let request = try makeRequest(path: path, method: "GET")
        return try await send(request)
    }

    private func postNoBody<T: Decodable>(_ path: String) async throws -> T {
        let request = try makeRequest(path: path, method: "POST")
        return try await send(request)
    }

    private func makeRequest(path: String, method: String) throws -> URLRequest {
        let url = URL(string: path, relativeTo: configuration.baseURL)!
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let sessionToken {
            request.setValue("\(Self.sessionCookieName)=\(sessionToken)", forHTTPHeaderField: "Cookie")
        }
        return request
    }

    private func send<T: Decodable>(_ request: URLRequest) async throws -> T {
        let (data, response) = try await session.data(for: request)
        _ = try validate(response: response, data: data)
        return try decode(T.self, from: data)
    }

    @discardableResult
    private func validate(response: URLResponse, data: Data) throws -> HTTPURLResponse {
        guard let http = response as? HTTPURLResponse else {
            throw ClientError.invalidResponse
        }
        guard (200..<300).contains(http.statusCode) else {
            if let payload = try? decoder.decode(ApiErrorPayload.self, from: data) {
                throw ClientError.server(payload)
            }
            throw ClientError.invalidResponse
        }
        return http
    }

    private func decode<T: Decodable>(_ type: T.Type, from data: Data) throws -> T {
        if data.isEmpty, let empty = EmptyResponse() as? T {
            return empty
        }
        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw ClientError.decoding(error)
        }
    }
}
