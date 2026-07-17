import Foundation

/// State for the settings screen: the signed-in user, subscription status,
/// the max-articles preference, and the OPML import/export and Stripe
/// customer portal actions against the API.
@MainActor
final class SettingsViewModel: ObservableObject {
    @Published private(set) var user: CurrentUser?
    @Published private(set) var subscription: SubscriptionInfo?
    /// True once the first load has completed, successfully or not; gates
    /// the form so it never renders half-empty before data arrives.
    @Published private(set) var hasLoaded = false
    /// True while a portal, import, or export request is in flight; disables
    /// the corresponding buttons.
    @Published private(set) var isBusy = false

    /// Editable copy of the max-articles-on-feed-add preference;
    /// `savedMaxArticles` mirrors the value the server has.
    @Published var maxArticles = 100
    @Published private(set) var savedMaxArticles = 100

    @Published var errorMessage: String?
    /// Success confirmation, e.g. after an OPML import.
    @Published var infoMessage: String?
    /// Stripe customer portal URL, presented in SFSafariViewController.
    @Published var portalItem: SafariItem?
    /// Exported OPML written to a temporary file for the share sheet.
    @Published var exportedOPML: OPMLExport?

    /// Called when the API reports 401: the session is gone server-side and
    /// the app should return to the login screen.
    var onSessionExpired: () -> Void = {}

    private let client: NetworkClient

    init(client: NetworkClient = .shared) {
        self.client = client
    }

    var maxArticlesEdited: Bool {
        maxArticles != savedMaxArticles
    }

    /// The Stripe portal only exists server-side when the subscription
    /// system is enabled, and only makes sense for accounts Stripe bills.
    var canManageSubscription: Bool {
        guard let status = subscription?.status else { return false }
        return status != "unlimited" && status != "admin"
    }

    /// Loads the signed-in user and subscription info concurrently.
    /// Fetching /auth/me also refreshes the CSRF token for the mutating
    /// actions on this screen.
    func load() async {
        do {
            async let me = client.fetchMe()
            async let subscription = client.getSubscriptionInfo()
            let user = try await me.user
            self.user = user
            self.subscription = try await subscription
            maxArticles = user.maxArticlesOnFeedAdd
            savedMaxArticles = user.maxArticlesOnFeedAdd
        } catch {
            handle(error)
        }
        hasLoaded = true
    }

    /// Creates a Stripe customer portal session and exposes its URL for
    /// in-app presentation.
    func openSubscriptionPortal() async {
        isBusy = true
        defer { isBusy = false }
        do {
            let url = try await client.createPortalSession()
            guard let item = SafariItem(url) else {
                errorMessage = "The server returned an invalid portal URL."
                return
            }
            portalItem = item
        } catch {
            handle(error)
        }
    }

    /// Uploads the OPML document at `url` (a security-scoped URL from the
    /// document picker) and reports how many feeds were imported.
    func importOPML(from url: URL) async {
        isBusy = true
        defer { isBusy = false }
        do {
            let accessing = url.startAccessingSecurityScopedResource()
            defer {
                if accessing { url.stopAccessingSecurityScopedResource() }
            }
            let data = try Data(contentsOf: url)
            let count = try await client.importOPML(data, filename: url.lastPathComponent)
            infoMessage = count == 1 ? "Imported 1 feed." : "Imported \(count) feeds."
        } catch {
            handle(error)
        }
    }

    /// Downloads the OPML export to a temporary file and exposes it for the
    /// share sheet.
    func exportOPML() async {
        isBusy = true
        defer { isBusy = false }
        do {
            let data = try await client.exportOPML()
            let fileURL = FileManager.default.temporaryDirectory
                .appendingPathComponent("goread2-subscriptions.opml")
            try data.write(to: fileURL, options: .atomic)
            exportedOPML = OPMLExport(url: fileURL)
        } catch {
            handle(error)
        }
    }

    func saveMaxArticles() async {
        // Matches the server-side binding on PUT /api/account/max-articles.
        guard (0...10_000).contains(maxArticles) else {
            errorMessage = "Max articles must be between 0 and 10,000."
            return
        }
        do {
            try await client.updateMaxArticles(maxArticles)
            savedMaxArticles = maxArticles
        } catch {
            handle(error)
        }
    }

    private func handle(_ error: Error) {
        if case NetworkError.unauthorized = error {
            onSessionExpired()
            return
        }
        errorMessage = error.localizedDescription
    }
}

/// Identifiable temporary-file URL box for sheet(item:) presentation of the
/// OPML export share sheet.
struct OPMLExport: Identifiable {
    let url: URL
    var id: URL { url }
}
