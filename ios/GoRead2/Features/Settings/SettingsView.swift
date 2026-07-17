import SwiftUI
import UniformTypeIdentifiers

/// Account management screen: the signed-in user, subscription status with
/// the Stripe customer portal, OPML import/export, the max-articles
/// preference, and sign-out.
struct SettingsView: View {
    @EnvironmentObject private var authManager: AuthManager
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = SettingsViewModel()

    @State private var showingImporter = false

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.hasLoaded {
                    settingsForm
                } else {
                    ProgressView()
                }
            }
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") {
                        dismiss()
                    }
                }
            }
        }
        .sheet(item: $viewModel.portalItem) { item in
            SafariView(url: item.url)
                .ignoresSafeArea()
        }
        .sheet(item: $viewModel.exportedOPML) { export in
            ActivityView(items: [export.url])
        }
        .fileImporter(isPresented: $showingImporter,
                      allowedContentTypes: opmlContentTypes) { result in
            switch result {
            case .success(let url):
                Task { await viewModel.importOPML(from: url) }
            case .failure(let error):
                viewModel.errorMessage = error.localizedDescription
            }
        }
        .alert("OPML Import", isPresented: infoBinding) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(viewModel.infoMessage ?? "")
        }
        .alert("Error", isPresented: errorBinding) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(viewModel.errorMessage ?? "")
        }
        .task {
            viewModel.onSessionExpired = { authManager.sessionExpired() }
            await viewModel.load()
        }
    }

    private var settingsForm: some View {
        Form {
            if let user = viewModel.user {
                accountSection(user)
            }
            if let subscription = viewModel.subscription {
                subscriptionSection(subscription)
            }
            preferencesSection
            opmlSection
            signOutSection
        }
    }

    // MARK: - Account

    private func accountSection(_ user: CurrentUser) -> some View {
        Section("Account") {
            HStack(spacing: 12) {
                AsyncImage(url: URL(string: user.avatar)) { image in
                    image.resizable().scaledToFill()
                } placeholder: {
                    Image(systemName: "person.crop.circle.fill")
                        .resizable()
                        .foregroundStyle(.secondary)
                }
                .frame(width: 48, height: 48)
                .clipShape(Circle())

                VStack(alignment: .leading, spacing: 2) {
                    Text(user.name)
                        .font(.headline)
                    Text(user.email)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            }
            .padding(.vertical, 4)
        }
    }

    // MARK: - Subscription

    private func subscriptionSection(_ subscription: SubscriptionInfo) -> some View {
        Section("Subscription") {
            LabeledContent("Status", value: statusLabel(subscription.status))

            if subscription.status == "trial" {
                LabeledContent("Trial ends") {
                    Text(trialEndsLabel(subscription))
                }
            }
            if let nextBillingDate = subscription.nextBillingDate {
                LabeledContent("Next billing",
                               value: nextBillingDate.formatted(date: .abbreviated, time: .omitted))
            }

            LabeledContent("Feeds", value: feedUsageLabel(subscription))

            if viewModel.canManageSubscription {
                Button("Manage Subscription") {
                    Task { await viewModel.openSubscriptionPortal() }
                }
                .disabled(viewModel.isBusy)
            }
        }
    }

    private func statusLabel(_ status: String) -> String {
        switch status {
        case "trial": return "Trial"
        case "active": return "Active"
        case "cancelled": return "Cancelled"
        case "expired": return "Expired"
        case "admin": return "Admin"
        case "unlimited": return "Unlimited"
        default: return status.capitalized
        }
    }

    private func trialEndsLabel(_ subscription: SubscriptionInfo) -> String {
        let date = subscription.trialEndsAt.formatted(date: .abbreviated, time: .omitted)
        guard let days = subscription.trialDaysRemaining else { return date }
        return days == 1 ? "\(date) (1 day left)" : "\(date) (\(days) days left)"
    }

    private func feedUsageLabel(_ subscription: SubscriptionInfo) -> String {
        if subscription.feedLimit < 0 {
            return "\(subscription.currentFeeds)"
        }
        return "\(subscription.currentFeeds) of \(subscription.feedLimit)"
    }

    // MARK: - Preferences

    private var preferencesSection: some View {
        Section {
            HStack {
                Text("Max articles on feed add")
                Spacer()
                TextField("Max", value: $viewModel.maxArticles, format: .number)
                    .keyboardType(.numberPad)
                    .multilineTextAlignment(.trailing)
                    .frame(maxWidth: 80)
            }
            if viewModel.maxArticlesEdited {
                Button("Save") {
                    Task { await viewModel.saveMaxArticles() }
                }
            }
        } header: {
            Text("Preferences")
        } footer: {
            Text("Limits how many articles are imported when subscribing to a new feed. 0 means no limit.")
        }
    }

    // MARK: - OPML

    private var opmlSection: some View {
        Section {
            Button {
                showingImporter = true
            } label: {
                Label("Import OPML", systemImage: "square.and.arrow.down")
            }
            Button {
                Task { await viewModel.exportOPML() }
            } label: {
                Label("Export OPML", systemImage: "square.and.arrow.up")
            }
        } header: {
            Text("Feeds")
        } footer: {
            Text("OPML files carry feed subscriptions between RSS readers.")
        }
        .disabled(viewModel.isBusy)
    }

    // MARK: - Sign out

    private var signOutSection: some View {
        Section {
            Button("Sign Out", role: .destructive) {
                Task { await authManager.signOut() }
            }
        }
    }

    // MARK: - Helpers

    /// OPML documents commonly use the .opml extension, which does not
    /// conform to public.xml, so both types are accepted.
    private var opmlContentTypes: [UTType] {
        var types: [UTType] = [.xml]
        if let opml = UTType(filenameExtension: "opml") {
            types.append(opml)
        }
        return types
    }

    private var infoBinding: Binding<Bool> {
        Binding(
            get: { viewModel.infoMessage != nil },
            set: { if !$0 { viewModel.infoMessage = nil } }
        )
    }

    private var errorBinding: Binding<Bool> {
        Binding(
            get: { viewModel.errorMessage != nil },
            set: { if !$0 { viewModel.errorMessage = nil } }
        )
    }
}

#Preview {
    SettingsView()
        .environmentObject(AuthManager())
}
