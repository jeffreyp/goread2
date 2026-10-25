import Foundation
import SwiftUI
import UniformTypeIdentifiers

/// The OPML import and export API calls, shared by the settings screen and,
/// on macOS, the File menu commands.
struct OPMLTransfer {
    private let client: NetworkClient

    init(client: NetworkClient = .shared) {
        self.client = client
    }

    /// Uploads the OPML document at `url`, a security-scoped URL from a file
    /// panel, and returns the confirmation message for the import.
    func importFile(at url: URL) async throws -> String {
        let accessing = url.startAccessingSecurityScopedResource()
        defer {
            if accessing { url.stopAccessingSecurityScopedResource() }
        }
        let data = try Data(contentsOf: url)
        let count = try await client.importOPML(data, filename: url.lastPathComponent)
        return count == 1
            ? "Successfully imported 1 feed from OPML file"
            : "Successfully imported \(count) feeds from OPML file"
    }

    static let exportFilename = "goread2-subscriptions.opml"

    /// Downloads the account's subscriptions as OPML, ready for presentation.
    func export() async throws -> ExportedFile {
        try ExportedFile(data: await client.exportOPML(),
                         filename: Self.exportFilename,
                         contentType: .opml)
    }
}

extension UTType {
    /// OPML documents use the .opml extension, which is not a registered
    /// system type and does not conform to public.xml, so the type is
    /// derived from the extension with XML as the fallback.
    static let opml = UTType(filenameExtension: "opml") ?? .xml

    /// Types the OPML open panel accepts. Some readers export subscriptions
    /// with a .xml extension instead.
    static let opmlImportTypes: [UTType] = {
        guard let opml = UTType(filenameExtension: "opml") else { return [.xml] }
        return [opml, .xml]
    }()
}
