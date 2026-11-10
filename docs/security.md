# Security

The security controls in GoRead2, where each is implemented, and how account data is deleted and retained. GoRead2 runs only on Google App Engine, which serves it over HTTPS.

## Table of Contents

- [Authentication and Sessions](#authentication-and-sessions)
- [Authorization](#authorization)
- [CSRF Protection](#csrf-protection)
- [Rate Limiting](#rate-limiting)
- [Input Validation](#input-validation)
- [Cross-Origin Requests](#cross-origin-requests)
- [Content Security Policy](#content-security-policy)
- [Cookie and Tracking Inventory](#cookie-and-tracking-inventory)
- [Account Deletion](#account-deletion)
- [Data Retention](#data-retention)
- [Logging and Audit](#logging-and-audit)
- [Regression Tests and Dependency Scanning](#regression-tests-and-dependency-scanning)
- [Reporting Security Issues](#reporting-security-issues)
- [Troubleshooting](#troubleshooting)
- [Related Documentation](#related-documentation)

## Authentication and Sessions

GoRead2 signs users in with Google OAuth 2.0 and stores no passwords. [authentication.md](authentication.md) describes the flow and session handling in full.

- **Session cookie**: `HttpOnly`, so scripts cannot read it, and `Secure` in production, which means App Engine or `ENVIRONMENT=production`. A session lasts 7 days and is renewed on use.
- **SameSite**: the session and OAuth state cookies use `Lax` mode, which withholds them from cross-site subrequests and POSTs while still sending the OAuth state cookie on Google's top-level redirect back to `/auth/callback`.
- **Cleanup**: the `/cron/cleanup-sessions` job deletes expired sessions once a day.
- **Environment isolation**: local and production servers use different cookie names (see [Cookie and Tracking Inventory](#cookie-and-tracking-inventory)), so a browser can be signed in to both.

## Authorization

- **User data isolation**: per-user records, such as subscriptions, read and star state, and sessions, are always looked up by the signed-in user's ID. Feeds and articles are shared between users. Article state changes (`POST /api/articles/:id/read` and `/star`) also check that the caller subscribes to the article's feed, since article IDs are sequential and guessable, and return `403 Forbidden` otherwise.
- **Admin endpoints**: `/admin/*` requires a session for an admin account (see [api.md](api.md#admin-endpoints)).
- **Admin CLI**: `cmd/admin` and `admin.sh` require an `ADMIN_TOKEN` that matches a token stored, as a SHA-256 hash, in the database. Tokens can be listed and revoked (see [admin.md](admin.md#admin-token-system)).
- **Initial admins**: accounts whose email is in `INITIAL_ADMIN_EMAILS` are made admins at sign-in and at startup (see [admin.md](admin.md)).
- **Cron and task endpoints**: in production, `/cron/*` requires App Engine's `X-Appengine-Cron` header and `/tasks/*` requires `X-AppEngine-QueueName`. App Engine strips both headers from external requests. Outside App Engine, both require an admin session and an `X-Admin-Token` header equal to the server's `ADMIN_TOKEN`, and reject every request when `ADMIN_TOKEN` is unset.
- **Unmatched paths**: return an explicit `404` from `middleware.NotFoundHandler` in `internal/middleware/not_found.go`, because Gin's built-in fallback loses its response body under the gzip middleware.

### Local Dev Login

The dev login described in [authentication.md](authentication.md#local-dev-login) signs in as any typed Google ID, which would impersonate the owner of that ID on a shared database. Three guards keep it local:

1. **Build tag**: the handler and its routes are compiled only with `-tags devlogin`. `make build`, `make dev`, and the App Engine deploy omit the tag, so their binaries contain no dev login code. `TestDefaultBuildHasNoDevLogin` checks this for the default build.
2. **Deployed environments**: a tagged server refuses to register the routes when `GAE_ENV` is set or `ENVIRONMENT=production`, and logs why.
3. **Local database only**: it also refuses when `GOOGLE_CLOUD_PROJECT` is set without `DATASTORE_EMULATOR_HOST`. Staging shares the production Datastore, so any real Datastore is treated as production data. Without `GOOGLE_CLOUD_PROJECT` the server uses the local SQLite file.

`TestDevLoginAllowed` and `TestRegisterDevLoginRefusedOnAppEngine` cover the runtime guards; they run in `test.sh` under the `devlogin` tag. A tagged server that registers the routes logs a warning at startup.

## CSRF Protection

Every state-changing API request (`POST`, `PUT`, `DELETE`) must carry a valid token in the `X-CSRF-Token` header. `internal/auth/csrf.go` implements the tokens:

- Each token is an HMAC-SHA256 of the session ID under `CSRF_SECRET`, so the server stores no tokens.
- Tokens are compared in constant time, and remain valid for the life of the session.
- Production requires `CSRF_SECRET`, so tokens survive restarts and are valid on every instance. A local server without it generates a secret at each start (see [configuration.md](configuration.md#security)).

Clients read the token from `GET /auth/me` and send it with each mutating request. The web frontend's `app.js`, `account.js`, and `modals.js` do this through a shared `getAuthHeaders` helper:

```javascript
const { csrf_token } = await (await fetch('/auth/me')).json();

fetch('/api/feeds', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf_token },
    body: JSON.stringify({ url: feedUrl })
});
```

## Rate Limiting

Requests are rate limited per client IP address. On App Engine the address comes from the `X-Appengine-User-Ip` header, which Google sets and clients cannot forge; elsewhere it is the address of the TCP connection. Headers such as `X-Forwarded-For` are ignored, so a client cannot evade its limit by forging one. Manual feed refresh is also limited per user, so one account cannot bypass that limit by changing address. Rejected requests get `429 Too Many Requests`. [api.md](api.md#rate-limiting) lists the limit for each group of endpoints.

## Input Validation

- **Article links**: links from feed content are resolved against the feed's URL and restricted to the `http` and `https` schemes, both at storage time (`sanitizeArticleLink` in `internal/services/feed_service.go`) and at render time (`sanitizeUrl` in `web/static/js/app.js`), so a feed cannot store a `javascript:` URI or inject markup through a link attribute.
- **Article content**: the server sanitizes feed HTML, and the web frontend sanitizes it again with DOMPurify before rendering.
- **Feed URLs**: validated before they are fetched, including protection against server-side request forgery to internal addresses.
- **Request size**: request bodies are limited to 1 MB, and OPML uploads to 10 MB (`internal/middleware/body_limit.go`). A larger request gets `413`.
- **SQL**: SQLite queries use parameterized statements.
- **OPML**: import rejects documents nesting deeper than 50 levels or containing more than 50,000 XML elements (`internal/services/feed_service.go`), which prevents XML-bomb denial of service.

## Cross-Origin Requests

CORS is disabled by default (`internal/middleware/cors.go`). Setting `ALLOWED_ORIGIN` to an exact origin allows that single origin to make credentialed cross-origin requests (`GET, POST, PUT, DELETE, OPTIONS`, headers `Content-Type, Authorization, X-CSRF-Token`); any other origin, or no `ALLOWED_ORIGIN` at all, gets no CORS headers and falls back to the browser's same-origin policy.

## Content Security Policy

`internal/middleware/security_headers.go` sends an enforcing `Content-Security-Policy` header on every response. Setting `CSP_REPORT_ONLY=true` sends the same policy as `Content-Security-Policy-Report-Only` instead, so browsers log violations to the console without blocking them.

`script-src` allows only `'self'` and `https://cdn.jsdelivr.net` (DOMPurify). It does not allow `'unsafe-inline'`, so a script injected through feed content does not run even if it gets past sanitization. The frontend follows two rules to stay within this policy:

- Templates load only external scripts. For example, the subscription success and cancel pages use `web/static/js/subscription-redirect.js` rather than an inline redirect script.
- JavaScript that builds markup binds event handlers with `addEventListener`, never with inline `on*` attributes such as `onclick`.

`style-src` allows `https://fonts.googleapis.com` for the Inter font stylesheet, and still allows `'unsafe-inline'` because sanitized article content and templates carry inline `style` attributes.

## Cookie and Tracking Inventory

The web frontend sets the following cookies and browser storage entries. `web/templates/privacy.html` documents the same list for users and must be updated when an entry is added or removed.

| Name in production | Name on a local server | Type | Set by | Lifetime | Purpose |
|--------------------|------------------------|------|--------|----------|---------|
| `session_id` | `session_id_local` | Cookie, strictly necessary | `internal/auth/session.go` | 7 days, renewed on use | Authenticated session |
| `oauth_state` | `oauth_state_local` | Cookie, strictly necessary | `internal/handlers/auth_handler.go` | 10 minutes | OAuth CSRF state, cleared on callback |
| `tabletHelpDismissed` | Same | `localStorage`, functional | `web/static/js/app.js` | Persistent | Hides the tablet help hint once dismissed |

GoRead2 uses no third-party analytics or tracking. Every page requests resources from Google Fonts (the Inter stylesheet) and cdn.jsdelivr.net (DOMPurify), which receive the visitor's IP address but set no cookies.

## Account Deletion

`DELETE /api/account` permanently deletes the signed-in user's account. The request must pass CSRF validation and repeat the account email as `confirm_email`, compared case-insensitively. The handler queues the deletion before revoking the user's sessions, so a queueing failure leaves the user signed in and able to retry; it then revokes every session and clears the cookie.

In production, once the deletion is queued, the handler also records a `deletion_requested_at` timestamp on the `User` record (`UserDataService.RequestDeletion`). This timestamp does two things: it lets the data retention run retry a deletion that both Cloud Tasks passes failed to complete (see [Data Retention](#data-retention) below), and it makes `SignInGoogleUser` refuse a re-sign-in on the Google identity while the deletion is in flight, returning `ErrAccountDeletionPending` instead of creating a session. Signing back in through the normal flow after the account is actually gone is unaffected, since deletion removes the `User` record.

`UserDataService.DeleteAccount` (`internal/services/user_data_service.go`) runs the deletion in this order, and every step is idempotent:

1. Delete the user's Stripe customers, which cancels any subscription (see [stripe.md](stripe.md#account-deletion)). If Stripe is not configured and the subscription is still live, the deletion stops here with an error.
2. Write the account tombstone
3. Delete `UserArticle`, `UserFeed`, and `Session` records in batches of 500, then the `UserActivity` record that tracks the account's last activity for retention
4. Pseudonymize audit log entries: the user's email becomes `deleted-user-<id>`, and the IP address of actions the user performed as an admin is cleared. Entries are kept for the admin trail.
5. Delete the `User` record last, so a partial failure leaves an account that a retry can find and finish

Shared `Feed` and `Article` records are not touched. Other App Engine instances can keep serving a deleted user from their in-memory session cache for up to its 10-minute TTL, so a second pass runs 15 minutes later to remove anything those requests recreated, including Stripe customers, which it finds by their `user_id` metadata (see [deployment.md](deployment.md#async-task-processing-cloud-tasks)).

**Account tombstones** stop a deleted account from starting a second free trial by signing in again. A tombstone is an `AccountTombstone` entity keyed by HMAC-SHA256 of the Google ID under `ACCOUNT_TOMBSTONE_KEY`, holding only the deletion time and an optional expiry. A self-requested deletion writes a tombstone that never expires. A deletion for inactivity writes one that expires 24 months later, so a returning user is offered a new trial after that. Writing a tombstone never shortens an existing one: a permanent tombstone stays permanent, and otherwise the later expiry is kept. An expired tombstone no longer counts at sign-in. A plain hash would not protect the numeric Google ID from brute force. On sign-in, a new account whose Google ID has an unexpired tombstone is created with its trial already ended. Without the key, deletion still works but writes no tombstone, and the sign-in check is skipped. The key is cached once read; a Secret Manager error other than a missing secret is not cached, so the next deletion or sign-in retries the lookup.

## Data Retention

`RetentionService` (`internal/services/retention_service.go`) applies the following policy on the first of each month through `/cron/apply-retention`, and applies the pending-deletion rule alone every day through `/cron/retry-pending-deletions`:

- **Inactive accounts**: an account is deleted, through the same path as `DELETE /api/account`, once it has had no activity for 24 months and its subscription status is `trial`, `cancelled`, or `expired`. Paying accounts (`active`, `past_due`, `unpaid`), admins, and accounts with granted free months are never deleted for inactivity. No warning is sent before deletion. The account's tombstone expires 24 months after the deletion (see [Account Deletion](#account-deletion)).
- **Pending self-requested deletions**: any account with a `deletion_requested_at` timestamp older than one hour is deleted, regardless of subscription status. This is the backstop for a `DELETE /api/account` deletion whose Cloud Tasks passes both failed (see [Account Deletion Failures](deployment.md#account-deletion-failures)). Because it runs daily, a paying subscriber whose deletion failed during an outage remains subscribed for about a day at most.
- **Account tombstones**: deleted once expired. Tombstones from self-requested deletions never expire.
- **Audit log IP addresses**: blanked on entries older than 12 months; the rest of each entry is kept. Each run reads only entries that still have an IP address, so an entry that a failed run missed is cleared by the next successful one.
- **Request logs**: Cloud Logging's `_Default` bucket keeps App Engine request logs, including IP addresses and user agents, for 30 days.

Activity is a `UserActivity` record per user holding the last time any session for that account was created or used. `SessionManager` writes it at sign-in and then at most once a day per user on each instance, rather than on every request. An account with no `UserActivity` record is not judged by its creation date: the first retention run records it as active as of that run, so it can be deleted for inactivity no sooner than 24 months later.

A failed deletion is logged and counted; the run continues with other accounts and then returns an error so that Cloud Tasks retries it. Each retry repeats only the work still outstanding.

## Logging and Audit

- **Audit log**: granting or revoking admin access and granting free months, through the admin API or CLI, write an audit log entry with the acting admin, the target user, the IP address, and the result. Viewing a user's details through the admin API is also recorded. [admin.md](admin.md#audit-logging) describes how to read it.
- **Security log lines**: the server logs a line starting `SECURITY:` for each unauthenticated request to a protected route, each non-admin request to an admin route, and each request rejected by a rate limit, with the client IP and request path.
- **Request tracing**: `401` and `403` responses from `RequireAuth` and `RequireAdmin` include a `request_id` field, taken from `X-Cloud-Trace-Context`, which App Engine sets, or `X-Request-ID`, or else generated. It correlates a client-reported failure with the server logs.
- **Error responses**: API errors return a short message for the user, without stack traces. Admin endpoints also return the underlying error text in a `details` field.

## Regression Tests and Dependency Scanning

`test/security/` is the consolidated, CI-gated regression suite for the controls above: CSRF token enforcement, auth-bypass (every `RequireAuth` route rejects a request with no session cookie), SSRF protection on `POST /api/feeds`, and free-trial feed-limit enforcement. It runs as a blocking step in the `security` job in `.github/workflows/test.yml`, alongside an advisory `govulncheck` scan, which reports known vulnerabilities reachable from the code. See [testing.md](testing.md#cicd-integration) for details.

Dependabot, configured in `.github/dependabot.yml`, opens pull requests for known-vulnerable or outdated dependencies on a weekly schedule, across four ecosystems:

- `gomod` (root `go.mod`) - Go module dependencies
- `npm` (root `package.json`) - frontend build tooling
- `bundler` (`ios/Gemfile`) - fastlane and its Ruby dependencies
- `github-actions` (`.github/workflows/`) - action versions pinned by commit SHA

Within each ecosystem, minor and patch updates are grouped into one weekly pull request, and each major version update arrives on its own. Dependabot flags any known-vulnerable version, whether or not the code reaches the vulnerable function.

## Reporting Security Issues

Report a vulnerability privately through GitHub's private vulnerability reporting: on the repository's **Security** tab, choose **Report a vulnerability**. Do not open a public issue. Include the affected endpoint or component, the steps to reproduce, and the impact.

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md#authentication-issues) for sign-in, CSRF, and session errors.

## Related Documentation

- [Authentication](authentication.md) - Session and OAuth implementation
- [Configuration Reference](configuration.md#security) - The variables that configure these controls
- [Admin Guide](admin.md) - Admin tokens and the audit log
- [API Reference](api.md) - Rate limits and authentication per endpoint
- [Testing Guide](testing.md) - Security regression suite
- [Troubleshooting Guide](troubleshooting.md) - Common issues
