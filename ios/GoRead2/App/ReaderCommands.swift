#if os(macOS)
import SwiftUI

/// The menu-bar actions a reader window offers. The window publishes them as
/// a focused scene value, so the commands drive whichever window is
/// frontmost and disable themselves when none is, for example while the
/// Settings window has focus or before anyone signs in.
///
/// The optional actions stand for state the window may not have yet: there
/// is no article list until a feed is selected, no article to open until one
/// is chosen, and Mark All Read follows the article list's own rule of
/// appearing only for All Articles, since the endpoint is account-wide.
struct ReaderActions {
    var addFeed: () -> Void
    var importOPML: () -> Void
    var exportOPML: () -> Void
    var refresh: () -> Void
    var selectNextArticle: (() -> Void)?
    var selectPreviousArticle: (() -> Void)?
    var openArticleInBrowser: (() -> Void)?
    var unreadOnly: Binding<Bool>?
    var markAllRead: (() -> Void)?
}

private struct ReaderActionsKey: FocusedValueKey {
    typealias Value = ReaderActions
}

extension FocusedValues {
    var readerActions: ReaderActions? {
        get { self[ReaderActionsKey.self] }
        set { self[ReaderActionsKey.self] = newValue }
    }
}

/// The app's File and View menu commands. Every item routes through the
/// frontmost window's `ReaderActions`, which is also what enables and
/// disables them.
struct ReaderCommands: Commands {
    @FocusedValue(\.readerActions) private var actions

    var body: some Commands {
        // Replaces New Window: one reader window is the whole app, and Mac
        // RSS readers put Add Feed on Cmd-N.
        CommandGroup(replacing: .newItem) {
            Button("Add Feed…") { actions?.addFeed() }
                .keyboardShortcut("n")
                .disabled(actions == nil)
            Divider()
            Button("Import OPML…") { actions?.importOPML() }
                .keyboardShortcut("o")
                .disabled(actions == nil)
            Button("Export OPML…") { actions?.exportOPML() }
                .keyboardShortcut("e", modifiers: [.command, .shift])
                .disabled(actions == nil)
            Divider()
            Button("Refresh Feeds") { actions?.refresh() }
                .keyboardShortcut("r")
                .disabled(actions == nil)
        }

        // Lands in the View menu, below the sidebar and toolbar items the
        // system puts there.
        CommandGroup(after: .sidebar) {
            Divider()
            Button("Next Article") { actions?.selectNextArticle?() }
                .keyboardShortcut(.downArrow, modifiers: .command)
                .disabled(actions?.selectNextArticle == nil)
            Button("Previous Article") { actions?.selectPreviousArticle?() }
                .keyboardShortcut(.upArrow, modifiers: .command)
                .disabled(actions?.selectPreviousArticle == nil)
            // Command-Return is the open-in-browser shortcut Mac RSS
            // readers use.
            Button("Open in Browser") { actions?.openArticleInBrowser?() }
                .keyboardShortcut(.return, modifiers: .command)
                .disabled(actions?.openArticleInBrowser == nil)
            Divider()
            unreadOnlyItem
            Button("Mark All Read") { actions?.markAllRead?() }
                .keyboardShortcut("k", modifiers: [.command, .shift])
                .disabled(actions?.markAllRead == nil)
        }
    }

    /// A toggle so the menu carries a checkmark for the current filter. The
    /// disabled placeholder keeps the item and its shortcut in place while
    /// no article list is on screen.
    @ViewBuilder
    private var unreadOnlyItem: some View {
        if let unreadOnly = actions?.unreadOnly {
            Toggle("Unread Only", isOn: unreadOnly)
                .keyboardShortcut("u", modifiers: [.command, .shift])
        } else {
            Button("Unread Only") {}
                .keyboardShortcut("u", modifiers: [.command, .shift])
                .disabled(true)
        }
    }
}
#endif
