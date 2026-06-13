package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/database"
)

// requestTraceID returns a trace/request ID for correlation in logs and error responses.
// Prefers X-Cloud-Trace-Context (set by App Engine), then X-Request-ID, then a random fallback.
func requestTraceID(c *gin.Context) string {
	if trace := c.GetHeader("X-Cloud-Trace-Context"); trace != "" {
		if i := strings.Index(trace, "/"); i > 0 {
			return trace[:i]
		}
		return trace
	}
	if id := c.GetHeader("X-Request-ID"); id != "" {
		return id
	}
	b := make([]byte, 8)
	if _, err := rand.Read(b); err == nil {
		return hex.EncodeToString(b)
	}
	return "unknown"
}

type contextKey string

const (
	UserContextKey    contextKey = "user"
	sessionContextKey contextKey = "session"
)

type Middleware struct {
	sessionManager *SessionManager
}

func NewMiddleware(sessionManager *SessionManager) *Middleware {
	return &Middleware{
		sessionManager: sessionManager,
	}
}

// getOrLoadSession returns the session for this request, loading it once and caching
// in the gin context so subsequent middleware in the same request skip the store lookup.
func (m *Middleware) getOrLoadSession(c *gin.Context) (*Session, bool) {
	if cached, ok := c.Get(string(sessionContextKey)); ok {
		if s, ok := cached.(*Session); ok {
			return s, true
		}
		// nil sentinel stored by a prior call that found no session
		return nil, false
	}
	session, exists := m.sessionManager.GetSessionFromRequest(c.Request)
	if exists {
		c.Set(string(sessionContextKey), session)
	} else {
		c.Set(string(sessionContextKey), (*Session)(nil))
	}
	return session, exists
}

// RequireAuth is a middleware that requires authentication
func (m *Middleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := m.getOrLoadSession(c)
		if !exists {
			traceID := requestTraceID(c)
			log.Printf("SECURITY: unauthenticated request to %s %s from IP %s (trace=%s)",
				c.Request.Method, c.Request.URL.Path, GetSecureClientIP(c), traceID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required", "request_id": traceID})
			c.Abort()
			return
		}

		// Refresh session to extend expiry for active users
		_ = m.sessionManager.RefreshSession(session.ID)

		// Add user to context
		c.Set(string(UserContextKey), session.User)
		c.Next()
	}
}

// RequireAuthPage is a middleware that requires authentication for HTML pages
// Redirects to login instead of returning JSON error
func (m *Middleware) RequireAuthPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := m.getOrLoadSession(c)
		if !exists {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		// Refresh session to extend expiry for active users
		_ = m.sessionManager.RefreshSession(session.ID)

		// Add user to context
		c.Set(string(UserContextKey), session.User)
		c.Next()
	}
}

// OptionalAuth is a middleware that adds user to context if authenticated
func (m *Middleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := m.getOrLoadSession(c)
		if exists {
			// Refresh session to extend expiry for active users
			_ = m.sessionManager.RefreshSession(session.ID)
			c.Set(string(UserContextKey), session.User)
		}
		c.Next()
	}
}

// RequireAdmin is a middleware that requires admin privileges
func (m *Middleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := m.getOrLoadSession(c)
		if !exists {
			traceID := requestTraceID(c)
			log.Printf("SECURITY: unauthenticated admin request to %s %s from IP %s (trace=%s)",
				c.Request.Method, c.Request.URL.Path, GetSecureClientIP(c), traceID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required", "request_id": traceID})
			c.Abort()
			return
		}

		if !session.User.IsAdmin {
			traceID := requestTraceID(c)
			log.Printf("SECURITY: non-admin access attempt to %s %s by user %d from IP %s (trace=%s)",
				c.Request.Method, c.Request.URL.Path, session.User.ID, GetSecureClientIP(c), traceID)
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required", "request_id": traceID})
			c.Abort()
			return
		}

		// Refresh session to extend expiry for active users
		_ = m.sessionManager.RefreshSession(session.ID)

		// Add user to context
		c.Set(string(UserContextKey), session.User)
		c.Next()
	}
}

// GetUserFromContext extracts the user from the Gin context
func GetUserFromContext(c *gin.Context) (*database.User, bool) {
	user, exists := c.Get(string(UserContextKey))
	if !exists {
		return nil, false
	}

	userObj, ok := user.(*database.User)
	return userObj, ok
}

// GetUserFromStdContext extracts the user from a standard context
func GetUserFromStdContext(ctx context.Context) (*database.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*database.User)
	return user, ok
}
