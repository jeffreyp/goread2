import Foundation

/// The last session's data, persisted for two uses. At launch a screen
/// renders it at once while the server is asked for fresh data
/// (stale-while-revalidate). Without a connection it is all the app has, and
/// the screens stay readable from it.
///
/// It holds the feed list and unread counts, the first page of every article
/// list the user has opened (each feed and All Articles, with and without
/// the unread-only filter), and the full content of articles the reader has
/// fetched, up to `contentLimit` of them.
///
/// Each part is one file under Caches, in a folder per backend host so
/// Debug and Release builds keep separate data, like `SessionStore`. The
/// system may purge Caches at any time, which only costs a launch its head
/// start or an offline session its articles, and nothing there is backed up.
///
/// The data belongs to the signed-in account, so `deactivate()` deletes it
/// when the session ends and drops any write that arrives after that, such
/// as a response from the old session landing late. `activate()` turns
/// writes back on for the next session.
///
/// List and feed reads are synchronous, so a screen can show its cached data
/// in its first frame; those files are small. Content reads are asynchronous.
/// Writes are encoded on the caller and written atomically on a background
/// queue, in the order they were made.
@MainActor
final class LaunchCache {
    static let shared = LaunchCache(baseURL: NetworkClient.shared.baseURL)

    /// The most article bodies kept on disk. Past this, the least recently
    /// written are deleted.
    static let contentLimit = 500

    struct FeedList: Codable {
        let feeds: [Feed]
        let unreadCounts: [Int: Int]
    }

    struct ArticlePage: Codable {
        let articles: [Article]
        let nextCursor: String?
        /// When the server last returned this page. Local changes, such as
        /// marking an article read, keep the date. Nil in files written
        /// before the field existed.
        let fetchedAt: Date?
    }

    private let directory: URL
    private var contentDirectory: URL {
        directory.appendingPathComponent("Content", isDirectory: true)
    }
    private let queue = DispatchQueue(label: "org.jeffreypratt.goread2.launch-cache", qos: .utility)
    private var isActive = false
    /// Content writes since the last check against `contentLimit`.
    private var contentWritesSincePrune = 0

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
        read(FeedList.self, from: url(for: "feed-list.json"))
    }

    func saveFeedList(feeds: [Feed], unreadCounts: [Int: Int]) {
        write(FeedList(feeds: feeds, unreadCounts: unreadCounts), to: url(for: "feed-list.json"))
    }

    // MARK: - Article lists

    /// The cached first page of the list for `selection` with the given
    /// filter, or nil when that list has never been loaded.
    func loadArticles(for selection: FeedSelection, unreadOnly: Bool) -> ArticlePage? {
        read(ArticlePage.self, from: articlesURL(feedID: selection.apiFeedID, unreadOnly: unreadOnly))
    }

    func saveArticles(_ page: ArticlePage, for selection: FeedSelection, unreadOnly: Bool) {
        write(page, to: articlesURL(feedID: selection.apiFeedID, unreadOnly: unreadOnly))
    }

    /// Deletes both cached lists for a feed the user has unsubscribed from.
    func removeArticles(forFeedID feedID: Int) {
        guard isActive else { return }
        let urls = [true, false].map { articlesURL(feedID: String(feedID), unreadOnly: $0) }
        queue.async {
            for url in urls {
                try? FileManager.default.removeItem(at: url)
            }
        }
    }

    private func articlesURL(feedID: String, unreadOnly: Bool) -> URL {
        url(for: "articles-\(feedID)-\(unreadOnly ? "unread" : "all").json")
    }

    // MARK: - Article content

    /// The full content of the article with `id`, if the reader has fetched
    /// it before.
    func loadContent(articleID id: Int) async -> String? {
        guard isActive else { return nil }
        let url = contentURL(articleID: id)
        return await withCheckedContinuation { continuation in
            // On the write queue, so a write made just before is visible.
            queue.async {
                let data = try? Data(contentsOf: url)
                continuation.resume(returning: data.flatMap { String(data: $0, encoding: .utf8) })
            }
        }
    }

    func saveContent(_ content: String, articleID id: Int) {
        guard isActive, !content.isEmpty else { return }
        let directory = contentDirectory
        let url = contentURL(articleID: id)
        let data = Data(content.utf8)
        contentWritesSincePrune += 1
        // Listing the folder on every write would be wasteful; checking now
        // and then lets it run over the limit by at most this much.
        let prune = contentWritesSincePrune >= 25
        if prune { contentWritesSincePrune = 0 }
        queue.async {
            do {
                try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
                try data.write(to: url, options: .atomic)
            } catch {
                return
            }
            if prune {
                Self.pruneContent(in: directory, limit: Self.contentLimit)
            }
        }
    }

    private func contentURL(articleID id: Int) -> URL {
        contentDirectory.appendingPathComponent("\(id).html")
    }

    /// Deletes the least recently written files in `directory` until at most
    /// `limit` remain.
    private nonisolated static func pruneContent(in directory: URL, limit: Int) {
        let fileManager = FileManager.default
        guard let files = try? fileManager.contentsOfDirectory(at: directory,
                                                               includingPropertiesForKeys: [.contentModificationDateKey]),
              files.count > limit else { return }
        let dated = files.map { file in
            (file, (try? file.resourceValues(forKeys: [.contentModificationDateKey]))?.contentModificationDate ?? .distantPast)
        }
        for (file, _) in dated.sorted(by: { $0.1 < $1.1 }).prefix(files.count - limit) {
            try? fileManager.removeItem(at: file)
        }
    }

    // MARK: - File access

    private func url(for name: String) -> URL {
        directory.appendingPathComponent(name)
    }

    private func read<T: Decodable>(_ type: T.Type, from url: URL) -> T? {
        // Nothing is shown from the cache while no session is active.
        guard isActive, let data = try? Data(contentsOf: url) else { return nil }
        // A file from an older build whose models no longer decode is
        // treated as absent; the next save replaces it.
        return try? JSONDecoder().decode(type, from: data)
    }

    private func write<T: Encodable>(_ value: T, to url: URL) {
        guard isActive, let data = try? JSONEncoder().encode(value) else { return }
        let directory = directory
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
