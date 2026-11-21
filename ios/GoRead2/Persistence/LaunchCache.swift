import Foundation

/// The last session's first-screen data, persisted so the next launch can
/// render it at once while the server is asked for fresh data
/// (stale-while-revalidate): the feed list, the unread counts, and the
/// unread-only first page of All Articles.
///
/// Each part is one JSON file under Caches, in a folder per backend host so
/// Debug and Release builds keep separate data, like `SessionStore`. The
/// system may purge Caches at any time, which only costs one launch its
/// head start, and nothing there is backed up.
///
/// The data belongs to the signed-in account, so `deactivate()` deletes it
/// when the session ends and drops any write that arrives after that, such
/// as a response from the old session landing late. `activate()` turns
/// writes back on for the next session.
///
/// Reads are synchronous, so a screen can show its cached data in its first
/// frame; the files are small. Writes are encoded on the caller and written
/// atomically on a background queue, in the order they were made.
@MainActor
final class LaunchCache {
    static let shared = LaunchCache(baseURL: NetworkClient.shared.baseURL)

    struct FeedList: Codable {
        let feeds: [Feed]
        let unreadCounts: [Int: Int]
    }

    struct ArticlePage: Codable {
        let articles: [Article]
        let nextCursor: String?
    }

    private enum Entry: String {
        case feedList = "feed-list.json"
        case allUnreadArticles = "all-unread-articles.json"
    }

    private let directory: URL
    private let queue = DispatchQueue(label: "org.jeffreypratt.goread2.launch-cache", qos: .utility)
    private var isActive = false

    init(baseURL: URL) {
        let caches = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask)[0]
        directory = caches
            .appendingPathComponent("LaunchCache", isDirectory: true)
            .appendingPathComponent(baseURL.host ?? "default", isDirectory: true)
    }

    // MARK: - Session lifecycle

    /// Allows writes; called when a session starts or is restored.
    func activate() {
        isActive = true
    }

    /// Deletes the cached data and drops writes until the next `activate()`;
    /// called whenever the session ends.
    func deactivate() {
        isActive = false
        let directory = directory
        queue.async {
            try? FileManager.default.removeItem(at: directory)
        }
    }

    // MARK: - Feed list

    func loadFeedList() -> FeedList? {
        read(FeedList.self, from: .feedList)
    }

    func saveFeedList(feeds: [Feed], unreadCounts: [Int: Int]) {
        write(FeedList(feeds: feeds, unreadCounts: unreadCounts), to: .feedList)
    }

    // MARK: - All Articles, unread only

    func loadAllUnreadArticles() -> ArticlePage? {
        read(ArticlePage.self, from: .allUnreadArticles)
    }

    func saveAllUnreadArticles(_ page: ArticlePage) {
        write(page, to: .allUnreadArticles)
    }

    // MARK: - File access

    private func url(for entry: Entry) -> URL {
        directory.appendingPathComponent(entry.rawValue)
    }

    private func read<T: Decodable>(_ type: T.Type, from entry: Entry) -> T? {
        // Nothing is shown from the cache while no session is active.
        guard isActive, let data = try? Data(contentsOf: url(for: entry)) else { return nil }
        // A file from an older build whose models no longer decode is
        // treated as absent; the next save replaces it.
        return try? JSONDecoder().decode(type, from: data)
    }

    private func write<T: Encodable>(_ value: T, to entry: Entry) {
        guard isActive, let data = try? JSONEncoder().encode(value) else { return }
        let directory = directory
        let url = url(for: entry)
        queue.async {
            do {
                try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
                try data.write(to: url, options: .atomic)
            } catch {
                // A cache that cannot be written costs only the next launch's
                // head start.
            }
        }
    }
}
