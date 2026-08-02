import SwiftUI

/// Histórico (Fase 4): séries reais já coletadas — ping/speed test do
/// agente (Fase 2) e anomalias estatísticas (Fase 7) — sem introduzir
/// biblioteca de gráfico nova (nenhuma existe no projeto ainda); segue o
/// mesmo padrão de lista paginada usado no resto do produto. Paridade com
/// apps/web/src/app/(dashboard)/history/page.tsx.
@MainActor
@Observable
final class HistoryViewModel {
    private(set) var pingTests: [PingTestRecord] = []
    private(set) var speedTests: [SpeedTestRecord] = []
    private(set) var anomalies: [Anomaly] = []
    private(set) var isLoading = true
    private(set) var errorMessage: String?

    private let client: APIClient

    init(client: APIClient) {
        self.client = client
    }

    func load() async {
        isLoading = true
        errorMessage = nil
        do {
            let orgs = try await client.organizations(page: 1, pageSize: 1)
            guard let org = orgs.items.first else {
                errorMessage = "Nenhuma organização cadastrada ainda."
                isLoading = false
                return
            }
            let sites = try await client.sites(organizationId: org.id, page: 1, pageSize: 1)
            guard let site = sites.items.first else {
                errorMessage = "Nenhum site cadastrado nesta organização ainda."
                isLoading = false
                return
            }
            async let pingResult = client.pingTests(siteId: site.id, page: 1, pageSize: 20)
            async let speedResult = client.speedTests(siteId: site.id, page: 1, pageSize: 20)
            async let anomaliesResult = client.anomalies(siteId: site.id, page: 1, pageSize: 20)
            pingTests = try await pingResult.items
            speedTests = try await speedResult.items
            anomalies = try await anomaliesResult.items
        } catch {
            errorMessage = "Não foi possível carregar o histórico. Verifique a conexão com o backend."
        }
        isLoading = false
    }
}

struct HistoryView: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var viewModel: HistoryViewModel

    init(client: APIClient) {
        _viewModel = State(initialValue: HistoryViewModel(client: client))
    }

    var body: some View {
        List {
            if viewModel.isLoading {
                ProgressView("Carregando…")
            } else if let errorMessage = viewModel.errorMessage {
                Text(errorMessage)
                    .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
            } else {
                Section("Speed tests (\(viewModel.speedTests.count))") {
                    if viewModel.speedTests.isEmpty {
                        Text("Nenhum speed test registrado ainda.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    } else {
                        ForEach(viewModel.speedTests) { t in
                            VStack(alignment: .leading, spacing: 2) {
                                HStack {
                                    Text("\(formatMbps(t.downloadMbps)) ↓ / \(formatMbps(t.uploadMbps)) ↑")
                                        .font(.subheadline)
                                    Spacer()
                                    Text(t.mode.uppercased())
                                        .font(.caption)
                                        .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                                }
                                Text(t.executedAt)
                                    .font(.caption)
                                    .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                            }
                        }
                    }
                }

                Section("Testes de ping (\(viewModel.pingTests.count))") {
                    if viewModel.pingTests.isEmpty {
                        Text("Nenhum teste de ping registrado ainda.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    } else {
                        ForEach(viewModel.pingTests) { t in
                            VStack(alignment: .leading, spacing: 2) {
                                HStack {
                                    Text(t.target).font(.subheadline)
                                    Spacer()
                                    Text(formatMs(t.latencyMsP50))
                                        .font(.caption)
                                        .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                                }
                                Text(t.executedAt)
                                    .font(.caption)
                                    .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                            }
                        }
                    }
                }

                Section("Anomalias (\(viewModel.anomalies.count))") {
                    if viewModel.anomalies.isEmpty {
                        Text("Nenhuma anomalia detectada ainda.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    } else {
                        ForEach(viewModel.anomalies) { a in
                            VStack(alignment: .leading, spacing: 2) {
                                Text("\(a.metric): \(String(format: "%.1f", a.value)) (esperado \(String(format: "%.1f", a.bucketMean)))")
                                    .font(.subheadline)
                                Text(a.observedAt)
                                    .font(.caption)
                                    .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("Histórico")
        .task {
            await viewModel.load()
        }
        .refreshable {
            await viewModel.load()
        }
    }
}

private func formatMbps(_ value: Double?) -> String {
    guard let value else { return "Indisponível" }
    return String(format: "%.1f Mbps", value)
}

private func formatMs(_ value: Double?) -> String {
    guard let value else { return "Indisponível" }
    return String(format: "%.1f ms", value)
}

#Preview {
    NavigationStack {
        HistoryView(client: APIClient())
    }
}
