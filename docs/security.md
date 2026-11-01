# Security Guidelines

This document outlines the security measures implemented in GoRead2 and how to use them safely.

## Table of Contents

- [Admin Access Security](#admin-access-security)
- [Deployment Security Recommendations](#deployment-security-recommendations)
- [Security Features](#security-features)
- [Reporting Security Issues](#reporting-security-issues)
- [General Security Best Practices](#general-security-best-practices)
- [Related Documentation](#related-documentation)

## Admin Access Security

### CLI Tool Security (CRITICAL)

The admin CLI tool uses database-based token authentication to prevent privilege escalation attacks. See [admin.md](admin.md) for complete details on the admin token system.

**Required for all admin commands:**
```bash
export ADMIN_TOKEN="your-64-char-token"
```

**Example usage:**
```bash
# Create first admin token (bootstrap mode)
ADMIN_TOKEN="bootstrap" go run cmd/admin/main.go create-token "Initial setup"

# Use token for admin commands
export ADMIN_TOKEN="generated_64_char_token"
go run cmd/admin/main.go list-users
go run cmd/admin/main.go set-admin user@example.com true
go run cmd/admin/main.go grant-months user@example.com 12
```

### Web-Based Admin API

Authenticated admin endpoints are available for safer admin operations:

- `GET /admin/users` - Not yet implemented; returns `501` with a note to use the CLI instead
- `GET /admin/users/:email` - Get user information
- `POST /admin/users/:email/admin` - Set admin status
- `POST /admin/users/:email/free-months` - Grant free months

**Example API usage:**
```bash
# Set admin status via API (requires admin authentication)
curl -X POST "https://yourdomain.com/admin/users/user@example.com/admin" \
  -H "Content-Type: application/json" \
  -d '{"is_admin": true}' \
  -b "session_id=your_session"
```

### Initial Admin Setup

Set initial admin users via environment variable (users must sign in first):

```bash
export INITIAL_ADMIN_EMAILS="admin1@example.com,admin2@example.com"
```

This is the **recommended way** to grant initial admin access for new deployments.

## Deployment Security Recommendations

### Environment Variables

**Required for all deployments:**
```bash
# OAuth (required)
GOOGLE_CLIENT_ID="your-google-client-id"
GOOGLE_CLIENT_SECRET="your-google-client-secret"
GOOGLE_REDIRECT_URL="https://yourdomain.com/auth/callback"

# Admin security (highly recommended)
ADMIN_TOKEN="your-64-char-token"
INITIAL_ADMIN_EMAILS="your-admin-email@example.com"

# CSRF Protection (recommended for production)
# Generate with: openssl rand -base64 32
# Ensures CSRF tokens survive application restarts
CSRF_SECRET="your-base64-encoded-32-byte-secret"

# Optional features
SUBSCRIPTION_ENABLED=false  # Set to true for paid features
CSP_REPORT_ONLY=false       # Set to true to report CSP violations without blocking them
```

### Database Security

**For production deployments:**

1. **Use remote database** instead of local SQLite when possible
2. **Restrict filesystem access** to the application directory
3. **Set proper file permissions** on SQLite database file:
   ```bash
   chmod 600 goread2.db
   chown app-user:app-user goread2.db
   ```

### Server Security

1. **Use HTTPS** in production (required for OAuth)
2. **Restrict CLI access** to authorized system administrators only
3. **Use environment variables** instead of hardcoded credentials
4. **Regular security updates** of dependencies

## Security Features

### Authentication & Authorization

- **Google OAuth 2.0** - Secure authentication without password storage
- **Session management** - HTTP-only, secure (in production) session cookies with 7-day expiration
- **CSRF protection** - Token-based CSRF protection for all state-changing API operations
- **Admin middleware** - Proper privilege checks for sensitive operations
- **User data isolation** - Complete separation of user data. Article state mutations (`POST /api/articles/:id/read`, `/star`) verify the caller is subscribed to the article's feed before writing, since article IDs are sequential and guessable; an unauthorized request gets `403 Forbidden`.
- **Rate limiting** - IP-based rate limiting to prevent brute force and DoS attacks

### Session Security

- **Secure cookies** - The session and OAuth state cookies are marked Secure in production (App Engine and ENVIRONMENT=production)
- **HTTP-only cookies** - Prevents XSS attacks from accessing session tokens
- **SameSite protection** - Both cookies use Lax mode, which withholds them from cross-site subrequests and POSTs while still sending the OAuth state cookie on Google's top-level redirect back to `/auth/callback`
- **Automatic cleanup** - Expired sessions are cleaned up every 24 hours by the `/cron/cleanup-sessions` cron job (not an in-process timer)
- **Environment isolation** - Separate cookie names for local and production environments prevent authentication conflicts

#### Environment-Specific Cookies

GoRead2 uses environment-specific cookie names to prevent authentication state conflicts between local development and production deployments:

| Environment | Session Cookie | OAuth State Cookie |
|-------------|----------------|-------------------|
| **Local** (development) | `session_id_local` | `oauth_state_local` |
| **Production** (GAE) | `session_id` | `oauth_state` |

**Benefits:**
- No cross-environment interference - developers can be logged into both environments simultaneously
- Prevents accidental logout when switching between environments
- Reduces confusion during development and testing

The environment is automatically detected via `GAE_ENV` or `ENVIRONMENT` environment variables. See [authentication.md](authentication.md) for complete implementation details.

#### Cookie and Tracking Inventory

The web frontend sets the following cookies and browser storage entries. `web/templates/privacy.html` documents the same list for users and must be updated when an entry is added or removed.

| Name | Type | Set by | Lifetime | Purpose |
|------|------|--------|----------|---------|
| `session_id` | Cookie, strictly necessary | `internal/auth/session.go` | 7 days, renewed on use | Authenticated session |
| `oauth_state` | Cookie, strictly necessary | `internal/handlers/auth_handler.go` | 10 minutes | OAuth CSRF state, cleared on callback |
| `tabletHelpDismissed` | `localStorage`, functional | `web/static/js/app.js` | Persistent | Hides the tablet help hint once dismissed |

GoRead2 uses no third-party analytics or tracking. Every page requests resources from Google Fonts (the Inter stylesheet) and cdn.jsdelivr.net (DOMPurify), which receive the visitor's IP address but set no cookies.

### CSRF Protection

All state-changing API operations (POST, PUT, DELETE) require a valid CSRF token:

- **Stateless HMAC-based tokens** - Tokens derived from session IDs using HMAC-SHA256
- **No server-side storage** - Tokens survive application restarts (when CSRF_SECRET is configured)
- **Token validation** - Constant-time comparison prevents timing attacks
- **Session-bound expiration** - Tokens remain valid as long as the session is active

**✅ `app.js`, `account.js`, and `modals.js`, the official JavaScript files, include built-in CSRF protection.**

**Client implementation example:**
```javascript
// Get CSRF token from /auth/me response
const response = await fetch('/auth/me');
const data = await response.json();
const csrfToken = data.csrf_token;

// Include in all POST/PUT/DELETE requests
fetch('/api/feeds', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken
    },
    body: JSON.stringify({url: feedUrl})
});
```

The official implementation uses a helper method:
```javascript
// Store token from /auth/me
this.csrfToken = data.csrf_token;

// Helper method for headers
getAuthHeaders(includeContentType = true) {
    const headers = {};
    if (this.csrfToken) {
        headers['X-CSRF-Token'] = this.csrfToken;
    }
    if (includeContentType) {
        headers['Content-Type'] = 'application/json';
    }
    return headers;
}

// Usage in API calls
fetch('/api/feeds', {
    method: 'POST',
    headers: this.getAuthHeaders(),
    body: JSON.stringify({url: feedUrl})
});
```

### Rate Limiting

Protection against brute force and DoS attacks:

- **Auth endpoints** (`/auth/*`) - 10 requests/second, burst of 20
- **API endpoints** (`/api/*`) - 30 requests/second, burst of 50
- **Webhook endpoint** (`/webhooks/stripe`) - 5 requests/second, burst of 10
- **Data export** (`/api/account/export`) - 1 request/minute, burst of 3, because each export reads the user's full history
- **IP-based tracking** - Each client IP has independent limits
- **Automatic cleanup** - Old IP entries cleaned up hourly

### Input Validation

- **XSS protection** - All user inputs are properly escaped. Article links from feed content are resolved against the feed's URL and restricted to the `http`/`https` schemes, both at storage time (`sanitizeArticleLink` in `internal/services/feed_service.go`) and at render time (`sanitizeUrl` in `web/static/js/app.js`), so a malicious feed can't store a `javascript:` URI or inject markup through an unescaped link attribute.
- **URL validation** - Feed URLs are validated before processing
- **Request size limits** - File uploads limited to 10MB
- **SQL injection prevention** - All database queries use parameterized statements
- **OPML bomb protection** - OPML import rejects documents nesting deeper than 50 levels or containing more than 50,000 XML elements (`internal/services/feed_service.go`), preventing XML-bomb-style denial of service

### Cross-Origin Requests

CORS is disabled by default (`internal/middleware/cors.go`). Setting `ALLOWED_ORIGIN` to an exact origin allows that single origin to make credentialed cross-origin requests (`GET, POST, PUT, DELETE, OPTIONS`, headers `Content-Type, Authorization, X-CSRF-Token`); any other origin, or no `ALLOWED_ORIGIN` at all, gets no CORS headers and falls back to the browser's same-origin policy.

### Content Security Policy

`internal/middleware/security_headers.go` sends an enforcing `Content-Security-Policy` header on every response. Setting `CSP_REPORT_ONLY=true` sends the same policy as `Content-Security-Policy-Report-Only` instead, so browsers log violations to the console without blocking them.

`script-src` allows only `'self'` and `https://cdn.jsdelivr.net` (DOMPurify). It does not allow `'unsafe-inline'`, so a script injected through feed content does not run even if it gets past sanitization. The frontend follows two rules to stay within this policy:

- Templates load only external scripts. For example, the subscription success and cancel pages use `web/static/js/subscription-redirect.js` rather than an inline redirect script.
- JavaScript that builds markup binds event handlers with `addEventListener`, never with inline `on*` attributes such as `onclick`.

`style-src` allows `https://fonts.googleapis.com` for the Inter font stylesheet, and still allows `'unsafe-inline'` because sanitized article content and templates carry inline `style` attributes.

### Request Tracing

401/403 responses from `RequireAuth`/`RequireAdmin` include a `request_id` field, sourced from `X-Cloud-Trace-Context` (set by App Engine) or `X-Request-ID`, falling back to a random value. Use it to correlate a client-reported auth failure with server logs.

### Endpoint Protection

- **Cron endpoints** - Protected by App Engine cron header validation (production) or admin auth (local)
- **Admin endpoints** - Require authenticated admin session
- **Unmatched paths** - Return an explicit 404 via `middleware.NotFoundHandler` (`internal/middleware/not_found.go`). Gin's built-in fallback writes its 404 after the gzip middleware has closed its writer, so clients that accept gzip received an empty response that the App Engine frontend reported as status 200, which vulnerability scanners interpreted as a hit.

### Local Dev Login

The dev login described in [authentication.md](authentication.md#local-dev-login) signs in as any typed Google ID, which would impersonate the owner of that ID on a shared database. Three guards keep it local:

1. **Build tag**: the handler and its routes are compiled only with `-tags devlogin`. `make build`, `make dev`, and the App Engine deploy omit the tag, so their binaries contain no dev login code. `TestDefaultBuildHasNoDevLogin` checks this for the default build.
2. **Deployed environments**: a tagged server refuses to register the routes when `GAE_ENV` is set or `ENVIRONMENT=production`, and logs why.
3. **Local database only**: it also refuses when `GOOGLE_CLOUD_PROJECT` is set without `DATASTORE_EMULATOR_HOST`. Staging shares the production Datastore, so any real Datastore is treated as production data. Without `GOOGLE_CLOUD_PROJECT` the server uses the local SQLite file.

`TestDevLoginAllowed` and `TestRegisterDevLoginRefusedOnAppEngine` cover the runtime guards; they run in `test.sh` under the `devlogin` tag. A tagged server that registers the routes logs a warning at startup.

### Account Deletion

`DELETE /api/account` permanently deletes the signed-in user's account. The request must pass CSRF validation and repeat the account email as `confirm_email`, compared case-insensitively. The handler queues the deletion before revoking the user's sessions, so a queueing failure leaves the user signed in and able to retry; it then revokes every session and clears the cookie.

`UserDataService.DeleteAccount` (`internal/services/user_data_service.go`) runs the deletion in this order, and every step is idempotent:

1. Delete the user's Stripe customers, which cancels any subscription (see [stripe.md](stripe.md#account-deletion)). If Stripe is not configured and the subscription is still live, the deletion stops here with an error.
2. Write the account tombstone
3. Delete `UserArticle`, `UserFeed`, and `Session` records in batches of 500, then the `UserActivity` record that tracks the account's last activity for retention
4. Pseudonymize audit log entries: the user's email becomes `deleted-user-<id>`, and the IP address of actions the user performed as an admin is cleared. Entries are kept for the admin trail.
5. Delete the `User` record last, so a partial failure leaves an account that a retry can find and finish

Shared `Feed` and `Article` records are not touched. Other App Engine instances can keep serving a deleted user from their in-memory session cache for up to its 10-minute TTL, so a second pass runs 15 minutes later to remove anything those requests recreated, including Stripe customers, which it finds by their `user_id` metadata (see [deployment.md](deployment.md#async-task-processing-cloud-tasks)).

**Account tombstones** stop a deleted account from starting a second free trial by signing in again. A tombstone is an `AccountTombstone` entity keyed by HMAC-SHA256 of the Google ID under `ACCOUNT_TOMBSTONE_KEY`, holding only the deletion time. A plain hash would not protect the numeric Google ID from brute force. On sign-in, a new account whose Google ID has a tombstone is created with its trial already ended. Without the key, deletion still works but writes no tombstone, and the sign-in check is skipped. The key is cached once read; a Secret Manager error other than a missing secret is not cached, so the next deletion or sign-in retries the lookup.

### Data Retention

`RetentionService` (`internal/services/retention_service.go`) applies the retention policy on the first of each month through `/cron/apply-retention`:

- **Inactive accounts**: an account is deleted, through the same path as `DELETE /api/account`, once it has had no activity for 24 months and its subscription status is `trial`, `cancelled`, or `expired`. Paying accounts (`active`, `past_due`, `unpaid`), admins, and accounts with granted free months are never deleted for inactivity. No warning is sent before deletion.
- **Audit log IP addresses**: blanked on entries older than 12 months; the rest of each entry is kept.
- **Request logs**: Cloud Logging's `_Default` bucket keeps App Engine request logs, including IP addresses and user agents, for 30 days.

Activity is a `UserActivity` record per user holding the last time any session for that account was created or used. `SessionManager` writes it at sign-in and then at most once a day per user on each instance, rather than on every request. An account with no `UserActivity` record, such as one created before activity tracking existed, is not judged by its creation date: the first retention run records it as active as of that run, so it can be deleted for inactivity no sooner than 24 months later.

A failed deletion is logged and counted; the run continues with other accounts and then returns an error so that Cloud Tasks retries it. Each retry repeats only the work still outstanding.

### Regression Test Coverage

`test/security/` is the consolidated, CI-gated regression suite for the controls above: CSRF token enforcement, auth-bypass (every `RequireAuth` route rejects a request with no session cookie), SSRF protection on `POST /api/feeds`, and free-trial feed-limit enforcement. It runs as a blocking step in the `security` job in `.github/workflows/test.yml`, separate from the advisory `govulncheck` scan. See [testing.md](testing.md#cicd-integration) for details.

### Dependency Vulnerability Alerts

Dependabot is configured in `.github/dependabot.yml` and opens pull requests for known-vulnerable or outdated dependencies on a weekly schedule, across four ecosystems:

- `gomod` (root `go.mod`) - Go module dependencies
- `npm` (root `package.json`) - frontend build tooling
- `bundler` (`ios/Gemfile`) - fastlane and its Ruby dependencies
- `github-actions` (`.github/workflows/`) - action versions pinned by commit SHA

Within each ecosystem, minor and patch updates are grouped into a single weekly pull request. Major version updates still arrive as individual pull requests so that each breaking change can be reviewed on its own.

This complements the advisory `govulncheck` scan above: `govulncheck` checks whether a vulnerability is reachable from the code, while Dependabot flags any known-vulnerable version regardless of reachability and proposes the update itself.

### Audit & Monitoring

- **Admin action logging** - All privilege changes are logged
- **Failed authentication tracking** - Monitor for unauthorized access attempts
- **Error handling** - Secure error messages that don't leak sensitive information
- **Rate limit violations** - Logged for monitoring potential attacks

## Reporting Security Issues

To report a security vulnerability:

1. **Do NOT create a public GitHub issue**
2. **Email security concerns** to the maintainer privately
3. **Provide detailed information** about the vulnerability
4. **Allow reasonable time** for fixes before public disclosure

## General Security Best Practices

1. **Keep the application updated** with security patches
2. **Use strong, unique tokens** for admin access
3. **Limit admin privileges** to necessary users only
4. **Monitor admin actions** through application logs
5. **Use HTTPS** for all production deployments (enforced automatically on App Engine)
6. **Restrict server access** to authorized personnel only

## Related Documentation

- [Admin Guide](admin.md) - Admin token system
- [Authentication](authentication.md) - Session and OAuth implementation
- [API Reference](api.md) - CSRF and rate limiting per endpoint
- [Testing Guide](testing.md) - Security regression suite
- [Troubleshooting Guide](troubleshooting.md) - Common issues
