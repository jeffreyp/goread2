import Foundation

/// State for the feed list screen: the user's subscriptions plus per-feed
/// unread counts, and the add/delete/refresh actions against the API.
///
/// It opens on the last session's feeds and counts from `LaunchCache`, when
/// there are any, and writes every change back so the next launch opens on
/// the current state.
@MainActor
final class FeedListViewModel: ObservableObject {
    @Published private(set) var feeds: [Feed] = [] {
        didSet { saveToCache() }
    }
    @Published private(set) var unreadCounts: [Int: Int] = [:] {
        didSet { saveToCache() }
    }
    /// True once the first load from the server has completed, successfully
    /// or not.
    @Published private(set) var hasLoaded = false
    /// True while the list shows the last session's data from the cache and
    /// the server has not yet answered.
    @Published private(set) var isShowingCachedData = false
    /// True when the latest load failed, so an empty list is a failure to
    /// reach the server rather than an account with no subscriptions.
    @Published private(set) var loadFailed = false
    @Published var errorMessage: String?

    /// Called when the API reports 401: the session is gone server-side and
    /// the app should return to the login screen.
    var onSessionExpired: () -> Void = {}

    private let client: NetworkClient
    private let launchCache: LaunchCache
    /// Off while the initializer fills in the cached data, which needs no
    /// writing back.
    private var savesToCache = false

    init(client: NetworkClient = .shared, launchCache: LaunchCache? = nil) {
        self.client = client
        // A default argument cannot reach the main actor's shared cache.
        self.launchCache = launchCache ?? .shared
        if let cached = self.launchCache.loadFeedList() {
            feeds = cached.feeds
            unreadCounts = cached.unreadCounts
            isShowingCachedData = true
        }
        savesToCache = true
    }

    /// True once there is something to show: cached data, or the server's
    /// answer to the first load. Gates the empty state so it never flashes
    /// before data arrives.
    var isReady: Bool {
        hasLoaded || isShowingCachedData
    }

    var totalUnread: Int {
        unreadCounts.values.reduce(0, +)
    }

    func unreadCount(for feed: Feed) -> Int {
        unreadCounts[feed.id] ?? 0
    }

    /// Loads subscriptions and unread counts concurrently. On failure, any
    /// cached data stays on screen.
    func load() async {
        do {
            async let feeds = client.getFeeds()
            async let counts = client.getUnreadCounts()
            let (loadedFeeds, loadedCounts) = try await (feeds, counts)
            self.feeds = loadedFeeds
            self.unreadCounts = loadedCounts
            isShowingCachedData = false
            loadFailed = false
        } catch {
            loadFailed = !(error is CancellationError)
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

    /// Applies a read-state change reported by the article list to the
    /// feed's count locally, so opening an article does not cost a request.
    /// Counts never drop below zero; the next refresh corrects any drift.
    func adjustUnreadCount(feedID: Int, by delta: Int) {
        unreadCounts[feedID] = max(0, (unreadCounts[feedID] ?? 0) + delta)
    }

    /// Pull-to-refresh: asks the server to re-fetch the user's due feeds, then
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

    private func saveToCache() {
        guard savesToCache else { return }
        launchCache.saveFeedList(feeds: feeds, unreadCounts: unreadCounts)
    }

    private func handle(_ error: Error) {
        if error is CancellationError { return }
        if case NetworkError.unauthorized = error {
            onSessionExpired()
            return
        }
        errorMessage = error.localizedDescription
    }
}
