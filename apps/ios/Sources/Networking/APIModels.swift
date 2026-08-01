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

public struct Command: Codable, Sendable, Identifiable {
    public let id: String
    public let siteId: String
    public let agentId: String
    public let type: String
    public let status: String
    public let result: PingCommandResult?
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
