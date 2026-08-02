import Foundation

// Tipos espelhando packages/contracts/openapi.yaml. Nesta fase são escritos à
// mão, como no cliente web (ver apps/web/src/lib/api-types.ts) — a Fase 2 deve
// introduzir geração automática a partir do OpenAPI (ADR-005) para os dois
// clientes ao mesmo tempo, eliminando o risco de divergência manual.

public enum Role: String, Codable, Sendable {
    case owner
    case administrator
    case operatorRole = "operator"
    case viewer
    case auditor
}

public struct User: Codable, Sendable, Identifiable, Equatable {
    public let id: String
    public let organizationId: String
    public let email: String
    public let role: Role
    public let mfaEnrolled: Bool
    public let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case organizationId = "organization_id"
        case email
        case role
        case mfaEnrolled = "mfa_enrolled"
        case createdAt = "created_at"
    }
}

public struct AuthSession: Codable, Sendable {
    public let user: User
    public let expiresAt: Date

    enum CodingKeys: String, CodingKey {
        case user
        case expiresAt = "expires_at"
    }
}

public struct ApiErrorPayload: Codable, Sendable, Error {
    public let error: String
    public let message: String
    public let correlationId: String

    enum CodingKeys: String, CodingKey {
        case error
        case message
        case correlationId = "correlation_id"
    }
}

public struct Organization: Codable, Sendable, Identifiable, Equatable {
    public let id: String
    public let name: String
    public let planTier: String
    public let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case planTier = "plan_tier"
        case createdAt = "created_at"
    }
}

public struct Site: Codable, Sendable, Identifiable, Equatable {
    public let id: String
    public let organizationId: String
    public let name: String
    public let timezone: String
    public let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case organizationId = "organization_id"
        case name
        case timezone
        case createdAt = "created_at"
    }
}

public struct PingCommandResult: Codable, Sendable {
    public let target: String
    public let protocolName: String
    public let latencyMsP50: Double?
    public let latencyMsP95: Double?
    public let latencyMsP99: Double?
    public let jitterMs: Double?
    public let packetLossPct: Double?

    enum CodingKeys: String, CodingKey {
        case target
        case protocolName = "protocol"
        case latencyMsP50 = "latency_ms_p50"
        case latencyMsP95 = "latency_ms_p95"
        case latencyMsP99 = "latency_ms_p99"
        case jitterMs = "jitter_ms"
        case packetLossPct = "packet_loss_pct"
    }
}

public struct BatchPingCommandResult: Codable, Sendable {
    public let protocolName: String
    public let results: [PingCommandResult]

    enum CodingKeys: String, CodingKey {
        case protocolName = "protocol"
        case results
    }
}

public struct DnsLookupCommandResult: Codable, Sendable {
    public let hostname: String
    public let addresses: [String]
    public let durationMs: Double

    enum CodingKeys: String, CodingKey {
        case hostname
        case addresses
        case durationMs = "duration_ms"
    }
}

public struct TracerouteHop: Codable, Sendable, Identifiable {
    public let hop: Int
    public let address: String
    public let rttMs: Double?
    public var id: Int { hop }

    enum CodingKeys: String, CodingKey {
        case hop
        case address
        case rttMs = "rtt_ms"
    }
}

public struct TracerouteCommandResult: Codable, Sendable {
    public let target: String
    public let reached: Bool
    public let hops: [TracerouteHop]
}

public struct SslCheckCommandResult: Codable, Sendable {
    public let target: String
    public let port: Int
    public let validNow: Bool
    public let verifyError: String
    public let notBefore: String
    public let notAfter: String
    public let daysUntilExpiry: Int
    public let issuer: String
    public let subject: String
    public let dnsNames: [String]

    enum CodingKeys: String, CodingKey {
        case target
        case port
        case validNow = "valid_now"
        case verifyError = "verify_error"
        case notBefore = "not_before"
        case notAfter = "not_after"
        case daysUntilExpiry = "days_until_expiry"
        case issuer
        case subject
        case dnsNames = "dns_names"
    }
}

