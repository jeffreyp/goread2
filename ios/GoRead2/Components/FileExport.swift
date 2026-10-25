import SwiftUI
import UniformTypeIdentifiers
#if os(iOS)
import UIKit
#endif

/// A finished download, such as the OPML or the account data export,
/// waiting to be handed to the user. The iOS share sheet needs a file on
/// disk, so the document is written to a temporary copy there; the macOS
/// save panel writes the document itself and needs only the bytes.
struct ExportedFile: Identifiable {
    let id = UUID()
    let filename: String
    let contentType: UTType

    #if os(iOS)
    let url: URL
    #else
    let data: Data
    #endif

    init(data: Data, filename: String, contentType: UTType) throws {
        self.filename = filename
        self.contentType = contentType
        #if os(iOS)
        let fileURL = FileManager.default.temporaryDirectory
            .appendingPathComponent(filename)
        try data.write(to: fileURL, options: .atomic)
        url = fileURL
        #else
        self.data = data
        #endif
    }

    /// The filename without its extension, which the save panel appends
    /// from the content type.
    var baseName: String {
        (filename as NSString).deletingPathExtension
    }
}

#if os(iOS)
/// UIActivityViewController wrapper for the system share sheet.
struct ActivityView: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ controller: UIActivityViewController, context: Context) {}
}
#endif

#if os(macOS)
/// An exported file, in the form `fileExporter` writes through the save
/// panel.
struct ExportedDocument: FileDocument {
    static var readableContentTypes: [UTType] { [.opml, .xml, .json] }

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

extension View {
    /// Offers a finished export to the user: the share sheet on iOS, and on
    /// macOS the save panel `fileExporter` presents, which is what grants a
    /// sandboxed app write access to the chosen location. `onError` reports
    /// a failed save; the iOS share sheet has no equivalent failure.
    @ViewBuilder
    func fileExport(item: Binding<ExportedFile?>,
                    onError: @escaping (String) -> Void) -> some View {
        #if os(iOS)
        sheet(item: item) { export in
            ActivityView(items: [export.url])
        }
        #else
        fileExporter(isPresented: Binding(get: { item.wrappedValue != nil },
                                          set: { if !$0 { item.wrappedValue = nil } }),
                     document: item.wrappedValue.map { ExportedDocument(data: $0.data) },
                     contentType: item.wrappedValue?.contentType ?? .data,
                     defaultFilename: item.wrappedValue?.baseName) { result in
            item.wrappedValue = nil
            if case .failure(let error) = result {
                onError(error.localizedDescription)
            }
        }
        #endif
    }
}
