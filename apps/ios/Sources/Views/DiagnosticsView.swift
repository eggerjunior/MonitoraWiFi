import Observation
import SwiftUI

/// Primeira ferramenta real da Fase 5 (diagnósticos sob demanda): dispara um
/// comando de ping executado de verdade pelo agente do site — nunca simula
/// um resultado antes de o agente responder (Seção 2.1). Paridade com a
/// mesma ferramenta no web (apps/web/src/components/PingTool.tsx).
@MainActor
@Observable
final class DiagnosticsViewModel {
    private(set) var siteId: String?
    private(set) var isLoadingSite = true
    private(set) var loadError: String?

    var target = "1.1.1.1"
    var protocolName = "icmp"
    private(set) var command: Command?
    private(set) var isSubmitting = false
    private(set) var submitError: String?

    private let client: APIClient
    private var pollTask: Task<Void, Never>?

    init(client: APIClient) {
        self.client = client
    }

    func loadSite() async {
        isLoadingSite = true
        loadError = nil
        do {
            let orgs = try await client.organizations(page: 1, pageSize: 1)
            guard let org = orgs.items.first else {
                loadError = "Nenhuma organização cadastrada ainda."
                isLoadingSite = false
                return
            }
            let sites = try await client.sites(organizationId: org.id, page: 1, pageSize: 1)
            guard let site = sites.items.first else {
                loadError = "Nenhum site cadastrado nesta organização ainda."
                isLoadingSite = false
                return
            }
            siteId = site.id
        } catch {
            loadError = "Não foi possível carregar o site. Verifique a conexão com o backend."
        }
        isLoadingSite = false
    }

    func runPing() async {
        guard let siteId else { return }
        pollTask?.cancel()
        submitError = nil
        command = nil
        isSubmitting = true
        do {
            let created = try await client.createPingCommand(siteId: siteId, target: target, protocolName: protocolName)
            command = created
            startPolling(commandId: created.id)
        } catch let error as APIClient.ClientError {
            submitError = Self.message(for: error)
        } catch {
            submitError = "Erro de rede ao criar o comando."
        }
        isSubmitting = false
    }

    private func startPolling(commandId: String) {
        pollTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(1))
                guard let self, !Task.isCancelled else { return }
                guard let updated = try? await self.client.getCommand(id: commandId) else { continue }
                self.command = updated
                if updated.status != "pending" && updated.status != "claimed" {
                    return
                }
            }
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

struct DiagnosticsView: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var viewModel: DiagnosticsViewModel

    init(client: APIClient) {
        _viewModel = State(initialValue: DiagnosticsViewModel(client: client))
    }

    private static let protocols = ["icmp", "tcp", "http", "dns"]

    var body: some View {
        List {
            if viewModel.isLoadingSite {
                ProgressView("Carregando…")
            } else if let loadError = viewModel.loadError {
                Text(loadError)
                    .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
            } else {
                Section("Ping sob demanda") {
                    TextField("Alvo (ex.: 1.1.1.1)", text: $viewModel.target)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()

                    Picker("Protocolo", selection: $viewModel.protocolName) {
                        ForEach(Self.protocols, id: \.self) { p in
                            Text(p.uppercased()).tag(p)
                        }
                    }

                    Button {
                        Task { await viewModel.runPing() }
                    } label: {
                        if viewModel.isSubmitting {
                            ProgressView()
                        } else {
                            Text("Executar")
                        }
                    }
                    .disabled(viewModel.isSubmitting || viewModel.target.isEmpty)
                }

                if let submitError = viewModel.submitError {
                    Section {
                        Text(submitError)
                            .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                    }
                }

                if let command = viewModel.command {
                    Section("Resultado") {
                        LabeledContent("Status", value: statusLabel(command.status))
                        if command.status == "completed", let result = command.result {
                            LabeledContent("p50", value: formatMs(result.latencyMsP50))
                            LabeledContent("Perda", value: formatPct(result.packetLossPct))
                            LabeledContent("Jitter", value: formatMs(result.jitterMs))
                        }
                        if command.status == "failed" {
                            Text(command.error ?? "Falha não especificada.")
                                .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                        }
                    }
                }
            }
        }
        .navigationTitle("Ferramentas")
        .task {
            await viewModel.loadSite()
        }
    }

    private func statusLabel(_ status: String) -> String {
        switch status {
        case "pending": return "Aguardando agente…"
        case "claimed": return "Executando…"
        case "completed": return "Concluído"
        case "failed": return "Falhou"
        default: return status
        }
    }

    private func formatMs(_ value: Double?) -> String {
        guard let value else { return "Indisponível" }
        return String(format: "%.1f ms", value)
    }

    private func formatPct(_ value: Double?) -> String {
        guard let value else { return "Indisponível" }
        return String(format: "%.1f%%", value)
    }
}

#Preview {
    NavigationStack {
        DiagnosticsView(client: APIClient())
    }
}
