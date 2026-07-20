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
- **User data isolation** - Complete separation of user data
- **Rate limiting** - IP-based rate limiting to prevent brute force and DoS attacks

### Session Security

- **Secure cookies** - Automatically enabled in production (App Engine and ENVIRONMENT=production)
- **HTTP-only cookies** - Prevents XSS attacks from accessing session tokens
- **SameSite protection** - Lax mode prevents CSRF attacks via cross-site requests
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
- **IP-based tracking** - Each client IP has independent limits
- **Automatic cleanup** - Old IP entries cleaned up hourly

### Input Validation

- **XSS protection** - All user inputs are properly escaped
- **URL validation** - Feed URLs are validated before processing
- **Request size limits** - File uploads limited to 10MB
- **SQL injection prevention** - All database queries use parameterized statements
- **OPML bomb protection** - OPML import rejects documents nesting deeper than 50 levels or containing more than 50,000 XML elements (`internal/services/feed_service.go`), preventing XML-bomb-style denial of service

### Cross-Origin Requests

CORS is disabled by default (`internal/middleware/cors.go`). Setting `ALLOWED_ORIGIN` to an exact origin allows that single origin to make credentialed cross-origin requests (`GET, POST, PUT, DELETE, OPTIONS`, headers `Content-Type, Authorization, X-CSRF-Token`); any other origin, or no `ALLOWED_ORIGIN` at all, gets no CORS headers and falls back to the browser's same-origin policy.

### Request Tracing

401/403 responses from `RequireAuth`/`RequireAdmin` include a `request_id` field, sourced from `X-Cloud-Trace-Context` (set by App Engine) or `X-Request-ID`, falling back to a random value. Use it to correlate a client-reported auth failure with server logs.

### Endpoint Protection

- **Debug endpoints** (`/api/debug/*`) - Restricted to admin users only
- **Cron endpoints** - Protected by App Engine cron header validation (production) or admin auth (local)
- **Admin endpoints** - Require authenticated admin session
- **Unmatched paths** - Return an explicit 404 via `middleware.NotFoundHandler` (`internal/middleware/not_found.go`). Gin's built-in fallback writes its 404 after the gzip middleware has closed its writer, so clients that accept gzip received an empty response that the App Engine frontend reported as status 200, which vulnerability scanners interpreted as a hit.

### Regression Test Coverage

`test/security/` is the consolidated, CI-gated regression suite for the controls above: CSRF token enforcement, auth-bypass (every `RequireAuth` route rejects a request with no session cookie), SSRF protection on `POST /api/feeds`, and free-trial feed-limit enforcement. It runs as a blocking step in the `security` job in `.github/workflows/test.yml`, separate from the advisory `govulncheck` scan. See [testing.md](testing.md#cicd-integration) for details.

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
