import SwiftUI
#if os(iOS)
import UIKit
#else
import AppKit
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
    /// Offers a just-written file to the user: the share sheet on iOS, and
    /// on macOS the file revealed in the Finder, where the export already
    /// lives on disk. gr-mwzq.7 replaces the macOS side with a save panel.
    @ViewBuilder
    func fileExport(item: Binding<OPMLExport?>) -> some View {
        #if os(iOS)
        sheet(item: item) { export in
            ActivityView(items: [export.url])
        }
        #else
        modifier(FinderRevealer(item: item))
        #endif
    }
}

#if os(macOS)
private struct FinderRevealer: ViewModifier {
    @Binding var item: OPMLExport?

    func body(content: Content) -> some View {
        content.task(id: item?.url) {
            guard let url = item?.url else { return }
            NSWorkspace.shared.activateFileViewerSelecting([url])
            item = nil
        }
    }
}
#endif
