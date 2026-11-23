import Combine
import Foundation

/// State for the feed list screen: the user's subscriptions plus per-feed
/// unread counts, and the add/delete/refresh actions against the API.
///
/// It opens on the last session's feeds and counts from `LaunchCache`, when
/// there are any, and writes every change back so the next launch opens on
/// the current state.
///
/// Without a connection the cached data stays on screen without an alert,
/// unsubscribing is disabled, and the list reloads once a connection
/// returns.
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
    /// True after a request failed for lack of a connection, until a later
    /// one succeeds or the device's network path returns.
    @Published private(set) var lostConnection = false
    @Published var errorMessage: String?

    /// Called when the API reports 401: the session is gone server-side and
    /// the app should return to the login screen.
    var onSessionExpired: () -> Void = {}

    private let client: NetworkClient
    private let launchCache: LaunchCache
    private let connectivity: ConnectivityMonitor
    private var connectivityObserver: AnyCancellable?
    /// Off while the initializer fills in the cached data, which needs no
    /// writing back.
    private var savesToCache = false

    init(client: NetworkClient = .shared,
         launchCache: LaunchCache? = nil,
         connectivity: ConnectivityMonitor? = nil) {
        self.client = client
        // A default argument cannot reach the main actor's shared instances.
        self.launchCache = launchCache ?? .shared
        self.connectivity = connectivity ?? .shared
        if let cached = self.launchCache.loadFeedList() {
            feeds = cached.feeds
            unreadCounts = cached.unreadCounts
            isShowingCachedData = true
        }
        savesToCache = true
        connectivityObserver = self.connectivity.$isOffline
            .removeDuplicates()
            .dropFirst()
            .sink { [weak self] isOffline in
                self?.connectivityChanged(isOffline: isOffline)
            }
    }

    /// Re-enables actions when the network path returns, and retries a load
    /// that failed.
    private func connectivityChanged(isOffline: Bool) {
        // Views read `isOffline`, which depends on the monitor.
        objectWillChange.send()
        guard !isOffline else { return }
        lostConnection = false
        if loadFailed {
            Task { await load() }
        }
    }

    /// True when there is no network path, or the latest request found no
    /// connection.
    var isOffline: Bool {
        connectivity.isOffline || lostConnection
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
            lostConnection = false
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
            lostConnection = false
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
            launchCache.removeArticles(forFeedID: feed.id)
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
        switch error {
        case NetworkError.unauthorized:
            onSessionExpired()
            return
        case NetworkError.noConnection:
            lostConnection = true
            // With feeds on screen the cached list is still usable, and the
            // article list's banner says it is offline.
            if !feeds.isEmpty { return }
        default:
            break
        }
        errorMessage = error.localizedDescription
    }
}