/// Formato do resultado depende de `Command.type` — cada caso só decodifica
/// com sucesso a partir do JSON do tipo correspondente (os três shapes têm
/// campos obrigatórios mutuamente exclusivos), nunca inventamos qual é.
public enum CommandResult: Sendable {
    case ping(PingCommandResult)
    case batchPing(BatchPingCommandResult)
    case dnsLookup(DnsLookupCommandResult)
    case traceroute(TracerouteCommandResult)
    case sslCheck(SslCheckCommandResult)
    case httpRequest(HttpRequestCommandResult)
    case lanScan(LanScanCommandResult)
    case wakeOnLan(WakeOnLanCommandResult)
    case portScan(PortScanCommandResult)
    case dnsResolverCompare(DnsResolverCompareCommandResult)
}

extension CommandResult: Decodable {
    public init(from decoder: Decoder) throws {
        if let v = try? PingCommandResult(from: decoder) {
            self = .ping(v)
        } else if let v = try? BatchPingCommandResult(from: decoder) {
            self = .batchPing(v)
        } else if let v = try? DnsLookupCommandResult(from: decoder) {
            self = .dnsLookup(v)
        } else if let v = try? TracerouteCommandResult(from: decoder) {
            self = .traceroute(v)
        } else if let v = try? SslCheckCommandResult(from: decoder) {
            self = .sslCheck(v)
        } else if let v = try? HttpRequestCommandResult(from: decoder) {
            self = .httpRequest(v)
        } else if let v = try? LanScanCommandResult(from: decoder) {
            self = .lanScan(v)
        } else if let v = try? WakeOnLanCommandResult(from: decoder) {
            self = .wakeOnLan(v)
        } else if let v = try? PortScanCommandResult(from: decoder) {
            self = .portScan(v)
        } else if let v = try? DnsResolverCompareCommandResult(from: decoder) {
            self = .dnsResolverCompare(v)
        } else {
            throw DecodingError.dataCorruptedError(in: try decoder.singleValueContainer(), debugDescription: "Formato de resultado de comando não reconhecido")
        }
    }
}

public struct Command: Decodable, Sendable, Identifiable {
    public let id: String
    public let siteId: String
    public let agentId: String
    public let type: String
    public let status: String
    public let result: CommandResult?
    public let error: String?

    enum CodingKeys: String, CodingKey {
        case id
        case siteId = "site_id"
        case agentId = "agent_id"
        case type
        case status
        case result
        case error
    }
}

public struct HttpRequestCommandResult: Codable, Sendable {
    public let url: String
    public let method: String
    public let statusCode: Int
    public let statusText: String
    public let headers: [String: String]
    public let bodySnippet: String
    public let bodyTruncated: Bool
    public let contentLength: Int
    public let durationMs: Double

    enum CodingKeys: String, CodingKey {
        case url
        case method
        case statusCode = "status_code"
        case statusText = "status_text"
        case headers
        case bodySnippet = "body_snippet"
        case bodyTruncated = "body_truncated"
        case contentLength = "content_length"
        case durationMs = "duration_ms"
    }
}

public struct LanScanCommandResult: Codable, Sendable {
    public let cidr: String
    public let hosts: [String]
}

public struct WakeOnLanCommandResult: Codable, Sendable {
    public let macAddress: String
    public let broadcastIp: String
    public let port: Int

    enum CodingKeys: String, CodingKey {
        case macAddress = "mac_address"
        case broadcastIp = "broadcast_ip"
        case port
    }
}

public struct PortScanCommandResult: Codable, Sendable {
    public let target: String
    public let openPorts: [Int]

    enum CodingKeys: String, CodingKey {
        case target
        case openPorts = "open_ports"
    }
}

/// Resultado da resolução contra um resolvedor específico dentro da
/// comparação (Fase 2) — nunca inventa endereço quando `error` vem
/// preenchido (Seção 2.1).
public struct DnsResolverResult: Codable, Sendable, Identifiable {
    public let resolver: String
    public let addresses: [String]
    public let durationMs: Double
    public let error: String
    public var id: String { resolver }

