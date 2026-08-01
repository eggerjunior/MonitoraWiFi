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
