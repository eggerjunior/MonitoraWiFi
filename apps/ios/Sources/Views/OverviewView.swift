import SwiftUI

@MainActor
@Observable
final class OverviewViewModel {
    private(set) var organizations: [Organization] = []
    private(set) var sitesByOrganization: [String: [Site]] = [:]
    private(set) var isLoading = false
    private(set) var errorMessage: String?

    private let client: APIClient

    init(client: APIClient = APIClient()) {
        self.client = client
    }

    func load() async {
        isLoading = true
        errorMessage = nil
        do {
            let page = try await client.organizations()
            organizations = page.items
            for org in page.items {
                let sites = try await client.sites(organizationId: org.id)
                sitesByOrganization[org.id] = sites.items
            }
        } catch {
            errorMessage = "Não foi possível carregar organizações/sites. Verifique a conexão com o backend."
        }
        isLoading = false
    }
}

struct OverviewView: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @State private var viewModel = OverviewViewModel()
    @State private var showingSettings = false

    var body: some View {
        List {
            Section {
                LabeledContent("Suporte a LiDAR neste aparelho") {
                    Text(LiDARCapabilityChecker.isLiDARAvailable ? "Disponível" : "Indisponível")
                        .foregroundStyle(
                            LiDARCapabilityChecker.isLiDARAvailable
                                ? Color.egger(.success, scheme: colorScheme)
                                : Color.egger(.unavailable, scheme: colorScheme)
                        )
                }
            } footer: {
                Text("Detecção de hardware (ARKit) — o levantamento espacial guiado chega na Fase 6. Sem LiDAR, o app usa o fluxo manual de planta (Seção 6.4).")
            }

            if viewModel.isLoading {
                ProgressView("Carregando…")
            } else if let error = viewModel.errorMessage {
                Text(error)
                    .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
            } else if viewModel.organizations.isEmpty {
                Text("Nenhuma organização cadastrada ainda.")
                    .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
            } else {
                ForEach(viewModel.organizations) { org in
                    Section(org.name) {
                        let sites = viewModel.sitesByOrganization[org.id] ?? []
                        if sites.isEmpty {
                            Text("Nenhum site cadastrado.")
                                .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                        } else {
                            ForEach(sites) { site in
                                LabeledContent(site.name, value: site.timezone)
                            }
                        }
                    }
                }
            }

            Section {
                versionFooter
            }
        }
        .navigationTitle("Visão geral")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Configurações", systemImage: "gearshape") {
                    showingSettings = true
                }
            }
        }
        .sheet(isPresented: $showingSettings) {
            SettingsView()
        }
        .task {
            await viewModel.load()
        }
        .refreshable {
            await viewModel.load()
        }
    }

    @ViewBuilder
    private var versionFooter: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text("\(VersionManager.currentVersionString) — \(VersionManager.buildDateString)")
                .font(.caption)
                .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))

            if let url = VersionManager.commitURL {
                Link("commit \(VersionManager.gitCommit)", destination: url)
                    .font(.caption)
            } else {
                Text("commit \(VersionManager.gitCommit)")
                    .font(.caption)
                    .foregroundStyle(Color.egger(.textDisabled, scheme: colorScheme))
            }
        }
    }
}

#Preview {
    NavigationStack {
        OverviewView()
    }
    .environment(SessionStore())
}
