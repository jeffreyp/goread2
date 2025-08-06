package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"goread2/internal/auth"
)

type AuthHandler struct {
	authService    *auth.AuthService
	sessionManager *auth.SessionManager
}

func NewAuthHandler(authService *auth.AuthService, sessionManager *auth.SessionManager) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		sessionManager: sessionManager,
	}
}

func (ah *AuthHandler) Login(c *gin.Context) {
	// Generate state parameter for CSRF protection
	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
		return
	}

	// Store state in session/cookie for validation
	c.SetCookie("oauth_state", state, 600, "/", "", false, true) // 10 minutes

	authURL := ah.authService.GetAuthURL(state)
	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

func (ah *AuthHandler) Callback(c *gin.Context) {
	// Verify state parameter
	storedState, err := c.Cookie("oauth_state")
	if err != nil || storedState != c.Query("state") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	// Clear the state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	// Handle the OAuth callback
	user, err := ah.authService.HandleCallback(code)
	if err != nil {
		log.Printf("OAuth callback error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate", "details": err.Error()})
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

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":     user.ID,
			"email":  user.Email,
			"name":   user.Name,
			"avatar": user.Avatar,
		},
	})
}

func generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
