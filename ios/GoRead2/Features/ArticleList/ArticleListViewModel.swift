import Foundation

/// State for the article list screen: one page-at-a-time cursor pagination
/// over /api/feeds/:id/articles, an unread-only filter, and optimistic
/// mark-read on opening an article.
@MainActor
final class ArticleListViewModel: ObservableObject {
    @Published private(set) var articles: [Article] = []
    /// True once the first load has completed, successfully or not; gates
    /// the empty state so it never flashes before data arrives.
    @Published private(set) var hasLoaded = false
    @Published private(set) var isLoadingMore = false
    @Published private(set) var unreadOnly = false
    @Published var errorMessage: String?

    /// Called when the API reports 401: the session is gone server-side and
    /// the app should return to the login screen.
    var onSessionExpired: () -> Void = {}

    private let selection: FeedSelection
    private let client: NetworkClient
    private var nextCursor: String?

    init(selection: FeedSelection, client: NetworkClient = .shared) {
        self.selection = selection
        self.client = client
    }

    /// True while the server reports another page after the loaded ones.
    var hasMorePages: Bool {
        nextCursor != nil
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

    /// Appends the next page. No-op while a page is already loading or when
    /// the cursor is exhausted.
    func loadMore() async {
        guard let cursor = nextCursor, !isLoadingMore else { return }
        isLoadingMore = true
        defer { isLoadingMore = false }
        do {
            let page = try await client.getArticles(feedID: selection.apiFeedID,
                                                    cursor: cursor,
                                                    unreadOnly: unreadOnly)
            // New articles arriving between page fetches can shift the
            // windows; drop any rows the list already has.
            let loadedIDs = Set(articles.map(\.id))
            articles += page.articles.filter { !loadedIDs.contains($0.id) }
            nextCursor = page.nextCursor
        } catch {
            handle(error)
        }
    }

    func toggleUnreadFilter() async {
        unreadOnly.toggle()
        articles = []
        nextCursor = nil
        hasLoaded = false
        await load()
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
        do {
            try await client.markRead(articleID: article.id,
                                      isRead: isRead,
                                      feedID: article.feedId,
                                      wasRead: !isRead)
        } catch {
            if let index = articles.firstIndex(where: { $0.id == article.id }) {
                articles[index].isRead = !isRead
            }
            handle(error)
        }
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
        if case NetworkError.unauthorized = error {
            onSessionExpired()
            return
        }
        errorMessage = error.localizedDescription
    }
}
