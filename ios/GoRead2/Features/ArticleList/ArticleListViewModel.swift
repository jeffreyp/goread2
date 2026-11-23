import Combine
import Foundation

/// State for the article list screen: one page-at-a-time cursor pagination
/// over /api/feeds/:id/articles, an unread-only filter, and optimistic
/// mark-read on opening an article.
///
/// Every list starts from its last first page in `LaunchCache` when there
/// is one, for its feed and filter, and writes the first page back on every
/// change. Without a connection the cached page stays on screen with the
/// time it was fetched, actions that need the server are disabled, and the
/// reader shows the content of articles it has fetched before.
@MainActor
final class ArticleListViewModel: ObservableObject {
    /// Sentinel article ID for the position one past the last article, where
    /// the reader shows the end-of-list page instead of an article. Real
    /// article IDs from the server are positive.
    static let endOfListID = -1

    @Published private(set) var articles: [Article] = [] {
        didSet { saveToCache() }
    }
    /// True once the first load from the server has completed, successfully
    /// or not.
    @Published private(set) var hasLoaded = false
    /// True while the list shows a first page from the cache: the server
    /// has not answered yet, or the load failed.
    @Published private(set) var isShowingCachedData = false
    /// When the server returned the first page on screen, or nil before any
    /// load and for a cached page written before the date was kept.
    @Published private(set) var fetchedAt: Date?
    /// True when the latest load failed, so an empty list is a failure to
    /// reach the server rather than a feed with no articles.
    @Published private(set) var loadFailed = false
    /// True after a request failed for lack of a connection, until a later
    /// one succeeds or the device's network path returns.
    @Published private(set) var lostConnection = false
    @Published private(set) var isLoadingMore = false
    /// The article the end-of-list page follows: the one the reader moved
    /// forward from to reach it. Pages loaded after that move put further
    /// articles after it, and the reader continues into them.
    @Published private(set) var endOfListAnchorID: Int?
    /// True while the end-of-list page looks for further articles in the
    /// background. An unread-only list can return several empty pages in a
    /// row while the server scans past read articles, so the reader shows
    /// the end-of-list page during the search instead of waiting on it.
    @Published private(set) var isSearchingForMore = false
    /// The All Articles stream opens unread-only, since its common use is
    /// catching up; per-feed lists open showing everything.
    @Published private(set) var unreadOnly: Bool
    @Published var errorMessage: String?
    /// Articles whose full content could not be fetched (or is genuinely
    /// empty); the reader falls back to the description for these instead of
    /// showing a loading placeholder forever.
    @Published private(set) var contentUnavailable: Set<Int> = []

    /// Called when the API reports 401: the session is gone server-side and
    /// the app should return to the login screen.
    var onSessionExpired: () -> Void = {}
    /// Called with an article's feed ID and the change to that feed's unread
    /// count (-1 when marked read, +1 when marked unread, and the reverse
    /// when a failed request is rolled back). The split view uses it to keep
    /// the sidebar's badges current without refetching every count.
    var onUnreadCountChange: (_ feedID: Int, _ delta: Int) -> Void = { _, _ in }

    /// Server-side unread counts include only articles published within this
    /// window (unreadCountWindowDays in the Go database layer), so read-state
    /// changes on older articles leave the counts alone.
    private static let unreadCountWindow: TimeInterval = 90 * 24 * 60 * 60

    private let selection: FeedSelection
    private let client: NetworkClient
    private let launchCache: LaunchCache
    private let connectivity: ConnectivityMonitor
    private var connectivityObserver: AnyCancellable?
    private var nextCursor: String?
    /// The number of leading `articles` that came with the first page, and
    /// the cursor that followed it: what the cache holds. Later pages are
    /// left out, so the cached cursor still continues right after them.
    private var firstPageCount = 0
    private var firstPageCursor: String?
    /// Off while the initializer fills in the cached page, which needs no
    /// writing back.
    private var savesToCache = false
    /// The next-page fetch in flight, shared so a second caller waits for it
    /// instead of returning early.
    private var loadMoreTask: Task<Bool, Never>?
    /// The end-of-list page's background search, while it runs.
    private var searchTask: Task<Void, Never>?
    /// Full-content fetches currently in flight, keyed by article ID.
    private var contentFetches: Set<Int> = []
    /// Articles in `contentUnavailable` only because the fetch found no
    /// connection; they are fetched again once one returns.
    private var contentMissingOffline: Set<Int> = []

