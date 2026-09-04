import SwiftUI

@main
struct GoRead2App: App {
    var body: some Scene {
        WindowGroup {
            ContentView()
            #if os(macOS)
                // Floor for the three-pane layout: the sidebar and article
                // list minimums plus room for the reader.
                .frame(minWidth: 720, minHeight: 480)
            #endif
        }
        #if os(macOS)
        // Wide enough that all three panes open tiled rather than the
        // sidebar starting collapsed.
        .defaultSize(width: 1200, height: 800)
        #endif
    }
}
