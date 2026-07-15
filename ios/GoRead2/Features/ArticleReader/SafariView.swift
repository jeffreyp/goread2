import SafariServices
import SwiftUI

/// SFSafariViewController wrapper for showing external web pages in-app.
/// SFSafariViewController accepts only http/https URLs; callers guard the
/// scheme before presenting.
struct SafariView: UIViewControllerRepresentable {
    let url: URL

    func makeUIViewController(context: Context) -> SFSafariViewController {
        SFSafariViewController(url: url)
    }

    func updateUIViewController(_ controller: SFSafariViewController, context: Context) {}
}

/// Identifiable URL box for sheet(item:) presentation of SafariView.
struct SafariItem: Identifiable {
    let url: URL
    var id: URL { url }

    /// Returns nil for URLs SFSafariViewController cannot show (non-http
    /// schemes such as mailto).
    init?(_ url: URL?) {
        guard let url, let scheme = url.scheme?.lowercased(),
              scheme == "http" || scheme == "https" else { return nil }
        self.url = url
    }
}