    init(selection: FeedSelection,
         client: NetworkClient = .shared,
         launchCache: LaunchCache? = nil,
         connectivity: ConnectivityMonitor? = nil) {
        self.selection = selection
        self.client = client
        // A default argument cannot reach the main actor's shared instances.
        self.launchCache = launchCache ?? .shared
        self.connectivity = connectivity ?? .shared
        if case .all = selection {
            unreadOnly = true
        } else {
            unreadOnly = false
        }
        restoreFromCache()
        savesToCache = true
        connectivityObserver = self.connectivity.$isOffline
            .removeDuplicates()
            .dropFirst()
            .sink { [weak self] isOffline in
                self?.connectivityChanged(isOffline: isOffline)
            }
    }

    /// Fills the list from the cached first page for the current filter,
    /// when there is one.
    private func restoreFromCache() {
        guard let cached = launchCache.loadArticles(for: selection, unreadOnly: unreadOnly) else { return }
        articles = cached.articles
        nextCursor = cached.nextCursor
        firstPageCount = cached.articles.count
        firstPageCursor = cached.nextCursor
        fetchedAt = cached.fetchedAt
        isShowingCachedData = true
    }

    /// Re-enables actions when the network path returns, and replaces a
    /// cached list left on screen by a failed load.
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
    /// connection. Read and star changes, mark-all-read, and further pages
    /// are unavailable meanwhile.
    var isOffline: Bool {
        connectivity.isOffline || lostConnection
    }

    /// True while the list on screen is a cached copy because the server
    /// could not be reached, which the list notes with `fetchedAt`.
    var isShowingStaleData: Bool {
        isShowingCachedData && loadFailed
    }

    /// True once there is something to show: the cached first page, or the
    /// server's answer to the first load. Gates the empty state so it never
    /// flashes before data arrives.
    var isReady: Bool {
        hasLoaded || isShowingCachedData
    }

    /// True while the server reports another page after the loaded ones.
    var hasMorePages: Bool {
        nextCursor != nil
    }

    /// The first loaded article after the end-of-list page's anchor, or nil
    /// when none has loaded. A refresh that drops the anchor also clears
    /// this, leaving the reader on the end-of-list page.
    var articleAfterEndOfList: Article? {
        guard let endOfListAnchorID,
              let index = articles.firstIndex(where: { $0.id == endOfListAnchorID }),
              articles.indices.contains(index + 1) else { return nil }
        return articles[index + 1]
    }

    /// The number of loaded articles still unread.
    var unreadLoadedCount: Int {
        articles.filter { !$0.isRead }.count
    }

    /// The ID the reader should show after moving one article forward
    /// (`offset` 1) or back (`offset` -1) from `currentID`, or nil when
    /// there is nowhere to go.
    ///
    /// Moving forward from the last loaded article lands on the end-of-list
    /// page (`endOfListID`) at once, whatever the read state of the list,
    /// and starts a background search for further pages. From the
    /// end-of-list page, moving back returns to the article it follows, and
    /// moving forward continues into any article loaded since. With no
    /// current article, or one that is no longer loaded, the move selects
    /// the first article.
    func destination(from currentID: Int?, offset: Int) -> Int? {
        guard !articles.isEmpty else { return nil }
        if currentID == Self.endOfListID {
            if offset > 0 { return articleAfterEndOfList?.id }
            if let endOfListAnchorID, articles.contains(where: { $0.id == endOfListAnchorID }) {
                return endOfListAnchorID
            }
            return articles[articles.count - 1].id
        }
        guard let index = articles.firstIndex(where: { $0.id == currentID }) else {
            return articles[0].id
        }
        let target = index + offset
        if articles.indices.contains(target) {
            // Approaching the end of the loaded pages: fetch the next one so
            // "next" keeps working past the page boundary.
            if offset > 0, target >= articles.count - 3 {
                Task { await loadMore() }
            }
            return articles[target].id
        }
        guard offset > 0 else { return nil }
        endOfListAnchorID = articles[index].id
        searchForMore()
        return Self.endOfListID
    }

    /// Loads further pages in the background until one adds an article
    /// after the end-of-list anchor, the server has no more, or a fetch
    /// fails. No-op while a search is already running.
    private func searchForMore() {
        guard searchTask == nil, hasMorePages else { return }
        isSearchingForMore = true
        searchTask = Task {
            while !Task.isCancelled, hasMorePages, articleAfterEndOfList == nil {
                guard await loadMore() else { break }
            }
            // A cancelled search has already been cleared, and a newer one
            // may have taken its place.
            guard !Task.isCancelled else { return }
            isSearchingForMore = false
            searchTask = nil
        }
    }

