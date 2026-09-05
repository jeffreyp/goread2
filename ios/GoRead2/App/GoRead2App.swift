import SwiftUI

@main
struct GoRead2App: App {
    /// One session owner for the whole app. On macOS the Settings scene is a
    /// sibling of the window group rather than a child of ContentView, so it
    /// cannot inherit the manager from the window's environment; signing out
    /// in Settings has to move the same object the window is observing.
    @StateObject private var authManager = AuthManager()

    var body: some Scene {
        mainWindow

        #if os(macOS)
        // Puts Settings in the app menu under its standard Cmd-, shortcut,
        // where Mac users look for it, instead of a toolbar button.
        Settings {
            SettingsView()
                .environmentObject(authManager)
        }
        #endif
    }

    #if os(macOS)
    private var mainWindow: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(authManager)
                // Floor for the three-pane layout: the sidebar and article
                // list minimums plus room for the reader.
                .frame(minWidth: 720, minHeight: 480)
        }
        // Wide enough that all three panes open tiled rather than the sidebar
        // starting collapsed.
        .defaultSize(width: 1200, height: 800)
    }
    #else
    private var mainWindow: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(authManager)
        }
    }
    #endif
}
