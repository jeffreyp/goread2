import SwiftUI

/// The app's primary navigation screen: the user's subscribed feeds with
/// unread counts, plus add, delete, and manual refresh.
struct FeedListView: View {
    @EnvironmentObject private var authManager: AuthManager
    @StateObject private var viewModel = FeedListViewModel()

    @State private var showingAddFeed = false
    @State private var newFeedURL = ""

    var body: some View {
        NavigationStack {
            Group {
                if !viewModel.hasLoaded {
                    ProgressView()
                } else if viewModel.feeds.isEmpty {
                    emptyState
                } else {
                    feedList
                }
            }
            .navigationTitle("Feeds")
            .toolbar {
                ToolbarItem(placement: .primaryAction) {
                    Button {
                        showingAddFeed = true
                    } label: {
                        Label("Add Feed", systemImage: "plus")
                    }
                }
                ToolbarItem(placement: .secondaryAction) {
                    Button(role: .destructive) {
                        Task { await authManager.signOut() }
                    } label: {
                        Label("Sign Out", systemImage: "rectangle.portrait.and.arrow.right")
                    }
                }
            }
            .navigationDestination(for: FeedSelection.self) { selection in
                ArticleListView(selection: selection)
            }
        }
        .alert("Add Feed", isPresented: $showingAddFeed) {
            TextField("Feed URL", text: $newFeedURL)
                .textInputAutocapitalization(.never)
                .keyboardType(.URL)
                .autocorrectionDisabled()
            Button("Add") {
                let url = newFeedURL
                newFeedURL = ""
                Task { await viewModel.addFeed(url: url) }
            }
            Button("Cancel", role: .cancel) {
                newFeedURL = ""
            }
        } message: {
            Text("Enter the URL of an RSS or Atom feed, or of a site that links to one.")
        }
        .alert("Error", isPresented: errorBinding) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(viewModel.errorMessage ?? "")
        }
        .task {
            viewModel.onSessionExpired = { authManager.sessionExpired() }
            await viewModel.load()
        }
    }

    private var feedList: some View {
        List {
            NavigationLink(value: FeedSelection.all) {
                Label("All Articles", systemImage: "tray.full")
                    .badge(viewModel.totalUnread)
            }

            Section("Subscriptions") {
                ForEach(viewModel.feeds) { feed in
                    NavigationLink(value: FeedSelection.feed(feed)) {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(feed.title)
                                .lineLimit(1)
                            if !feed.description.isEmpty {
                                Text(feed.description)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                    .lineLimit(1)
                            }
                        }
                        .badge(viewModel.unreadCount(for: feed))
                    }
                    .swipeActions(edge: .trailing) {
                        Button(role: .destructive) {
                            Task { await viewModel.delete(feed) }
                        } label: {
                            Label("Unsubscribe", systemImage: "trash")
                        }
                    }
                }
            }
        }
        .refreshable {
            await viewModel.refresh()
        }
        .onAppear {
            // Re-fires when a pushed article screen pops; picks up counts
            // changed by read actions. The initial appearance is covered by
            // .task, so skip until that load finishes.
            guard viewModel.hasLoaded else { return }
            Task { await viewModel.refreshUnreadCounts() }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 12) {
            Image(systemName: "tray")
                .font(.system(size: 48))
                .foregroundStyle(.secondary)
            Text("No Feeds")
                .font(.title2.bold())
            Text("Subscribe to an RSS feed to start reading.")
                .foregroundStyle(.secondary)
            Button("Add Feed") {
                showingAddFeed = true
            }
            .buttonStyle(.borderedProminent)
            .padding(.top, 8)
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

#Preview {
    FeedListView()
        .environmentObject(AuthManager())
}
