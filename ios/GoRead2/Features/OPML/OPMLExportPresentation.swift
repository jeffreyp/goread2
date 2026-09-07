import SwiftUI
#if os(iOS)
import UIKit
#endif

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

extension View {
    /// Offers a finished export to the user: the share sheet on iOS, and on
    /// macOS the save panel `fileExporter` presents, which is what grants a
    /// sandboxed app write access to the chosen location. `onError` reports
    /// a failed save; the iOS share sheet has no equivalent failure.
    @ViewBuilder
    func fileExport(item: Binding<OPMLExport?>,
                    onError: @escaping (String) -> Void) -> some View {
        #if os(iOS)
        sheet(item: item) { export in
            ActivityView(items: [export.url])
        }
        #else
        fileExporter(isPresented: Binding(get: { item.wrappedValue != nil },
                                          set: { if !$0 { item.wrappedValue = nil } }),
                     document: item.wrappedValue.map { OPMLDocument(data: $0.data) },
                     contentType: .opml,
                     defaultFilename: OPMLExport.baseName) { result in
            item.wrappedValue = nil
            if case .failure(let error) = result {
                onError(error.localizedDescription)
            }
        }
        #endif
    }
}
