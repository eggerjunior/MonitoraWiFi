import Network
import simd

/// Funções puras usadas pelo levantamento espacial (Fase 6) — extraídas de
/// `SpatialSurveyView`/`SpatialSurveyViewModel` justamente para serem
/// testáveis sem depender de uma sessão ARKit/Network real (que não roda em
/// CI headless nem neste ambiente de desenvolvimento — ver apps/ios/README.md).
public enum SpatialSurveyMath {
    /// Extrai a posição (x, y, z) da coluna de translação de uma pose ARKit
    /// (`ARAnchor.transform`/`ARCamera.transform`) — relativa à origem da
    /// sessão, nunca georreferenciada.
    public static func position(from transform: simd_float4x4) -> SIMD3<Float> {
        SIMD3(transform.columns.3.x, transform.columns.3.y, transform.columns.3.z)
    }

    /// Rótulo de qualidade a partir do RTT medido no próprio ponto — nunca a
    /// partir de RSSI (indisponível via API pública do iOS, ver capability
    /// matrix). Faixas escolhidas para RTT ao próprio backend (não ao
    /// gateway local), por isso mais generosas que um ping LAN puro.
    public static func rttQualityLabel(rttMs: Double?) -> String {
        guard let rttMs else { return "falhou" }
        switch rttMs {
        case ..<50: return "boa"
        case 50..<150: return "média"
        default: return "ruim"
        }
    }

    /// Mapeia o tipo de interface do `NWPathMonitor` pro vocabulário fixo
    /// aceito pelo backend (`interface_type`: wifi | cellular | wired |
    /// other) — nunca envia um valor fora desse conjunto.
    public static func interfaceType(from path: NWPath) -> String {
        if path.usesInterfaceType(.wifi) { return "wifi" }
        if path.usesInterfaceType(.cellular) { return "cellular" }
        if path.usesInterfaceType(.wiredEthernet) { return "wired" }
        return "other"
    }
}
