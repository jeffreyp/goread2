import SwiftUI

/// iPhone root: the feed list at the base of a navigation stack, with
/// article lists and the reader pushed on top.
///
/// The feed list loads once. After that its unread badges follow read-state
/// changes reported by the pushed article list, and resync with the server
/// on returning to the app, on pull-to-refresh, and after Mark All Read.
struct StackRootView: View {
    @StateObject private var feedViewModel = FeedListViewModel()
    /// Starts empty so a brand-new account lands on the feed list's welcome
    /// screen; once the feed list, cached or loaded, has at least one
    /// subscription, All Articles is pushed automatically so unread
    /// articles are visible without an extra tap.
    @State private var path = NavigationPath()
    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        NavigationStack(path: $path) {
            FeedListView(viewModel: feedViewModel)
                .navigationDestination(for: FeedSelection.self) { selection in
                    ArticleListScreen(
                        selection: selection,
                        onUnreadCountChange: { [feedViewModel] feedID, delta in
                            feedViewModel.adjustUnreadCount(feedID: feedID, by: delta)
                        },
                        refreshUnreadCounts: { [feedViewModel] in
                            await feedViewModel.refreshUnreadCounts()
                        })
                }
        }
        // The feed list may already hold the last session's data, which
        // onChange does not report.
        .onAppear { pushAllArticlesIfNeeded() }
        .onChange(of: feedViewModel.hasLoaded) { _ in pushAllArticlesIfNeeded() }
        .onChange(of: feedViewModel.feeds.count) { _ in pushAllArticlesIfNeeded() }
        .onChange(of: scenePhase) { phase in
            // Picks up reads made on other devices and corrects any drift in
            // the locally adjusted counts.
            guard phase == .active, feedViewModel.hasLoaded else { return }
            Task { await feedViewModel.refreshUnreadCounts() }
        }
    }

    private func pushAllArticlesIfNeeded() {
        guard feedViewModel.isReady, path.isEmpty, !feedViewModel.feeds.isEmpty else { return }
        path.append(FeedSelection.all)
    }
}

#Preview {
    StackRootView()
        .environmentObject(AuthManager())
}
