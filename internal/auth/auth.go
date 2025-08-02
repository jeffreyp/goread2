package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"goread2/internal/database"
)

type AuthService struct {
	db     database.Database
	config *oauth2.Config
}

type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func NewAuthService(db database.Database) *AuthService {
	config := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &AuthService{
		db:     db,
		config: config,
	}
}

func (a *AuthService) GetAuthURL(state string) string {
	return a.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (a *AuthService) HandleCallback(code string) (*database.User, error) {
	ctx := context.Background()

	// Exchange code for token
	token, err := a.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user info from Google
	client := a.config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

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
