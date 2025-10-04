# Security Guidelines

This document outlines the security measures implemented in GoRead2 and how to use them safely.

## Admin Access Security

### CLI Tool Security (CRITICAL)

The admin CLI tool now requires environment-based authentication to prevent privilege escalation attacks:

**Required for all admin commands:**
```bash
export ADMIN_TOKEN="your-secure-random-token"
```

**Required for sensitive operations (set-admin, grant-months):**
```bash
export ADMIN_TOKEN_VERIFY="your-secure-random-token"  # Must match ADMIN_TOKEN
```

**Example usage:**
```bash
# Set secure tokens (use a long, random value)
export ADMIN_TOKEN="$(openssl rand -hex 32)"
export ADMIN_TOKEN_VERIFY="$ADMIN_TOKEN"

# Now you can run admin commands safely
go run cmd/admin/main.go list-users
go run cmd/admin/main.go set-admin user@example.com true
go run cmd/admin/main.go grant-months user@example.com 12
```

### Web-Based Admin API

New authenticated admin endpoints are available for safer admin operations:

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
ADMIN_TOKEN="$(openssl rand -hex 32)"
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

**Client implementation:**
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

- **v2.x** - Enhanced security implementation
  - Added CSRF protection for all state-changing operations
  - Added rate limiting for auth and API endpoints
  - Added secure cookie flag for production environments
  - Restricted debug endpoints to admin-only access
  - Enhanced cron endpoint authentication
- **v1.x** - Initial security implementation with admin CLI protection
  - Added environment-based admin token requirements
  - Added web-based admin API with proper authentication
  - Added initial admin user configuration
  - Improved audit logging for admin actions

---

**Remember:** Security is a shared responsibility. Follow these guidelines and keep your deployment secure.