    /// Stops the end-of-list search and forgets its anchor, for a reload
    /// that replaces the list.
    private func resetEndOfList() {
        searchTask?.cancel()
        searchTask = nil
        isSearchingForMore = false
        endOfListAnchorID = nil
    }

    /// Loads the first page, discarding any previously loaded pages. Used
    /// for the initial load, pull-to-refresh, and filter changes.
    func load() async {
        resetEndOfList()
        do {
            let page = try await client.getArticles(feedID: selection.apiFeedID,
                                                    unreadOnly: unreadOnly)
            firstPageCount = page.articles.count
            firstPageCursor = page.nextCursor
            nextCursor = page.nextCursor
            isShowingCachedData = false
            loadFailed = false
            lostConnection = false
            fetchedAt = Date()
            // Content that could not be fetched offline may be fetched now.
            let missingOffline = contentMissingOffline
            contentUnavailable.subtract(missingOffline)
            contentMissingOffline = []
            // Set before the articles so their write to the cache goes
            // through.
            hasLoaded = true
            articles = keepingLoadedContent(page.articles)
            // The reader fetches content when an article opens, so one open
            // now would otherwise wait on a placeholder.
            for id in missingOffline {
                Task { await loadContent(for: id) }
            }
        } catch {
            loadFailed = !(error is CancellationError)
            handle(error)
        }
        hasLoaded = true
    }

    /// `fresh` with the full content already fetched for any of its
    /// articles carried over, which the list API omits. The reader fetches
    /// content only when an article opens, so an article open across a
    /// reload would otherwise lose its content.
    private func keepingLoadedContent(_ fresh: [Article]) -> [Article] {
        var loaded: [Int: String] = [:]
        for article in articles where !article.content.isEmpty {
            loaded[article.id] = article.content
        }
        guard !loaded.isEmpty else { return fresh }
        return fresh.map { article in
            guard article.content.isEmpty, let content = loaded[article.id] else { return article }
            var article = article
            article.content = content
            return article
        }
    }

    /// Appends the next page, and returns whether a page was loaded. While
    /// a page is already loading, waits for that fetch and returns its
    /// result. No-op when the cursor is exhausted.
    @discardableResult
    func loadMore() async -> Bool {
        if let loadMoreTask {
            return await loadMoreTask.value
        }
        guard let cursor = nextCursor, !isOffline else { return false }
        let task = Task { await loadPage(after: cursor) }
        loadMoreTask = task
        let loaded = await task.value
        loadMoreTask = nil
        return loaded
    }

    private func loadPage(after cursor: String) async -> Bool {
        isLoadingMore = true
        defer { isLoadingMore = false }
        do {
            let page = try await client.getArticles(feedID: selection.apiFeedID,
                                                    cursor: cursor,
                                                    unreadOnly: unreadOnly)
            // A refresh or filter change during the fetch replaced the list;
            // this page belongs to the old one.
            guard nextCursor == cursor else { return false }
            // New articles arriving between page fetches can shift the
            // windows; drop any rows the list already has.
            let loadedIDs = Set(articles.map(\.id))
            articles += page.articles.filter { !loadedIDs.contains($0.id) }
            nextCursor = page.nextCursor
            return true
        } catch {
            handle(error)
            return false
        }
    }

    func toggleUnreadFilter() async {
        unreadOnly.toggle()
        // Cleared before the articles, so emptying the list does not
        // overwrite the cache, and the other filter's cache is not written
        // while it is restored.
        hasLoaded = false
        isShowingCachedData = false
        loadFailed = false
        fetchedAt = nil
        firstPageCount = 0
        firstPageCursor = nil
        savesToCache = false
        articles = []
        nextCursor = nil
        restoreFromCache()
        savesToCache = true
        await load()
    }

