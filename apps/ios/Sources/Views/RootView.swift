import SwiftUI

enum AppSection: String, CaseIterable, Identifiable {
    case overview = "Visão geral"
    case network = "Rede"
    case map = "Mapa"
    case tools = "Ferramentas"
    case alerts = "Alertas"
    case history = "Histórico"

    var id: String { rawValue }

    var systemImage: String {
        switch self {
        case .overview: "gauge"
        case .network: "network"
        case .map: "map"
        case .tools: "wrench.and.screwdriver"
        case .alerts: "bell"
        case .history: "clock.arrow.circlepath"
        }
    }
}

/// Navegação adaptativa (Seção 15): `TabView` em iPhone (compacto),
/// `NavigationSplitView` em iPad (regular) — mesmas 6 seções nos dois casos,
/// só muda o chrome de navegação.
struct RootView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(SessionStore.self) private var session
    @State private var selection: AppSection = .overview

    var body: some View {
        if horizontalSizeClass == .regular {
            NavigationSplitView {
                // `List(_:selection:rowContent:)` com Binding não-opcional foi
                // removido do SwiftUI — a API de seleção de lista exige
                // Binding<SelectionValue?>. `selection` aqui nunca é nil na
                // prática (sempre há uma seção selecionada), então o setter
                // simplesmente ignora um nil vindo da lista.
                List(AppSection.allCases, selection: Binding(
                    get: { selection },
                    set: { newValue in
                        if let newValue { selection = newValue }
                    }
                )) { section in
                    Label(section.rawValue, systemImage: section.systemImage)
                        .tag(section)
                }
                .navigationTitle("Egger")
            } detail: {
                NavigationStack {
                    destination(for: selection)
                }
            }
        } else {
            TabView(selection: $selection) {
                ForEach(AppSection.allCases) { section in
                    NavigationStack {
                        destination(for: section)
                    }
                    .tabItem { Label(section.rawValue, systemImage: section.systemImage) }
                    .tag(section)
                }
            }
        }
    }

    @ViewBuilder
    private func destination(for section: AppSection) -> some View {
        switch section {
        case .overview:
            OverviewView(client: session.client)
        case .tools:
            DiagnosticsView(client: session.client)
        case .network:
            NetworkView(client: session.client)
        case .map:
            SpatialSurveyView(client: session.client)
        case .alerts:
            AlertsView(client: session.client)
        case .history:
            HistoryView(client: session.client)
        }
    }
}

#Preview {
    RootView()
        .environment(SessionStore())
}
