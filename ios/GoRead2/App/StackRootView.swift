import SwiftUI

/// iPhone root: the feed list at the base of a navigation stack, with
/// article lists and the reader pushed on top.
struct StackRootView: View {
    @StateObject private var feedViewModel = FeedListViewModel()
    /// Launches with All Articles pushed so unread articles are visible
    /// immediately; the feed list stays one back-swipe away.
    @State private var path = NavigationPath([FeedSelection.all])

    var body: some View {
        NavigationStack(path: $path) {
            FeedListView(viewModel: feedViewModel)
                .navigationDestination(for: FeedSelection.self) { selection in
                    ArticleListScreen(selection: selection)
                }
        }
    }
}

#Preview {
    StackRootView()
        .environmentObject(AuthManager())
}
