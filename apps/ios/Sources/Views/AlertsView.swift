import SwiftUI

/// Alertas (Fase 4): única fonte real hoje é o worker de anomalias
/// estatísticas (Fase 7, `GET /sites/{id}/anomalies`) — nunca reportado sem
/// histórico suficiente. Não existe schema de alerta com
/// severidade/status próprio ainda (ver docs/architecture/05-modelo-dados.md,
/// entidade ALERT — não implementada); severidade aqui é derivada do
/// z-score na própria UI. Paridade com
/// apps/web/src/app/(dashboard)/alerts/page.tsx.
private let criticalZScore = 5.0

@MainActor
@Observable
final class AlertsViewModel {
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
            let page = try await client.anomalies(siteId: site.id, page: 1, pageSize: 50)
            anomalies = page.items
        } catch {
            errorMessage = "Não foi possível carregar as anomalias. Verifique a conexão com o backend."
        }
        isLoading = false
    }
}

struct AlertsView: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var viewModel: AlertsViewModel

    init(client: APIClient) {
        _viewModel = State(initialValue: AlertsViewModel(client: client))
    }

    var body: some View {
        List {
            if viewModel.isLoading {
                ProgressView("Carregando…")
            } else if let errorMessage = viewModel.errorMessage {
                Text(errorMessage)
                    .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
            } else if viewModel.anomalies.isEmpty {
                Section {
                    Text("Nenhuma anomalia detectada ainda — pode ser que ainda não haja histórico suficiente, ou que a rede esteja dentro do padrão.")
                        .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                }
            } else {
                ForEach(viewModel.anomalies) { anomaly in
                    Section {
                        AnomalyRow(anomaly: anomaly, colorScheme: colorScheme)
                    }
                }
            }

            Section {
                Text("Hoje só a latência de ping é monitorada (ping_latency_ms_p50) — cobertura de speed test ainda não implementada.")
                    .font(.caption)
                    .foregroundStyle(Color.egger(.textDisabled, scheme: colorScheme))
            }
        }
        .navigationTitle("Alertas")
        .task {
            await viewModel.load()
        }
        .refreshable {
            await viewModel.load()
        }
    }
}

private struct AnomalyRow: View {
    let anomaly: Anomaly
    let colorScheme: ColorScheme

    private var isCritical: Bool { abs(anomaly.zScore) >= criticalZScore }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(metricLabel(anomaly.metric)).font(.headline)
                Spacer()
                Text(isCritical ? "Crítico" : "Atenção")
                    .font(.caption)
                    .foregroundStyle(Color.egger(isCritical ? .critical : .warning, scheme: colorScheme))
            }
            LabeledContent("Valor observado", value: String(format: "%.1f", anomaly.value))
            LabeledContent("Média esperada", value: String(format: "%.1f", anomaly.bucketMean))
            LabeledContent("Z-score", value: String(format: "%.2f", anomaly.zScore))
            Text("Observado em \(anomaly.observedAt)")
                .font(.caption)
                .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
        }
    }

    private func metricLabel(_ metric: String) -> String {
        metric == "ping_latency_ms_p50" ? "Latência de ping (p50)" : metric
    }
}

#Preview {
    NavigationStack {
        AlertsView(client: APIClient())
    }
}
