import SwiftUI

struct SettingsView: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var showingChangelog = false

    var body: some View {
        NavigationStack {
            List {
                if case .authenticated(let user) = session.state {
                    Section("Conta") {
                        LabeledContent("E-mail", value: user.email)
                        LabeledContent("Papel", value: user.role.rawValue.capitalized)
                    }
                }

                Section {
                    Button {
                        showingChangelog = true
                    } label: {
                        LabeledContent(
                            "Versão \(VersionManager.currentVersionString)",
                            value: VersionManager.buildDateString
                        )
                    }
                }

                Section {
                    Button("Sair", role: .destructive) {
                        Task {
                            await session.logout()
                            dismiss()
                        }
                    }
                }
            }
            .navigationTitle("Configurações")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Fechar") { dismiss() }
                }
            }
            .sheet(isPresented: $showingChangelog) {
                ChangelogView()
            }
        }
    }
}

struct ChangelogView: View {
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List(VersionHistory.entries) { entry in
                Section {
                    ForEach(entry.changes, id: \.self) { change in
                        Text(change)
                    }
                } header: {
                    HStack {
                        Text("\(entry.version) (Build \(entry.build))")
                        if entry.isCurrent {
                            Text("Atual")
                                .font(.caption2.bold())
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(.green.opacity(0.2), in: Capsule())
                        }
                        Spacer()
                        Text(entry.isCurrent ? VersionManager.buildDateString : entry.date)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .navigationTitle("Histórico de versões")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Fechar") { dismiss() }
                }
            }
        }
    }
}

#Preview {
    SettingsView()
        .environment(SessionStore())
}
