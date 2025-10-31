package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// CSRFManager manages CSRF tokens using stateless HMAC-based generation
// Tokens are derived from session IDs using HMAC-SHA256, eliminating the need
// for server-side storage and ensuring tokens survive application restarts
type CSRFManager struct {
	secret []byte
}

// NewCSRFManager creates a new CSRF manager with HMAC-based token generation
func NewCSRFManager() *CSRFManager {
	secret := getOrGenerateSecret()

	return &CSRFManager{
		secret: secret,
	}
}

// getOrGenerateSecret retrieves the CSRF secret from environment or generates one
// In production, CSRF_SECRET should be set to ensure consistency across instances
// In development/testing, we generate a random secret per process
func getOrGenerateSecret() []byte {
	// Try to get secret from environment first
	secretStr := os.Getenv("CSRF_SECRET")
	if secretStr != "" {
		// Decode base64-encoded secret
		secret, err := base64.StdEncoding.DecodeString(secretStr)
		if err != nil {
			log.Printf("Warning: Invalid CSRF_SECRET format, generating new secret: %v", err)
		} else if len(secret) >= 32 {
			return secret
		} else {
			log.Printf("Warning: CSRF_SECRET too short (need >= 32 bytes), generating new secret")
		}
	}

	// Generate random secret (32 bytes for HMAC-SHA256)
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		log.Fatalf("Failed to generate CSRF secret: %v", err)
	}

	// In production, warn that a persistent secret should be configured
	if os.Getenv("GAE_ENV") == "standard" || os.Getenv("ENVIRONMENT") == "production" {
		log.Printf("WARNING: CSRF_SECRET not set in production. Tokens will be invalidated on restart.")
		log.Printf("Generate a CSRF_SECRET with: openssl rand -base64 32")
	}

	return secret
}

// GenerateToken generates a CSRF token for a session using HMAC-SHA256
// The token is deterministically derived from the session ID, making it stateless
func (cm *CSRFManager) GenerateToken(sessionID string) (string, error) {
	// Generate HMAC-SHA256 of session ID
	mac := hmac.New(sha256.New, cm.secret)
	mac.Write([]byte(sessionID))
	tokenBytes := mac.Sum(nil)

	// Encode as base64 for transmission
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	return token, nil
}

// ValidateToken validates a CSRF token for a session by recomputing the HMAC
// This is stateless - no database or memory lookup required
func (cm *CSRFManager) ValidateToken(sessionID, token string) bool {
	// Generate expected token for this session
	expectedToken, err := cm.GenerateToken(sessionID)
	if err != nil {
		return false
	}

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(expectedToken), []byte(token)) == 1
}

// DeleteToken is a no-op in the stateless implementation
// CSRF tokens are tied to session lifetime, so deleting the session invalidates the token
func (cm *CSRFManager) DeleteToken(sessionID string) {
	// No-op: stateless tokens don't require deletion
	// Token becomes invalid when session is deleted
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
