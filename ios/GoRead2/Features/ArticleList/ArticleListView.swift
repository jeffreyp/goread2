import SwiftUI

/// Paginated article list for one feed or the merged "All Articles" stream:
/// infinite scroll via cursor pagination, an unread-only filter, and
/// pull-to-refresh. Opening an article marks it read immediately. Rows offer
/// star (leading) and mark read/unread (trailing) swipe actions, and the
/// All Articles list carries a mark-all-read toolbar button (the endpoint is
/// account-wide, so per-feed lists do not offer it).
///
/// On iPhone the rows are navigation links that push the reader; on iPad,
/// where `selectedArticleID` is provided, the rows drive the split view's
/// article selection instead.
struct ArticleListView: View {
    @EnvironmentObject private var authManager: AuthManager
    @ObservedObject var viewModel: ArticleListViewModel
    let selection: FeedSelection
    var selectedArticleID: Binding<Int?>?

    @State private var showMarkAllReadConfirmation = false

    var body: some View {
        Group {
            if !viewModel.hasLoaded {
                ProgressView()
            } else if viewModel.articles.isEmpty {
                emptyState
            } else {
                articleList
            }
        }
        .navigationTitle(selection.title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItemGroup(placement: .primaryAction) {
                if case .all = selection {
                    Button {
                        showMarkAllReadConfirmation = true
                    } label: {
                        Label("Mark All Read", systemImage: "checkmark.circle")
                    }
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
                Task { await viewModel.markAllRead() }
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
            // mid-scroll.
            guard !viewModel.hasLoaded else { return }
            await viewModel.load()
        }
    }

    private var articleList: some View {
        Group {
            if let selectedArticleID {
                List(selection: selectedArticleID) { articleRows }
            } else {
                List { articleRows }
            }
        }
        .listStyle(.plain)
        .refreshable {
            await viewModel.load()
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
                }
                .swipeActions(edge: .trailing) {
                    Button {
                        Task { await viewModel.setRead(article, isRead: !article.isRead) }
                    } label: {
                        Label(article.isRead ? "Mark Unread" : "Mark Read",
                              systemImage: article.isRead ? "circle.fill" : "checkmark.circle")
                    }
                    .tint(.blue)
                }
        }

        if viewModel.hasMorePages {
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
        EmptyStateView(systemImage: viewModel.unreadOnly ? "checkmark.circle" : "tray",
                       title: viewModel.unreadOnly ? "All Caught Up" : "No Articles",
                       message: viewModel.unreadOnly
                           ? "Every article here has been read."
                           : "Articles will appear once this feed has content.")
    }

    private var errorBinding: Binding<Bool> {
        Binding(
            get: { viewModel.errorMessage != nil },
            set: { if !$0 { viewModel.errorMessage = nil } }
        )
    }
}

/// iPhone wrapper for the pushed article list: owns the view model for the
/// screen's lifetime and hosts the reader's push destination.
struct ArticleListScreen: View {
    @StateObject private var viewModel: ArticleListViewModel

    private let selection: FeedSelection

    init(selection: FeedSelection) {
        self.selection = selection
        _viewModel = StateObject(wrappedValue: ArticleListViewModel(selection: selection))
    }

    var body: some View {
        ArticleListView(viewModel: viewModel, selection: selection)
            .navigationDestination(for: Article.self) { article in
                ArticleReaderScreen(viewModel: viewModel, articleID: article.id)
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
