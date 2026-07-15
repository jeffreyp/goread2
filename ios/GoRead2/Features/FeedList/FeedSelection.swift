import Foundation

/// Which stream of articles a feed-list row opens: a single feed or the
/// merged "All Articles" stream, which the backend addresses as feed ID
/// "all".
enum FeedSelection: Hashable {
    case all
    case feed(Feed)

    var title: String {
        switch self {
        case .all:
            return "All Articles"
        case .feed(let feed):
            return feed.title
        }
    }

    /// Path component for /api/feeds/:id/articles.
    var apiFeedID: String {
        switch self {
        case .all:
            return "all"
        case .feed(let feed):
            return String(feed.id)
        }
    }
}
