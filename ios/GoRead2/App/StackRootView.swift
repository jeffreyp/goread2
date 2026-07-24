import SwiftUI

/// iPhone root: the feed list at the base of a navigation stack, with
/// article lists and the reader pushed on top.
struct StackRootView: View {
    @StateObject private var feedViewModel = FeedListViewModel()
    /// Starts empty so a brand-new account lands on the feed list's welcome
    /// screen; once the feed list confirms there is at least one
    /// subscription, All Articles is pushed automatically so unread
    /// articles are visible without an extra tap.
    @State private var path = NavigationPath()

    var body: some View {
        NavigationStack(path: $path) {
            FeedListView(viewModel: feedViewModel)
                .navigationDestination(for: FeedSelection.self) { selection in
                    ArticleListScreen(selection: selection)
                }
        }
        .onChange(of: feedViewModel.hasLoaded) { _ in pushAllArticlesIfNeeded() }
        .onChange(of: feedViewModel.feeds.count) { _ in pushAllArticlesIfNeeded() }
    }

    private func pushAllArticlesIfNeeded() {
        guard feedViewModel.hasLoaded, path.isEmpty, !feedViewModel.feeds.isEmpty else { return }
        path.append(FeedSelection.all)
    }
}

#Preview {
    StackRootView()
        .environmentObject(AuthManager())
}
