import Foundation

/// Shared networking layer for the GoRead2 REST API. All feature screens go
/// through this client so cookies (session), CSRF handling, JSON coding, and
/// error mapping live in one place.
///
/// Uses URLSession.shared, whose cookie store is HTTPCookieStorage.shared;
/// the OAuth flow injects the session cookie there and every request here
/// carries it automatically.
final class NetworkClient {
    static let shared = NetworkClient()

    let baseURL: URL

    /// CSRF token from the last GET /auth/me, injected as X-CSRF-Token into
    /// every mutating request.
    private(set) var csrfToken: String?

    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder

    init(baseURL: URL? = nil, session: URLSession = .shared) {
        if let baseURL {
            self.baseURL = baseURL
        } else {
            guard let urlString = Bundle.main.object(forInfoDictionaryKey: "APIBaseURL") as? String,
                  let url = URL(string: urlString) else {
                fatalError("APIBaseURL missing from Info.plist")
            }
            self.baseURL = url
        }
        self.session = session

        decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        decoder.dateDecodingStrategy = .custom({ @Sendable decoder in
            try Self.decodeRFC3339Date(decoder)
        })

        encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
    }

    // MARK: - Feeds

    func getFeeds() async throws -> [Feed] {
        try await decode([Feed]?.self, from: get("/api/feeds")) ?? []
    }

    func addFeed(url: String) async throws -> Feed {
        struct Request: Encodable { let url: String }
        return try await decode(Feed.self, from: post("/api/feeds", body: Request(url: url)))
    }

    func deleteFeed(id: Int) async throws {
        _ = try await send(request(path: "/api/feeds/\(id)", method: "DELETE"))
    }

    func refreshFeeds() async throws {
        _ = try await post("/api/feeds/refresh")
    }

    func getUnreadCounts() async throws -> [Int: Int] {
        let counts = try await decode([String: Int].self, from: get("/api/feeds/unread-counts"))
        return counts.reduce(into: [:]) { result, entry in
            if let id = Int(entry.key) { result[id] = entry.value }
        }
    }

    // MARK: - Articles

    /// feedID "all" returns articles across every subscribed feed; the
    /// backend special-cases it.
    func getArticles(feedID: String = "all",
                     cursor: String? = nil,
                     unreadOnly: Bool = false,
                     limit: Int = 50) async throws -> PaginatedArticles {
        var query = [URLQueryItem(name: "limit", value: String(limit))]
        if let cursor { query.append(URLQueryItem(name: "cursor", value: cursor)) }
        if unreadOnly { query.append(URLQueryItem(name: "unread_only", value: "true")) }
        return try await decode(PaginatedArticles.self,
                                from: get("/api/feeds/\(feedID)/articles", query: query))
    }

    /// Fetches a single article, including the full content that the list
    /// endpoint often omits.
    func getArticle(id: Int) async throws -> Article {
        try await decode(Article.self, from: get("/api/articles/\(id)"))
    }

    func markRead(articleID: Int, isRead: Bool, feedID: Int, wasRead: Bool) async throws {
        struct Request: Encodable {
            let isRead: Bool
            let feedId: Int
            let wasRead: Bool
        }
        _ = try await post("/api/articles/\(articleID)/read",
                           body: Request(isRead: isRead, feedId: feedID, wasRead: wasRead))
    }

    func toggleStar(articleID: Int) async throws {
        _ = try await post("/api/articles/\(articleID)/star")
    }

    /// Returns the number of articles marked read.
    @discardableResult
    func markAllRead() async throws -> Int {
        struct Response: Decodable { let articlesCount: Int }
        return try await decode(Response.self, from: post("/api/articles/mark-all-read")).articlesCount
    }

    // MARK: - OPML

    /// Returns the number of feeds imported.
    func importOPML(_ opmlData: Data, filename: String = "subscriptions.opml") async throws -> Int {
        let boundary = "goread2-\(UUID().uuidString)"
        var body = Data()
        body.append(Data("--\(boundary)\r\n".utf8))
        body.append(Data("Content-Disposition: form-data; name=\"opml\"; filename=\"\(filename)\"\r\n".utf8))
        body.append(Data("Content-Type: application/xml\r\n\r\n".utf8))
        body.append(opmlData)
        body.append(Data("\r\n--\(boundary)--\r\n".utf8))

        var req = request(path: "/api/feeds/import", method: "POST")
        req.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
        req.httpBody = body

        struct Response: Decodable { let importedCount: Int }
        return try await decode(Response.self, from: send(req)).importedCount
    }

    /// Returns the OPML document for the share sheet.
    func exportOPML() async throws -> Data {
        try await get("/api/feeds/export")
    }

    // MARK: - Account and subscription

