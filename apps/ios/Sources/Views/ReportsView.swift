import Foundation
import SwiftUI

/// Relatórios (Fase 7, motor de correlação): diagnósticos e recomendações
/// são calculados pelo worker (apps/worker/internal/diagnostics) a partir
/// de anomalias reais — nunca gerados sem evidência. Relatórios são
/// agregados sob demanda pelo backend (POST /sites/{id}/reports), não
/// pré-gerados. Paridade com
/// apps/web/src/app/(dashboard)/reports/page.tsx.
@MainActor
@Observable
final class ReportsViewModel {
    private(set) var siteId: String?
    private(set) var diagnoses: [Diagnosis] = []
    private(set) var recommendations: [Recommendation] = []
    private(set) var reports: [Report] = []
    private(set) var isLoading = true
    private(set) var errorMessage: String?

    private(set) var isGenerating = false
    private(set) var generateError: String?

    private(set) var expandedContent: [String: ReportContent] = [:]
    private(set) var expandingReportId: String?

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
            siteId = site.id
            async let diagnosesResult = client.diagnoses(siteId: site.id, page: 1, pageSize: 50)
            async let recommendationsResult = client.recommendations(siteId: site.id, page: 1, pageSize: 50)
            async let reportsResult = client.reports(siteId: site.id, page: 1, pageSize: 20)
            diagnoses = try await diagnosesResult.items
            recommendations = try await recommendationsResult.items
            reports = try await reportsResult.items
        } catch {
            errorMessage = "Não foi possível carregar os relatórios. Verifique a conexão com o backend."
        }
        isLoading = false
    }

    func generateReport() async {
        guard let siteId else { return }
        generateError = nil
        isGenerating = true
        do {
            let created = try await client.createReport(siteId: siteId)
            reports.insert(created, at: 0)
        } catch let error as APIClient.ClientError {
            generateError = Self.message(for: error)
        } catch {
            generateError = "Erro de rede ao gerar o relatório."
        }
        isGenerating = false
    }

    func toggleExpand(reportId: String) async {
        if expandedContent[reportId] != nil {
            expandedContent[reportId] = nil
            return
        }
        expandingReportId = reportId
        do {
            let full = try await client.report(id: reportId)
            expandedContent[reportId] = full.content
        } catch {
            // Silencioso: a linha do relatório simplesmente não expande —
            // nunca mostra conteúdo inventado no lugar de um erro.
        }
        expandingReportId = nil
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

struct ReportsView: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var viewModel: ReportsViewModel

    init(client: APIClient) {
        _viewModel = State(initialValue: ReportsViewModel(client: client))
    }

    var body: some View {
        List {
            if viewModel.isLoading {
                ProgressView("Carregando…")
            } else if let errorMessage = viewModel.errorMessage {
                Text(errorMessage)
                    .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
            } else {
                Section("Diagnósticos (\(viewModel.diagnoses.count))") {
                    if viewModel.diagnoses.isEmpty {
                        Text("Nenhum diagnóstico ainda — pode ser que ainda não haja evidência suficiente (anomalias reais) ou que a rede esteja dentro do padrão.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    } else {
                        ForEach(viewModel.diagnoses) { d in
                            DiagnosisRow(diagnosis: d, colorScheme: colorScheme)
                        }
                    }
                }

                Section("Recomendações (\(viewModel.recommendations.count))") {
                    if viewModel.recommendations.isEmpty {
                        Text("Nenhuma recomendação ainda — cada recomendação exige um diagnóstico real por trás, nunca é gerada sozinha.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    } else {
                        ForEach(viewModel.recommendations) { r in
                            RecommendationRow(recommendation: r, colorScheme: colorScheme)
                        }
                    }
                }

                Section {
                    Button {
                        Task { await viewModel.generateReport() }
                    } label: {
                        if viewModel.isGenerating {
                            ProgressView()
                        } else {
                            Text("Gerar relatório (últimos 7 dias)")
                        }
                    }
                    .disabled(viewModel.isGenerating)

                    if let generateError = viewModel.generateError {
                        Text(generateError)
                            .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                    }
                } header: {
                    Text("Relatórios gerados (\(viewModel.reports.count))")
                }

                if viewModel.reports.isEmpty {
                    Section {
                        Text("Nenhum relatório gerado ainda.")
                            .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                    }
                } else {
                    ForEach(viewModel.reports) { report in
                        Section {
                            ReportRow(
                                report: report,
                                content: viewModel.expandedContent[report.id],
                                isExpanding: viewModel.expandingReportId == report.id,
                                colorScheme: colorScheme,
                                onToggle: { Task { await viewModel.toggleExpand(reportId: report.id) } }
                            )
                        }
                    }
                }
            }
        }
        .navigationTitle("Relatórios")
        .task {
            await viewModel.load()
        }
        .refreshable {
            await viewModel.load()
        }
    }
}

private struct DiagnosisRow: View {
    let diagnosis: Diagnosis
    let colorScheme: ColorScheme

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(categoryLabel(diagnosis.category)).font(.headline)
                Spacer()
                LevelBadge(label: "impacto", level: diagnosis.impact, colorScheme: colorScheme)
                LevelBadge(label: "risco", level: diagnosis.risk, colorScheme: colorScheme)
            }
            Text(diagnosis.summary)
                .font(.subheadline)
                .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
            Text("Confiança: \(Int(diagnosis.confidence * 100))% · \(diagnosis.evidence.count) \(diagnosis.evidence.count == 1 ? "anomalia como evidência" : "anomalias como evidência")")
                .font(.caption)
                .foregroundStyle(Color.egger(.textDisabled, scheme: colorScheme))
        }
    }
}

