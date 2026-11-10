# Security

The security controls in GoRead2 and where each is implemented. GoRead2 runs only on Google App Engine, which serves it over HTTPS.

## Table of Contents

- [Authentication and Sessions](#authentication-and-sessions)
- [Authorization](#authorization)
- [CSRF Protection](#csrf-protection)
- [Rate Limiting](#rate-limiting)
- [Input Validation](#input-validation)
- [Cross-Origin Requests](#cross-origin-requests)
- [Content Security Policy](#content-security-policy)
- [Personal Data](#personal-data)
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
- **Environment isolation**: local and production servers use different cookie names (see [privacy.md](privacy.md#cookies-and-browser-storage)), so a browser can be signed in to both.

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

## Personal Data

[privacy.md](privacy.md) describes the personal data GoRead2 stores, the cookies it sets, and how accounts are exported, deleted, and retained.

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
- [Privacy and Personal Data](privacy.md) - Stored data, cookies, export, deletion, and retention
- [Admin Guide](admin.md) - Admin tokens and the audit log
- [API Reference](api.md) - Rate limits and authentication per endpoint
- [Testing Guide](testing.md) - Security regression suite
- [Troubleshooting Guide](troubleshooting.md) - Common issues
