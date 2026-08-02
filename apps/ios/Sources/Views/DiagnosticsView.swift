import Observation
import SwiftUI

/// Ferramentas de diagnóstico sob demanda (Fase 5): ping, DNS lookup e
/// traceroute disparam um comando real executado pelo agente do site —
/// nunca simulam um resultado antes de o agente responder (Seção 2.1). A
/// calculadora de sub-rede é cálculo puro (sem agente/rede envolvida).
/// Paridade com apps/web/src/components/{Ping,DnsLookup,Traceroute}Tool.tsx.
@MainActor
@Observable
final class DiagnosticsViewModel {
    private(set) var siteId: String?
    private(set) var isLoadingSite = true
    private(set) var loadError: String?

    var target = "1.1.1.1"
    var protocolName = "icmp"
    var batchTargetsRaw = "1.1.1.1\n8.8.8.8"
    var batchProtocolName = "icmp"
    var hostname = "example.com"
    var tracerouteTarget = "1.1.1.1"
    var sslCheckTarget = "example.com"
    var sslCheckPort = "443"
    var rdapQuery = "example.com"
    var httpRequestURL = "http://localhost"
    var httpRequestMethod = "GET"
    var httpRequestBody = ""
    var lanScanCIDR = "192.168.1.0/24"
    var wolMACAddress = ""
    var wolBroadcastIP = "255.255.255.255"
    var portScanTarget = "192.168.1.1"
    var portScanStart = "1"
    var portScanEnd = "1024"

    private(set) var pingCommand: Command?
    private(set) var batchPingCommand: Command?
    private(set) var dnsCommand: Command?
    private(set) var tracerouteCommand: Command?
    private(set) var sslCheckCommand: Command?
    private(set) var httpRequestCommand: Command?
    private(set) var lanScanCommand: Command?
    private(set) var wolCommand: Command?
    private(set) var portScanCommand: Command?
    private(set) var rdapResult: RdapResult?
    private(set) var rdapError: String?
    private(set) var isRdapLoading = false

    static let maxBatchTargets = 20

    var batchTargets: [String] {
        batchTargetsRaw
            .split(separator: "\n")
            .map { $0.trimmingCharacters(in: .whitespaces) }
            .filter { !$0.isEmpty }
    }
    private(set) var isSubmitting = false
    private(set) var submitError: String?