private struct RecommendationRow: View {
    let recommendation: Recommendation
    let colorScheme: ColorScheme

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text("Ação recomendada").font(.headline)
                Spacer()
                LevelBadge(label: "impacto", level: recommendation.impact, colorScheme: colorScheme)
                LevelBadge(label: "risco", level: recommendation.risk, colorScheme: colorScheme)
            }
            Text(recommendation.action)
                .font(.subheadline)
                .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
            Text("Confiança: \(Int(recommendation.confidence * 100))%")
                .font(.caption)
                .foregroundStyle(Color.egger(.textDisabled, scheme: colorScheme))
        }
    }
}

private struct ReportRow: View {
    let report: Report
    let content: ReportContent?
    let isExpanding: Bool
    let colorScheme: ColorScheme
    let onToggle: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text("\(shortDate(report.periodStart)) – \(shortDate(report.periodEnd))")
                Spacer()
                Button(content != nil ? "Recolher" : "Ver conteúdo", action: onToggle)
                    .font(.caption)
                    .disabled(isExpanding)
            }
            Text("Gerado em \(report.generatedAt)")
                .font(.caption)
                .foregroundStyle(Color.egger(.textDisabled, scheme: colorScheme))

            if isExpanding {
                ProgressView()
            } else if let content {
                Divider()
                Text("\(content.anomalyCount) \(content.anomalyCount == 1 ? "anomalia" : "anomalias") no período · \(content.diagnoses.count) \(content.diagnoses.count == 1 ? "diagnóstico" : "diagnósticos") · \(content.recommendations.count) \(content.recommendations.count == 1 ? "recomendação" : "recomendações")")
                    .font(.caption)
                    .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
                ForEach(Array(content.diagnoses.enumerated()), id: \.offset) { item in
                    Text(item.element.summary)
                        .font(.caption)
                }
            }
        }
    }

    private func shortDate(_ isoString: String) -> String {
        guard let date = ISO8601DateFormatter().date(from: isoString) else { return isoString }
        return date.formatted(date: .abbreviated, time: .omitted)
    }
}

private struct LevelBadge: View {
    let label: String
    let level: String
    let colorScheme: ColorScheme

    private var token: EggerColorToken {
        switch level {
        case "high": .critical
        case "medium": .warning
        default: .success
        }
    }

    private var levelLabel: String {
        switch level {
        case "high": "alto"
        case "medium": "médio"
        default: "baixo"
        }
    }

    var body: some View {
        Text("\(label): \(levelLabel)")
            .font(.caption2)
            .foregroundStyle(Color.egger(token, scheme: colorScheme))
    }
}

private func categoryLabel(_ category: String) -> String {
    switch category {
    case "internet_slow": "Internet lenta"
    case "wifi_slow": "Wi-Fi lento"
    default: category
    }
}

#Preview {
    NavigationStack {
        ReportsView(client: APIClient())
    }
}
