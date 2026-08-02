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
