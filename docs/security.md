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
- **Session management** - HTTP-only cookies with CSRF protection
- **Admin middleware** - Proper privilege checks for sensitive operations
- **User data isolation** - Complete separation of user data

### Input Validation

- **XSS protection** - All user inputs are properly escaped
- **CSRF protection** - State parameter validation for OAuth
- **URL validation** - Feed URLs are validated before processing

### Audit & Monitoring

- **Admin action logging** - All privilege changes are logged
- **Failed authentication tracking** - Monitor for unauthorized access attempts
- **Error handling** - Secure error messages that don't leak sensitive information

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

- **v1.x** - Initial security implementation with admin CLI protection
- Added environment-based admin token requirements
- Added web-based admin API with proper authentication
- Added initial admin user configuration
- Improved audit logging for admin actions

---

**Remember:** Security is a shared responsibility. Follow these guidelines and keep your deployment secure.