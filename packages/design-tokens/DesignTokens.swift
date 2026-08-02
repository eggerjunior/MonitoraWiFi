// GERADO AUTOMATICAMENTE por packages/design-tokens/scripts/generate.mjs
// Não editar à mão — edite tokens.json e rode `node scripts/generate.mjs`.

import SwiftUI

/// Token de cor semântico. O valor concreto depende do esquema (claro/escuro),
/// resolvido em runtime via `Color.egger(_:for:)`.
public enum EggerColorToken: String, CaseIterable, Sendable {
    case background
    case surface
    case surfaceRaised
    case border
    case textPrimary
    case textSecondary
    case textDisabled
    case accent
    case accentPressed
    case success
    case warning
    case critical
    case info
    case unavailable
}

public extension Color {
    static func egger(_ token: EggerColorToken, scheme: ColorScheme) -> Color {
        switch scheme {
        case .dark:
            switch token {
        case .background: return Color(hex: "#0E1116")
        case .surface: return Color(hex: "#161B22")
        case .surfaceRaised: return Color(hex: "#1E242D")
        case .border: return Color(hex: "#2A313C")
        case .textPrimary: return Color(hex: "#F2F4F7")
        case .textSecondary: return Color(hex: "#AEB4BF")
        case .textDisabled: return Color(hex: "#7B8394")
        case .accent: return Color(hex: "#4C9AFF")
        case .accentPressed: return Color(hex: "#7AB6FF")
        case .success: return Color(hex: "#3FCB78")
        case .warning: return Color(hex: "#E0A526")
        case .critical: return Color(hex: "#F26D6D")
        case .info: return Color(hex: "#4C9AFF")
        case .unavailable: return Color(hex: "#6B7280")
            }
        default:
            switch token {
        case .background: return Color(hex: "#F5F6F8")
        case .surface: return Color(hex: "#FFFFFF")
        case .surfaceRaised: return Color(hex: "#FFFFFF")
        case .border: return Color(hex: "#D8DBE0")
        case .textPrimary: return Color(hex: "#12151A")
        case .textSecondary: return Color(hex: "#4B5563")
        case .textDisabled: return Color(hex: "#6D7787")
        case .accent: return Color(hex: "#0A6CFF")
        case .accentPressed: return Color(hex: "#0854CC")
        case .success: return Color(hex: "#0F893E")
        case .warning: return Color(hex: "#A3690A")
        case .critical: return Color(hex: "#C22C2C")
        case .info: return Color(hex: "#0A6CFF")
        case .unavailable: return Color(hex: "#8A8F98")
            }
        }
    }

    init(hex: String) {
        var value: UInt64 = 0
        Scanner(string: hex.replacingOccurrences(of: "#", with: "")).scanHexInt64(&value)
        let r = Double((value >> 16) & 0xFF) / 255
        let g = Double((value >> 8) & 0xFF) / 255
        let b = Double(value & 0xFF) / 255
        self.init(.sRGB, red: r, green: g, blue: b, opacity: 1)
    }
}

/// Fonte de proveniência de uma métrica (Seção 2.1 do documento-fonte): toda
/// métrica exibida indica de onde veio, nunca é inventada.
public enum EggerMetricSource: String, CaseIterable, Sendable, Codable {
    case unifiLocalApi = "unifi_local_api"
    case unifiSiteManager = "unifi_site_manager"
    case agentIcmp = "agent_icmp"
    case agentTcp = "agent_tcp"
    case agentUdp = "agent_udp"
    case agentDns = "agent_dns"
    case agentHttp = "agent_http"
    case snmp = "snmp"
    case arkit = "arkit"
    case estimated = "estimated"
    case userDeclared = "user_declared"
    case unavailable = "unavailable"
}
