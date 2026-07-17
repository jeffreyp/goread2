import SwiftUI

/// Centred placeholder for empty lists and unselected split-view columns:
/// an icon above a title and a supporting message. The icon scales with
/// Dynamic Type alongside the text.
struct EmptyStateView: View {
    let systemImage: String
    let title: String
    let message: String

    @ScaledMetric(relativeTo: .largeTitle) private var iconSize: CGFloat = 48

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: systemImage)
                .font(.system(size: iconSize))
                .foregroundStyle(.secondary)
            Text(title)
                .font(.title2.bold())
            Text(message)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
        }
        .padding()
    }
}

#Preview {
    EmptyStateView(systemImage: "tray",
                   title: "No Feeds",
                   message: "Subscribe to an RSS feed to start reading.")
}
