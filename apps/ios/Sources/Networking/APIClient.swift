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

    /// Mede o RTT real até o backend a partir de onde o dispositivo está
    /// agora — usado pelo levantamento espacial (Fase 6) pra atribuir uma
    /// qualidade de rede a cada ponto capturado. Reaproveita `/auth/me`
    /// (sessão já autenticada) em vez de criar um endpoint só pra isso;
    /// nunca inventa um valor quando a chamada falha (retorna nil).
    public func measureRTTToBackend() async -> Double? {
        let start = Date()
        do {
            _ = try await get("/auth/me") as User
        } catch {
            return nil
        }
        return Date().timeIntervalSince(start) * 1000
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

    public func createDNSLookupCommand(siteId: String, hostname: String) async throws -> Command {
        struct Params: Encodable { let hostname: String }
        struct Body: Encodable { let type: String; let params: Params }
        return try await postJSON("/sites/\(siteId)/commands", body: Body(type: "dns_lookup", params: Params(hostname: hostname)))
    }

    public func createDNSResolverCompareCommand(siteId: String, hostname: String) async throws -> Command {
        struct Params: Encodable { let hostname: String }
        struct Body: Encodable { let type: String; let params: Params }
        return try await postJSON("/sites/\(siteId)/commands", body: Body(type: "dns_resolver_compare", params: Params(hostname: hostname)))
    }

    public func createTracerouteCommand(siteId: String, target: String) async throws -> Command {
        struct Params: Encodable { let target: String }
        struct Body: Encodable { let type: String; let params: Params }
        return try await postJSON("/sites/\(siteId)/commands", body: Body(type: "traceroute", params: Params(target: target)))
    }

    /// Ping em lote (Fase 5) — mesma fila de comando sob demanda, testando
    /// vários alvos numa única execução real do agente.
    public func createBatchPingCommand(siteId: String, targets: [String], protocolName: String) async throws -> Command {
        struct Params: Encodable {
            let targets: [String]
            let protocolName: String
            enum CodingKeys: String, CodingKey {
                case targets
                case protocolName = "protocol"
            }
        }
        struct Body: Encodable { let type: String; let params: Params }
        return try await postJSON("/sites/\(siteId)/commands", body: Body(type: "batch_ping", params: Params(targets: targets, protocolName: protocolName)))
    }

    /// Verificação de certificado SSL/TLS sob demanda (Fase 5) — handshake
    /// real feito pelo agente do site, nunca simulado localmente.
    public func createSslCheckCommand(siteId: String, target: String, port: Int) async throws -> Command {
        struct Params: Encodable { let target: String; let port: Int }
        struct Body: Encodable { let type: String; let params: Params }
        return try await postJSON("/sites/\(siteId)/commands", body: Body(type: "ssl_check", params: Params(target: target, port: port)))
    }

    /// HTTP client sob demanda (Fase 5) — requisição real feita pelo
    /// agente do site, nunca simulada localmente.
    public func createHTTPRequestCommand(siteId: String, url: String, method: String, body: String?) async throws -> Command {
        struct Params: Encodable { let url: String; let method: String; let body: String? }
        struct Body: Encodable { let type: String; let params: Params }
        return try await postJSON("/sites/\(siteId)/commands", body: Body(type: "http_request", params: Params(url: url, method: method, body: body)))
    }

    /// LAN scanner (Fase 5) — varre um bloco CIDR privado (RFC 1918, no
    /// máximo /22) por hosts reais, executado pelo agente do site.
    public func createLANScanCommand(siteId: String, cidr: String) async throws -> Command {
        struct Params: Encodable { let cidr: String }
        struct Body: Encodable { let type: String; let params: Params }
        return try await postJSON("/sites/\(siteId)/commands", body: Body(type: "lan_scan", params: Params(cidr: cidr)))
    }

    /// Wake-on-LAN (Fase 5, ADR-008) — o magic packet é enviado
    /// exclusivamente pelo agente do site (nunca pelo app iOS, que tem
    /// restrições de plataforma documentadas no ADR-008 pra
    /// broadcast/multicast).
    public func createWakeOnLANCommand(siteId: String, macAddress: String, broadcastIP: String) async throws -> Command {
        struct Params: Encodable { let macAddress: String; let broadcastIp: String
            enum CodingKeys: String, CodingKey { case macAddress = "mac_address"; case broadcastIp = "broadcast_ip" }
        }
        struct Body: Encodable { let type: String; let params: Params }
        return try await postJSON("/sites/\(siteId)/commands", body: Body(type: "wake_on_lan", params: Params(macAddress: macAddress, broadcastIp: broadcastIP)))
    }

    /// Port scanner (Fase 5) — só aceita IPv4 privado literal (RFC 1918)
    /// como alvo (mitigação do threat-model.md §5), executado pelo agente.
    public func createPortScanCommand(siteId: String, target: String, startPort: Int, endPort: Int) async throws -> Command {
        struct Params: Encodable { let target: String; let startPort: Int; let endPort: Int
            enum CodingKeys: String, CodingKey { case target; case startPort = "start_port"; case endPort = "end_port" }
        }
        struct Body: Encodable { let type: String; let params: Params }
        return try await postJSON("/sites/\(siteId)/commands", body: Body(type: "port_scan", params: Params(target: target, startPort: startPort, endPort: endPort)))
    }

    /// RDAP/WHOIS (Fase 5) — consulta pública sobre domínio/IP, resolvida
    /// pelo backend via bootstrap real da IANA. Não passa pelo agente do
    /// site (a informação é da internet, não da LAN).
    public func rdapLookup(query: String) async throws -> RdapResult {
        let encoded = query.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? query
        return try await get("/rdap/lookup?query=\(encoded)")
    }

    /// Anomalias estatísticas (Fase 7, worker) — só leitura.
    public func anomalies(siteId: String, page: Int = 1, pageSize: Int = 50) async throws -> Page<Anomaly> {
        try await get("/sites/\(siteId)/anomalies?page=\(page)&page_size=\(pageSize)")
    }

    /// Histórico de ping/speed test do agente (Fase 2/4) — série real, não
    /// o resultado sob demanda de PingCommandResult (Fase 5).
    public func pingTests(siteId: String, page: Int = 1, pageSize: Int = 20) async throws -> Page<PingTestRecord> {
        try await get("/sites/\(siteId)/ping-tests?page=\(page)&page_size=\(pageSize)")
    }

    public func speedTests(siteId: String, page: Int = 1, pageSize: Int = 20) async throws -> Page<SpeedTestRecord> {
        try await get("/sites/\(siteId)/speed-tests?page=\(page)&page_size=\(pageSize)")
    }

    /// Inventário UniFi (Fase 3/4, início) — sincronizado pelo agente local,
    /// nunca chamado diretamente pelo app (ADR-001).
    public func uniFiDevices(siteId: String) async throws -> UniFiDeviceList {
        try await get("/sites/\(siteId)/unifi/devices")
    }

    public func uniFiClients(siteId: String) async throws -> UniFiClientList {
        try await get("/sites/\(siteId)/unifi/clients")
    }

    /// Envia um levantamento espacial completo (Fase 6) — metadados + todas
    /// as amostras de uma vez, ao final da caminhada guiada.
    public func createSpatialSurvey(siteId: String, request: CreateSpatialSurveyRequest) async throws -> SpatialSurvey {
        try await postJSON("/sites/\(siteId)/spatial-surveys", body: request)
    }

    public func spatialSurveys(siteId: String, page: Int = 1, pageSize: Int = 20) async throws -> Page<SpatialSurvey> {
        try await get("/sites/\(siteId)/spatial-surveys?page=\(page)&page_size=\(pageSize)")
    }

    public func spatialSurvey(id: String) async throws -> SpatialSurvey {
        try await get("/spatial-surveys/\(id)")
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
