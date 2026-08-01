import ARKit

/// Detecção de suporte a LiDAR (Seção 6.2, passo 4: "Detectar suporte ao
/// LiDAR"). Esta é uma checagem de capacidade de hardware — não inicia sessão
/// de câmera nem pede qualquer permissão (ver docs/architecture/01-limitacoes-tecnicas.md
/// §1.6): só dispositivos Pro/Pro Max (iPhone) e iPad Pro (2020+) retornam
/// verdadeiro aqui.
public enum LiDARCapabilityChecker {
    public static var isLiDARAvailable: Bool {
        ARWorldTrackingConfiguration.supportsSceneReconstruction(.mesh)
    }
}
