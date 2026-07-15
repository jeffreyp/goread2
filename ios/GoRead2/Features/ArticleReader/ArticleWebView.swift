import SwiftUI
import WebKit

/// WKWebView wrapper that renders one article's HTML inside a stylesheet
/// matching the web app's reading typography, with automatic Dark Mode via
/// prefers-color-scheme. Tapped links are handed to `onLinkTap` instead of
/// navigating in place. Horizontal swipes report an offset (-1 for a right
/// swipe, +1 for a left swipe) through `onSwipe` so the reader can move
/// between articles.
struct ArticleWebView: UIViewRepresentable {
    let article: Article
    let onLinkTap: (URL) -> Void
    var onSwipe: ((Int) -> Void)?

    func makeUIView(context: Context) -> WKWebView {
        let webView = WKWebView()
        webView.navigationDelegate = context.coordinator
        webView.uiDelegate = context.coordinator
        // A transparent web view lets the SwiftUI systemBackground show
        // through, avoiding a white flash in Dark Mode while HTML loads.
        webView.isOpaque = false
        webView.backgroundColor = .clear
        webView.scrollView.backgroundColor = .clear

        let nextSwipe = UISwipeGestureRecognizer(
            target: context.coordinator, action: #selector(Coordinator.didSwipeToNext))
        nextSwipe.direction = .left
        nextSwipe.delegate = context.coordinator
        webView.addGestureRecognizer(nextSwipe)

        let previousSwipe = UISwipeGestureRecognizer(
            target: context.coordinator, action: #selector(Coordinator.didSwipeToPrevious))
        previousSwipe.direction = .right
        previousSwipe.delegate = context.coordinator
        webView.addGestureRecognizer(previousSwipe)

        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        context.coordinator.onLinkTap = onLinkTap
        context.coordinator.onSwipe = onSwipe
        // updateUIView also fires for unrelated state changes (star toggles,
        // sheet presentation); only reload when showing a different article,
        // or the scroll position would reset.
        guard context.coordinator.loadedArticleID != article.id else { return }
        context.coordinator.loadedArticleID = article.id
        // The article URL as base resolves relative image and link paths in
        // feed content.
        webView.loadHTMLString(Self.page(for: article), baseURL: URL(string: article.url))
        webView.scrollView.setContentOffset(.zero, animated: false)
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(onLinkTap: onLinkTap)
    }

    final class Coordinator: NSObject, WKNavigationDelegate, WKUIDelegate,
                             UIGestureRecognizerDelegate {
        var onLinkTap: (URL) -> Void
        var onSwipe: ((Int) -> Void)?
        var loadedArticleID: Int?

        init(onLinkTap: @escaping (URL) -> Void) {
            self.onLinkTap = onLinkTap
        }

        @objc func didSwipeToNext() { onSwipe?(1) }
        @objc func didSwipeToPrevious() { onSwipe?(-1) }

        // The scroll view's pan recognizer would otherwise claim every touch
        // and the swipes would never fire.
        func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer,
                               shouldRecognizeSimultaneouslyWith
                               otherGestureRecognizer: UIGestureRecognizer) -> Bool {
            true
        }

        // Touches starting at the leading edge belong to the navigation
        // controller's interactive pop; the previous-article swipe must not
        // race it.
        func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer,
                               shouldReceive touch: UITouch) -> Bool {
            guard let swipe = gestureRecognizer as? UISwipeGestureRecognizer,
                  swipe.direction == .right,
                  let window = touch.window else { return true }
            return touch.location(in: window).x > 44
        }

        func webView(_ webView: WKWebView,
                     decidePolicyFor navigationAction: WKNavigationAction,
                     decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
            // Main-frame link taps leave the reader; everything else (the
            // initial loadHTMLString, iframes) renders in place.
            if navigationAction.navigationType == .linkActivated,
               navigationAction.targetFrame?.isMainFrame != false,
               let url = navigationAction.request.url {
                decisionHandler(.cancel)
                onLinkTap(url)
                return
            }
            decisionHandler(.allow)
        }

        // target="_blank" links ask for a new web view; open them the same
        // way as ordinary link taps instead.
        func webView(_ webView: WKWebView,
                     createWebViewWith configuration: WKWebViewConfiguration,
                     for navigationAction: WKNavigationAction,
                     windowFeatures: WKWindowFeatures) -> WKWebView? {
            if let url = navigationAction.request.url {
                onLinkTap(url)
            }
            return nil
        }
    }

    // MARK: - HTML template

    /// Wraps the article in a full page: title, meta line (feed, author,
    /// date), and the feed-provided content HTML.
    static func page(for article: Article) -> String {
        var meta = [String]()
        if let feedTitle = article.feedTitle, !feedTitle.isEmpty {
            meta.append(escape(feedTitle))
        }
        if !article.author.isEmpty {
            meta.append(escape(article.author))
        }
        meta.append(escape(article.publishedAt.formatted(date: .abbreviated, time: .shortened)))

        let body = article.content.isEmpty ? article.description : article.content

        return """
        <!DOCTYPE html>
        <html>
        <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <style>\(stylesheet)</style>
        </head>
        <body>
        <h1 class="title">\(escape(article.title))</h1>
        <div class="meta">\(meta.joined(separator: " · "))</div>
        <div class="content">\(body)</div>
        </body>
        </html>
        """
    }

    /// Mirrors the web app's .article-content rules (styles.css), with the
    /// system font standing in for Inter and a prefers-color-scheme block for
    /// Dark Mode.
    private static let stylesheet = """
    :root { color-scheme: light dark; }
    body {
        font-family: -apple-system, system-ui, sans-serif;
        margin: 0;
        padding: 20px;
        font-size: 17px;
        line-height: 1.6;
        color: #3c4043;
        background: transparent;
        -webkit-text-size-adjust: 100%;
        overflow-wrap: break-word;
    }
    h1.title {
        font-size: 26px;
        font-weight: 600;
        line-height: 1.3;
        color: #202124;
        margin: 0 0 12px;
    }
    .meta {
        font-size: 14px;
        color: #5f6368;
        margin-bottom: 20px;
        padding-bottom: 16px;
        border-bottom: 1px solid #e1e5e9;
    }
    .content { line-height: 1.7; }
    .content p { margin: 0 0 16px; }
    img, video, iframe { max-width: 100%; height: auto; }
    img { margin: 16px 0; }
    a { color: #1a73e8; text-decoration: none; }
    ul, ol { margin: 16px 0; padding-left: 32px; }
    li { margin-bottom: 8px; }
    blockquote {
        margin: 16px 0;
        padding-left: 12px;
        border-left: 3px solid #e1e5e9;
        color: #5f6368;
    }
    pre {
        overflow-x: auto;
        padding: 12px;
        background: rgba(128, 128, 128, 0.1);
        border-radius: 6px;
    }
    code { font-family: ui-monospace, monospace; font-size: 15px; }
    figure { margin: 16px 0; }
    @media (prefers-color-scheme: dark) {
        body { color: #e8eaed; }
        h1.title { color: #f1f3f4; }
        .meta { color: #9aa0a6; border-color: #3c4043; }
        a { color: #8ab4f8; }
        blockquote { border-color: #3c4043; color: #9aa0a6; }
    }
    """

    private static func escape(_ text: String) -> String {
        text.replacingOccurrences(of: "&", with: "&amp;")
            .replacingOccurrences(of: "<", with: "&lt;")
            .replacingOccurrences(of: ">", with: "&gt;")
    }
}
