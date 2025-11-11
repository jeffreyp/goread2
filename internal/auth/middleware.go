package auth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/database"
)

type contextKey string

const UserContextKey contextKey = "user"

type Middleware struct {
	sessionManager *SessionManager
}

func NewMiddleware(sessionManager *SessionManager) *Middleware {
	return &Middleware{
		sessionManager: sessionManager,
	}
}

// RequireAuth is a middleware that requires authentication
func (m *Middleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := m.sessionManager.GetSessionFromRequest(c.Request)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// Add user to context
		c.Set(string(UserContextKey), session.User)
		c.Next()
	}
}

// RequireAuthPage is a middleware that requires authentication for HTML pages
// Redirects to login instead of returning JSON error
func (m *Middleware) RequireAuthPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := m.sessionManager.GetSessionFromRequest(c.Request)
		if !exists {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		// Add user to context
		c.Set(string(UserContextKey), session.User)
		c.Next()
	}
}

// OptionalAuth is a middleware that adds user to context if authenticated
func (m *Middleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := m.sessionManager.GetSessionFromRequest(c.Request)
		if exists {
			c.Set(string(UserContextKey), session.User)
		}
		c.Next()
	}
}

// RequireAdmin is a middleware that requires admin privileges
func (m *Middleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := m.sessionManager.GetSessionFromRequest(c.Request)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		if !session.User.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
			c.Abort()
			return
		}

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
