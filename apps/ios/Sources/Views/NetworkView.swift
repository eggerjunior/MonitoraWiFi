import Observation
import SwiftUI

/// Dashboard "Rede" (Fase 3/4, início): inventário de dispositivos UniFi e
/// clientes conectados, sincronizado pelo agente local — nunca simulado.
/// Detalhe de rádio/porta ainda não confirmado contra a instalação real
/// (capability-matrix.md "a validar") — não exibido até ser validado.
/// Paridade com apps/web/src/app/(dashboard)/{devices,wifi,clients}/page.tsx.
@MainActor
@Observable
final class NetworkViewModel {
    private(set) var devices: [UniFiDevice] = []
    private(set) var clients: [UniFiClient] = []
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
            async let devicesResult = client.uniFiDevices(siteId: site.id)
            async let clientsResult = client.uniFiClients(siteId: site.id)
            devices = try await devicesResult.items
            clients = try await clientsResult.items
        } catch {
            errorMessage = "Não foi possível carregar o inventário UniFi. Verifique a conexão com o backend."
        }
        isLoading = false
    }

    var accessPoints: [UniFiDevice] {
        devices.filter { $0.features.contains("accessPoint") }
    }

    var switches: [UniFiDevice] {
        devices.filter { $0.features.contains("switching") }
    }

    func wirelessClientCount(forDeviceExternalID externalID: String) -> Int {
        clients.filter { $0.type == "WIRELESS" && $0.uplinkDeviceId == externalID }.count
    }

    func wiredClientCount(forDeviceExternalID externalID: String) -> Int {
        clients.filter { $0.type == "WIRED" && $0.uplinkDeviceId == externalID }.count
    }

    /// Nome do dispositivo upstream (topologia dispositivo→dispositivo,
    /// confirmado em 2026-08-02) — nil pro dispositivo raiz (gateway) ou
    /// se o uplink apontar pra um device que não está mais na lista atual.
    func uplinkName(for device: UniFiDevice) -> String? {
        guard !device.uplinkDeviceId.isEmpty else { return nil }
        return devices.first { $0.externalId == device.uplinkDeviceId }?.name
    }
}

struct NetworkView: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var viewModel: NetworkViewModel

    init(client: APIClient) {
        _viewModel = State(initialValue: NetworkViewModel(client: client))
    }

    var body: some View {
        List {
            if viewModel.isLoading {
                ProgressView("Carregando…")
            } else if let errorMessage = viewModel.errorMessage {
                Text(errorMessage)
                    .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
            } else {
                Section("Wi-Fi (\(viewModel.accessPoints.count))") {
                    if viewModel.accessPoints.isEmpty {
                        Text("Nenhum access point sincronizado ainda.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    } else {
                        ForEach(viewModel.accessPoints) { ap in
                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text(ap.name).font(.headline)
                                    Spacer()
                                    Text(ap.state)
                                        .font(.caption)
                                        .foregroundStyle(ap.state == "ONLINE" ? Color.egger(.success, scheme: colorScheme) : Color.egger(.textSecondary, scheme: colorScheme))
                                }
                                Text("\(ap.model) · \(viewModel.wirelessClientCount(forDeviceExternalID: ap.externalId)) clientes sem fio")
                                    .font(.caption)
                                    .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                            }
                        }
                    }
                }

                Section("Switches (\(viewModel.switches.count))") {
                    if viewModel.switches.isEmpty {
                        Text("Nenhum switch sincronizado ainda.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    } else {
                        ForEach(viewModel.switches) { sw in
                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text(sw.name).font(.headline)
                                    Spacer()
                                    Text(sw.state)
                                        .font(.caption)
                                        .foregroundStyle(sw.state == "ONLINE" ? Color.egger(.success, scheme: colorScheme) : Color.egger(.textSecondary, scheme: colorScheme))
                                }
                                Text("\(sw.model) · \(viewModel.wiredClientCount(forDeviceExternalID: sw.externalId)) clientes cabeados")
                                    .font(.caption)
                                    .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                            }
                        }
                    }
                }

                Section("Dispositivos (\(viewModel.devices.count))") {
                    if viewModel.devices.isEmpty {
                        Text("Nenhum dispositivo sincronizado ainda. Requer um agente local com a integração UniFi configurada.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    } else {
                        ForEach(viewModel.devices) { device in
                            VStack(alignment: .leading, spacing: 2) {
                                Text(device.name).font(.body)
                                Text("\(device.model) · \(device.firmwareVersion.isEmpty ? "firmware indisponível" : device.firmwareVersion)")
                                    .font(.caption)
                                    .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                                if let uplinkName = viewModel.uplinkName(for: device) {
                                    Text("Conectado a: \(uplinkName)")
                                        .font(.caption)
                                        .foregroundStyle(Color.egger(.textDisabled, scheme: colorScheme))
                                }
                            }
                        }
                    }
                }

                Section("Clientes (\(viewModel.clients.count))") {
                    if viewModel.clients.isEmpty {
                        Text("Nenhum cliente sincronizado ainda.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    } else {
                        ForEach(viewModel.clients) { c in
                            LabeledContent(c.name.isEmpty ? "Sem nome" : c.name, value: c.type)
                        }
                    }
                }

                Section {
                    Text("Canal, potência e utilização por rádio, e estatística por porta ainda não confirmados contra a instalação real — não exibidos até serem validados.")
                        .font(.caption)
                        .foregroundStyle(Color.egger(.textDisabled, scheme: colorScheme))
                }
            }
        }
        .navigationTitle("Rede")
        .task {
            await viewModel.load()
        }
        .refreshable {
            await viewModel.load()
        }
    }
}

#Preview {
    NavigationStack {
        NetworkView(client: APIClient())
    }
}