    func getAccountStats() async throws -> AccountStats {
        try await decode(AccountStats.self, from: get("/api/account/stats"))
    }

    func getSubscriptionInfo() async throws -> SubscriptionInfo {
        try await decode(SubscriptionInfo.self, from: get("/api/subscription"))
    }

    func updateMaxArticles(_ maxArticles: Int) async throws {
        struct Request: Encodable { let maxArticles: Int }
        var req = request(path: "/api/account/max-articles", method: "PUT")
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = try encoder.encode(Request(maxArticles: maxArticles))
        _ = try await send(req)
    }

    /// Returns the Stripe customer portal URL to open in a browser.
    func createPortalSession() async throws -> URL {
        struct Response: Decodable { let portalUrl: URL }
        return try await decode(Response.self, from: post("/api/subscription/portal")).portalUrl
    }

    // MARK: - Auth

    /// Fetches the signed-in user and stores the CSRF token for subsequent
    /// mutating requests.
    @discardableResult
    func fetchMe() async throws -> MeResponse {
        let me = try await decode(MeResponse.self, from: get("/auth/me"))
        if let token = me.csrfToken, !token.isEmpty {
            csrfToken = token
        }
        return me
    }

    /// Exchanges the one-time code from the goread2://auth callback for the
    /// session token (mobile auth handoff).
    func exchangeAuthCode(_ code: String) async throws -> TokenResponse {
        struct Request: Encodable { let code: String }
        return try await decode(TokenResponse.self, from: post("/auth/token", body: Request(code: code)))
    }

    func logout() async throws {
        _ = try await post("/auth/logout")
        csrfToken = nil
    }

    // MARK: - Request building

    private func request(path: String, method: String, query: [URLQueryItem] = []) -> URLRequest {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)!
        components.path = path
        if !query.isEmpty {
            components.queryItems = query
        }
        var req = URLRequest(url: components.url!)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Accept")
        if method != "GET", let csrfToken {
            req.setValue(csrfToken, forHTTPHeaderField: "X-CSRF-Token")
        }
        return req
    }

    private func get(_ path: String, query: [URLQueryItem] = []) async throws -> Data {
        try await send(request(path: path, method: "GET", query: query))
    }

    private func post(_ path: String) async throws -> Data {
        try await send(request(path: path, method: "POST"))
    }

    private func post<Body: Encodable>(_ path: String, body: Body) async throws -> Data {
        var req = request(path: path, method: "POST")
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = try encoder.encode(body)
        return try await send(req)
    }

    // MARK: - Response handling

    private func send(_ request: URLRequest) async throws -> Data {
        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: request)
        } catch {
            throw NetworkError.noConnection
        }

        guard let http = response as? HTTPURLResponse else {
            throw NetworkError.serverError(statusCode: 0, message: "Invalid response")
        }

        switch http.statusCode {
        case 200...299:
            return data
        case 401:
            throw NetworkError.unauthorized
        case 402:
            throw NetworkError.paymentRequired(message: errorMessage(in: data))
        case 404:
            throw NetworkError.notFound
        default:
            throw NetworkError.serverError(statusCode: http.statusCode,
                                           message: errorMessage(in: data))
        }
    }

    private func decode<T: Decodable>(_ type: T.Type, from data: Data) throws -> T {
        do {
            return try decoder.decode(type, from: data)
        } catch {
            throw NetworkError.decodingError(underlying: error)
        }
    }

    /// Extracts the {"error": "..."} envelope every handler uses for failures.
    private func errorMessage(in data: Data) -> String {
        struct ErrorResponse: Decodable { let error: String }
        if let response = try? JSONDecoder().decode(ErrorResponse.self, from: data) {
            return response.error
        }
        return "The server returned an unexpected response."
    }

    // MARK: - Date decoding

    /// Go's encoding/json emits time.Time as RFC 3339 with optional
    /// fractional seconds (RFC3339Nano), which no single built-in strategy
    /// parses.
    private static func decodeRFC3339Date(_ decoder: Decoder) throws -> Date {
        let container = try decoder.singleValueContainer()
        let string = try container.decode(String.self)

        let fractional = ISO8601DateFormatter()
        fractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = fractional.date(from: string) {
            return date
        }

        let plain = ISO8601DateFormatter()
        plain.formatOptions = [.withInternetDateTime]
        if let date = plain.date(from: string) {
            return date
        }

        // RFC3339Nano can carry more fractional digits than
        // ISO8601DateFormatter accepts.
        let nano = DateFormatter()
        nano.locale = Locale(identifier: "en_US_POSIX")
        nano.dateFormat = "yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSSXXXXX"
        if let date = nano.date(from: string) {
            return date
        }

        throw DecodingError.dataCorruptedError(in: container,
                                               debugDescription: "Unparseable RFC 3339 date: \(string)")
    }
}
