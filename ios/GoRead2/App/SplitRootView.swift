import SwiftUI

/// iPad root: a three-column split view (feeds sidebar, article list,
/// reader) mirroring the web app's three-pane layout. The system-provided
/// sidebar toggle collapses columns, and in portrait the sidebar overlays
/// instead of tiling. Hardware keyboard shortcuts match the web app:
/// j/k select the next/previous article, m toggles read, s toggles the
/// star, and r refreshes every feed.
struct SplitRootView: View {
    @StateObject private var feedViewModel = FeedListViewModel()
    @State private var columnVisibility: NavigationSplitViewVisibility = .automatic
    @State private var feedSelection: FeedSelection?
    @State private var articleViewModel: ArticleListViewModel?
    @State private var selectedArticleID: Int?

    /// Launches with All Articles selected so unread articles are visible
    /// immediately, without a trip to the sidebar (which starts hidden in
    /// portrait).
    init() {
        _feedSelection = State(initialValue: .all)
        _articleViewModel = State(initialValue: ArticleListViewModel(selection: .all))
    }

    var body: some View {
        NavigationSplitView(columnVisibility: $columnVisibility) {
            FeedListView(viewModel: feedViewModel,
                         sidebarSelection: $feedSelection,
                         refreshAction: refreshAllPanes)
        } content: {
            if let articleViewModel, let feedSelection {
                ArticleListView(viewModel: articleViewModel,
                                selection: feedSelection,
                                selectedArticleID: $selectedArticleID,
                                refreshAction: refreshAllPanes)
                    .id(feedSelection)
            } else {
                EmptyStateView(systemImage: "tray.full",
                               title: "No Feed Selected",
                               message: "Choose a feed from the sidebar.")
            }
        } detail: {
            if let articleViewModel, let articleID = selectedArticleID {
                ArticleReaderView(viewModel: articleViewModel,
                                  currentID: readerSelection(fallback: articleID))
            } else {
                EmptyStateView(systemImage: "doc.text",
                               title: "No Article Selected",
                               message: "Choose an article from the list.")
            }
        }
        .navigationSplitViewStyle(.balanced)
        .onChange(of: feedSelection) { selection in
            selectedArticleID = nil
            articleViewModel = selection.map { ArticleListViewModel(selection: $0) }
        }
        .onChange(of: selectedArticleID) { _ in
            // Opening an article marks it read; keep the sidebar's unread
            // badges current.
            guard feedViewModel.hasLoaded else { return }
            Task { await feedViewModel.refreshUnreadCounts() }
        }
        .background(shortcutButtons)
    }

    /// Pull-to-refresh (either pane) and the r shortcut: one server-side
    /// refresh that updates all three panes. The sidebar reloads its feeds
    /// and counts, the article list re-queries, and the reader opens the
    /// first newly arrived unread article.
    private func refreshAllPanes() async {
        await feedViewModel.refresh()
        guard let articleViewModel else { return }
        let knownIDs = Set(articleViewModel.articles.map(\.id))
        await articleViewModel.load()
        // The selection changing mid-refresh replaces the view model; the
        // new articles belong to the old list, so leave the reader alone.
        guard articleViewModel === self.articleViewModel,
              let firstNew = articleViewModel.articles.first(where: {
                  !$0.isRead && !knownIDs.contains($0.id)
              }) else { return }
        selectedArticleID = firstNew.id
    }

    /// The reader drives article selection through this binding, so its
    /// previous/next controls also move the list highlight.
    private func readerSelection(fallback articleID: Int) -> Binding<Int> {
        Binding(
            get: { selectedArticleID ?? articleID },
            set: { selectedArticleID = $0 }
        )
    }

    // MARK: - Keyboard shortcuts

    /// Buttons with zero opacity still register their keyboard shortcuts,
    /// which is what makes these work without visible chrome.
    private var shortcutButtons: some View {
        Group {
            Button("Next Article") { moveSelection(1) }
                .keyboardShortcut("j", modifiers: [])
            Button("Previous Article") { moveSelection(-1) }
                .keyboardShortcut("k", modifiers: [])
            Button("Mark Read or Unread") { toggleSelectedRead() }
                .keyboardShortcut("m", modifiers: [])
            Button("Star or Unstar") { toggleSelectedStar() }
                .keyboardShortcut("s", modifiers: [])
            Button("Refresh Feeds") { Task { await refreshAllPanes() } }
                .keyboardShortcut("r", modifiers: [])
        }
        .opacity(0)
        .accessibilityHidden(true)
    }

    private var selectedArticle: Article? {
        guard let articleViewModel, let selectedArticleID else { return nil }
        return articleViewModel.articles.first { $0.id == selectedArticleID }
    }

    private func moveSelection(_ offset: Int) {
        guard let viewModel = articleViewModel else { return }
        let articles = viewModel.articles
        guard !articles.isEmpty else { return }
        // From the caught-up screen, k returns to the last article and j
        // stays put.
        if selectedArticleID == ArticleReaderView.caughtUpID {
            if offset < 0 {
                selectedArticleID = articles[articles.count - 1].id
            }
            return
        }
        var newIndex = 0
        if let selectedArticleID,
           let index = articles.firstIndex(where: { $0.id == selectedArticleID }) {
            newIndex = index + offset
        }
        guard articles.indices.contains(newIndex) else {
            // One past the last article is the caught-up screen, reachable
            // once nothing unread remains.
            if offset > 0, newIndex == articles.count, viewModel.isCaughtUp {
                selectedArticleID = ArticleReaderView.caughtUpID
            }
            return
        }
        selectedArticleID = articles[newIndex].id
        // Approaching the end of the loaded pages: fetch the next one so
        // "next" keeps working past the page boundary.
        if offset > 0, newIndex >= articles.count - 3 {
            Task { await viewModel.loadMore() }
        }
    }

    private func toggleSelectedRead() {
        guard let viewModel = articleViewModel, let article = selectedArticle else { return }
        Task { await viewModel.setRead(article, isRead: !article.isRead) }
    }

    private func toggleSelectedStar() {
        guard let viewModel = articleViewModel, let article = selectedArticle else { return }
        Task { await viewModel.toggleStar(article) }
    }
}

#Preview {
    SplitRootView()
        .environmentObject(AuthManager())
}
