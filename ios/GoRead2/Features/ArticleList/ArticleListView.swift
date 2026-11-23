import SwiftUI

/// Paginated article list for one feed or the merged "All Articles" stream:
/// infinite scroll via cursor pagination, an unread-only filter, and
/// pull-to-refresh. Opening an article marks it read immediately. Rows offer
/// star (leading) and mark read/unread (trailing) swipe actions, and the
/// All Articles list carries a mark-all-read toolbar button (the endpoint is
/// account-wide, so per-feed lists do not offer it).
///
/// When the server cannot be reached, a cached list stays on screen under a
/// banner giving the time it was fetched, and the actions that need the
/// server are disabled until a connection returns.
///
/// On iPhone the rows are navigation links that push the reader; on iPad,
/// where `selectedArticleID` is provided, the rows drive the split view's
/// article selection instead, and the first unread article is pre-selected
/// once the initial load completes.
struct ArticleListView: View {
    @EnvironmentObject private var authManager: AuthManager
    @ObservedObject var viewModel: ArticleListViewModel
    let selection: FeedSelection
    var selectedArticleID: Binding<Int?>?
    /// Replaces the default pull-to-refresh (re-querying this list). The
    /// iPad split view passes an action that refreshes every pane at once.
    var refreshAction: (() async -> Void)?
    /// Replaces the default mark-all-read (updating this list only). The
    /// iPad split view passes an action that also refreshes the sidebar's
    /// unread badges.
    var markAllReadAction: (() async -> Void)?

    @State private var showMarkAllReadConfirmation = false

    var body: some View {
        Group {
            if !viewModel.isReady {
                ProgressView()
            } else if viewModel.articles.isEmpty, viewModel.loadFailed {
                loadFailedState
            } else if viewModel.articles.isEmpty {
                emptyState
            } else {
                articleList
            }
        }
        .navigationTitle(selection.title)
        .inlineNavigationTitle()
        .toolbar {
            ToolbarItemGroup(placement: .primaryAction) {
                if case .all = selection {
                    Button {
                        showMarkAllReadConfirmation = true
                    } label: {
                        Label("Mark All Read", systemImage: "checkmark.circle")
                    }
                    .disabled(viewModel.isOffline)
                }
                Button {
                    Task { await viewModel.toggleUnreadFilter() }
                } label: {
                    Label("Unread Only",
                          systemImage: viewModel.unreadOnly
                              ? "line.3.horizontal.decrease.circle.fill"
                              : "line.3.horizontal.decrease.circle")
                }
            }
        }
        .confirmationDialog("Mark all articles as read?",
                            isPresented: $showMarkAllReadConfirmation,
                            titleVisibility: .visible) {
            Button("Mark All Read") {
                Task {
                    if let markAllReadAction {
                        await markAllReadAction()
                    } else {
                        await viewModel.markAllRead()
                    }
                }
            }
        }
        .alert("Error", isPresented: errorBinding) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(viewModel.errorMessage ?? "")
        }
        .task {
            viewModel.onSessionExpired = { authManager.sessionExpired() }
            // .task re-fires when a pushed reader screen pops; only the
            // first appearance should load, or pagination state would reset
            // mid-scroll. Cached data from the last session does not count
            // as loaded, so it is always revalidated.
            guard !viewModel.hasLoaded else { return }
            await viewModel.load()
            // On iPad, pre-select the first unread article so the reader
            // pane isn't empty on launch, matching Mail/News-style split
            // views. iPhone has no selectedArticleID binding, and an
            // existing selection (e.g. restored across a feed re-selection)
            // is left alone. This waits for the server's list rather than
            // using the cached one: opening an article marks it read, and a
            // cached article may since have been read on another device.
            if let selectedArticleID, selectedArticleID.wrappedValue == nil {
                selectedArticleID.wrappedValue = viewModel.articles.first(where: { !$0.isRead })?.id
            }
        }
    }

    private var articleList: some View {
        Group {
            if let selectedArticleID {
                // List(selection:) highlights the row on a programmatic
                // selection change (e.g. the j/k shortcuts in SplitRootView)
                // but does not scroll to reveal it, so the highlight can
                // move off-screen while paging. Follow it explicitly.
                ScrollViewReader { proxy in
                    List(selection: selectedArticleID) { articleRows }
                        .onChange(of: selectedArticleID.wrappedValue) { newValue in
                            guard let newValue else { return }
                            withAnimation {
                                proxy.scrollTo(newValue, anchor: .center)
                            }
                        }
                }
            } else {
                List { articleRows }
            }
        }
        .listStyle(.plain)
        .safeAreaInset(edge: .top, spacing: 0) {
            if viewModel.isShowingStaleData {
                OfflineBanner(fetchedAt: viewModel.fetchedAt, isOffline: viewModel.isOffline)
            }
        }
        .refreshable {
            if let refreshAction {
                await refreshAction()
            } else {
                await viewModel.load()
            }
        }
    }

    @ViewBuilder
    private var articleRows: some View {
        ForEach(viewModel.articles) { article in
            row(for: article)
                .swipeActions(edge: .leading) {
                    Button {
                        Task { await viewModel.toggleStar(article) }
                    } label: {
                        Label(article.isStarred ? "Unstar" : "Star",
                              systemImage: article.isStarred ? "star.slash" : "star")
                    }
                    .tint(.yellow)
                    .disabled(viewModel.isOffline)
                }
                .swipeActions(edge: .trailing) {
                    Button {
                        Task { await viewModel.setRead(article, isRead: !article.isRead) }
                    } label: {
                        Label(article.isRead ? "Mark Unread" : "Mark Read",
                              systemImage: article.isRead ? "circle.fill" : "checkmark.circle")
                    }
                    .tint(.blue)
                    .disabled(viewModel.isOffline)
                }
        }

        // Offline the next page cannot load, so the row would spin forever.
        if viewModel.hasMorePages, !viewModel.isOffline {
            HStack {
                Spacer()
                ProgressView()
                Spacer()
            }
            .onAppear {
                Task { await viewModel.loadMore() }
            }
        }
    }

    @ViewBuilder
    private func row(for article: Article) -> some View {
        if selectedArticleID != nil {
            ArticleRow(article: article)
                .tag(article.id)
        } else {
            NavigationLink(value: article) {
                ArticleRow(article: article)
            }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 16) {
            EmptyStateView(systemImage: viewModel.unreadOnly ? "checkmark.circle" : "tray",
                           title: viewModel.unreadOnly ? "All Caught Up!" : "No Articles Yet",
                           message: viewModel.unreadOnly
                               ? "You've read all your articles. Great job! New articles will appear here as they're published."
                               : "This feed doesn't have any articles yet. Check back later for new content.")
            // The toolbar filter also switches modes, but it is easy to miss
            // from an empty screen, so the caught-up state offers its own way
            // to reach the read articles.
            if viewModel.unreadOnly {
                Button("View All Articles") {
                    Task { await viewModel.toggleUnreadFilter() }
                }
            }
        }
    }

    /// A failed load with nothing cached for this list, worded like the
    /// feed list's, rather than an empty state that would suggest the list
    /// has no articles.
    private var loadFailedState: some View {
        VStack(spacing: 8) {
            EmptyStateView(systemImage: "wifi.exclamationmark",
                           title: "Couldn't Load Articles",
                           message: "Check the connection and try again.")
            Button("Try Again") {
                Task { await viewModel.load() }
            }
            .buttonStyle(.borderedProminent)
        }
    }

    private var errorBinding: Binding<Bool> {
        Binding(
            get: { viewModel.errorMessage != nil },
            set: { if !$0 { viewModel.errorMessage = nil } }
        )
    }
}

