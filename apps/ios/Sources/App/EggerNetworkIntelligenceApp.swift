import SwiftUI

@main
struct EggerNetworkIntelligenceApp: App {
    @State private var session = SessionStore()

    var body: some Scene {
        WindowGroup {
            AppRootSwitchView()
                .environment(session)
        }
    }
}

/// Alterna entre login e o shell principal com base no estado real de
/// autenticação (validado contra o backend em `bootstrap()`) — nunca assume
/// sessão válida só porque um token existe localmente.
private struct AppRootSwitchView: View {
    @Environment(SessionStore.self) private var session

    var body: some View {
        Group {
            switch session.state {
            case .checking:
                ProgressView("Carregando…")
            case .unauthenticated:
                LoginView()
            case .authenticated:
                RootView()
            }
        }
        .task {
            await session.bootstrap()
        }
    }
}
