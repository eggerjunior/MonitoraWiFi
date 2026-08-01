import Testing
@testable import EggerNetworkIntelligence

@Suite("VersionManager e VersionHistory")
struct VersionManagerTests {
    @Test("commitURL é nil quando o commit é 'dev' (build local)")
    func commitURLIsNilForDevBuild() {
        // Em builds de teste/simulador sem GIT_COMMIT injetado, o valor lido
        // do Info.plist é o fallback "dev" — nunca deve virar um link quebrado.
        if VersionManager.gitCommit == "dev" {
            #expect(VersionManager.commitURL == nil)
        }
    }

    @Test("exatamente uma entrada do changelog é a atual")
    func exactlyOneCurrentEntry() {
        let currentEntries = VersionHistory.entries.filter(\.isCurrent)
        #expect(currentEntries.count == 1)
    }

    @Test("a entrada atual do changelog corresponde ao topo da lista")
    func currentEntryIsFirst() {
        #expect(VersionHistory.entries.first?.isCurrent == true)
    }
}
