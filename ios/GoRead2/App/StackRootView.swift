import SwiftUI

/// iPhone root: the feed list at the base of a navigation stack, with
/// article lists and the reader pushed on top.
struct StackRootView: View {
    @StateObject private var feedViewModel = FeedListViewModel()

    var body: some View {
        NavigationStack {
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
