#if os(macOS)
import Foundation

/// State behind the File menu's OPML commands: the open panel's presentation
/// flag, the finished export awaiting the save panel, and the result
/// messages. The Settings window runs the same transfers through
/// `SettingsViewModel`, so the menu commands work without opening it.
@MainActor
final class OPMLActionModel: ObservableObject {
    @Published var isImporting = false
    @Published var export: OPMLExport?
    /// The one result message on screen, if any. A single alert carries both
    /// outcomes, since only one can be pending at a time.
    @Published var alert: OPMLAlert?
    /// True while an import or export request is in flight; the menu
    /// commands stay enabled, but a second request is ignored.
    @Published private(set) var isBusy = false

    /// Called when the API reports 401, matching the other screens: the
    /// session is gone server-side and the app returns to the login screen.
    var onSessionExpired: () -> Void = {}
    /// Called after a successful import, so the feed list picks up the new
    /// subscriptions.
    var onImported: () async -> Void = {}

    private let transfer: OPMLTransfer

    init(transfer: OPMLTransfer = OPMLTransfer()) {
        self.transfer = transfer
    }

    func beginImport() {
        guard !isBusy else { return }
        isImporting = true
    }

    func importFile(at url: URL) async {
        isBusy = true
        defer { isBusy = false }
        do {
            let confirmation = try await transfer.importFile(at: url)
            alert = OPMLAlert(title: "OPML Import", message: confirmation)
            await onImported()
        } catch {
            handle(error)
        }
    }

    func exportSubscriptions() async {
        guard !isBusy else { return }
        isBusy = true
        defer { isBusy = false }
        do {
            export = try await transfer.export()
        } catch {
            handle(error)
        }
    }

    /// Reports a failure raised outside a transfer, such as a file panel
    /// that could not hand over the chosen file.
    func report(_ error: Error) {
        handle(error)
    }

    func report(message: String) {
        alert = OPMLAlert(title: "Error", message: message)
    }

    private func handle(_ error: Error) {
        if error is CancellationError { return }
        if case NetworkError.unauthorized = error {
            onSessionExpired()
            return
        }
        report(message: error.localizedDescription)
    }
}

/// A pending result message for the OPML commands.
struct OPMLAlert: Identifiable {
    let id = UUID()
    let title: String
    let message: String
}
#endif
