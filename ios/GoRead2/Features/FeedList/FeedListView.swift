import SwiftUI

/// The app's primary navigation screen: the user's subscribed feeds with
/// unread counts, plus add, delete, and manual refresh. On iPhone the rows
/// are navigation links that push an article list; on iPad, where
/// `sidebarSelection` is provided, the rows drive the split view's sidebar
/// selection instead.
struct FeedListView: View {
    @EnvironmentObject private var authManager: AuthManager
    @ObservedObject var viewModel: FeedListViewModel
    var sidebarSelection: Binding<FeedSelection?>?
    /// Replaces the default pull-to-refresh (refreshing feeds and counts).
    /// The iPad split view passes an action that refreshes every pane at
    /// once.
    var refreshAction: (() async -> Void)?

    @State private var showingAddFeed = false
    @State private var showingSettings = false

    var body: some View {
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
                Button {
                    showingSettings = true
                } label: {
                    Label("Settings", systemImage: "gear")
                }
            }
        }
        .sheet(isPresented: $showingSettings) {
            SettingsView()
        }
        .sheet(isPresented: $showingAddFeed) {
            AddFeedView { url in
                try await viewModel.addFeed(url: url)
            }
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
        Group {
            if let sidebarSelection {
                List(selection: sidebarSelection) { feedRows }
                    // The system sidebar list style is translucent by
                    // default, so on iPad the content column's selected-
                    // article highlight can bleed through behind it in
                    // landscape. An opaque background keeps the sidebar
                    // fully covering whatever renders underneath.
                    .scrollContentBackground(.hidden)
                    .background(Color(.systemBackground))
            } else {
                List { feedRows }
            }
        }
        .refreshable {
            if let refreshAction {
                await refreshAction()
            } else {
                await viewModel.refresh()
            }
        }
        .onAppear {
            // Re-fires when a pushed article screen pops; picks up counts
            // changed by read actions. The initial appearance is covered by
            // .task, so skip until that load finishes.
            guard viewModel.hasLoaded else { return }
            Task { await viewModel.refreshUnreadCounts() }
        }
    }

    @ViewBuilder
    private var feedRows: some View {
        row(for: .all) {
            Label("All Articles", systemImage: "tray.full")
                .badge(viewModel.totalUnread)
        }

        Section("Subscriptions") {
            ForEach(viewModel.feeds) { feed in
                row(for: .feed(feed)) {
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

    @ViewBuilder
    private func row<Content: View>(for selection: FeedSelection,
                                    @ViewBuilder content: () -> Content) -> some View {
        if sidebarSelection != nil {
            content().tag(selection)
        } else {
            NavigationLink(value: selection, label: content)
        }
    }

    private var emptyState: some View {
        VStack(spacing: 8) {
            EmptyStateView(systemImage: "tray",
                           title: "Welcome to GoRead2!",
                           message: "Get started by adding your first RSS feed. You can add feeds from your favorite blogs, news sites, and more.")
            Button("Add Your First Feed") {
                showingAddFeed = true
            }
            .buttonStyle(.borderedProminent)
        }
    }

    private var errorBinding: Binding<Bool> {
        Binding(
            get: { viewModel.errorMessage != nil },
            set: { if !$0 { viewModel.errorMessage = nil } }
        )
    }
}

#Preview {
    NavigationStack {
        FeedListView(viewModel: FeedListViewModel())
            .environmentObject(AuthManager())
    }
}
