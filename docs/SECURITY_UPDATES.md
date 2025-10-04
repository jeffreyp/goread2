# Security Updates - v2.0

This document summarizes the security enhancements implemented in v2.0.

## Overview

Five critical security vulnerabilities were identified and fixed:

1. ✅ Insecure session cookies
2. ✅ Missing CSRF protection
3. ✅ Weak cron endpoint authentication
4. ✅ Debug endpoints exposed to all users
5. ✅ Missing rate limiting

## Changes Made

### 1. Secure Session Cookies

**Files Modified:**
- `internal/auth/session.go` (lines 8, 101-113, 117-129)

**Changes:**
- Session cookies now use `Secure` flag in production environments
- Automatically enabled when `GAE_ENV=standard` or `ENVIRONMENT=production`
- Prevents session hijacking over unencrypted HTTP connections

**Testing:** All session tests pass ✓

---

### 2. CSRF Protection

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

**Client Integration Required:**
```javascript
// Get CSRF token
const response = await fetch('/auth/me');
const data = await response.json();
const csrfToken = data.csrf_token;

// Include in all POST/PUT/DELETE requests
fetch('/api/feeds', {
    method: 'POST',
    headers: {
        'X-CSRF-Token': csrfToken,
        'Content-Type': 'application/json'
    },
    body: JSON.stringify({url: feedUrl})
});
```

**Testing:** 8 new CSRF tests, all passing ✓

---

### 3. Enhanced Cron Endpoint Authentication

**Files Modified:**
- `internal/handlers/feed_handler.go` (lines 8, 224-242)
- `main.go` (lines 189-197)

**Changes:**
- **Production (App Engine):** Validates `X-Appengine-Cron` header
- **Local/Other:** Requires authenticated admin user
- Prevents unauthorized feed refresh operations

**Testing:** Verified in integration tests ✓

---

### 4. Admin-Only Debug Endpoints

**Files Modified:**
- `main.go` (lines 225-232)

**Changes:**
- Moved debug endpoints from `/api/feeds/:id/debug` to `/api/debug/feeds/:id`
- Moved debug endpoints from `/api/debug/article` (admin-only)
- Moved debug endpoints from `/api/debug/subscriptions` (admin-only)
- Now requires authenticated admin session
- Prevents information disclosure to regular users

**Testing:** Verified endpoint protection ✓

---

### 5. Rate Limiting

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

---

## Additional Security Improvements

### 6. Removed Sensitive Logging
- `internal/handlers/payment_handler.go:80-83` - Removed Stripe secret prefix logging

### 7. Reduced Error Verbosity
- `internal/handlers/auth_handler.go:60-64` - Removed internal error details from responses

---

## Documentation Updates

### Backend Documentation Updates:
1. **docs/security.md** - Added comprehensive security feature documentation
2. **docs/api.md** - Updated with CSRF requirements and rate limiting info
3. **test/helpers/http.go** - Updated test helpers to support CSRF

### Frontend Implementation Updates:
1. **web/static/js/app.js** - Added CSRF token storage and usage for main app
2. **web/static/js/account.js** - Added CSRF token support for account page
3. **web/static/js/modals.js** - Added CSRF token for OPML import
4. **web/static/js/*.min.js** - Rebuilt minified versions with CSRF support

### New Documentation Files:
1. **docs/SECURITY_UPDATES.md** - This document

---

## Testing Summary

### New Tests Added:
- 8 CSRF token tests (`internal/auth/csrf_test.go`)
- 6 Rate limiter tests (`internal/auth/rate_limiter_test.go`)

### Test Results:
- ✅ All auth package tests passing (20 tests)
- ✅ All handler tests passing
- ✅ All service tests passing
- ✅ Core API integration tests passing (5 test suites)
- ✅ Build successful
- ✅ Linter clean (0 issues)

---

## Breaking Changes

### Client-Side Changes Required:

All POST, PUT, and DELETE requests now require a CSRF token.

**✅ The official JavaScript files (`app.js`, `account.js`, `modals.js`) have been updated with CSRF support.**

If you have custom client code, update it as follows:

**Before:**
```javascript
fetch('/api/feeds', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({url: feedUrl})
});
```

**After:**
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

**Implementation Details:**

The official client files implement CSRF protection as follows:

1. **Token Storage** - CSRF token stored on authentication:
   ```javascript
   // From /auth/me response
   this.csrfToken = data.csrf_token;
   ```

2. **Helper Method** - Centralized header management:
   ```javascript
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
   ```

3. **Usage** - All state-changing requests:
   ```javascript
   fetch('/api/feeds', {
       method: 'POST',
       headers: this.getAuthHeaders(),
       body: JSON.stringify({url: feedUrl})
   });
   ```

### API Endpoint Changes:

**Debug endpoints moved (admin-only):**
- `/api/feeds/:id/debug` → `/api/debug/feeds/:id`
- `/api/debug/article` (new location)
- `/api/debug/subscriptions` (new location)

---

## Deployment Notes

### Environment Variables:

No new environment variables required. The following are automatically detected:

- `GAE_ENV=standard` - Enables secure cookies in App Engine
- `ENVIRONMENT=production` - Enables secure cookies in other environments

### Backward Compatibility:

- ✅ All existing authenticated endpoints continue to work
- ✅ No database schema changes required
- ✅ Official client-side code updated with CSRF token support
- ⚠️ Custom client implementations must be updated to include CSRF tokens

---

## Security Checklist for Deployments

- [x] Update client-side JavaScript to include CSRF tokens (✅ Complete in v2.0)
- [ ] Verify rate limiting is working (check for 429 responses under load)
- [ ] Confirm secure cookies are enabled in production
- [ ] Test CSRF protection on all state-changing operations
- [ ] Verify debug endpoints are admin-only
- [ ] Monitor logs for rate limit violations

---

## Version History

**v2.0** - Security hardening release
- Added CSRF protection
- Added rate limiting
- Added secure cookie support
- Restricted debug endpoints
- Enhanced cron authentication

---

**Last Updated:** 2025-10-04
**Reviewed By:** Security audit completed
**Status:** Production ready ✓
