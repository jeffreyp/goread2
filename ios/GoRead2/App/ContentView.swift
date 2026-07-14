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
            case .signedIn(let user):
                // Placeholder until the feed list screen (gr-e8b5) lands.
                VStack(spacing: 12) {
                    Image(systemName: "book")
                        .font(.system(size: 48))
                        .foregroundStyle(.tint)
                    Text("GoRead2")
                        .font(.largeTitle.bold())
                    Text("Signed in as \(user.email)")
                        .foregroundStyle(.secondary)
                    Button("Sign Out") {
                        Task { await authManager.signOut() }
                    }
                    .buttonStyle(.bordered)
                }
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
