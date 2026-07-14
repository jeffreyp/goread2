import Foundation

/// Mirrors services.SubscriptionInfo from GET /api/subscription.
struct SubscriptionInfo: Codable, Hashable {
    /// "trial", "active", "cancelled", "expired", "admin", or "unlimited"
    /// when the subscription system is disabled server-side.
    let status: String
    let subscriptionId: String
    let trialEndsAt: Date
    let lastPaymentDate: Date
    let nextBillingDate: Date?
    let currentFeeds: Int
    /// -1 means unlimited.
    let feedLimit: Int
    let canAddFeeds: Bool
    let isActive: Bool
    let trialDaysRemaining: Int?
}
