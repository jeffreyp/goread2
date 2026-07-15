import SwiftUI

/// Paginated article list for one feed or the merged "All Articles" stream:
/// infinite scroll via cursor pagination, an unread-only filter, and
/// pull-to-refresh. Opening an article marks it read immediately.
struct ArticleListView: View {
    @EnvironmentObject private var authManager: AuthManager
    @StateObject private var viewModel: ArticleListViewModel

    private let selection: FeedSelection

    init(selection: FeedSelection) {
        self.selection = selection
        _viewModel = StateObject(wrappedValue: ArticleListViewModel(selection: selection))
    }

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
            ToolbarItem(placement: .primaryAction) {
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
        .navigationDestination(for: Article.self) { article in
            ArticleReaderPlaceholderView(article: article)
                .task { await viewModel.markRead(article) }
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
        List {
            ForEach(viewModel.articles) { article in
                NavigationLink(value: article) {
                    ArticleRow(article: article)
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
        .listStyle(.plain)
        .refreshable {
            await viewModel.load()
        }
    }

    private var emptyState: some View {
        VStack(spacing: 12) {
            Image(systemName: viewModel.unreadOnly ? "checkmark.circle" : "tray")
                .font(.system(size: 48))
                .foregroundStyle(.secondary)
            Text(viewModel.unreadOnly ? "All Caught Up" : "No Articles")
                .font(.title2.bold())
            Text(viewModel.unreadOnly
                 ? "Every article here has been read."
                 : "Articles will appear once this feed has content.")
                .foregroundStyle(.secondary)
        }
        .padding()
    }

    private var errorBinding: Binding<Bool> {
        Binding(
            get: { viewModel.errorMessage != nil },
            set: { if !$0 { viewModel.errorMessage = nil } }
        )
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

/// Placeholder destination until the article reader screen (gr-znsq) lands.
private struct ArticleReaderPlaceholderView: View {
    let article: Article

    var body: some View {
        VStack(spacing: 12) {
            Text(article.title)
                .font(.title3.bold())
                .multilineTextAlignment(.center)
            Text("Reader coming soon")
                .foregroundStyle(.secondary)
        }
        .padding()
        .navigationTitle(article.feedTitle ?? "")
        .navigationBarTitleDisplayMode(.inline)
    }
}

#Preview {
    NavigationStack {
        ArticleListView(selection: .all)
            .environmentObject(AuthManager())
    }
}
