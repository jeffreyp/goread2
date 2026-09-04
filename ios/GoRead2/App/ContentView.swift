import SwiftUI

struct ContentView: View {
    @StateObject private var authManager = AuthManager()

    var body: some View {
        Group {
            switch authManager.state {
            case .checking:
                ProgressView()
            case .signedOut:
                LoginView()
            case .signedIn:
                #if os(macOS)
                // Mac windows are always wide enough for the three-pane
                // layout; there is no compact equivalent.
                SplitRootView()
                #else
                if UIDevice.current.userInterfaceIdiom == .pad {
                    SplitRootView()
                } else {
                    StackRootView()
                }
                #endif
            }
        }
        .environmentObject(authManager)
        .task {
            await authManager.bootstrap()
        }
    }
}

#Preview {
    ContentView()
}
