# Security Guidelines

This document outlines the security measures implemented in GoRead2 and how to use them safely.

## Table of Contents

- [Admin Access Security](#admin-access-security)
- [Deployment Security Recommendations](#deployment-security-recommendations)
- [Security Features](#security-features)
- [Reporting Security Issues](#reporting-security-issues)
- [Security Best Practices for Self-Hosters](#security-best-practices-for-self-hosters)
- [Version History](#version-history)

## Admin Access Security

### CLI Tool Security (CRITICAL)

The admin CLI tool uses database-based token authentication to prevent privilege escalation attacks. See [ADMIN.md](ADMIN.md) for complete details on the admin token system.

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

- `GET /admin/users` - List all users (requires admin auth)
- `GET /admin/users/:email` - Get user information
- `POST /admin/users/:email/admin` - Set admin status
- `POST /admin/users/:email/free-months` - Grant free months

**Example API usage:**
```bash
# Set admin status via API (requires admin authentication)
curl -X POST "https://yourdomain.com/admin/users/user@example.com/admin" \
  -H "Content-Type: application/json" \
  -d '{"is_admin": true}' \
  -b "session_cookie=your_session"
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
- **Automatic cleanup** - Expired sessions are cleaned up hourly

### CSRF Protection

All state-changing API operations (POST, PUT, DELETE) require a valid CSRF token:

- **Token generation** - Cryptographically secure random tokens (32 bytes)
- **Token validation** - Constant-time comparison prevents timing attacks
- **Token expiration** - Tokens expire after 24 hours
- **Automatic cleanup** - Expired tokens are removed hourly

**✅ The official JavaScript files (`app.js`, `account.js`, `modals.js`) include built-in CSRF protection.**

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
- **IP-based tracking** - Each client IP has independent limits
- **Automatic cleanup** - Old IP entries cleaned up hourly

### Input Validation

- **XSS protection** - All user inputs are properly escaped
- **URL validation** - Feed URLs are validated before processing
- **Request size limits** - File uploads limited to 10MB
- **SQL injection prevention** - All database queries use parameterized statements

### Endpoint Protection

- **Debug endpoints** - Restricted to admin users only (moved to `/api/debug/*`)
- **Cron endpoints** - Protected by App Engine cron header validation (production) or admin auth (local)
- **Admin endpoints** - Require authenticated admin session

### Audit & Monitoring

- **Admin action logging** - All privilege changes are logged
- **Failed authentication tracking** - Monitor for unauthorized access attempts
- **Error handling** - Secure error messages that don't leak sensitive information
- **Rate limit violations** - Logged for monitoring potential attacks

## Reporting Security Issues

If you discover a security vulnerability, please:

1. **Do NOT create a public GitHub issue**
2. **Email security concerns** to the maintainer privately
3. **Provide detailed information** about the vulnerability
4. **Allow reasonable time** for fixes before public disclosure

## Security Best Practices for Self-Hosters

1. **Keep the application updated** with security patches
2. **Use strong, unique tokens** for admin access
3. **Limit admin privileges** to necessary users only
4. **Monitor admin actions** through application logs
5. **Use HTTPS** for all production deployments
6. **Restrict server access** to authorized personnel only

## Version History

### v2.0 - Security Hardening Release (2025-10-04)

This release addressed five critical security vulnerabilities and implemented comprehensive security improvements.

#### Vulnerabilities Fixed

1. ✅ **Insecure session cookies** - Added secure flag for production environments
2. ✅ **Missing CSRF protection** - Implemented token-based CSRF protection
3. ✅ **Weak cron endpoint authentication** - Enhanced authentication for scheduled tasks
4. ✅ **Debug endpoints exposed to all users** - Restricted to admin-only access
5. ✅ **Missing rate limiting** - Added IP-based rate limiting

#### 1. Secure Session Cookies

**Files Modified:**
- `internal/auth/session.go` (lines 8, 101-113, 117-129)

**Changes:**
- Session cookies now use `Secure` flag in production environments
- Automatically enabled when `GAE_ENV=standard` or `ENVIRONMENT=production`
- Prevents session hijacking over unencrypted HTTP connections

**Testing:** All session tests pass ✓

#### 2. CSRF Protection

**Files Created:**
- `internal/auth/csrf.go` - Complete CSRF token management system
- `internal/auth/csrf_test.go` - Comprehensive test suite

**Files Modified:**
- `internal/handlers/auth_handler.go` - CSRF token generation in `/auth/me`
- `main.go` - Applied CSRF middleware to API and admin routes
- `test/helpers/http.go` - Test helper updated to include CSRF tokens

**Features:**
- Cryptographically secure 32-byte random tokens
- Constant-time comparison prevents timing attacks
- 24-hour token expiration
- Automatic cleanup of expired tokens
- Tokens tied to session ID

**Testing:** 8 new CSRF tests, all passing ✓

#### 3. Enhanced Cron Endpoint Authentication

**Files Modified:**
- `internal/handlers/feed_handler.go` (lines 8, 224-242)
- `main.go` (lines 189-197)

**Changes:**
- **Production (App Engine):** Validates `X-Appengine-Cron` header
- **Local/Other:** Requires authenticated admin user
- Prevents unauthorized feed refresh operations

**Testing:** Verified in integration tests ✓

#### 4. Admin-Only Debug Endpoints

**Files Modified:**
- `main.go` (lines 225-232)

**Changes:**
- Moved debug endpoints from `/api/feeds/:id/debug` to `/api/debug/feeds/:id`
- Moved debug endpoints from `/api/debug/article` (admin-only)
- Moved debug endpoints from `/api/debug/subscriptions` (admin-only)
- Now requires authenticated admin session
- Prevents information disclosure to regular users

**Testing:** Verified endpoint protection ✓

#### 5. Rate Limiting

**Files Created:**
- `internal/auth/rate_limiter.go` - IP-based rate limiter
- `internal/auth/rate_limiter_test.go` - Comprehensive test suite

**Files Modified:**
- `main.go` (lines 51-54, 187, 208)

**Features:**
- **Auth endpoints** (`/auth/*`): 10 requests/second, burst of 20
- **API endpoints** (`/api/*`): 30 requests/second, burst of 50
- IP-based tracking with independent limits per client
- Automatic cleanup of old IP entries
- Returns `429 Too Many Requests` when limit exceeded

**Testing:** 6 new rate limiter tests, all passing ✓

#### Additional Security Improvements

**6. Removed Sensitive Logging**
- `internal/handlers/payment_handler.go:80-83` - Removed Stripe secret prefix logging

**7. Reduced Error Verbosity**
- `internal/handlers/auth_handler.go:60-64` - Removed internal error details from responses

#### Documentation Updates

**Backend Documentation:**
1. **docs/SECURITY.md** - Added comprehensive security feature documentation (this file)
2. **docs/ADMIN.md** - Added admin token security system details
3. **docs/API.md** - Updated with CSRF requirements and rate limiting info
4. **test/helpers/http.go** - Updated test helpers to support CSRF

**Frontend Implementation:**
1. **web/static/js/app.js** - Added CSRF token storage and usage for main app
2. **web/static/js/account.js** - Added CSRF token support for account page
3. **web/static/js/modals.js** - Added CSRF token for OPML import
4. **web/static/js/*.min.js** - Rebuilt minified versions with CSRF support

#### Testing Summary

**New Tests Added:**
- 8 CSRF token tests (`internal/auth/csrf_test.go`)
- 6 Rate limiter tests (`internal/auth/rate_limiter_test.go`)

**Test Results:**
- ✅ All auth package tests passing (20 tests)
- ✅ All handler tests passing
- ✅ All service tests passing
- ✅ Core API integration tests passing (5 test suites)
- ✅ Build successful
- ✅ Linter clean (0 issues)

#### Breaking Changes

**Client-Side Changes Required:**

All POST, PUT, and DELETE requests now require a CSRF token.

**✅ The official JavaScript files (`app.js`, `account.js`, `modals.js`) have been updated with CSRF support.**

If you have custom client code, update it to include CSRF tokens:

```javascript
// First, get the CSRF token from /auth/me
const authResponse = await fetch('/auth/me');
const authData = await authResponse.json();

// Then include it in subsequent requests
fetch('/api/feeds', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': authData.csrf_token
    },
    body: JSON.stringify({url: feedUrl})
});
```

**API Endpoint Changes:**

Debug endpoints moved (admin-only):
- `/api/feeds/:id/debug` → `/api/debug/feeds/:id`
- `/api/debug/article` (new location)
- `/api/debug/subscriptions` (new location)

#### Deployment Notes

**Environment Variables:**

No new environment variables required. The following are automatically detected:

- `GAE_ENV=standard` - Enables secure cookies in App Engine
- `ENVIRONMENT=production` - Enables secure cookies in other environments

**Backward Compatibility:**

- ✅ All existing authenticated endpoints continue to work
- ✅ No database schema changes required
- ✅ Official client-side code updated with CSRF token support
- ⚠️ Custom client implementations must be updated to include CSRF tokens

#### Security Checklist for Deployments

- [x] Update client-side JavaScript to include CSRF tokens (✅ Complete in v2.0)
- [ ] Verify rate limiting is working (check for 429 responses under load)
- [ ] Confirm secure cookies are enabled in production
- [ ] Test CSRF protection on all state-changing operations
- [ ] Verify debug endpoints are admin-only
- [ ] Monitor logs for rate limit violations

### v1.x - Initial Security Implementation

- **Admin CLI protection** - Environment-based admin token requirements
- **Web-based admin API** - Proper authentication for admin operations
- **Initial admin user configuration** - Environment variable setup
- **Audit logging** - Improved logging for admin actions

---

**Last Updated:** 2025-10-04
**Status:** Production ready ✓

**Remember:** Security is a shared responsibility. Follow these guidelines and keep your deployment secure.