/// iPhone wrapper for the pushed article list: owns the view model for the
/// screen's lifetime and hosts the reader's push destination. It keeps the
/// feed list underneath current without that list refetching when it
/// reappears: read-state changes are reported through `onUnreadCountChange`,
/// and pull-to-refresh and Mark All Read end with `refreshUnreadCounts`.
struct ArticleListScreen: View {
    @StateObject private var viewModel: ArticleListViewModel

    private let selection: FeedSelection
    private let onUnreadCountChange: (_ feedID: Int, _ delta: Int) -> Void
    private let refreshUnreadCounts: () async -> Void

    init(selection: FeedSelection,
         onUnreadCountChange: @escaping (_ feedID: Int, _ delta: Int) -> Void = { _, _ in },
         refreshUnreadCounts: @escaping () async -> Void = {}) {
        self.selection = selection
        self.onUnreadCountChange = onUnreadCountChange
        self.refreshUnreadCounts = refreshUnreadCounts
        _viewModel = StateObject(wrappedValue: ArticleListViewModel(selection: selection))
    }

    var body: some View {
        ArticleListView(viewModel: viewModel,
                        selection: selection,
                        refreshAction: {
                            await viewModel.load()
                            await refreshUnreadCounts()
                        },
                        markAllReadAction: {
                            await viewModel.markAllRead()
                            await refreshUnreadCounts()
                        })
            .navigationDestination(for: Article.self) { article in
                ArticleReaderScreen(viewModel: viewModel, articleID: article.id)
            }
            .onAppear {
                viewModel.onUnreadCountChange = onUnreadCountChange
            }
    }
}

/// One line in the article list: unread indicator, title, feed name (for the
/// merged stream) and publication date.
private struct ArticleRow: View {
    let article: Article

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            Circle()
                .fill(.blue)
                .frame(width: 8, height: 8)
                .padding(.top, 6)
                .opacity(article.isRead ? 0 : 1)

            VStack(alignment: .leading, spacing: 2) {
                Text(article.title)
                    .font(.headline)
                    .fontWeight(article.isRead ? .regular : .semibold)
                    .foregroundStyle(article.isRead ? .secondary : .primary)
                    .lineLimit(2)

                HStack(spacing: 4) {
                    if article.isStarred {
                        Image(systemName: "star.fill")
                            .foregroundStyle(.yellow)
                    }
                    if let feedTitle = article.feedTitle, !feedTitle.isEmpty {
                        Text(feedTitle)
                            .lineLimit(1)
                        Text("·")
                    }
                    Text(article.publishedAt, format: .relative(presentation: .named))
                }
                .font(.caption)
                .foregroundStyle(.secondary)
            }
        }
        .padding(.vertical, 2)
    }
}

#Preview {
    NavigationStack {
        ArticleListScreen(selection: .all)
            .environmentObject(AuthManager())
    }
}
