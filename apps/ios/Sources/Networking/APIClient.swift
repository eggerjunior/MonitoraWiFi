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
        /// Usado apenas como fallback se `APIBaseURL` não estiver no
        /// Info.plist (não deveria acontecer num build gerado por
        /// `project.yml`).
        public static let developmentDefault = Configuration(
            baseURL: URL(string: "http://localhost:8080/api/v1")!
        )

        /// Lê `API_BASE_URL` (injetado no Info.plist via `project.yml`,
        /// mesmo padrão do `GIT_COMMIT`) — permite trocar o backend de
        /// destino num build sem alterar código, e evita o erro de deixar
        /// `localhost` hardcoded num build de produção/TestFlight.
        public static var fromInfoPlist: Configuration {
            if let urlString = Bundle.main.object(forInfoDictionaryKey: "APIBaseURL") as? String,
               let url = URL(string: urlString) {
                return Configuration(baseURL: url)
            }
            return developmentDefault
        }
    }

    private static let sessionCookieName = "egger_session"

    private let configuration: Configuration
    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder
    private var sessionToken: String?

    public init(configuration: Configuration = .fromInfoPlist) {
        self.configuration = configuration

        // httpShouldSetCookies = false: por padrão, a URLSession intercepta o
        // cabeçalho Set-Cookie da resposta para popular seu próprio
        // HTTPCookieStorage — e ao fazer isso, ele deixa de aparecer em
        // `HTTPURLResponse.value(forHTTPHeaderField: "Set-Cookie")`, mesmo
        // com configuração `.ephemeral`. Como este cliente gerencia o token
        // de sessão manualmente (para poder persisti-lo no Keychain), a
        // URLSession não pode ficar no meio do caminho consumindo o header
        // antes da gente conseguir lê-lo.
        let sessionConfig = URLSessionConfiguration.ephemeral
        sessionConfig.httpShouldSetCookies = false
        sessionConfig.httpCookieAcceptPolicy = .never
        self.session = URLSession(configuration: sessionConfig)

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

    /// Dispara um comando de ping sob demanda (Fase 5, início — executado de
    /// verdade pelo agente do site, nunca simulado localmente).
    public func createPingCommand(siteId: String, target: String, protocolName: String) async throws -> Command {
        struct Params: Encodable {
            let target: String
            let protocolName: String
            enum CodingKeys: String, CodingKey {
                case target
                case protocolName = "protocol"
            }
        }
        struct Body: Encodable {
            let type: String
            let params: Params
        }
        return try await postJSON("/sites/\(siteId)/commands", body: Body(type: "ping", params: Params(target: target, protocolName: protocolName)))
    }

    /// Consulta o status/resultado de um comando — usado para polling
    /// enquanto status é pending/claimed.
    public func getCommand(id: String) async throws -> Command {
        try await get("/commands/\(id)")
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

    private func postJSON<Body: Encodable, T: Decodable>(_ path: String, body: Body) async throws -> T {
        var request = try makeRequest(path: path, method: "POST")
        request.httpBody = try encoder.encode(body)
        return try await send(request)
    }

    private func makeRequest(path: String, method: String) throws -> URLRequest {
        // `URL(string:relativeTo:)` trata uma string começando com "/" como
        // caminho absoluto (RFC 3986) — isso DESCARTA qualquer componente de
        // path do baseURL (ex.: "/api/v1"), em vez de ser relativo a ele.
        // Como `baseURL` sempre inclui "/api/v1", montamos a URL final por
        // concatenação de string (preservando querystring), não por
        // resolução de URL relativa.
        var base = configuration.baseURL.absoluteString
        if !base.hasSuffix("/") {
            base += "/"
        }
        let trimmedPath = path.hasPrefix("/") ? String(path.dropFirst()) : path
        guard let url = URL(string: base + trimmedPath) else {
            throw ClientError.invalidResponse
        }
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
