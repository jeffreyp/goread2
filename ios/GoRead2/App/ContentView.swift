import SwiftUI

struct ContentView: View {
    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "book")
                .font(.system(size: 48))
                .foregroundStyle(.tint)
            Text("GoRead2")
                .font(.largeTitle.bold())
        }
    }
}

#Preview {
    ContentView()
}
