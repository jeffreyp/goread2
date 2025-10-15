# Authentication

This document describes the authentication architecture and implementation details for GoRead2.

## Table of Contents

- [Overview](#overview)
- [Authentication Flow](#authentication-flow)
- [Session Management](#session-management)
- [Environment Isolation](#environment-isolation)
- [Security Considerations](#security-considerations)
- [Implementation Details](#implementation-details)

## Overview

GoRead2 uses Google OAuth 2.0 for user authentication, providing a secure, password-less login experience. The system implements session-based authentication with HTTP-only cookies and CSRF protection.

### Key Features

- **Google OAuth 2.0** - Leverages Google's authentication infrastructure
- **Session-based authentication** - Secure session management with database-backed storage
- **Environment isolation** - Separate authentication states for local and production deployments
- **CSRF protection** - Token-based protection for state-changing operations
- **Automatic session cleanup** - Expired sessions are automatically removed

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

## Session Management

### Session Creation

Sessions are created after successful OAuth authentication:

```go
session, err := sessionManager.CreateSession(user)
// Session includes:
// - Unique session ID (base64-encoded 32 random bytes)
// - User ID
// - Creation timestamp
// - Expiration timestamp (7 days from creation)
```

### Session Storage

Sessions are stored in the database with the following schema:

```go
type Session struct {
    ID        string    // Session identifier (used in cookie)
    UserID    int       // Associated user
    CreatedAt time.Time // When session was created
    ExpiresAt time.Time // When session expires
}
```

### Session Validation

On each authenticated request:

1. Extract session ID from cookie
2. Look up session in database
3. Check if session has expired
4. Load associated user
5. Inject user into request context

### Session Cleanup

Expired sessions are automatically cleaned up every hour by a background goroutine.

## Environment Isolation

GoRead2 implements environment-specific cookie names to prevent authentication state conflicts between local development and production deployments.

### Cookie Names by Environment

| Environment | Session Cookie | OAuth State Cookie |
|-------------|----------------|-------------------|
| **Local** (development) | `session_id_local` | `oauth_state_local` |
| **Production** (GAE) | `session_id` | `oauth_state` |

### Environment Detection

The environment is determined by checking environment variables:

```go
isProduction := os.Getenv("GAE_ENV") == "standard" ||
                os.Getenv("ENVIRONMENT") == "production"
```

### Benefits

- **No cross-environment interference** - Local and production sessions are completely isolated
- **Simultaneous testing** - Developers can be logged into both environments at once
- **Reduced confusion** - No unexpected logouts when switching between environments

### Implementation

The `SessionManager` uses the `getCookieName()` helper method to return the appropriate cookie name:

```go
func (sm *SessionManager) getCookieName() string {
    isProduction := os.Getenv("GAE_ENV") == "standard" ||
                    os.Getenv("ENVIRONMENT") == "production"
    if isProduction {
        return "session_id"
    }
    return "session_id_local"
}
```

Similarly, the auth handler uses `getOAuthStateCookieName()` for OAuth state cookies.

## Security Considerations

### Cookie Attributes

Session cookies are configured with security-conscious attributes:

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

- **HTTP-Only cookies** - Cannot be accessed via JavaScript, preventing XSS attacks
- **Secure flag in production** - Cookies only transmitted over HTTPS in production
- **SameSite Lax** - Protects against CSRF attacks by default
- **Session expiration** - Sessions automatically expire after 7 days
- **Automatic cleanup** - Expired sessions removed from database

### CSRF Protection

State-changing operations (POST, PUT, DELETE, PATCH) require CSRF tokens:

1. CSRF token generated when user loads authenticated page
2. Token included in request headers: `X-CSRF-Token`
3. Middleware validates token before processing request

See [SECURITY.md](SECURITY.md) for more details on security implementation.

## Implementation Details

### File Structure

- `internal/auth/session.go` - Session manager implementation
- `internal/auth/middleware.go` - Authentication middleware
- `internal/handlers/auth_handler.go` - OAuth handlers
- `internal/auth/csrf.go` - CSRF token management

### Configuration

Required environment variables:

```bash
# Google OAuth (required)
GOOGLE_CLIENT_ID="your-client-id"
GOOGLE_CLIENT_SECRET="your-client-secret"
GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback"

# Optional: Force production mode
ENVIRONMENT="production"
```

### Testing

Authentication tests use environment-specific cookie names automatically:

```go
// Tests automatically use local cookie names
req.AddCookie(&http.Cookie{
    Name:  "session_id_local",  // Local mode in tests
    Value: session.ID,
})
```

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/auth/login` | GET | Initiate OAuth flow |
| `/auth/callback` | GET | OAuth callback handler |
| `/auth/logout` | POST | End session |
| `/auth/me` | GET | Get current user info |

### Middleware Usage

```go
// Optional authentication (user context if logged in)
api.Use(authMiddleware.OptionalAuth())

// Required authentication (returns 401 if not logged in)
api.Use(authMiddleware.RequireAuth())

// CSRF protection for state-changing operations
api.Use(authMiddleware.CSRFMiddleware(csrfManager))
```

## Related Documentation

- [SECURITY.md](SECURITY.md) - Security implementation and best practices
- [API.md](API.md) - API endpoints and usage
- [SETUP.md](SETUP.md) - Initial setup and configuration
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common authentication issues

## Troubleshooting

### "Invalid state parameter" error

This usually indicates:
- Cookie was deleted between login initiation and callback
- Request came from different browser/incognito window
- OAuth state cookie expired (10 minute timeout)

**Solution**: Try logging in again.

### Logged out when switching environments

If using the same browser for both local and production:
- Ensure you're not using an old version (pre-environment isolation)
- Clear cookies and log in again
- Verify environment variables are set correctly

### Session expires too quickly

Sessions last 7 days by default. If experiencing premature expiration:
- Check server time is correct
- Verify database is persisting sessions
- Check browser cookie settings

### Can't stay logged in

Common causes:
- Browser blocking cookies
- Incognito/private browsing mode
- Cookie settings too restrictive

**Solution**: Allow cookies for the application domain.
