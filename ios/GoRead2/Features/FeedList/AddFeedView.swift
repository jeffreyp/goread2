import SwiftUI

/// Sheet for subscribing to a new feed. Owns the submit lifecycle so a
/// failed add shows its error inline and keeps the typed URL for
/// correction; the sheet dismisses only when the add succeeds.
struct AddFeedView: View {
    /// Performs the add against the API; throws on failure.
    let addFeed: (String) async throws -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var url = ""
    @State private var isSubmitting = false
    @State private var errorMessage: String?
    @FocusState private var urlFieldFocused: Bool

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    HStack {
                        TextField("example.com or https://example.com/feed.xml",
                                  text: $url)
                            .keyboardType(.URL)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                            .focused($urlFieldFocused)
                            .onSubmit(submit)
                        if url.isEmpty {
                            PasteButton(payloadType: String.self) { strings in
                                guard let pasted = strings.first else { return }
                                Task { @MainActor in
                                    url = pasted.trimmingCharacters(in: .whitespacesAndNewlines)
                                }
                            }
                            .labelStyle(.iconOnly)
                            .buttonBorderShape(.capsule)
                            .controlSize(.small)
                        }
                    }
                } header: {
                    Text("Website or Feed URL")
                } footer: {
                    Text("Enter a website domain (e.g., \"slashdot.org\") or direct feed URL")
                }

                if let errorMessage {
                    Section {
                        Label(errorMessage, systemImage: "exclamationmark.triangle")
                            .foregroundStyle(.red)
                    }
                }
            }
            .navigationTitle("Add Feed")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                        .disabled(isSubmitting)
                }
                ToolbarItem(placement: .confirmationAction) {
                    if isSubmitting {
                        ProgressView()
                    } else {
                        Button("Add", action: submit)
                            .disabled(trimmedURL.isEmpty)
                    }
                }
            }
            .disabled(isSubmitting)
        }
        .presentationDetents([.medium])
        .interactiveDismissDisabled(isSubmitting)
        .onAppear { urlFieldFocused = true }
    }

    private var trimmedURL: String {
        url.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func submit() {
        let target = trimmedURL
        guard !target.isEmpty, !isSubmitting else { return }
        errorMessage = nil
        isSubmitting = true
        Task {
            do {
                try await addFeed(target)
                dismiss()
            } catch {
                errorMessage = error.localizedDescription
                isSubmitting = false
                urlFieldFocused = true
            }
        }
    }
}

#Preview {
    AddFeedView { _ in
        try await Task.sleep(nanoseconds: 1_000_000_000)
    }
}
