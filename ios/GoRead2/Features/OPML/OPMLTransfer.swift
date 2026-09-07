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

    /// Downloads the account's subscriptions as OPML, ready for presentation.
    func export() async throws -> OPMLExport {
        try OPMLExport(data: await client.exportOPML())
    }
}

/// A finished OPML export waiting to be handed to the user. The iOS share
/// sheet needs a file on disk, so the document is written to a temporary
/// copy there; the macOS save panel writes the document itself and needs
/// only the bytes.
struct OPMLExport: Identifiable {
    static let baseName = "goread2-subscriptions"
    static let filename = "\(baseName).opml"

    let id = UUID()

    #if os(iOS)
    let url: URL
    #else
    let data: Data
    #endif

    init(data: Data) throws {
        #if os(iOS)
        let fileURL = FileManager.default.temporaryDirectory
            .appendingPathComponent(Self.filename)
        try data.write(to: fileURL, options: .atomic)
        url = fileURL
        #else
        self.data = data
        #endif
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

#if os(macOS)
/// The exported subscriptions, in the form `fileExporter` writes through the
/// save panel.
struct OPMLDocument: FileDocument {
    static var readableContentTypes: [UTType] { [.opml, .xml] }

    let data: Data

    init(data: Data) {
        self.data = data
    }

    init(configuration: ReadConfiguration) throws {
        guard let contents = configuration.file.regularFileContents else {
            throw CocoaError(.fileReadCorruptFile)
        }
        data = contents
    }

    func fileWrapper(configuration: WriteConfiguration) throws -> FileWrapper {
        FileWrapper(regularFileWithContents: data)
    }
}
#endif
