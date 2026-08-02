import Observation
import RealityKit
import SwiftUI

/// Tela de topo do levantamento espacial (Fase 6) — ocupa o slot ".map" já
/// reservado em RootView desde a Fase 1. Mostra levantamentos já enviados
/// (metadados, sem replay 3D nesta primeira fatia) e permite iniciar um novo.
@MainActor
@Observable
final class SpatialSurveyListViewModel {
    private(set) var siteId: String?
    private(set) var isLoading = true
    private(set) var loadError: String?
    private(set) var surveys: [SpatialSurvey] = []

    private let client: APIClient
    init(client: APIClient) { self.client = client }

    func load() async {
        isLoading = true
        loadError = nil
        do {
            let orgs = try await client.organizations(page: 1, pageSize: 1)
            guard let org = orgs.items.first else {
                loadError = "Nenhuma organização cadastrada ainda."
                isLoading = false
                return
            }
            let sites = try await client.sites(organizationId: org.id, page: 1, pageSize: 1)
            guard let site = sites.items.first else {
                loadError = "Nenhum site cadastrado nesta organização ainda."
                isLoading = false
                return
            }
            siteId = site.id
            let page = try await client.spatialSurveys(siteId: site.id, page: 1, pageSize: 50)
            surveys = page.items
        } catch {
            loadError = "Não foi possível carregar os levantamentos."
        }
        isLoading = false
    }
}

struct SpatialSurveyView: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var viewModel: SpatialSurveyListViewModel
    private let client: APIClient
    @State private var showingCapture = false

    init(client: APIClient) {
        self.client = client
        _viewModel = State(initialValue: SpatialSurveyListViewModel(client: client))
    }

    var body: some View {
        List {
            if viewModel.isLoading {
                ProgressView("Carregando…")
            } else if let loadError = viewModel.loadError {
                Text(loadError)
                    .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
            } else {
                Section {
                    LabeledContent("LiDAR neste aparelho") {
                        Text(LiDARCapabilityChecker.isLiDARAvailable ? "Disponível" : "Indisponível")
                    }
                    Button {
                        showingCapture = true
                    } label: {
                        Text("Novo levantamento")
                    }
                }

                Section("Levantamentos") {
                    if viewModel.surveys.isEmpty {
                        Text("Nenhum levantamento ainda.")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(viewModel.surveys) { survey in
                            VStack(alignment: .leading, spacing: 2) {
                                Text(survey.name).fontWeight(.medium)
                                Text("\(survey.sampleCount) amostra(s) · \(survey.lidarUsed ? "LiDAR" : "sem LiDAR") · \(survey.deviceModel)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("Levantamento espacial")
        .task { await viewModel.load() }
        .sheet(isPresented: $showingCapture, onDismiss: {
            Task { await viewModel.load() }
        }) {
            if let siteId = viewModel.siteId {
                NavigationStack {
                    SpatialSurveyCaptureView(client: client, siteId: siteId)
                }
            }
        }
    }
}

struct SpatialSurveyCaptureView: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme
    @State private var viewModel: SpatialSurveyViewModel
    @State private var arViewBox = ARViewBox()
    let siteId: String

    init(client: APIClient, siteId: String) {
        self.siteId = siteId
        _viewModel = State(initialValue: SpatialSurveyViewModel(client: client))
    }

    var body: some View {
        ZStack(alignment: .bottom) {
            if viewModel.isCapturing {
                SpatialSurveyARViewRepresentable(arView: Binding(
                    get: { arViewBox.value },
                    set: { arViewBox.value = $0 }
                ))
                .ignoresSafeArea()

                VStack(spacing: 12) {
                    if let label = viewModel.lastCaptureLabel {
                        Text("Última amostra: \(label) (\(viewModel.capturedCount) no total)")
                            .font(.caption)
                            .padding(6)
                            .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 8))
                    }
                    HStack {
                        Button {
                            guard let av = arViewBox.value else { return }
                            Task { await viewModel.captureSample(arView: av) }
                        } label: {
                            Text("Capturar aqui")
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.borderedProminent)

                        Button {
                            Task {
                                await viewModel.endSessionAndUpload(siteId: siteId)
                                if viewModel.createdSurvey != nil { dismiss() }
                            }
                        } label: {
                            Text("Finalizar")
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.bordered)
                        .disabled(viewModel.capturedCount == 0 || viewModel.isSubmitting)
                    }
                    if let error = viewModel.submitError {
                        Text(error).font(.caption).foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                    }
                }
                .padding()
            } else {
                Form {
                    Section("Nome do levantamento") {
                        TextField("ex.: Térreo", text: $viewModel.surveyName)
                    }
                    Section {
                        Text(viewModel.isLiDARAvailable
                             ? "Este aparelho tem LiDAR — a malha do ambiente será reconstruída em tempo real durante a captura."
                             : "Este aparelho não tem LiDAR — o tracking usa pontos de referência (sem reconstrução de malha).")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    Button {
                        viewModel.beginSession()
                    } label: {
                        Text("Iniciar captura")
                    }
                }
            }
        }
        .navigationTitle("Novo levantamento")
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button("Cancelar") { dismiss() }
            }
        }
    }
}

/// `ARView?` precisa de um wrapper de referência pra ser usado como
/// `@State` + `Binding` sem reconstruir a view a cada frame da sessão.
@Observable
final class ARViewBox {
    var value: ARView?
}
