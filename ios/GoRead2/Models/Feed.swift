import Foundation

/// Mirrors the JSON emitted for database.Feed by the Go handlers.
struct Feed: Codable, Identifiable, Hashable {
    let id: Int
    let title: String
    let url: String
    let description: String
    let createdAt: Date
    let updatedAt: Date
    let lastFetch: Date
}
