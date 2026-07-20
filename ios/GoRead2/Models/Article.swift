import Foundation

/// Mirrors the JSON emitted for database.Article by the Go handlers.
struct Article: Codable, Identifiable, Hashable {
    let id: Int
    let feedId: Int
    let feedTitle: String?
    let title: String
    let url: String
    /// Often empty in list responses; the reader lazily fetches the full
    /// content via GET /api/articles/:id and fills it in.
    var content: String
    let description: String
    let author: String
    let publishedAt: Date
    let createdAt: Date
    var isRead: Bool
    var isStarred: Bool
}

/// One page of articles from GET /api/feeds/:id/articles.
struct PaginatedArticles: Decodable {
    let articles: [Article]
    /// Cursor for the next page; nil when there are no more pages.
    let nextCursor: String?

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        articles = try container.decodeIfPresent([Article].self, forKey: .articles) ?? []
        let cursor = try container.decodeIfPresent(String.self, forKey: .nextCursor)
        nextCursor = (cursor?.isEmpty ?? true) ? nil : cursor
    }

    private enum CodingKeys: String, CodingKey {
        case articles
        case nextCursor
    }
}