    enum CodingKeys: String, CodingKey {
        case resolver
        case addresses
        case durationMs = "duration_ms"
        case error
    }
}

/// Comparação entre resolvedores DNS (lista fixa: sistema, Cloudflare,
/// Google, Quad9 — ver apps/local-agent/internal/probes/probes.go,
/// KnownResolvers).
public struct DnsResolverCompareCommandResult: Codable, Sendable {
    public let hostname: String
    public let resolvers: [DnsResolverResult]
}

public struct RdapEvent: Codable, Sendable {
    public let action: String
    public let date: String
}

public struct RdapResult: Codable, Sendable {
    public let query: String
    public let server: String
    public let objectClassName: String
    public let handle: String
    public let name: String
    public let status: [String]
    public let events: [RdapEvent]
    public let nameservers: [String]?

    enum CodingKeys: String, CodingKey {
        case query
        case server
        case objectClassName = "object_class_name"
        case handle
        case name
        case status
        case events
        case nameservers
    }
}

/// Anomalia estatística (Fase 7, worker) — nunca reportada sem histórico
/// suficiente no bucket (hora do dia × dia da semana).
public struct Anomaly: Codable, Sendable, Identifiable {
    public let id: String
    public let metric: String
    public let observedAt: String
    public let value: Double
    public let bucketMean: Double
    public let bucketSize: Int
    public let zScore: Double
    public let detectedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case metric
        case observedAt = "observed_at"
        case value
        case bucketMean = "bucket_mean"
        case bucketSize = "bucket_size"
        case zScore = "z_score"
        case detectedAt = "detected_at"
    }
}

/// Resultado de ping periódico do agente (Fase 2) — histórico real, não o
/// ping sob demanda de PingCommandResult (Fase 5).
public struct PingTestRecord: Codable, Sendable, Identifiable {
    public let id: String
    public let target: String
    public let protocolName: String
    public let latencyMsP50: Double?
    public let packetLossPct: Double?
    public let jitterMs: Double?
    public let executedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case target
        case protocolName = "protocol"
        case latencyMsP50 = "latency_ms_p50"
        case packetLossPct = "packet_loss_pct"
        case jitterMs = "jitter_ms"
        case executedAt = "executed_at"
    }
}

public struct SpeedTestRecord: Codable, Sendable, Identifiable {
    public let id: String
    public let mode: String
    public let downloadMbps: Double?
    public let uploadMbps: Double?
    public let bufferbloatMs: Double?
    public let executedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case mode
        case downloadMbps = "download_mbps"
        case uploadMbps = "upload_mbps"
        case bufferbloatMs = "bufferbloat_ms"
        case executedAt = "executed_at"
    }
}

public struct UniFiDevice: Codable, Sendable, Identifiable {
    public let id: String
    public let externalId: String
    public let macAddress: String
    public let ipAddress: String
    public let name: String
    public let model: String
    public let state: String
    public let firmwareVersion: String
    public let features: [String]
    public let interfaces: [String]
    /// external_id do dispositivo upstream (ex.: o switch a que um AP está
    /// conectado) — vazio pro dispositivo raiz (gateway). Confirmado em
    /// 2026-08-02 contra a instalação real.
    public let uplinkDeviceId: String

    enum CodingKeys: String, CodingKey {
        case id
        case externalId = "external_id"
        case macAddress = "mac_address"
        case ipAddress = "ip_address"
        case name
        case model
        case state
        case firmwareVersion = "firmware_version"
        case features
        case interfaces
        case uplinkDeviceId = "uplink_device_id"
    }
}

public struct UniFiDeviceList: Codable, Sendable {
    public let items: [UniFiDevice]
}

public struct UniFiClient: Codable, Sendable, Identifiable {
    public let id: String
    public let type: String
    public let name: String
    public let ipAddress: String
    public let macAddress: String
    public let uplinkDeviceId: String

