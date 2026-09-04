import SwiftUI
#if os(iOS)
import SafariServices
#endif

#if os(iOS)
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
#endif

/// Identifiable URL box for presenting a web page.
struct SafariItem: Identifiable {
    let url: URL
    var id: URL { url }

    /// Returns nil for URLs the in-app browser cannot show (non-http
    /// schemes such as mailto).
    init?(_ url: URL?) {
        guard let url, let scheme = url.scheme?.lowercased(),
              scheme == "http" || scheme == "https" else { return nil }
        self.url = url
    }
}

extension View {
    /// Shows `item` as an in-app browser sheet on iOS. macOS has no
    /// SFSafariViewController, and Mac apps are expected to hand external
    /// pages to the default browser, so the URL opens there and the binding
    /// clears immediately.
    @ViewBuilder
    func webPage(item: Binding<SafariItem?>) -> some View {
        #if os(iOS)
        sheet(item: item) { entry in
            SafariView(url: entry.url)
                .ignoresSafeArea()
        }
        #else
        modifier(ExternalWebPageOpener(item: item))
        #endif
    }
}

#if os(macOS)
private struct ExternalWebPageOpener: ViewModifier {
    @Binding var item: SafariItem?
    @Environment(\.openURL) private var openURL

    func body(content: Content) -> some View {
        content.task(id: item?.url) {
            guard let url = item?.url else { return }
            openURL(url)
            item = nil
        }
    }
}
#endif
