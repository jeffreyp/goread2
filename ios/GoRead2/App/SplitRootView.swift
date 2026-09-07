import SwiftUI
import UniformTypeIdentifiers

/// iPad and Mac root: a three-column split view (feeds sidebar, article
/// list, reader) mirroring the web app's three-pane layout. The system-provided
/// sidebar toggle collapses columns, and in portrait the sidebar overlays
/// instead of tiling. Hardware keyboard shortcuts match the web app:
/// j/k select the next/previous article, m toggles read, s toggles the
/// star, and r refreshes every feed.
///
/// The sidebar starts with nothing selected; once the feed list finishes
/// loading, `selectAllArticlesIfNeeded` picks All Articles for an account
/// with subscriptions, or opens the sidebar for a brand-new account so its
/// welcome screen is reachable instead of an empty All Articles list.
///
/// On macOS this view is also the menu bar's target: it publishes its
/// actions as a focused scene value for `ReaderCommands` and hosts the
/// sheets, panels, and dialogs those commands raise.
struct SplitRootView: View {
    @StateObject private var feedViewModel = FeedListViewModel()
    @State private var columnVisibility: NavigationSplitViewVisibility = .automatic
    @State private var feedSelection: FeedSelection?
    @State private var articleViewModel: ArticleListViewModel?
    @State private var selectedArticleID: Int?

    #if os(macOS)
    @EnvironmentObject private var authManager: AuthManager
    /// The File menu's OPML commands, which reach the API without the
    /// Settings window being open.
    @StateObject private var opml = OPMLActionModel()
    @State private var showingAddFeed = false
    @State private var showingMarkAllReadConfirmation = false
    #endif

    var body: some View {
        menuBarSupport(splitView)
    }

    private var splitView: some View {
        NavigationSplitView(columnVisibility: $columnVisibility) {
            FeedListView(viewModel: feedViewModel,
                         sidebarSelection: $feedSelection,
                         refreshAction: refreshAllPanes)
                .splitColumnWidth(min: 180, ideal: 240, max: 360)
        } content: {
            if let articleViewModel, let feedSelection {
                ArticleListView(viewModel: articleViewModel,
                                selection: feedSelection,
                                selectedArticleID: $selectedArticleID,
                                refreshAction: refreshAllPanes)
                    .id(feedSelection)
                    .splitColumnWidth(min: 260, ideal: 340, max: 520)
            } else {
                EmptyStateView(systemImage: "tray.full",
                               title: "No Feed Selected",
                               message: "Choose a feed from the sidebar.")
                    .splitColumnWidth(min: 260, ideal: 340, max: 520)
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
        .onChange(of: feedViewModel.hasLoaded) { _ in selectAllArticlesIfNeeded() }
        .onChange(of: feedViewModel.feeds.count) { _ in selectAllArticlesIfNeeded() }
        .onChange(of: selectedArticleID) { _ in
            // Opening an article marks it read; keep the sidebar's unread
            // badges current.
            guard feedViewModel.hasLoaded else { return }
            Task { await feedViewModel.refreshUnreadCounts() }
        }
        .background(shortcutButtons)
    }

    /// Picks the initial sidebar selection once the feed list load settles:
    /// All Articles for an account with subscriptions (so unread articles
    /// are visible without a trip to the sidebar), or nothing for a
    /// brand-new account, forcing the sidebar open so the welcome screen's
    /// "Add Your First Feed" is reachable instead of hidden behind an empty
    /// All Articles list. Only runs once, before any explicit selection.
    private func selectAllArticlesIfNeeded() {
        guard feedViewModel.hasLoaded, feedSelection == nil else { return }
        if feedViewModel.feeds.isEmpty {
            columnVisibility = .all
        } else {
            feedSelection = .all
        }
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

    // MARK: - Menu bar

    #if os(macOS)
    /// Everything the File and View menu commands need: the presentations
    /// they raise, and the actions themselves, published for whichever
    /// window is frontmost.
    ///
    /// SwiftUI honours one sheet-style presentation per view, and on macOS
    /// the file panels, the confirmation dialog, and the alert are all
    /// sheets. Attaching them to one view leaves only the first of them
    /// working, so each hangs off its own zero-size background instead.
    @ViewBuilder
    private func menuBarSupport<Content: View>(_ content: Content) -> some View {
        content
            .sheet(isPresented: $showingAddFeed) {
                AddFeedView { url in
                    try await feedViewModel.addFeed(url: url)
                }
            }
            .background(markAllReadConfirmation)
            .background(opmlImporter)
            .background(opmlExporter)
            .background(opmlAlert)
            .focusedSceneValue(\.readerActions, readerActions)
            .task {
                opml.onSessionExpired = { authManager.sessionExpired() }
                opml.onImported = { await feedViewModel.refresh() }
            }
    }

    private var markAllReadConfirmation: some View {
        presentationHost
            .confirmationDialog("Mark all articles as read?",
                                isPresented: $showingMarkAllReadConfirmation,
                                titleVisibility: .visible) {
                Button("Mark All Read") { markAllRead() }
            }
    }

    private var opmlImporter: some View {
        presentationHost
            .fileImporter(isPresented: $opml.isImporting,
                          allowedContentTypes: UTType.opmlImportTypes) { result in
                switch result {
                case .success(let url):
                    Task { await opml.importFile(at: url) }
                case .failure(let error):
                    opml.report(error)
                }
            }
    }

    private var opmlExporter: some View {
        presentationHost
            .fileExport(item: $opml.export) { opml.report(message: $0) }
    }

    private var opmlAlert: some View {
        presentationHost
            .alert(opml.alert?.title ?? "", isPresented: alertBinding) {
                Button("OK", role: .cancel) {}
            } message: {
                Text(opml.alert?.message ?? "")
            }
    }

    /// Carries one presentation and nothing else, so it never covers or
    /// intercepts anything in the window it sits behind.
    private var presentationHost: some View {
        Color.clear.allowsHitTesting(false)
    }

    private var readerActions: ReaderActions {
        ReaderActions(
            addFeed: { showingAddFeed = true },
            importOPML: { opml.beginImport() },
            exportOPML: { Task { await opml.exportSubscriptions() } },
            refresh: { Task { await refreshAllPanes() } },
            selectNextArticle: articleViewModel.map { _ in { moveSelection(1) } },
            selectPreviousArticle: articleViewModel.map { _ in { moveSelection(-1) } },
            unreadOnly: articleViewModel.map { viewModel in
                Binding(
                    get: { viewModel.unreadOnly },
                    set: { _ in Task { await viewModel.toggleUnreadFilter() } }
                )
            },
            // The endpoint is account-wide, so the command follows the
            // article list's rule of offering this from All Articles only.
            markAllRead: feedSelection == .all && articleViewModel != nil
                ? { showingMarkAllReadConfirmation = true }
                : nil
        )
    }

    /// Marks every article read from the menu, then refreshes the sidebar's
    /// unread badges, which the account-wide endpoint has just zeroed.
    private func markAllRead() {
        guard let articleViewModel else { return }
        Task {
            await articleViewModel.markAllRead()
            await feedViewModel.refreshUnreadCounts()
        }
    }

    private var alertBinding: Binding<Bool> {
        Binding(
            get: { opml.alert != nil },
            set: { if !$0 { opml.alert = nil } }
        )
    }
    #else
    /// iOS has no menu bar, so the window needs no command wiring.
    private func menuBarSupport<Content: View>(_ content: Content) -> Content {
        content
    }
    #endif
}

#Preview {
    SplitRootView()
        .environmentObject(AuthManager())
}
