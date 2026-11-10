import Foundation

/// State for the article list screen: one page-at-a-time cursor pagination
/// over /api/feeds/:id/articles, an unread-only filter, and optimistic
/// mark-read on opening an article.
@MainActor
final class ArticleListViewModel: ObservableObject {
    /// Sentinel article ID for the position one past the last article, where
    /// the reader shows the end-of-list page instead of an article. Real
    /// article IDs from the server are positive.
    static let endOfListID = -1

    @Published private(set) var articles: [Article] = []
    /// True once the first load has completed, successfully or not; gates
    /// the empty state so it never flashes before data arrives.
    @Published private(set) var hasLoaded = false
    @Published private(set) var isLoadingMore = false
    /// True while a move past the last loaded article waits for the next
    /// page; the reader shows progress instead of appearing to ignore the
    /// move.
    @Published private(set) var isLoadingNextArticle = false
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
    private var nextCursor: String?
    /// The next-page fetch in flight, shared so a second caller waits for it
    /// instead of returning early.
    private var loadMoreTask: Task<Bool, Never>?
    /// Full-content fetches currently in flight, keyed by article ID.
    private var contentFetches: Set<Int> = []

    init(selection: FeedSelection, client: NetworkClient = .shared) {
        self.selection = selection
        self.client = client
        if case .all = selection {
            unreadOnly = true
        } else {
            unreadOnly = false
        }
    }

    /// True while the server reports another page after the loaded ones.
    var hasMorePages: Bool {
        nextCursor != nil
    }

    /// True when every loaded article is read and the server has no further
    /// pages. The end-of-list page words itself as "all caught up" only
    /// then; otherwise it reports how many articles are still unread.
    var isCaughtUp: Bool {
        !hasMorePages && articles.allSatisfy(\.isRead)
    }

    /// The number of loaded articles still unread.
    var unreadLoadedCount: Int {
        articles.filter { !$0.isRead }.count
    }

    /// The ID the reader should show after moving one article forward
    /// (`offset` 1) or back (`offset` -1) from `currentID`, or nil when
    /// there is nowhere to go.
    ///
    /// Moving forward from the last loaded article waits for the next page
    /// when the server has one, and otherwise lands on the end-of-list page
    /// (`endOfListID`) whatever the read state of the list. It returns nil
    /// there only when the next page fails to load, which is reported
    /// through `errorMessage`. From the end-of-list page, moving back
    /// returns to the last article. With no current article, or one that is
    /// no longer loaded, the move selects the first article.
    func destination(from currentID: Int?, offset: Int) async -> Int? {
        guard !articles.isEmpty else { return nil }
        if currentID == Self.endOfListID {
            return offset < 0 ? articles[articles.count - 1].id : nil
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

        isLoadingNextArticle = hasMorePages
        defer { isLoadingNextArticle = false }
        while hasMorePages {
            let loaded = await loadMore()
            // A refresh during the wait can drop the current article.
            guard let index = articles.firstIndex(where: { $0.id == currentID }) else { return nil }
            if articles.indices.contains(index + 1) {
                return articles[index + 1].id
            }
            guard loaded else { return nil }
        }
        return Self.endOfListID
    }

    /// Loads the first page, discarding any previously loaded pages. Used
    /// for the initial load, pull-to-refresh, and filter changes.
    func load() async {
        do {
            let page = try await client.getArticles(feedID: selection.apiFeedID,
                                                    unreadOnly: unreadOnly)
            articles = page.articles
            nextCursor = page.nextCursor
        } catch {
            handle(error)
        }
        hasLoaded = true
    }

    /// Appends the next page, and returns whether a page was loaded. While
    /// a page is already loading, waits for that fetch and returns its
    /// result. No-op when the cursor is exhausted.
    @discardableResult
    func loadMore() async -> Bool {
        if let loadMoreTask {
            return await loadMoreTask.value
        }
        guard let cursor = nextCursor else { return false }
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
        articles = []
        nextCursor = nil
        hasLoaded = false
        await load()
    }

    /// Fetches the full content of the article with `id` and fills it into
    /// the list, like the web app does on selection: the list API often
    /// omits content, leaving only the truncated description. Safe to call
    /// repeatedly (the reader prefetches neighbours); requests already
    /// satisfied, in flight, or known to fail are no-ops.
    func loadContent(for id: Int) async {
        guard let index = articles.firstIndex(where: { $0.id == id }),
              articles[index].content.isEmpty,
              !contentFetches.contains(id),
              !contentUnavailable.contains(id) else { return }
        contentFetches.insert(id)
        defer { contentFetches.remove(id) }
        do {
            let full = try await client.getArticle(id: id)
            guard let index = articles.firstIndex(where: { $0.id == id }) else { return }
            if full.content.isEmpty {
                contentUnavailable.insert(id)
            } else {
                articles[index].content = full.content
            }
        } catch {
            // A cancelled fetch (navigating away mid-load) may be retried
            // later; only real failures fall back to the description.
            guard !Task.isCancelled else { return }
            if case NetworkError.unauthorized = error {
                onSessionExpired()
            }
            contentUnavailable.insert(id)
        }
    }

    /// Marks `article` read, optimistically in the list and then on the
    /// server. Articles already read are left alone so re-opening one does
    /// not disturb server-side unread counts.
    func markRead(_ article: Article) async {
        await setRead(article, isRead: true)
    }

    /// Sets the read state of `article`, optimistically in the list and then
    /// on the server, reverting on failure. No-op when the state already
    /// matches, so server-side unread counts are never double-adjusted.
    func setRead(_ article: Article, isRead: Bool) async {
        guard article.isRead != isRead,
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
    /// the server, reverting on failure.
    func toggleStar(_ article: Article) async {
        guard let index = articles.firstIndex(where: { $0.id == article.id }) else { return }
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
        do {
            try await client.markAllRead()
            for index in articles.indices {
                articles[index].isRead = true
            }
        } catch {
            handle(error)
        }
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
