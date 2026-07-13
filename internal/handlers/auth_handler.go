package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/secrets"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication setup failed. Please try signing in again."})
		return
	}

	// Mobile clients (client=ios) get the goread2:// handoff on callback
	// instead of the redirect to /.
	mobile := c.Query("client") == "ios"

	// Store state in session manager for one-time use validation
	ah.sessionManager.StoreOAuthState(state, mobile)

	// Store state in cookie for validation (backward compatibility)
	// Use environment-specific cookie name to avoid conflicts
	c.SetCookie(getOAuthStateCookieName(), state, 600, "/", "", false, true) // 10 minutes

	authURL := ah.authService.GetAuthURL(state, c.Request.Host)
	if mobile {
		// The mobile flow opens this endpoint as a top-level navigation
		// inside ASWebAuthenticationSession, so the state cookie above must
		// land in that browser context; redirect straight to Google rather
		// than returning JSON the web frontend would navigate to itself.
		c.Redirect(http.StatusFound, authURL)
		return
	}
	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

func (ah *AuthHandler) Callback(c *gin.Context) {
	// Verify state parameter from cookie
	storedState, err := c.Cookie(getOAuthStateCookieName())
	queryState := c.Query("state")

	if err != nil || storedState != queryState {
		log.Printf("SECURITY: OAuth state mismatch from IP %s (cookie=%v query=%v)", auth.GetSecureClientIP(c), storedState != "", queryState != "")
		c.JSON(http.StatusBadRequest, gin.H{"error": "The OAuth state parameter is not valid."})
		return
	}

	// Validate and consume state (one-time use check)
	valid, mobile := ah.sessionManager.ValidateAndConsumeOAuthState(queryState)
	if !valid {
		log.Printf("SECURITY: OAuth state expired or replayed from IP %s", auth.GetSecureClientIP(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "The OAuth state parameter has expired or has already been used. Please try signing in again."})
		return
	}

	// Clear the state cookie
	c.SetCookie(getOAuthStateCookieName(), "", -1, "/", "", false, true)

	code := c.Query("code")
	if code == "" {
		ah.callbackError(c, mobile, http.StatusBadRequest, "The authorization code is missing from the OAuth callback.")
		return
	}

	// Handle the OAuth callback
	user, err := ah.authService.HandleCallback(code, c.Request.Host)
	if err != nil {
		log.Printf("OAuth callback error: %v", err)
		ah.callbackError(c, mobile, http.StatusInternalServerError, "Authentication failed. Please try signing in again.")
		return
	}

	// Invalidate any pre-existing session to prevent session fixation attacks
	if oldSession, exists := ah.sessionManager.GetSessionFromRequest(c.Request); exists {
		ah.sessionManager.DeleteSession(oldSession.ID)
		ah.csrfManager.DeleteToken(oldSession.ID)
	}

	// Create session
	session, err := ah.sessionManager.CreateSession(user)
	if err != nil {
		ah.callbackError(c, mobile, http.StatusInternalServerError, "Session creation failed. Please try signing in again.")
		return
	}

	if mobile {
		// Hand the session off via a one-time code so the session token never
		// appears in the goread2:// URL. No session cookie is set here: the
		// ASWebAuthenticationSession browser context is discarded after the
		// redirect, and the app claims the session via POST /auth/token.
		authCode, err := ah.sessionManager.CreateAuthCode(session)
		if err != nil {
			ah.callbackError(c, mobile, http.StatusInternalServerError, "Session creation failed. Please try signing in again.")
			return
		}
		c.Redirect(http.StatusFound, mobileCallbackURL+"?code="+url.QueryEscape(authCode))
		return
	}

	// Set session cookie
	ah.sessionManager.SetSessionCookie(c.Writer, session)

	// Redirect to app
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

// mobileCallbackURL is the custom URL scheme the iOS app registers; redirecting
// to it completes the app's ASWebAuthenticationSession.
const mobileCallbackURL = "goread2://auth"

// callbackError reports a callback failure appropriately per client: mobile
// flows get a redirect to the app's URL scheme (so the auth sheet dismisses
// and the app can show the error), web flows get the JSON error as before.
func (ah *AuthHandler) callbackError(c *gin.Context, mobile bool, status int, message string) {
	if mobile {
		c.Redirect(http.StatusFound, mobileCallbackURL+"?error="+url.QueryEscape(message))
		return
	}
	c.JSON(status, gin.H{"error": message})
}

// Token exchanges a one-time code minted by the mobile OAuth callback for the
// session token. The app stores the token as its session cookie; subsequent
// API calls then authenticate exactly like the web frontend's.
func (ah *AuthHandler) Token(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A one-time authorization code is required."})
		return
	}

	sessionID, expiresAt, ok := ah.sessionManager.ExchangeAuthCode(req.Code)
	if !ok {
		log.Printf("SECURITY: invalid or expired auth code exchange attempt from IP %s", auth.GetSecureClientIP(c))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "The authorization code is invalid or has expired. Please try signing in again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_token": sessionID,
		"cookie_name":   ah.sessionManager.SessionCookieName(),
		"expires_at":    expiresAt,
	})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
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
				c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to perform this action."})
				return
			}
		} else {
			// In non-App Engine environments, require admin session + ADMIN_TOKEN header.
			user, exists := auth.GetUserFromContext(c)
			if !exists || !user.IsAdmin {
				log.Printf("Unauthorized cron request - requires admin authentication")
				c.JSON(http.StatusForbidden, gin.H{"error": "Admin access is required to perform this action."})
				return
			}
			expectedToken, err := secrets.GetAdminToken(context.Background())
			if err != nil {
				log.Printf("Warning: failed to load ADMIN_TOKEN from Secret Manager: %v", err)
			}
			if expectedToken == "" || c.GetHeader("X-Admin-Token") != expectedToken {
				log.Printf("Unauthorized cron request - invalid or missing X-Admin-Token from IP: %s", auth.GetSecureClientIP(c))
				c.JSON(http.StatusForbidden, gin.H{"error": "A valid X-Admin-Token header is required for admin access."})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean up sessions. Please try again."})
		return
	}

	// Also cleanup in-memory cache
	ah.sessionManager.CleanupExpiredCache()
	ah.sessionManager.CleanupExpiredOAuthStates()
	ah.sessionManager.CleanupExpiredAuthCodes()

	log.Printf("Session cleanup completed successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Session cleanup completed"})
}