    enum CodingKeys: String, CodingKey {
        case id
        case type
        case name
        case ipAddress = "ip_address"
        case macAddress = "mac_address"
        case uplinkDeviceId = "uplink_device_id"
    }
}

public struct UniFiClientList: Codable, Sendable {
    public let items: [UniFiClient]
}

public struct Page<Item: Codable & Sendable>: Codable, Sendable {
    public let items: [Item]
    public let page: Int
    public let pageSize: Int
    public let total: Int

    enum CodingKeys: String, CodingKey {
        case items
        case page
        case pageSize = "page_size"
        case total
    }
}

// MARK: - Levantamento espacial (Fase 6, "Spatial WiFi Survey")

/// Amostra capturada durante a caminhada guiada — posição real do ARKit
/// (world tracking, relativa à origem da sessão, não georreferenciada) +
/// qualidade de rede medida no próprio ponto. Deliberadamente sem RSSI/canal/
/// PHY rate: nenhum desses campos é obtido pelo iPhone (ver capability
/// matrix) nem por esta versão da Network API do UniFi (confirmado ausente
/// em `/clients`) — nunca inventar dado que a plataforma não expõe.
public struct SpatialSurveySample: Codable, Sendable {
    public let positionX: Double
    public let positionY: Double
    public let positionZ: Double
    public let ssid: String?
    public let bssid: String?
    public let rttMs: Double?
    public let isExpensive: Bool
    public let isConstrained: Bool
    public let interfaceType: String
    public let capturedAt: String

    enum CodingKeys: String, CodingKey {
        case positionX = "position_x"
        case positionY = "position_y"
        case positionZ = "position_z"
        case ssid
        case bssid
        case rttMs = "rtt_ms"
        case isExpensive = "is_expensive"
        case isConstrained = "is_constrained"
        case interfaceType = "interface_type"
        case capturedAt = "captured_at"
    }

    public init(positionX: Double, positionY: Double, positionZ: Double, ssid: String?, bssid: String?, rttMs: Double?, isExpensive: Bool, isConstrained: Bool, interfaceType: String, capturedAt: String) {
        self.positionX = positionX
        self.positionY = positionY
        self.positionZ = positionZ
        self.ssid = ssid
        self.bssid = bssid
        self.rttMs = rttMs
        self.isExpensive = isExpensive
        self.isConstrained = isConstrained
        self.interfaceType = interfaceType
        self.capturedAt = capturedAt
    }
}

public struct CreateSpatialSurveyRequest: Encodable, Sendable {
    public let name: String
    public let deviceModel: String
    public let lidarUsed: Bool
    public let startedAt: String
    public let finishedAt: String
    public let samples: [SpatialSurveySample]

    enum CodingKeys: String, CodingKey {
        case name
        case deviceModel = "device_model"
        case lidarUsed = "lidar_used"
        case startedAt = "started_at"
        case finishedAt = "finished_at"
        case samples
    }

    public init(name: String, deviceModel: String, lidarUsed: Bool, startedAt: String, finishedAt: String, samples: [SpatialSurveySample]) {
        self.name = name
        self.deviceModel = deviceModel
        self.lidarUsed = lidarUsed
        self.startedAt = startedAt
        self.finishedAt = finishedAt
        self.samples = samples
    }
}

public struct SpatialSurvey: Codable, Sendable, Identifiable {
    public let id: String
    public let siteId: String
    public let createdBy: String
    public let name: String
    public let deviceModel: String
    public let lidarUsed: Bool
    public let startedAt: String
    public let finishedAt: String
    public let sampleCount: Int
    public let samples: [SpatialSurveySample]?

    enum CodingKeys: String, CodingKey {
        case id
        case siteId = "site_id"
        case createdBy = "created_by"
        case name
        case deviceModel = "device_model"
        case lidarUsed = "lidar_used"
        case startedAt = "started_at"
        case finishedAt = "finished_at"
        case sampleCount = "sample_count"
        case samples
    }
}
