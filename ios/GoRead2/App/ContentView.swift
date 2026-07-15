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
                FeedListView()
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
