package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/jeffreyp/goread2/internal/config"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/secrets"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// redactEmail redacts an email address for logging, keeping only first char and domain
func redactEmail(email string) string {
	if len(email) == 0 {
		return "***"
	}
	// Find @ symbol
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}
	if atIndex <= 0 {
		return "***"
	}
	// Show first char + *** + @domain
	return string(email[0]) + "***" + email[atIndex:]
}

type AuthService struct {
	db     database.Database
	config *oauth2.Config

	// redirectURLsByHost maps a request's Host header to the OAuth redirect
	// URL registered for that host in the Google OAuth client. Only hosts
	// derived from GOOGLE_REDIRECT_URL (production) and STAGING_REDIRECT_URL
	// (staging) are ever present, so an unrecognized or spoofed Host header
	// simply falls through to the production default in config.RedirectURL —
	// it can never select an arbitrary attacker-supplied URL.
	redirectURLsByHost map[string]string
}

type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func NewAuthService(db database.Database) *AuthService {
	ctx := context.Background()

	// Get OAuth credentials from secrets or environment
	clientID, clientSecret, err := secrets.GetOAuthCredentials(ctx)
	if err != nil {
		// Fall back to environment variables for backwards compatibility
		clientID = os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &AuthService{
		db:                 db,
		config:             config,
		redirectURLsByHost: buildRedirectURLsByHost(config.RedirectURL, os.Getenv("STAGING_REDIRECT_URL")),
	}
}

// buildRedirectURLsByHost derives a Host -> redirect URL allowlist from the
// production and staging redirect URLs. Both must be exact, pre-registered
// redirect URIs on the Google OAuth client (Google rejects anything else).
func buildRedirectURLsByHost(redirectURLs ...string) map[string]string {
	byHost := make(map[string]string, len(redirectURLs))
	for _, redirectURL := range redirectURLs {
		if redirectURL == "" {
			continue
		}
		u, err := url.Parse(redirectURL)
		if err != nil || u.Host == "" {
			log.Printf("WARNING: could not parse redirect URL %q: %v", redirectURL, err)
			continue
		}
		byHost[u.Host] = redirectURL
	}
	return byHost
}

// configForHost returns the OAuth config to use for a request, with
// RedirectURL selected by matching host against the allowlist. Unrecognized
// hosts (including local dev) fall back to the production default.
func (a *AuthService) configForHost(host string) oauth2.Config {
	cfg := *a.config
	if redirectURL, ok := a.redirectURLsByHost[host]; ok {
		cfg.RedirectURL = redirectURL
	}
	return cfg
}

// GetAuthURL builds the Google OAuth consent URL. host is the incoming
// request's Host header, used to pick the matching registered redirect URI
// (production vs. staging) — see configForHost.
func (a *AuthService) GetAuthURL(state, host string) string {
	cfg := a.configForHost(host)
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// HandleCallback exchanges the OAuth code for a user. host must match the
// Host used when GetAuthURL generated the original auth request, since
// Google's token exchange requires the redirect_uri to match exactly.
func (a *AuthService) HandleCallback(code, host string) (*database.User, error) {
	ctx := context.Background()
	cfg := a.configForHost(host)

	// Exchange code for token
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token (code length: %d): %w", len(code), err)
	}

	// Get user info from Google
	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	if resp.StatusCode != 200 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("failed to get user info: unexpected status %d", resp.StatusCode)
	}
	defer func() { _ = resp.Body.Close() }()

	var googleUser GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Check if user exists in our database
	user, err := a.db.GetUserByGoogleID(googleUser.ID)
	if err != nil {
		// User doesn't exist, create new user
		user = &database.User{
			GoogleID:  googleUser.ID,
			Email:     googleUser.Email,
			Name:      googleUser.Name,
			Avatar:    googleUser.Picture,
			CreatedAt: time.Now(),
		}

		if err := a.db.CreateUser(user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	return user, nil
}

func (a *AuthService) ValidateConfig() error {
	if a.config.ClientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID environment variable is required")
	}
	if a.config.ClientSecret == "" {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET environment variable is required")
	}
	if a.config.RedirectURL == "" {
		return fmt.Errorf("GOOGLE_REDIRECT_URL environment variable is required")
	}
	return nil
}

// InitializeAdminUsers grants admin privileges to users specified in INITIAL_ADMIN_EMAILS
// This should be called on application startup to ensure initial admin access
func (a *AuthService) InitializeAdminUsers() error {
	cfg := config.Get()

	if len(cfg.InitialAdminEmails) == 0 {
		log.Println("No initial admin emails configured (INITIAL_ADMIN_EMAILS)")
		return nil
	}

	log.Printf("Initializing admin privileges for %d users", len(cfg.InitialAdminEmails))

	for _, email := range cfg.InitialAdminEmails {
		user, err := a.db.GetUserByEmail(email)
		if err != nil {
			log.Printf("Warning: Initial admin user not found: %s (user must sign in first)", redactEmail(email))
			continue
		}

		if user.IsAdmin {
			log.Printf("User %s already has admin privileges", redactEmail(email))
			continue
		}

		err = a.db.SetUserAdmin(user.ID, true)
		if err != nil {
			log.Printf("Error granting admin privileges to %s: %v", redactEmail(email), err)
			continue
		}

		log.Printf("✅ Granted admin privileges to user: %s (%s)", user.Name, redactEmail(user.Email))
	}

	return nil
}
