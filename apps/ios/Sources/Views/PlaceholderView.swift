import SwiftUI

/// Estado vazio honesto (Seção 15: "Estados vazios") — nunca dado simulado.
/// Cada seção aponta para a fase do roadmap em que será implementada de
/// verdade.
struct PlaceholderView: View {
    let section: AppSection

    private var phase: String {
        switch section {
        case .overview: "Fase 1"
        case .network: "Fase 4 (Dashboards)"
        case .map: "Fase 6 (LiDAR)"
        case .tools: "Fase 5 (Diagnósticos)"
        case .alerts: "Fase 4 (Dashboards)"
        case .history: "Fase 4 (Dashboards)"
        case .reports: "Fase 7 (Inteligência)"
        }
    }

    var body: some View {
        ContentUnavailableView {
            Label(section.rawValue, systemImage: section.systemImage)
        } description: {
            Text("Em construção — chega na \(phase). Nenhum dado simulado é exibido aqui.")
        }
        .navigationTitle(section.rawValue)
    }
}

#Preview {
    NavigationStack {
        PlaceholderView(section: .network)
    }
}
