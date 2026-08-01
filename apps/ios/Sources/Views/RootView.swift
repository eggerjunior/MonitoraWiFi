import SwiftUI

enum AppSection: String, CaseIterable, Identifiable {
    case overview = "Visão geral"
    case network = "Rede"
    case map = "Mapa"
    case tools = "Ferramentas"
    case alerts = "Alertas"

    var id: String { rawValue }

    var systemImage: String {
        switch self {
        case .overview: "gauge"
        case .network: "network"
        case .map: "map"
        case .tools: "wrench.and.screwdriver"
        case .alerts: "bell"
        }
    }
}

/// Navegação adaptativa (Seção 15): `TabView` em iPhone (compacto),
/// `NavigationSplitView` em iPad (regular) — mesmas 5 seções nos dois casos,
/// só muda o chrome de navegação.
struct RootView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var selection: AppSection = .overview

    var body: some View {
        if horizontalSizeClass == .regular {
            NavigationSplitView {
                List(AppSection.allCases, selection: $selection) { section in
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
            OverviewView()
        case .network, .map, .tools:
            PlaceholderView(section: section)
        case .alerts:
            PlaceholderView(section: section)
        }
    }
}

#Preview {
    RootView()
        .environment(SessionStore())
}
