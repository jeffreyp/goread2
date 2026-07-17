import SwiftUI

struct LoginView: View {
    @EnvironmentObject private var authManager: AuthManager
    @State private var isSigningIn = false
    @ScaledMetric(relativeTo: .largeTitle) private var iconSize: CGFloat = 56

    var body: some View {
        VStack(spacing: 24) {
            Spacer()

            VStack(spacing: 12) {
                Image(systemName: "book")
                    .font(.system(size: iconSize))
                    .foregroundStyle(.tint)
                Text("GoRead2")
                    .font(.largeTitle.bold())
                Text("An RSS reader")
                    .foregroundStyle(.secondary)
            }

            Spacer()

            if let error = authManager.errorMessage {
                Text(error)
                    .font(.callout)
                    .foregroundStyle(.red)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal)
            }

            Button {
                isSigningIn = true
                Task {
                    await authManager.signIn()
                    isSigningIn = false
                }
            } label: {
                if isSigningIn {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                } else {
                    Text("Sign in with Google")
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)
            .disabled(isSigningIn)
            .padding(.horizontal, 32)
            .padding(.bottom, 48)
        }
    }
}

#Preview {
    LoginView()
        .environmentObject(AuthManager())
}
