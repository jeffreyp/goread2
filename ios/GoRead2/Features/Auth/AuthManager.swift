import AuthenticationServices
import Foundation

/// Owns the app's session state: bootstrapping from the persisted session,
/// the ASWebAuthenticationSession sign-in flow, and sign-out.
@MainActor
final class AuthManager: ObservableObject {
    enum SessionState {
        /// Before `bootstrap` has looked for a persisted session.
        case checking
        case signedOut
        case signedIn
    }

    @Published private(set) var state: SessionState = .checking
    @Published var errorMessage: String?
    /// Confirmation shown on the login screen, e.g. after account deletion.
    @Published var noticeMessage: String?

    private let client: NetworkClient
    private let webSession = WebAuthSession()

    init(client: NetworkClient = .shared) {
        self.client = client
    }

    /// Restores the persisted session on launch. A stored credential shows
    /// the signed-in screens at once, while GET /auth/me validates it and
    /// fetches the CSRF token in the background; the first screen's data
    /// loads in parallel. Mutating requests wait for that token inside
    /// `NetworkClient`.
    ///
    /// Only a 401 signs the user out. Any other failure, such as no
    /// connection, leaves the session in place: the screens report their own
    /// load errors, and the next mutating request retries the check.
    func bootstrap() async {
        guard let credential = client.credential else {
            state = .signedOut
            return
        }
        client.prefetchLaunchData()
        state = .signedIn
        do {
            try await client.fetchMe()
        } catch NetworkError.unauthorized {
            // Leave alone a session that replaced this one during the check.
            if client.credential == credential {
                clearLocalSession()
            }
        } catch {
            // Not evidence that the session is invalid.
        }
    }

    /// The ?client= value identifying this build to /auth/login. The server
    /// gives both native clients the goread2:// handoff; the distinct values
    /// keep the two apps separable in server logs.
    static let clientName = {
        #if os(macOS)
        "macos"
        #else
        "ios"
        #endif
    }()

    /// Runs the Google OAuth flow: /auth/login?client=ios (or client=macos)
    /// inside ASWebAuthenticationSession, then exchanges the one-time code
    /// from the goread2://auth callback for the session token.
    func signIn() async {
        errorMessage = nil
        noticeMessage = nil

        var components = URLComponents(url: client.baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/auth/login"
        components.queryItems = [URLQueryItem(name: "client", value: Self.clientName)]

        do {
            let callbackURL = try await webSession.authenticate(url: components.url!,
                                                                callbackScheme: "goread2")
            let query = URLComponents(url: callbackURL, resolvingAgainstBaseURL: false)?.queryItems

            if let serverError = query?.first(where: { $0.name == "error" })?.value {
                errorMessage = serverError
                return
            }
            guard let code = query?.first(where: { $0.name == "code" })?.value else {
                errorMessage = "Sign-in failed: the callback carried no authorization code."
                return
            }

            let token = try await client.exchangeAuthCode(code)
            try client.storeSession(token)
            try await client.fetchMe()
            state = .signedIn
        } catch let authError as ASWebAuthenticationSessionError where authError.code == .canceledLogin {
            // The user dismissed the sheet; not an error.
        } catch is CancellationError {
            // The task was cancelled (e.g. the view went away); not an error.
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// Invalidates the session server-side, then clears local state.
    func signOut() async {
        try? await client.logout()
        clearLocalSession()
    }

    /// Called when an API request fails with 401: the session is gone
    /// server-side, so only local state needs clearing.
    func sessionExpired() {
        clearLocalSession()
    }

    /// Called after DELETE /api/account succeeds. The server has already
    /// revoked every session, so only local state needs clearing.
    func accountDeleted() {
        clearLocalSession()
        noticeMessage = "The account has been deleted."
    }

    private func clearLocalSession() {
        client.clearSession()
        state = .signedOut
    }
}
