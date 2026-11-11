import SwiftUI

/// Full-article reading screen: the content renders in a WKWebView, and a
/// bottom toolbar carries previous/next navigation, star, open-in-browser,
/// and share. Swiping left/right on the content also moves to the
/// next/previous article. Opening an article (including via previous/next)
/// marks it read. The standard edge-swipe pops back to the article list.
/// Advancing past the last loaded article always lands on an end-of-list
/// page at once, so finishing the list gets an explicit signal instead of a
/// dead gesture or a wait. Further pages load behind that page, and the
/// reader moves into the first article they bring.
///
/// The view shares the list's view model so star and read changes made here
/// show up in the list rows immediately. The current article is a binding:
/// on iPad the split view passes its selection through, so previous/next
/// moves the list highlight as well; on iPhone `ArticleReaderScreen` holds
/// it as local state.
struct ArticleReaderView: View {
    @ObservedObject var viewModel: ArticleListViewModel
    @Binding var currentID: Int
    @State private var safariItem: SafariItem?
    @Environment(\.openURL) private var openURL

    private var currentIndex: Int? {
        viewModel.articles.firstIndex { $0.id == currentID }
    }

    private var article: Article? {
        currentIndex.map { viewModel.articles[$0] }
    }

    var body: some View {
        Group {
            if currentID == ArticleListViewModel.endOfListID {
                endOfListView
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
                        openURL(url)
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
        .onChange(of: viewModel.articleAfterEndOfList?.id) { nextID in
            // The end-of-list search found another article: continue into
            // it, as moving forward would have.
            if currentID == ArticleListViewModel.endOfListID, let nextID {
                currentID = nextID
            }
        }
        .navigationTitle(article?.feedTitle ?? "")
        .inlineNavigationTitle()
        .toolbar { toolbarContent }
        .webPage(item: $safariItem)
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
        ToolbarItemGroup(placement: readerToolbarPlacement) {
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

                // Hidden rather than shown inert when the source URL is
                // missing or is a scheme no browser opens.
                if let sourcePage = SafariItem(URL(string: article.url)) {
                    Button {
                        safariItem = sourcePage
                    } label: {
                        Label("Open in Browser", systemImage: "safari")
                    }
                }

                if let url = URL(string: article.url) {
                    ShareLink(item: url) {
                        Label("Share", systemImage: "square.and.arrow.up")
                    }
                }
            }
        }
    }

    /// The end-of-list page, shown one position past the last article.
    /// It congratulates when no loaded article is unread, and otherwise
    /// says how many are. While further pages load behind it, a progress
    /// line says so. Swiping right returns to the previous article,
    /// mirroring the reader's swipe navigation.
    private var endOfListView: some View {
        VStack(spacing: 16) {
            let unread = viewModel.unreadLoadedCount
            if unread == 0 {
                EmptyStateView(systemImage: "checkmark.circle",
                               title: "All Caught Up!",
                               message: "You've read all your articles. Great job! "
                                   + "New articles will appear here as they're published.")
            } else {
                EmptyStateView(systemImage: "flag.checkered",
                               title: "That's the Last Article",
                               message: unread == 1
                                   ? "1 article in this list is still unread."
                                   : "\(unread) articles in this list are still unread.")
            }
            if viewModel.isSearchingForMore {
                HStack(spacing: 8) {
                    ProgressView()
                    Text("Checking for older articles")
                }
                .font(.footnote)
                .foregroundStyle(.secondary)
            }
        }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .contentShape(Rectangle())
            .gesture(DragGesture().onEnded { value in
                if value.translation.width > 50 {
                    move(-1)
                }
            })
    }

    private func canMove(_ offset: Int) -> Bool {
        if currentID == ArticleListViewModel.endOfListID {
            if offset > 0 { return viewModel.articleAfterEndOfList != nil }
            return !viewModel.articles.isEmpty
        }
        guard let index = currentIndex else { return false }
        // Forward always leads somewhere: the next article or the
        // end-of-list page.
        return offset > 0 || index > 0
    }

    private func move(_ offset: Int) {
        guard canMove(offset),
              let destination = viewModel.destination(from: currentID, offset: offset) else { return }
        currentID = destination
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