    /// Fetches the full content of the article with `id` and fills it into
    /// the list, like the web app does on selection: the list API often
    /// omits content, leaving only the truncated description. Safe to call
    /// repeatedly (the reader prefetches neighbours); requests already
    /// satisfied, in flight, or known to fail are no-ops.
    ///
    /// Content fetched once is kept in `LaunchCache` and read from there
    /// afterwards, which is what lets the reader work offline. Without a
    /// connection, an article never fetched shows its description until a
    /// connection returns and the list reloads.
    func loadContent(for id: Int) async {
        guard let index = articles.firstIndex(where: { $0.id == id }),
              articles[index].content.isEmpty,
              !contentFetches.contains(id),
              !contentUnavailable.contains(id) else { return }
        contentFetches.insert(id)
        defer { contentFetches.remove(id) }
        if let cached = await launchCache.loadContent(articleID: id) {
            if let index = articles.firstIndex(where: { $0.id == id }) {
                articles[index].content = cached
            }
            return
        }
        guard !isOffline else {
            markContentMissingOffline(id)
            return
        }
        do {
            let full = try await client.getArticle(id: id)
            guard let index = articles.firstIndex(where: { $0.id == id }) else { return }
            if full.content.isEmpty {
                contentUnavailable.insert(id)
            } else {
                articles[index].content = full.content
                launchCache.saveContent(full.content, articleID: id)
            }
        } catch {
            // A cancelled fetch (navigating away mid-load) may be retried
            // later; only real failures fall back to the description.
            guard !Task.isCancelled else { return }
            switch error {
            case NetworkError.unauthorized:
                onSessionExpired()
                contentUnavailable.insert(id)
            case NetworkError.noConnection:
                lostConnection = true
                markContentMissingOffline(id)
            default:
                contentUnavailable.insert(id)
            }
        }
    }

    private func markContentMissingOffline(_ id: Int) {
        contentUnavailable.insert(id)
        contentMissingOffline.insert(id)
    }

    /// Marks `article` read, optimistically in the list and then on the
    /// server. Articles already read are left alone so re-opening one does
    /// not disturb server-side unread counts. Offline the article stays
    /// unread, so opening it from the cache changes nothing.
    func markRead(_ article: Article) async {
        await setRead(article, isRead: true)
    }

    /// Sets the read state of `article`, optimistically in the list and then
    /// on the server, reverting on failure. No-op when the state already
    /// matches, so server-side unread counts are never double-adjusted, and
    /// while offline.
    func setRead(_ article: Article, isRead: Bool) async {
        guard !isOffline,
              article.isRead != isRead,
              let index = articles.firstIndex(where: { $0.id == article.id }) else { return }
        articles[index].isRead = isRead
        reportUnreadChange(for: article, isRead: isRead)
        do {
            try await client.markRead(articleID: article.id,
                                      isRead: isRead,
                                      feedID: article.feedId,
                                      wasRead: !isRead)
        } catch {
            if let index = articles.firstIndex(where: { $0.id == article.id }) {
                articles[index].isRead = !isRead
            }
            reportUnreadChange(for: article, isRead: !isRead)
            handle(error)
        }
    }

    private func reportUnreadChange(for article: Article, isRead: Bool) {
        guard article.publishedAt >= Date().addingTimeInterval(-Self.unreadCountWindow) else { return }
        onUnreadCountChange(article.feedId, isRead ? -1 : 1)
    }

    /// Toggles the star on `article`, optimistically in the list and then on
    /// the server, reverting on failure. No-op while offline.
    func toggleStar(_ article: Article) async {
        guard !isOffline, let index = articles.firstIndex(where: { $0.id == article.id }) else { return }
        articles[index].isStarred.toggle()
        do {
            try await client.toggleStar(articleID: article.id)
        } catch {
            if let index = articles.firstIndex(where: { $0.id == article.id }) {
                articles[index].isStarred.toggle()
            }
            handle(error)
        }
    }

    /// Marks every article read. The endpoint is account-wide, so this is
    /// only offered from the All Articles list; loaded rows update in place
    /// after the server confirms.
    func markAllRead() async {
        guard !isOffline else { return }
        do {
            try await client.markAllRead()
            for index in articles.indices {
                articles[index].isRead = true
            }
        } catch {
            handle(error)
        }
    }

    /// Writes the first page back to the cache after any change to it, such
    /// as a read or star toggle. Full content is left out: it is fetched
    /// again when an article opens, and would make every write large.
    /// The full content is kept separately, per article.
    private func saveToCache() {
        guard savesToCache, isReady else { return }
        let page = articles.prefix(firstPageCount).map { article in
            var article = article
            article.content = ""
            return article
        }
        launchCache.saveArticles(.init(articles: page, nextCursor: firstPageCursor, fetchedAt: fetchedAt),
                                 for: selection,
                                 unreadOnly: unreadOnly)
    }

    private func handle(_ error: Error) {
        if error is CancellationError { return }
        switch error {
        case NetworkError.unauthorized:
            onSessionExpired()
            return
        case NetworkError.noConnection:
            lostConnection = true
            // The list notes that it is offline; an alert adds nothing
            // while there are articles to read.
            if !articles.isEmpty { return }
        default:
            break
        }
        errorMessage = error.localizedDescription
    }
}
