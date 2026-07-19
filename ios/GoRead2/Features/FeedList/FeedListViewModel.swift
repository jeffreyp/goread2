import Foundation

/// State for the feed list screen: the user's subscriptions plus per-feed
/// unread counts, and the add/delete/refresh actions against the API.
@MainActor
final class FeedListViewModel: ObservableObject {
    @Published private(set) var feeds: [Feed] = []
    @Published private(set) var unreadCounts: [Int: Int] = [:]
    /// True once the first load has completed, successfully or not; gates
    /// the empty state so it never flashes before data arrives.
    @Published private(set) var hasLoaded = false
    @Published var errorMessage: String?

    /// Called when the API reports 401: the session is gone server-side and
    /// the app should return to the login screen.
    var onSessionExpired: () -> Void = {}

    private let client: NetworkClient

    init(client: NetworkClient = .shared) {
        self.client = client
    }

    var totalUnread: Int {
        unreadCounts.values.reduce(0, +)
    }

    func unreadCount(for feed: Feed) -> Int {
        unreadCounts[feed.id] ?? 0
    }

    /// Loads subscriptions and unread counts concurrently.
    func load() async {
        do {
            async let feeds = client.getFeeds()
            async let counts = client.getUnreadCounts()
            self.feeds = try await feeds
            self.unreadCounts = try await counts
        } catch {
            handle(error)
        }
        hasLoaded = true
    }

    /// Refreshes only the unread counts. Used when the list reappears, e.g.
    /// after reading articles on a pushed screen.
    func refreshUnreadCounts() async {
        do {
            unreadCounts = try await client.getUnreadCounts()
        } catch {
            handle(error)
        }
    }

    /// Pull-to-refresh: asks the server to re-fetch every feed, then
    /// reloads the list and counts.
    func refresh() async {
        do {
            try await client.refreshFeeds()
        } catch {
            handle(error)
            return
        }
        await load()
    }

    /// Subscribes to the feed at `url`, then reloads so ordering and counts
    /// match the server. Throws so the add-feed sheet can show the error
    /// inline; session expiry is still routed to `onSessionExpired`.
    func addFeed(url: String) async throws {
        do {
            _ = try await client.addFeed(url: url)
        } catch {
            if case NetworkError.unauthorized = error {
                onSessionExpired()
            }
            throw error
        }
        await load()
    }

    func delete(_ feed: Feed) async {
        do {
            try await client.deleteFeed(id: feed.id)
            feeds.removeAll { $0.id == feed.id }
            unreadCounts[feed.id] = nil
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
