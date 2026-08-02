import ARKit
import CoreLocation
import Foundation
import Network
import NetworkExtension
import Observation
import RealityKit

/// Coordena a captura de um levantamento espacial (Fase 6): sessão ARKit
/// (via SpatialSurveyARViewRepresentable), amostras em memória durante a
/// caminhada, envio completo ao backend ao final. Nunca dispara um teste de
/// rede novo por amostra através do agente (saturaria a LAN — ver
/// `04-estrategia-lidar.md`); em vez disso mede RTT ao backend a partir do
/// próprio ponto, que é o dado que de fato varia com a posição do usuário
/// (o agente é estacionário, então seu ping não variaria por posição).
@MainActor
@Observable
final class SpatialSurveyViewModel: NSObject {
    private(set) var isCapturing = false
    private(set) var capturedCount = 0
    private(set) var lastCaptureLabel: String?
    private(set) var isSubmitting = false
    private(set) var submitError: String?
    private(set) var createdSurvey: SpatialSurvey?

    var surveyName = ""

    private var samples: [SpatialSurveySample] = []
    private var startedAt: Date?
    private let pathMonitor = NWPathMonitor()
    private var currentPath: NWPath?
    private let locationManager = CLLocationManager()
    private let client: APIClient

    init(client: APIClient) {
        self.client = client
        super.init()
        locationManager.delegate = self
    }

    var isLiDARAvailable: Bool {
        LiDARCapabilityChecker.isLiDARAvailable
    }

    func beginSession() {
        isCapturing = true
        capturedCount = 0
        samples = []
        startedAt = Date()
        locationManager.requestWhenInUseAuthorization()
        pathMonitor.pathUpdateHandler = { [weak self] path in
            Task { @MainActor in self?.currentPath = path }
        }
        pathMonitor.start(queue: .main)
    }

    /// Captura uma amostra na posição atual da câmera — chamado pelo botão
    /// "Capturar aqui", nunca automaticamente por tap num ponto arbitrário da
    /// tela (o usuário precisa estar fisicamente no local que quer medir).
    func captureSample(arView: ARView) async {
        guard let frame = arView.session.currentFrame else { return }
        let transform = frame.camera.transform
        let position = SpatialSurveyMath.position(from: transform)

        let rttMs = await client.measureRTTToBackend()
        let network = await Self.fetchCurrentNetwork()
        let path = currentPath
        let interfaceType = path.map(SpatialSurveyMath.interfaceType(from:)) ?? "other"

        let sample = SpatialSurveySample(
            positionX: Double(position.x),
            positionY: Double(position.y),
            positionZ: Double(position.z),
            ssid: network?.ssid,
            bssid: network?.bssid,
            rttMs: rttMs,
            isExpensive: path?.isExpensive ?? false,
            isConstrained: path?.isConstrained ?? false,
            interfaceType: interfaceType,
            capturedAt: ISO8601DateFormatter().string(from: Date())
        )
        samples.append(sample)
        capturedCount = samples.count
        lastCaptureLabel = SpatialSurveyMath.rttQualityLabel(rttMs: rttMs)

        SpatialSampleMarker.addMarker(to: arView, at: transform, qualityLabel: lastCaptureLabel ?? "falhou")
    }

    private static func fetchCurrentNetwork() async -> NEHotspotNetwork? {
        await withCheckedContinuation { continuation in
            NEHotspotNetwork.fetchCurrent { network in
                continuation.resume(returning: network)
            }
        }
    }

    func endSessionAndUpload(siteId: String) async {
        guard let startedAt, !samples.isEmpty else { return }
        isSubmitting = true
        submitError = nil

        let request = CreateSpatialSurveyRequest(
            name: surveyName.isEmpty ? "Levantamento \(ISO8601DateFormatter().string(from: Date()))" : surveyName,
            deviceModel: deviceModelIdentifier(),
            lidarUsed: isLiDARAvailable,
            startedAt: ISO8601DateFormatter().string(from: startedAt),
            finishedAt: ISO8601DateFormatter().string(from: Date()),
            samples: samples
        )

        do {
            createdSurvey = try await client.createSpatialSurvey(siteId: siteId, request: request)
            isCapturing = false
            pathMonitor.cancel()
        } catch let error as APIClient.ClientError {
            submitError = Self.message(for: error)
        } catch {
            submitError = "Erro de rede ao enviar o levantamento."
        }
        isSubmitting = false
    }

    private func deviceModelIdentifier() -> String {
        var systemInfo = utsname()
        uname(&systemInfo)
        let machineMirror = Mirror(reflecting: systemInfo.machine)
        return machineMirror.children.reduce(into: "") { identifier, element in
            guard let value = element.value as? Int8, value != 0 else { return }
            identifier += String(UnicodeScalar(UInt8(value)))
        }
    }

    private static func message(for error: APIClient.ClientError) -> String {
        switch error {
        case .server(let payload):
            return payload.message
        case .invalidResponse, .decoding:
            return "Não foi possível falar com o servidor."
        }
    }
}

extension SpatialSurveyViewModel: CLLocationManagerDelegate {}
