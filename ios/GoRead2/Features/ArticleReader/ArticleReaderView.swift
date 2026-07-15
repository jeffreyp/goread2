import SwiftUI

/// Full-article reading screen: the content renders in a WKWebView, and a
/// bottom toolbar carries previous/next navigation, star, open-in-browser,
/// and share. Swiping left/right on the content also moves to the
/// next/previous article. Opening an article (including via previous/next)
/// marks it read. The standard edge-swipe pops back to the article list.
///
/// The view shares the list's view model so star and read changes made here
/// show up in the list rows immediately.
struct ArticleReaderView: View {
    @ObservedObject var viewModel: ArticleListViewModel
    @State private var currentID: Int
    @State private var safariItem: SafariItem?

    init(viewModel: ArticleListViewModel, articleID: Int) {
        self.viewModel = viewModel
        _currentID = State(initialValue: articleID)
    }

    private var currentIndex: Int? {
        viewModel.articles.firstIndex { $0.id == currentID }
    }

    private var article: Article? {
        currentIndex.map { viewModel.articles[$0] }
    }

    var body: some View {
        Group {
            if let article {
                ArticleWebView(article: article, onLinkTap: { url in
                    if let item = SafariItem(url) {
                        safariItem = item
                    } else {
                        // Non-web links (mailto, app schemes) go to the
                        // system handler.
                        UIApplication.shared.open(url)
                    }
                }, onSwipe: { offset in
                    move(offset)
                })
            } else {
                // A refresh can replace the list while the reader is open;
                // the article may no longer be loaded.
                ContentUnavailableCompatView()
            }
        }
        .navigationTitle(article?.feedTitle ?? "")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar { toolbarContent }
        .sheet(item: $safariItem) { item in
            SafariView(url: item.url)
                .ignoresSafeArea()
        }
        .task(id: currentID) {
            if let article {
                await viewModel.markRead(article)
            }
        }
    }

    @ToolbarContentBuilder
    private var toolbarContent: some ToolbarContent {
        ToolbarItemGroup(placement: .bottomBar) {
            Button {
                move(-1)
            } label: {
                Label("Previous Article", systemImage: "chevron.up")
            }
            .disabled(!canMove(-1))

            Button {
                move(1)
            } label: {
                Label("Next Article", systemImage: "chevron.down")
            }
            .disabled(!canMove(1))

            Spacer()

            if let article {
                Button {
                    Task { await viewModel.toggleStar(article) }
                } label: {
                    Label(article.isStarred ? "Unstar" : "Star",
                          systemImage: article.isStarred ? "star.fill" : "star")
                }
                .tint(article.isStarred ? .yellow : nil)

                Button {
                    safariItem = SafariItem(URL(string: article.url))
                } label: {
                    Label("Open in Browser", systemImage: "safari")
                }

                if let url = URL(string: article.url) {
                    ShareLink(item: url) {
                        Label("Share", systemImage: "square.and.arrow.up")
                    }
                }
            }
        }
    }

    private func canMove(_ offset: Int) -> Bool {
        guard let index = currentIndex else { return false }
        return viewModel.articles.indices.contains(index + offset)
    }

    private func move(_ offset: Int) {
        guard let index = currentIndex,
              viewModel.articles.indices.contains(index + offset) else { return }
        currentID = viewModel.articles[index + offset].id
        // Approaching the end of the loaded pages: fetch the next one so
        // "next" keeps working past the page boundary.
        if offset > 0, index + offset >= viewModel.articles.count - 3 {
            Task { await viewModel.loadMore() }
        }
    }
}

/// Fallback when the current article is no longer in the loaded list.
private struct ContentUnavailableCompatView: View {
    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "doc.questionmark")
                .font(.system(size: 48))
                .foregroundStyle(.secondary)
            Text("Article Unavailable")
                .font(.title2.bold())
            Text("This article is no longer in the loaded list.")
                .foregroundStyle(.secondary)
        }
        .padding()
    }
}
