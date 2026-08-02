import simd
import Testing
@testable import EggerNetworkIntelligence

@Suite("SpatialSurveyMath")
struct SpatialSurveyMathTests {
    @Test("extrai a posição correta da coluna de translação de uma pose ARKit")
    func positionFromTransform() {
        var transform = matrix_identity_float4x4
        transform.columns.3 = SIMD4<Float>(1.5, 0.2, -3.7, 1)
        let position = SpatialSurveyMath.position(from: transform)
        #expect(position.x == 1.5)
        #expect(position.y == 0.2)
        #expect(position.z == -3.7)
    }

    @Test("rótulo de qualidade nunca inventa dado quando o RTT falhou (nil)")
    func rttQualityLabelNilIsFailed() {
        #expect(SpatialSurveyMath.rttQualityLabel(rttMs: nil) == "falhou")
    }

    @Test("rótulo de qualidade classifica as faixas corretamente", arguments: [
        (10.0, "boa"),
        (49.9, "boa"),
        (50.0, "média"),
        (149.9, "média"),
        (150.0, "ruim"),
        (500.0, "ruim"),
    ])
    func rttQualityLabelBuckets(rttMs: Double, expected: String) {
        #expect(SpatialSurveyMath.rttQualityLabel(rttMs: rttMs) == expected)
    }
}
