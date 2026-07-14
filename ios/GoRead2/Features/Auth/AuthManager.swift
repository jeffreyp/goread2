import AuthenticationServices
import Foundation

/// Owns the app's session state: bootstrapping from the persisted cookie,
/// the ASWebAuthenticationSession sign-in flow, and sign-out.
@MainActor
final class AuthManager: ObservableObject {
    enum SessionState {
        /// Checking the persisted session on launch.
        case checking
        case signedOut
        case signedIn(CurrentUser)
    }

    @Published private(set) var state: SessionState = .checking
    @Published var errorMessage: String?

    private let client: NetworkClient
    private let webSession = WebAuthSession()

    init(client: NetworkClient = .shared) {
        self.client = client
        // Session cookies arrive via Set-Cookie on API responses (the backend
        // refreshes the sliding session); accept them unconditionally.
        HTTPCookieStorage.shared.cookieAcceptPolicy = .always
    }

    /// Validates the persisted session cookie on launch. Also fetches the
    /// CSRF token when the session is still valid.
    func bootstrap() async {
        do {
            let me = try await client.fetchMe()
            state = .signedIn(me.user)
        } catch {
            state = .signedOut
        }
    }

    /// Runs the Google OAuth flow: /auth/login?client=ios inside
    /// ASWebAuthenticationSession, then exchanges the one-time code from the
    /// goread2://auth callback for the session token.
    func signIn() async {
        errorMessage = nil

        var components = URLComponents(url: client.baseURL, resolvingAgainstBaseURL: false)!
        components.path = "/auth/login"
        components.queryItems = [URLQueryItem(name: "client", value: "ios")]

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
            storeSessionCookie(token)
            let me = try await client.fetchMe()
            state = .signedIn(me.user)
        } catch let authError as ASWebAuthenticationSessionError where authError.code == .canceledLogin {
            // The user dismissed the sheet; not an error.
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

    private func storeSessionCookie(_ token: TokenResponse) {
        var properties: [HTTPCookiePropertyKey: Any] = [
            .name: token.cookieName,
            .value: token.sessionToken,
            .domain: client.baseURL.host ?? "",
            .path: "/",
            .expires: token.expiresAt,
        ]
        if client.baseURL.scheme == "https" {
            properties[.secure] = "TRUE"
        }
        if let cookie = HTTPCookie(properties: properties) {
            HTTPCookieStorage.shared.setCookie(cookie)
        }
    }

    private func clearLocalSession() {
        if let cookies = HTTPCookieStorage.shared.cookies(for: client.baseURL) {
            cookies.forEach(HTTPCookieStorage.shared.deleteCookie)
        }
        state = .signedOut
    }
}
