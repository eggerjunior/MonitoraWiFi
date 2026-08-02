import ARKit
import RealityKit
import SwiftUI
import UIKit

/// Ponte UIKit→SwiftUI pro `ARView` (RealityKit) — não existe equivalente
/// SwiftUI puro para uma sessão ARKit de mundo real com passthrough de
/// câmera em iPhone/iPad (diferente do RealityView do visionOS). Configura
/// `sceneReconstruction = .mesh` só quando o hardware suporta (LiDAR,
/// `LiDARCapabilityChecker.isLiDARAvailable`) — em hardware sem LiDAR, a
/// sessão ainda roda com tracking por feature points (fallback obrigatório
/// documentado em `docs/architecture/04-estrategia-lidar.md`).
struct SpatialSurveyARViewRepresentable: UIViewRepresentable {
    @Binding var arView: ARView?

    func makeUIView(context: Context) -> ARView {
        let view = ARView(frame: .zero)

        let configuration = ARWorldTrackingConfiguration()
        if LiDARCapabilityChecker.isLiDARAvailable {
            configuration.sceneReconstruction = .mesh
            // Visualização em tempo real da malha reconstruída — prova visual
            // ao usuário de que o LiDAR está capturando geometria de verdade,
            // sem precisar de renderização própria nesta primeira fatia.
            view.debugOptions.insert(.showSceneUnderstanding)
        }
        view.session.run(configuration)

        DispatchQueue.main.async {
            self.arView = view
        }

        return view
    }

    func updateUIView(_ uiView: ARView, context: Context) {}
}

/// Marcador esférico colorido pela qualidade do RTT medido naquele ponto —
/// nunca uma "visualização de ondas de rádio" (o overlay é sempre uma
/// representação direta do dado de rede medido, não uma simulação de
/// propagação de sinal, ver `04-estrategia-lidar.md`).
enum SpatialSampleMarker {
    static func addMarker(to arView: ARView, at transform: simd_float4x4, qualityLabel: String) {
        let color: UIColor
        switch qualityLabel {
        case "boa": color = .systemGreen
        case "média": color = .systemYellow
        case "ruim": color = .systemRed
        default: color = .systemGray
        }

        let mesh = MeshResource.generateSphere(radius: 0.05)
        let material = SimpleMaterial(color: color, isMetallic: false)
        let entity = ModelEntity(mesh: mesh, materials: [material])

        let anchor = AnchorEntity(world: transform)
        anchor.addChild(entity)
        arView.scene.addAnchor(anchor)
    }
}
