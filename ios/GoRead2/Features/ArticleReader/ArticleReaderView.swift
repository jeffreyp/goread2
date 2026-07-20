import SwiftUI

/// Full-article reading screen: the content renders in a WKWebView, and a
/// bottom toolbar carries previous/next navigation, star, open-in-browser,
/// and share. Swiping left/right on the content also moves to the
/// next/previous article. Opening an article (including via previous/next)
/// marks it read. The standard edge-swipe pops back to the article list.
/// Advancing past the last article when nothing unread remains lands on a
/// caught-up screen, so finishing the list gets an explicit signal instead
/// of a dead gesture.
///
/// The view shares the list's view model so star and read changes made here
/// show up in the list rows immediately. The current article is a binding:
/// on iPad the split view passes its selection through, so previous/next
/// moves the list highlight as well; on iPhone `ArticleReaderScreen` holds
/// it as local state.
struct ArticleReaderView: View {
    /// Sentinel `currentID` for the position one past the last article,
    /// where the reader shows the caught-up screen instead of an article.
    /// Real article IDs from the server are positive.
    static let caughtUpID = -1

    @ObservedObject var viewModel: ArticleListViewModel
    @Binding var currentID: Int
    @State private var safariItem: SafariItem?

    private var currentIndex: Int? {
        viewModel.articles.firstIndex { $0.id == currentID }
    }

    private var article: Article? {
        currentIndex.map { viewModel.articles[$0] }
    }

    var body: some View {
        Group {
            if currentID == Self.caughtUpID {
                caughtUpView
            } else if let article {
                ArticleWebView(article: article,
                               showsContentPlaceholder: article.content.isEmpty
                                   && !viewModel.contentUnavailable.contains(article.id),
                               onLinkTap: { url in
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
            guard let article else { return }
            async let read: Void = viewModel.markRead(article)
            // The list API often omits full content; fetch it like the web
            // app does on selection.
            await viewModel.loadContent(for: article.id)
            await read
            // Prefetch neighbours, after the current article, so swipe
            // navigation lands on already-loaded content.
            for offset in [-1, 1] {
                guard let index = currentIndex,
                      viewModel.articles.indices.contains(index + offset) else { continue }
                await viewModel.loadContent(for: viewModel.articles[index + offset].id)
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

    /// The caught-up screen, shown one position past the last article.
    /// Swiping right returns to the last article, mirroring the reader's
    /// swipe navigation.
    private var caughtUpView: some View {
        EmptyStateView(systemImage: "checkmark.circle",
                       title: "All Caught Up!",
                       message: "You've read all your articles. Great job! "
                           + "New articles will appear here as they're published.")
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .contentShape(Rectangle())
            .gesture(DragGesture().onEnded { value in
                if value.translation.width > 50 {
                    move(-1)
                }
            })
    }

    private func canMove(_ offset: Int) -> Bool {
        if currentID == Self.caughtUpID {
            return offset < 0 && !viewModel.articles.isEmpty
        }
        guard let index = currentIndex else { return false }
        if viewModel.articles.indices.contains(index + offset) { return true }
        // One past the last article is the caught-up screen, reachable once
        // nothing unread remains.
        return offset > 0 && viewModel.isCaughtUp
    }

    private func move(_ offset: Int) {
        guard canMove(offset) else { return }
        if currentID == Self.caughtUpID {
            currentID = viewModel.articles[viewModel.articles.count - 1].id
            return
        }
        guard let index = currentIndex else { return }
        let target = index + offset
        guard viewModel.articles.indices.contains(target) else {
            currentID = Self.caughtUpID
            return
        }
        currentID = viewModel.articles[target].id
        // Approaching the end of the loaded pages: fetch the next one so
        // "next" keeps working past the page boundary.
        if offset > 0, target >= viewModel.articles.count - 3 {
            Task { await viewModel.loadMore() }
        }
    }
}

/// iPhone wrapper for the pushed reader: holds the current article as local
/// state, seeded from the tapped row.
struct ArticleReaderScreen: View {
    @ObservedObject var viewModel: ArticleListViewModel
    @State private var currentID: Int

    init(viewModel: ArticleListViewModel, articleID: Int) {
        self.viewModel = viewModel
        _currentID = State(initialValue: articleID)
    }

    var body: some View {
        ArticleReaderView(viewModel: viewModel, currentID: $currentID)
    }
}

/// Fallback when the current article is no longer in the loaded list.
private struct ContentUnavailableCompatView: View {
    var body: some View {
        EmptyStateView(systemImage: "doc.questionmark",
                       title: "Article Unavailable",
                       message: "This article is no longer in the loaded list.")
    }
}
