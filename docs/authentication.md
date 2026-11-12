# Authentication

This document describes the authentication architecture and implementation details for GoRead2.

## Table of Contents

- [Overview](#overview)
- [Authentication Flow](#authentication-flow)
- [Session Management](#session-management)
- [Environment Isolation](#environment-isolation)
- [Security Considerations](#security-considerations)
- [Implementation Details](#implementation-details)
- [Troubleshooting](#troubleshooting)
- [Related Documentation](#related-documentation)

## Overview

GoRead2 uses Google OAuth 2.0 for user authentication. The system implements session-based authentication with HTTP-only cookies and CSRF protection.

## Authentication Flow

### 1. Login Initiation

```
User clicks "Sign in with Google"
  ↓
App generates OAuth state token
  ↓
State token stored in environment-specific cookie
  ↓
User redirected to Google OAuth consent screen
```

### 2. OAuth Callback

```
Google redirects back with authorization code
  ↓
App verifies state token matches
  ↓
Exchange authorization code for access token
  ↓
Retrieve user profile from Google
  ↓
Create or update user in database
  ↓
Create session and set session cookie
  ↓
Redirect to application
```

`AuthService.SignInGoogleUser` (`internal/auth/auth.go`) performs the create-or-load step. It creates an account only when the lookup by Google ID returns `database.ErrUserNotFound`. Any other lookup error fails the sign-in with a 500, so a transient database failure cannot create a second account for an existing Google identity. A new user whose Google ID matches an unexpired deleted-account tombstone is created with the free trial already ended, so deleting and recreating an account does not grant a second trial. An existing account with a deletion in flight (`DELETE /api/account` was called but has not finished) refuses the sign-in instead, returning `ErrAccountDeletionPending`; the callback reports this to the client as a 403 rather than creating a session. See [privacy.md](privacy.md#account-deletion).

### 3. Authenticated Requests

```
Request includes session cookie
  ↓
Middleware validates session
  ↓
Load user from database
  ↓
Inject user into request context
  ↓
Process request
```

### Native Client Flow (iOS and macOS)

Native apps cannot complete the web flow above: the OAuth sheet (ASWebAuthenticationSession) only dismisses when redirected to the app's registered `goread2://` URL scheme, and cookies set inside the sheet's browsing context never reach the app. The native flow therefore diverges at three points:

```
App opens /auth/login?client=ios (or ?client=macos) inside
ASWebAuthenticationSession
  ↓
Server stores the OAuth state flagged as native and 302-redirects to Google
(top-level navigation, so the state cookie lands in the sheet's browser context)
  ↓
Google redirects to /auth/callback; state validates as in the web flow
  ↓
Server creates the session, then redirects to goread2://auth?code=<one-time-code>
instead of / and sets no session cookie
  ↓
App exchanges the code via POST /auth/token for the session token
  ↓
App stores the token in the Keychain and sends it as the session
cookie; all later API calls authenticate exactly like browser requests
```

The `client` parameter accepts `ios` and `macos`, both of which take the handoff described above. Any other value, including an empty one, takes the web flow, so the two apps share one server-side code path and one `goread2://` URL scheme.

One-time codes are single-use and expire after 2 minutes. The code indirection keeps session tokens out of the `goread2://` callback URL, where they could leak into device logs. Callback failures on the native path redirect to `goread2://auth?error=<message>` so the auth sheet dismisses and the app can surface the error. `POST /auth/token` returns the environment-specific session cookie name alongside the token, so a client pointed at a local backend sets `session_id_local` without hardcoding the environment logic.

The app keeps the token and cookie name in the Keychain, through `SessionStore` in `ios/GoRead2/Networking/`, rather than in a cookie store. On iOS the item is accessible after first unlock and never leaves the device. On macOS it lives in the login keychain. `NetworkClient` uses a `URLSession` with cookie handling disabled and adds the `Cookie` header to each request itself, so the token never enters `HTTPCookieStorage` or its on-disk cookie file. The server extends a session's expiry on activity without re-issuing the cookie, so the stored token stays valid until a 401 or sign-out clears it. On first launch after updating from a build that kept the token in `HTTPCookieStorage`, the app moves it into the Keychain and deletes the old cookies.

Expired codes are purged by the same `/cron/cleanup-sessions` job that removes expired OAuth states.

### Local Dev Login

A server built with the `devlogin` build tag, which `make dev-login` does, replaces the Google consent screen with a local form that signs in as any typed Google ID, email, and name. It exists so that sign-up, data export, account deletion, and the deleted-account tombstone can be tested with disposable accounts. The code lives in `internal/handlers/dev_login.go`; the default build and the App Engine deploy compile `dev_login_off.go` in its place, which registers nothing.

Only the step that asks Google for the user's identity differs. `/auth/login` issues and stores the OAuth state as usual but points at `/auth/dev-login?state=...` instead of Google. The form submits to `/auth/dev-login/callback`, which consumes the state with the same checks as `/auth/callback`, then calls `SignInGoogleUser` and the same session and native handoff code. The web frontend and the iOS and macOS apps therefore use the dev login unchanged, including the `goread2://` code handoff. The form page widens its `form-action` CSP directive to `'self' goread2:`, because the native flow ends in a redirect to the app's URL scheme from the form's submission.

The server refuses to register the dev login on App Engine, with `ENVIRONMENT=production`, or against a real Datastore. See [security.md](security.md#local-dev-login) for the guards and [testing.md](testing.md#local-dev-login-make-dev-login) for usage.

## Session Management

### Session Creation and Storage

`SessionManager.CreateSession` (`internal/auth/session.go`) creates a session after successful OAuth authentication and stores it in the database: a unique session ID (base64-encoded 32 random bytes), the user ID, a creation timestamp, and an expiration timestamp 7 days out.

### Session Validation

On each authenticated request, the middleware extracts the session ID from the cookie, looks up the session, checks whether it has expired, loads the associated user, and injects the user into the request context. A valid session's expiration is then extended by another 7 days (`SessionManager.RefreshSession`), so an active user's session never hits the original deadline; only a session with no activity for 7 days actually expires.

### Session Cleanup

Expired sessions are removed by the `/cron/cleanup-sessions` endpoint (`AuthHandler.CleanupExpiredSessions`), triggered every 24 hours by `cron.yaml`, not a background goroutine. Running cleanup as a cron job rather than an in-process timer avoids keeping an otherwise-idle App Engine instance alive just to fire a periodic goroutine.

## Environment Isolation

GoRead2 implements environment-specific cookie names to prevent authentication state conflicts between local development and production deployments.

### Cookie Names by Environment

| Environment | Session Cookie | OAuth State Cookie |
|-------------|----------------|-------------------|
| **Local** (development) | `session_id_local` | `oauth_state_local` |
| **Production** (GAE) | `session_id` | `oauth_state` |

### Environment Detection

`SessionManager.getCookieName()` picks the session cookie name above based on the same environment check used throughout the server:

```go
isProduction := os.Getenv("GAE_ENV") == "standard" ||
                os.Getenv("ENVIRONMENT") == "production"
```

The auth handler's `getOAuthStateCookieName()` applies the same check to the OAuth state cookie.

## Security Considerations

### Cookie Attributes

Session cookies are configured with the following attributes. The OAuth state cookie, set by `setOAuthStateCookie` in `internal/handlers/auth_handler.go`, uses the same `HttpOnly`, `Secure`, `SameSite`, and `Path` values, with a 10-minute `MaxAge`:

```go
cookie := &http.Cookie{
    Name:     sm.getCookieName(),
    Value:    session.ID,
    Expires:  session.ExpiresAt,
    HttpOnly: true,                    // Prevents JavaScript access
    Secure:   isProduction,            // HTTPS-only in production
    SameSite: http.SameSiteLaxMode,   // CSRF protection
    Path:     "/",                     // Available to entire app
}
```

### Key Security Features

- **HTTP-Only cookies** - Not accessible to JavaScript, so a successful XSS attack still can't read the session cookie and steal the session
- **Secure flag in production** - Cookies only transmitted over HTTPS in production
- **SameSite Lax** - Protects against CSRF attacks by default
- **Session expiration** - Sessions expire after 7 days of inactivity (see [Session Validation](#session-validation) above)
- **Automatic cleanup** - Expired sessions removed from database
- **Session fixation protection** - Any existing session is invalidated before creating a new one on login

### CSRF Protection

`GET /auth/me` returns an HMAC-derived CSRF token alongside the user's profile; every state-changing request (`POST`, `PUT`, `DELETE`, `PATCH`) must carry it in the `X-CSRF-Token` header. See [security.md](security.md#csrf-protection) for how the token is generated and verified, and a client-side example.

## Implementation Details

### File Structure

- `internal/auth/session.go` - Session manager implementation
- `internal/auth/middleware.go` - Authentication middleware
- `internal/handlers/auth_handler.go` - OAuth handlers
- `internal/handlers/dev_login.go` - Local dev login, compiled only with the `devlogin` build tag
- `internal/auth/csrf.go` - CSRF token management

### Configuration

[configuration.md](configuration.md#authentication-and-sessions) lists the OAuth and session variables: the three required OAuth variables, `STAGING_REDIRECT_URL`, `SESSION_CACHE_TTL`, and `ENVIRONMENT`, which forces production behavior outside App Engine.

### Host-Aware Redirect URL (staging support)

Google OAuth requires the `redirect_uri` used in both the auth request and the token exchange to exactly match a URI pre-registered on the OAuth client. App Engine's per-SHA staging versions (`staging-<sha>-dot-...`) get a new hostname on every deploy, so they can never complete a login round-trip: Google would reject an unregistered redirect URI.

`deploy-staging.yml` also deploys the same build to a second, fixed-name `staging` version whose hostname never changes, so a human reviewer can log in on staging before approving a production promotion (see [deployment.md](deployment.md#automated-staging-deploys-githubworkflowsdeploy-stagingyml)). `internal/auth/auth.go` picks the redirect URL per-request by matching the incoming request's `Host` header against an allowlist built from `GOOGLE_REDIRECT_URL`, the production URL, and `STAGING_REDIRECT_URL`, the staging URL; both are added as registered redirect URIs on the *same* Google OAuth client (no second client needed).

A `Host` header that doesn't match either one (e.g. a per-SHA staging URL, or an attacker-supplied value) falls back to the production redirect URL, because the allowlist only ever contains these two configured URLs. It can never select an arbitrary URL, so there's no host-header-spoofing risk.

**Caveat**: staging shares the production Datastore and live Stripe keys. Do not exercise Stripe subscription flows on staging, since webhooks point at the production domain only.

**Local dev**: `STAGING_REDIRECT_URL` is unset locally, so the allowlist only contains whatever host `GOOGLE_REDIRECT_URL` resolves to (typically `localhost:8080`).

See [api.md](api.md#authentication-endpoints) for the auth endpoints and example requests, and [testing.md](testing.md) for the test helpers that set up an authenticated request, including the local cookie name tests use.

### Middleware Usage

```go
// Optional authentication (user context if logged in)
api.Use(authMiddleware.OptionalAuth())

// Required authentication (returns 401 if not logged in)
api.Use(authMiddleware.RequireAuth())

// CSRF protection for state-changing operations
api.Use(authMiddleware.CSRFMiddleware(csrfManager))
```

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md#authentication-issues) for OAuth errors, session problems, and login issues.

## Related Documentation

- [Security Guidelines](security.md) - Security implementation and best practices
- [API Reference](api.md) - API endpoints and usage
- [Setup Guide](setup.md) - Initial setup and configuration
- [Troubleshooting Guide](troubleshooting.md) - Common authentication issues
