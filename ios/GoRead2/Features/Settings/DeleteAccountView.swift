import SwiftUI

/// Confirmation for DELETE /api/account. States the consequences, matching
/// the web account page, and enables the delete button only once the
/// account's email address has been typed, which the server checks again.
struct DeleteAccountView: View {
    @ObservedObject var viewModel: SettingsViewModel
    @Environment(\.dismiss) private var dismiss

    @State private var typedEmail = ""

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text("This permanently deletes the GoRead2 account, its feeds, reading history, and settings. A GoRead2 Pro subscription is cancelled immediately, with no refund for the rest of the billing period. Stripe keeps the payment records it is legally required to retain.")
                    Text("To keep a copy of the data, use Export All Data before deleting. This cannot be undone.")
                        .foregroundStyle(.secondary)
                }

                Section {
                    TextField("Email address", text: $typedEmail)
                        .emailFieldInput()
                        .disabled(viewModel.isDeletingAccount)
                } header: {
                    Text("Type \(viewModel.user?.email ?? "the account's email address") to confirm")
                        .textCase(nil)
                } footer: {
                    if let error = viewModel.deleteAccountError {
                        Text(error)
                            .foregroundStyle(.red)
                    }
                }

                Section {
                    Button(role: .destructive) {
                        Task { await viewModel.deleteAccount(confirmEmail: typedEmail) }
                    } label: {
                        if viewModel.isDeletingAccount {
                            ProgressView()
                        } else {
                            Text("Delete Account")
                        }
                    }
                    .disabled(!viewModel.confirmationMatches(typedEmail) || viewModel.isDeletingAccount)
                }
            }
            .formStyle(.grouped)
            .navigationTitle("Delete Account")
            .inlineNavigationTitle()
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        dismiss()
                    }
                    .disabled(viewModel.isDeletingAccount)
                }
            }
        }
        .interactiveDismissDisabled(viewModel.isDeletingAccount)
        .macSheetSize(width: 460, height: 420)
    }
}
