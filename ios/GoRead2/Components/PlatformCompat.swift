import SwiftUI

/// The SwiftUI bridge for a platform's native view class, so a wrapper type
/// declares one conformance and supplies `makeUIView`/`makeNSView` under an
/// `#if` inside its body.
#if os(iOS)
typealias PlatformViewRepresentable = UIViewRepresentable
#else
typealias PlatformViewRepresentable = NSViewRepresentable
#endif

/// Shims for SwiftUI modifiers that exist only on one platform, so shared
/// views read the same on iOS and macOS instead of carrying an `#if` at
/// every call site.
extension View {
    /// Compact navigation title, matching the iOS look inside sheets and
    /// split-view columns. macOS has no title display mode, so the modifier
    /// is dropped there.
    @ViewBuilder
    func inlineNavigationTitle() -> some View {
        #if os(iOS)
        navigationBarTitleDisplayMode(.inline)
        #else
        self
        #endif
    }

    /// Half-height sheet on iOS. macOS sizes sheets from their content, so
    /// the modifier is dropped and a minimum width keeps the form readable.
    @ViewBuilder
    func mediumSheet() -> some View {
        #if os(iOS)
        presentationDetents([.medium])
        #else
        frame(minWidth: 420)
        #endif
    }
}

extension View {
    /// Number-only keyboard for numeric fields. macOS types numbers on the
    /// hardware keyboard and has no equivalent modifier.
    @ViewBuilder
    func numericKeyboard() -> some View {
        #if os(iOS)
        keyboardType(.numberPad)
        #else
        self
        #endif
    }

    /// Feed-address entry: the URL keyboard with autocapitalisation off on
    /// iOS, and autocorrection off on both platforms.
    @ViewBuilder
    func urlFieldInput() -> some View {
        #if os(iOS)
        keyboardType(.URL)
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
        #else
        autocorrectionDisabled()
        #endif
    }

    /// Bounds for a `NavigationSplitView` column. iOS and iPadOS size their
    /// columns from the device, so the constraint applies on macOS only,
    /// where a window can be any width and the three panes need floors to
    /// stay usable.
    @ViewBuilder
    func splitColumnWidth(min: CGFloat, ideal: CGFloat, max: CGFloat = .infinity) -> some View {
        #if os(macOS)
        navigationSplitViewColumnWidth(min: min, ideal: ideal, max: max)
        #else
        self
        #endif
    }

    /// Caps a full-width control on macOS, where a window is far wider than
    /// a phone and an edge-to-edge button looks wrong. iOS keeps the
    /// full-width layout.
    @ViewBuilder
    func macContentWidth(_ width: CGFloat) -> some View {
        #if os(macOS)
        frame(maxWidth: width)
        #else
        self
        #endif
    }

    /// Capsule border for small icon buttons. macOS gained the shape in
    /// 14.0, so on Ventura the button keeps its default border.
    @ViewBuilder
    func capsuleButtonShape() -> some View {
        #if os(macOS)
        if #available(macOS 14.0, *) {
            buttonBorderShape(.capsule)
        } else {
            self
        }
        #else
        buttonBorderShape(.capsule)
        #endif
    }

    /// Opaque sidebar background. The system sidebar list style is
    /// translucent, so on iPad the content column's selected-article
    /// highlight bleeds through in landscape. Mac sidebars are expected to
    /// keep the system material, so the override applies only on iOS.
    @ViewBuilder
    func opaqueSidebarBackground() -> some View {
        #if os(iOS)
        scrollContentBackground(.hidden)
            .background(Color(.systemBackground))
        #else
        self
        #endif
    }
}

/// Toolbar placement for the reader's article navigation controls: the
/// bottom bar on iOS, the window toolbar on macOS, which has no bottom bar.
var readerToolbarPlacement: ToolbarItemPlacement {
    #if os(iOS)
    .bottomBar
    #else
    .automatic
    #endif
}
