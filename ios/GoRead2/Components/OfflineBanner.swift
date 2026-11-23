import SwiftUI

/// A strip above a list shown from the cache because the server could not be
/// reached, saying why and how old the data is: "Offline · Last updated 5
/// minutes ago". The age updates every minute while the banner is up.
struct OfflineBanner: View {
    /// When the server returned the data on screen; nil for a cached copy
    /// written before the app kept the date.
    let fetchedAt: Date?
    /// False when the device has a connection but the server failed anyway.
    let isOffline: Bool

    var body: some View {
        TimelineView(.periodic(from: .now, by: 60)) { context in
            Label(message(now: context.date), systemImage: isOffline ? "wifi.slash" : "exclamationmark.icloud")
                .font(.footnote)
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 6)
                .padding(.horizontal)
                .background(.bar)
        }
        .accessibilityElement(children: .combine)
    }

    private func message(now: Date) -> String {
        let reason = isOffline ? "Offline" : "Couldn't refresh"
        guard let fetchedAt else { return "\(reason) · Showing saved articles" }
        // A fetch moments ago reads better as "just now" than "in 0 seconds".
        let age = fetchedAt > now.addingTimeInterval(-60)
            ? "just now"
            : fetchedAt.formatted(.relative(presentation: .named, unitsStyle: .wide))
        return "\(reason) · Last updated \(age)"
    }
}

#Preview {
    VStack {
        OfflineBanner(fetchedAt: .now.addingTimeInterval(-300), isOffline: true)
        OfflineBanner(fetchedAt: nil, isOffline: false)
    }
}
