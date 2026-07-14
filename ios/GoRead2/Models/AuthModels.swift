import Foundation

/// The "user" object of GET /auth/me.
struct CurrentUser: Codable, Hashable {
    let id: Int
    let email: String
    let name: String
    let avatar: String
    let createdAt: Date
    let maxArticlesOnFeedAdd: Int
}

/// Response of GET /auth/me. Fetching it also refreshes the CSRF token
/// NetworkClient injects into mutating requests.
struct MeResponse: Decodable {
    let user: CurrentUser
    let csrfToken: String?
}

/// Response of POST /auth/token, the mobile auth handoff. The session token
/// is stored as the session cookie in HTTPCookieStorage; API calls then
/// authenticate exactly like the web frontend's.
struct TokenResponse: Decodable {
    let sessionToken: String
    let cookieName: String
    let expiresAt: Date
}
