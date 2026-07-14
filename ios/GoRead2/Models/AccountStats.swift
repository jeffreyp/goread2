import Foundation

/// Mirrors the response of GET /api/account/stats.
struct AccountStats: Decodable {
    let totalFeeds: Int
    let totalArticles: Int
    let totalUnread: Int
    let activeFeeds: Int
    let subscriptionInfo: SubscriptionInfo?
    let feeds: [Feed]

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        totalFeeds = try container.decode(Int.self, forKey: .totalFeeds)
        totalArticles = try container.decode(Int.self, forKey: .totalArticles)
        totalUnread = try container.decode(Int.self, forKey: .totalUnread)
        activeFeeds = try container.decode(Int.self, forKey: .activeFeeds)
        subscriptionInfo = try container.decodeIfPresent(SubscriptionInfo.self, forKey: .subscriptionInfo)
        feeds = try container.decodeIfPresent([Feed].self, forKey: .feeds) ?? []
    }

    private enum CodingKeys: String, CodingKey {
        case totalFeeds
        case totalArticles
        case totalUnread
        case activeFeeds
        case subscriptionInfo
        case feeds
    }
}
