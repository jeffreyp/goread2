package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CSRFToken represents a CSRF token with expiration
type CSRFToken struct {
	Token     string
	ExpiresAt time.Time
}

// CSRFManager manages CSRF tokens
type CSRFManager struct {
	tokens map[string]*CSRFToken
	mu     sync.RWMutex
}

// NewCSRFManager creates a new CSRF manager
func NewCSRFManager() *CSRFManager {
	cm := &CSRFManager{
		tokens: make(map[string]*CSRFToken),
	}

	// Start cleanup goroutine
	go cm.cleanupExpiredTokens()

	return cm
}

// GenerateToken generates a new CSRF token for a session
func (cm *CSRFManager) GenerateToken(sessionID string) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(bytes)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.tokens[sessionID] = &CSRFToken{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	return token, nil
}

// ValidateToken validates a CSRF token for a session
func (cm *CSRFManager) ValidateToken(sessionID, token string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	storedToken, exists := cm.tokens[sessionID]
	if !exists {
		return false
	}

	// Check if token is expired
	if time.Now().After(storedToken.ExpiresAt) {
		return false
	}

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(storedToken.Token), []byte(token)) == 1
}

// DeleteToken removes a CSRF token for a session
func (cm *CSRFManager) DeleteToken(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.tokens, sessionID)
}

// cleanupExpiredTokens periodically removes expired tokens
func (cm *CSRFManager) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		cm.mu.Lock()
		now := time.Now()
		for sessionID, token := range cm.tokens {
			if now.After(token.ExpiresAt) {
				delete(cm.tokens, sessionID)
			}
		}
		cm.mu.Unlock()
	}
}

// CSRFMiddleware returns a Gin middleware that validates CSRF tokens
func (m *Middleware) CSRFMiddleware(csrfManager *CSRFManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF check for GET, HEAD, OPTIONS (safe methods)
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Get session
		session, exists := m.sessionManager.GetSessionFromRequest(c.Request)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// Get CSRF token from header
		csrfToken := c.GetHeader("X-CSRF-Token")
		if csrfToken == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token required"})
			c.Abort()
			return
		}

		// Validate CSRF token
		if !csrfManager.ValidateToken(session.ID, csrfToken) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid CSRF token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