    private let client: APIClient
    private var pollTasks: [String: Task<Void, Never>] = [:]

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
        submitError = nil
        isSubmitting = true
        do {
            let created = try await client.createPingCommand(siteId: siteId, target: target, protocolName: protocolName)
            pingCommand = created
            startPolling(commandId: created.id) { [weak self] updated in self?.pingCommand = updated }
        } catch let error as APIClient.ClientError {
            submitError = Self.message(for: error)
        } catch {
            submitError = "Erro de rede ao criar o comando."
        }
        isSubmitting = false
    }

    func runBatchPing() async {
        guard let siteId else { return }
        submitError = nil
        isSubmitting = true
        do {
            let created = try await client.createBatchPingCommand(siteId: siteId, targets: batchTargets, protocolName: batchProtocolName)
            batchPingCommand = created
            startPolling(commandId: created.id) { [weak self] updated in self?.batchPingCommand = updated }
        } catch let error as APIClient.ClientError {
            submitError = Self.message(for: error)
        } catch {
            submitError = "Erro de rede ao criar o comando."
        }
        isSubmitting = false
    }

    func runDNSLookup() async {
        guard let siteId else { return }
        submitError = nil
        isSubmitting = true
        do {
            let created = try await client.createDNSLookupCommand(siteId: siteId, hostname: hostname)
            dnsCommand = created
            startPolling(commandId: created.id) { [weak self] updated in self?.dnsCommand = updated }
        } catch let error as APIClient.ClientError {
            submitError = Self.message(for: error)
        } catch {
            submitError = "Erro de rede ao criar o comando."
        }
        isSubmitting = false
    }

    func runTraceroute() async {
        guard let siteId else { return }
        submitError = nil
        isSubmitting = true
        do {
            let created = try await client.createTracerouteCommand(siteId: siteId, target: tracerouteTarget)
            tracerouteCommand = created
            startPolling(commandId: created.id) { [weak self] updated in self?.tracerouteCommand = updated }
        } catch let error as APIClient.ClientError {
            submitError = Self.message(for: error)
        } catch {
            submitError = "Erro de rede ao criar o comando."
        }
        isSubmitting = false
    }

    func runSslCheck() async {
        guard let siteId else { return }
        let port = Int(sslCheckPort) ?? 443
        submitError = nil
        isSubmitting = true
        do {
            let created = try await client.createSslCheckCommand(siteId: siteId, target: sslCheckTarget, port: port)
            sslCheckCommand = created
            startPolling(commandId: created.id) { [weak self] updated in self?.sslCheckCommand = updated }
        } catch let error as APIClient.ClientError {
            submitError = Self.message(for: error)
        } catch {
            submitError = "Erro de rede ao criar o comando."
        }
        isSubmitting = false
    }

    func runHTTPRequest() async {
        guard let siteId else { return }
        submitError = nil
        isSubmitting = true
        do {
            let created = try await client.createHTTPRequestCommand(siteId: siteId, url: httpRequestURL, method: httpRequestMethod, body: httpRequestBody.isEmpty ? nil : httpRequestBody)
            httpRequestCommand = created
            startPolling(commandId: created.id) { [weak self] updated in self?.httpRequestCommand = updated }
        } catch let error as APIClient.ClientError {
            submitError = Self.message(for: error)
        } catch {
            submitError = "Erro de rede ao criar o comando."
        }
        isSubmitting = false
    }

    func runLANScan() async {
        guard let siteId else { return }
        submitError = nil
        isSubmitting = true
        do {
            let created = try await client.createLANScanCommand(siteId: siteId, cidr: lanScanCIDR)
            lanScanCommand = created
            startPolling(commandId: created.id) { [weak self] updated in self?.lanScanCommand = updated }
        } catch let error as APIClient.ClientError {
            submitError = Self.message(for: error)
        } catch {
            submitError = "Erro de rede ao criar o comando."
        }
        isSubmitting = false
    }

    func runWakeOnLAN() async {
        guard let siteId else { return }
        submitError = nil
        isSubmitting = true
        do {
            let created = try await client.createWakeOnLANCommand(siteId: siteId, macAddress: wolMACAddress, broadcastIP: wolBroadcastIP)
            wolCommand = created
            startPolling(commandId: created.id) { [weak self] updated in self?.wolCommand = updated }
        } catch let error as APIClient.ClientError {
            submitError = Self.message(for: error)
        } catch {
            submitError = "Erro de rede ao criar o comando."
        }
        isSubmitting = false
    }

    func runPortScan() async {
        guard let siteId else { return }
        let start = Int(portScanStart) ?? 1
        let end = Int(portScanEnd) ?? 1024
        submitError = nil
        isSubmitting = true
        do {
            let created = try await client.createPortScanCommand(siteId: siteId, target: portScanTarget, startPort: start, endPort: end)
            portScanCommand = created
            startPolling(commandId: created.id) { [weak self] updated in self?.portScanCommand = updated }
        } catch let error as APIClient.ClientError {
            submitError = Self.message(for: error)
        } catch {
            submitError = "Erro de rede ao criar o comando."
        }
        isSubmitting = false
    }

    func runRdapLookup() async {
        rdapError = nil
        rdapResult = nil
        isRdapLoading = true
        do {
            rdapResult = try await client.rdapLookup(query: rdapQuery)
        } catch let error as APIClient.ClientError {
            rdapError = Self.message(for: error)
        } catch {
            rdapError = "Erro de rede ao consultar RDAP."
        }
        isRdapLoading = false
    }

    private func startPolling(commandId: String, onUpdate: @escaping (Command) -> Void) {
        pollTasks[commandId]?.cancel()
        pollTasks[commandId] = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(1))
                guard let self, !Task.isCancelled else { return }
                guard let updated = try? await self.client.getCommand(id: commandId) else { continue }
                onUpdate(updated)
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
                pingSection
                batchPingSection
                dnsLookupSection
                tracerouteSection
                sslCheckSection
                httpRequestSection
                lanScanSection
                wakeOnLanSection
                portScanSection
            }

            if let submitError = viewModel.submitError {
                Section {
                    Text(submitError)
                        .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                }
            }

            rdapSection

            SubnetCalculatorSection()
        }
        .navigationTitle("Ferramentas")
        .task {
            await viewModel.loadSite()
        }
    }

    @ViewBuilder
    private var pingSection: some View {
        Section("Ping sob demanda") {
            TextField("Alvo (ex.: 1.1.1.1)", text: $viewModel.target)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            Picker("Protocolo", selection: $viewModel.protocolName) {
                ForEach(Self.protocols, id: \.self) { p in Text(p.uppercased()).tag(p) }
            }
            Button {
                Task { await viewModel.runPing() }
            } label: {
                Text("Executar")
            }
            .disabled(viewModel.isSubmitting || viewModel.target.isEmpty)

            if let command = viewModel.pingCommand {
                StatusRow(status: command.status, colorScheme: colorScheme)
                if command.status == "completed", case .ping(let result) = command.result {
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

    @ViewBuilder
    private var batchPingSection: some View {
        Section("Ping em lote") {
            TextField("Alvos, um por linha", text: $viewModel.batchTargetsRaw, axis: .vertical)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .lineLimit(3...6)
            Picker("Protocolo", selection: $viewModel.batchProtocolName) {
                ForEach(Self.protocols, id: \.self) { p in Text(p.uppercased()).tag(p) }
            }
            if viewModel.batchTargets.count > DiagnosticsViewModel.maxBatchTargets {
                Text("Máximo de \(DiagnosticsViewModel.maxBatchTargets) alvos por execução.")
                    .font(.caption)
                    .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
            }
            Button {
                Task { await viewModel.runBatchPing() }
            } label: {
                Text("Executar")
            }
            .disabled(viewModel.isSubmitting || viewModel.batchTargets.isEmpty || viewModel.batchTargets.count > DiagnosticsViewModel.maxBatchTargets)

            if let command = viewModel.batchPingCommand {
                StatusRow(status: command.status, colorScheme: colorScheme)
                if command.status == "completed", case .batchPing(let result) = command.result {
                    ForEach(result.results, id: \.target) { r in
                        HStack {
                            Text(r.target)
                            Spacer()
                            Text(formatMs(r.latencyMsP50))
                            Spacer()
                            Text(formatPct(r.packetLossPct))
                        }
                        .font(.system(.caption, design: .monospaced))
                    }
                }
                if command.status == "failed" {
                    Text(command.error ?? "Falha não especificada.")
                        .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                }
            }
        }
    }

    @ViewBuilder
    private var dnsLookupSection: some View {
        Section("DNS lookup sob demanda") {
            TextField("Hostname (ex.: example.com)", text: $viewModel.hostname)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            Button {
                Task { await viewModel.runDNSLookup() }
            } label: {
                Text("Executar")
            }
            .disabled(viewModel.isSubmitting || viewModel.hostname.isEmpty)

            if let command = viewModel.dnsCommand {
                StatusRow(status: command.status, colorScheme: colorScheme)
                if command.status == "completed", case .dnsLookup(let result) = command.result {
                    ForEach(result.addresses, id: \.self) { addr in
                        Text(addr).font(.system(.body, design: .monospaced))
                    }
                }
                if command.status == "failed" {
                    Text(command.error ?? "Falha não especificada.")
                        .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                }
            }
        }
    }

    @ViewBuilder
    private var tracerouteSection: some View {
        Section("Traceroute sob demanda") {
            TextField("Alvo (ex.: 1.1.1.1)", text: $viewModel.tracerouteTarget)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            Button {
                Task { await viewModel.runTraceroute() }
            } label: {
                Text("Executar")
            }
            .disabled(viewModel.isSubmitting || viewModel.tracerouteTarget.isEmpty)

            if let command = viewModel.tracerouteCommand {
                StatusRow(status: command.status, colorScheme: colorScheme)
                if command.status == "completed", case .traceroute(let result) = command.result {
                    Text(result.reached ? "Destino alcançado" : "Destino não alcançado dentro do limite de saltos")
                        .font(.caption)
                        .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    ForEach(result.hops) { hop in
                        HStack {
                            Text("\(hop.hop).")
                            Spacer()
                            Text(hop.address.isEmpty ? "* sem resposta *" : hop.address)
                            Spacer()
                            Text(hop.rttMs.map { String(format: "%.1f ms", $0) } ?? "—")
                        }
                        .font(.system(.caption, design: .monospaced))
                    }
                }
                if command.status == "failed" {
                    Text(command.error ?? "Falha não especificada.")
                        .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                }
            }
        }
    }

    @ViewBuilder
    private var sslCheckSection: some View {
        Section("SSL/TLS checker") {
            TextField("Host (ex.: example.com)", text: $viewModel.sslCheckTarget)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            TextField("Porta", text: $viewModel.sslCheckPort)
                .keyboardType(.numberPad)
            Button {
                Task { await viewModel.runSslCheck() }
            } label: {
                Text("Executar")
            }
            .disabled(viewModel.isSubmitting || viewModel.sslCheckTarget.isEmpty)

            if let command = viewModel.sslCheckCommand {
                StatusRow(status: command.status, colorScheme: colorScheme)
                if command.status == "completed", case .sslCheck(let result) = command.result {
                    Text(result.validNow ? "Cadeia de certificado válida" : "Cadeia inválida: \(result.verifyError)")
                        .foregroundStyle(Color.egger(result.validNow ? .success : .critical, scheme: colorScheme))
                    LabeledContent("Emissor", value: result.issuer)
                    LabeledContent("Assunto", value: result.subject)
                    LabeledContent("Expira em", value: "\(result.daysUntilExpiry) dia(s)")
                }
                if command.status == "failed" {
                    Text(command.error ?? "Falha não especificada.")
                        .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                }
            }
        }
    }

    @ViewBuilder
    private var httpRequestSection: some View {
        Section("HTTP client sob demanda") {
            Picker("Método", selection: $viewModel.httpRequestMethod) {
                ForEach(["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"], id: \.self) { m in
                    Text(m).tag(m)
                }
            }
            TextField("URL", text: $viewModel.httpRequestURL)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            Button {
                Task { await viewModel.runHTTPRequest() }
            } label: {
                Text("Executar")
            }
            .disabled(viewModel.isSubmitting || viewModel.httpRequestURL.isEmpty)

            if let command = viewModel.httpRequestCommand {
                StatusRow(status: command.status, colorScheme: colorScheme)
                if command.status == "completed", case .httpRequest(let result) = command.result {
                    LabeledContent("Status", value: "\(result.statusCode) \(result.statusText)")
                    LabeledContent("Tempo", value: String(format: "%.1f ms", result.durationMs))
                    Text(result.bodySnippet)
                        .font(.system(.caption, design: .monospaced))
                        .lineLimit(6)
                }
                if command.status == "failed" {
                    Text(command.error ?? "Falha não especificada.")
                        .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                }
            }
        }
    }

    @ViewBuilder
    private var lanScanSection: some View {
        Section("LAN scanner") {
            Text("Bloco CIDR privado (RFC 1918), no máximo /22.")
                .font(.caption)
                .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
            TextField("CIDR (ex.: 192.168.1.0/24)", text: $viewModel.lanScanCIDR)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            Button {
                Task { await viewModel.runLANScan() }
            } label: {
                Text("Executar")
            }
            .disabled(viewModel.isSubmitting || viewModel.lanScanCIDR.isEmpty)

            if let command = viewModel.lanScanCommand {
                StatusRow(status: command.status, colorScheme: colorScheme)
                if command.status == "completed", case .lanScan(let result) = command.result {
                    if result.hosts.isEmpty {
                        Text("Nenhum host respondeu.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    } else {
                        ForEach(result.hosts, id: \.self) { host in
                            Text(host).font(.system(.caption, design: .monospaced))
                        }
                    }
                }
                if command.status == "failed" {
                    Text(command.error ?? "Falha não especificada.")
                        .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                }
            }
        }
    }

    @ViewBuilder
    private var wakeOnLanSection: some View {
        Section("Wake-on-LAN") {
            TextField("Endereço MAC (ex.: aa:bb:cc:dd:ee:ff)", text: $viewModel.wolMACAddress)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            TextField("Broadcast IP", text: $viewModel.wolBroadcastIP)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            Button {
                Task { await viewModel.runWakeOnLAN() }
            } label: {
                Text("Ligar")
            }
            .disabled(viewModel.isSubmitting || viewModel.wolMACAddress.isEmpty)

            if let command = viewModel.wolCommand {
                StatusRow(status: command.status, colorScheme: colorScheme)
                if command.status == "completed" {
                    Text("Magic packet enviado.")
                        .foregroundStyle(Color.egger(.success, scheme: colorScheme))
                }
                if command.status == "failed" {
                    Text(command.error ?? "Falha não especificada.")
                        .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                }
            }
        }
    }

    @ViewBuilder
    private var portScanSection: some View {
        Section("Port scanner") {
            Text("Alvo precisa ser um IP privado (RFC 1918), no máximo 1024 portas.")
                .font(.caption)
                .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
            TextField("Alvo (IP)", text: $viewModel.portScanTarget)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            TextField("Porta inicial", text: $viewModel.portScanStart)
                .keyboardType(.numberPad)
            TextField("Porta final", text: $viewModel.portScanEnd)
                .keyboardType(.numberPad)
            Button {
                Task { await viewModel.runPortScan() }
            } label: {
                Text("Executar")
            }
            .disabled(viewModel.isSubmitting || viewModel.portScanTarget.isEmpty)

            if let command = viewModel.portScanCommand {
                StatusRow(status: command.status, colorScheme: colorScheme)
                if command.status == "completed", case .portScan(let result) = command.result {
                    if result.openPorts.isEmpty {
                        Text("Nenhuma porta aberta encontrada.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    } else {
                        Text(result.openPorts.map(String.init).joined(separator: ", "))
                            .font(.system(.caption, design: .monospaced))
                    }
                }
                if command.status == "failed" {
                    Text(command.error ?? "Falha não especificada.")
                        .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                }
            }
        }
    }

    @ViewBuilder
    private var rdapSection: some View {
        Section("RDAP / WHOIS") {
            TextField("Domínio ou IP", text: $viewModel.rdapQuery)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            Button {
                Task { await viewModel.runRdapLookup() }
            } label: {
                Text(viewModel.isRdapLoading ? "Consultando…" : "Consultar")
            }
            .disabled(viewModel.isRdapLoading || viewModel.rdapQuery.isEmpty)

            if let result = viewModel.rdapResult {
                LabeledContent("Nome", value: result.name.isEmpty ? result.query : result.name)
                if !result.handle.isEmpty {
                    LabeledContent("Handle", value: result.handle)
                }
                if !result.status.isEmpty {
                    LabeledContent("Status", value: result.status.joined(separator: ", "))
                }
                ForEach(result.events, id: \.action) { event in
                    LabeledContent(event.action, value: event.date)
                }
            }
            if let rdapError = viewModel.rdapError {
                Text(rdapError)
                    .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
            }
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

private struct StatusRow: View {
    let status: String
    let colorScheme: ColorScheme

    var body: some View {
        LabeledContent("Status", value: label)
    }

    private var label: String {
        switch status {
        case "pending": return "Aguardando agente…"
        case "claimed": return "Executando…"
        case "completed": return "Concluído"
        case "failed": return "Falhou"
        default: return status
        }
    }
}

#Preview {
    NavigationStack {
        DiagnosticsView(client: APIClient())
    }
}
