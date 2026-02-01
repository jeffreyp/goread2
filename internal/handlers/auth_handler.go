package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
)

type AuthHandler struct {
	authService    *auth.AuthService
	sessionManager *auth.SessionManager
	csrfManager    *auth.CSRFManager
}

func NewAuthHandler(authService *auth.AuthService, sessionManager *auth.SessionManager, csrfManager *auth.CSRFManager) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		sessionManager: sessionManager,
		csrfManager:    csrfManager,
	}
}

func (ah *AuthHandler) Login(c *gin.Context) {
	// Generate state parameter for CSRF protection
	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
		return
	}

	// Store state in session manager for one-time use validation
	ah.sessionManager.StoreOAuthState(state)

	// Store state in cookie for validation (backward compatibility)
	// Use environment-specific cookie name to avoid conflicts
	c.SetCookie(getOAuthStateCookieName(), state, 600, "/", "", false, true) // 10 minutes

	authURL := ah.authService.GetAuthURL(state)
	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

func (ah *AuthHandler) Callback(c *gin.Context) {
	// Verify state parameter from cookie
	storedState, err := c.Cookie(getOAuthStateCookieName())
	queryState := c.Query("state")

	if err != nil || storedState != queryState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	// Validate and consume state (one-time use check)
	if !ah.sessionManager.ValidateAndConsumeOAuthState(queryState) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "State parameter has expired or already been used"})
		return
	}

	// Clear the state cookie
	c.SetCookie(getOAuthStateCookieName(), "", -1, "/", "", false, true)

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	// Handle the OAuth callback
	user, err := ah.authService.HandleCallback(code)
	if err != nil {
		log.Printf("OAuth callback error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate"})
		return
	}

	// Create session
	session, err := ah.sessionManager.CreateSession(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Set session cookie
	ah.sessionManager.SetSessionCookie(c.Writer, session)

	// Redirect to app
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (ah *AuthHandler) Logout(c *gin.Context) {
	// Get session from request
	session, exists := ah.sessionManager.GetSessionFromRequest(c.Request)
	if exists {
		ah.sessionManager.DeleteSession(session.ID)
		// Delete CSRF token
		ah.csrfManager.DeleteToken(session.ID)
	}

	// Clear session cookie
	ah.sessionManager.ClearSessionCookie(c.Writer)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (ah *AuthHandler) Me(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Generate CSRF token for the session
	session, sessionExists := ah.sessionManager.GetSessionFromRequest(c.Request)
	var csrfToken string
	if sessionExists {
		token, err := ah.csrfManager.GenerateToken(session.ID)
		if err == nil {
			csrfToken = token
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":                       user.ID,
			"email":                    user.Email,
			"name":                     user.Name,
			"avatar":                   user.Avatar,
			"created_at":               user.CreatedAt,
			"max_articles_on_feed_add": user.MaxArticlesOnFeedAdd,
		},
		"csrf_token": csrfToken,
	})
}

func generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// getOAuthStateCookieName returns an environment-specific cookie name for OAuth state
// to prevent local and production authentication flows from conflicting
func getOAuthStateCookieName() string {
	isProduction := os.Getenv("GAE_ENV") == "standard" || os.Getenv("ENVIRONMENT") == "production"
	if isProduction {
		return "oauth_state"
	}
	return "oauth_state_local"
}

// CleanupExpiredSessions is a cron endpoint that removes expired sessions
func (ah *AuthHandler) CleanupExpiredSessions(c *gin.Context) {
	// If this is the cron endpoint, verify it's authorized
	if c.Request.URL.Path == "/cron/cleanup-sessions" {
		// In App Engine, verify the X-Appengine-Cron header
		if os.Getenv("GAE_ENV") == "standard" {
			cronHeader := c.GetHeader("X-Appengine-Cron")
			if cronHeader != "true" {
				log.Printf("Unauthorized cron request from IP: %s", auth.GetSecureClientIP(c))
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				return
			}
		} else {
			// In non-App Engine environments, require authentication with admin privileges
			user, exists := auth.GetUserFromContext(c)
			if !exists || !user.IsAdmin {
				log.Printf("Unauthorized cron request - requires admin authentication")
				c.JSON(http.StatusForbidden, gin.H{"error": "Admin authentication required"})
				return
			}
		}
		log.Printf("Cron session cleanup started")
	} else {
		log.Printf("Manual session cleanup started")
	}

	// Cleanup expired sessions from database
	if err := ah.sessionManager.CleanupExpiredSessions(); err != nil {
		log.Printf("Session cleanup failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Also cleanup in-memory cache
	ah.sessionManager.CleanupExpiredCache()
	ah.sessionManager.CleanupExpiredOAuthStates()

	log.Printf("Session cleanup completed successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Session cleanup completed"})
}
