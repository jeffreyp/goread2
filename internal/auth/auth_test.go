package auth

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/config"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/secrets"
)

// mockDBForAuth extends mockDB with specific behaviors for auth testing
type mockDBForAuth struct {
	mockDB
	users                 map[string]*database.User
	usersByEmail          map[string]*database.User
	usersByID             map[int]*database.User
	shouldFailCreate      bool
	shouldFailGetByGoogle bool
	shouldFailGetByEmail  bool
	shouldFailSetAdmin    bool
}

func newMockDBForAuth() *mockDBForAuth {
	return &mockDBForAuth{
		users:        make(map[string]*database.User),
		usersByEmail: make(map[string]*database.User),
		usersByID:    make(map[int]*database.User),
	}
}

func (m *mockDBForAuth) CreateUser(user *database.User) error {
	if m.shouldFailCreate {
		return errors.New("failed to create user")
	}
	user.ID = len(m.users) + 1
	m.users[user.GoogleID] = user
	m.usersByEmail[user.Email] = user
	m.usersByID[user.ID] = user
	return nil
}

func (m *mockDBForAuth) GetUserByGoogleID(googleID string) (*database.User, error) {
	if m.shouldFailGetByGoogle {
		return nil, errors.New("failed to get user by google ID")
	}
	user, exists := m.users[googleID]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *mockDBForAuth) GetUserByEmail(email string) (*database.User, error) {
	if m.shouldFailGetByEmail {
		return nil, errors.New("failed to get user by email")
	}
	user, exists := m.usersByEmail[email]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *mockDBForAuth) SetUserAdmin(userID int, isAdmin bool) error {
	if m.shouldFailSetAdmin {
		return errors.New("failed to set user admin")
	}
	if user, exists := m.usersByID[userID]; exists {
		user.IsAdmin = isAdmin
	}
	return nil
}

func (m *mockDBForAuth) UpdateUserMaxArticlesOnFeedAdd(userID int, maxArticles int) error {
	if user, exists := m.usersByID[userID]; exists {
		user.MaxArticlesOnFeedAdd = maxArticles
	}
	return nil
}

func (m *mockDBForAuth) CreateSession(*database.Session) error           { return nil }
func (m *mockDBForAuth) GetSession(string) (*database.Session, error)    { return nil, nil }
func (m *mockDBForAuth) UpdateSessionExpiry(string, time.Time) error     { return nil }
func (m *mockDBForAuth) DeleteSession(string) error                      { return nil }
func (m *mockDBForAuth) DeleteExpiredSessions() error                    { return nil }
func (m *mockDBForAuth) UpdateFeedCacheHeaders(feedID int, etag, lastModified string) error {
	return nil
}

func TestNewAuthService(t *testing.T) {
	db := newMockDBForAuth()

	// Set environment variables for OAuth config
	_ = os.Setenv("GOOGLE_CLIENT_ID", "test_client_id_123")
	_ = os.Setenv("GOOGLE_CLIENT_SECRET", "test_client_secret_456")
	_ = os.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/callback")
	defer func() {
		_ = os.Unsetenv("GOOGLE_CLIENT_ID")
		_ = os.Unsetenv("GOOGLE_CLIENT_SECRET")
		_ = os.Unsetenv("GOOGLE_REDIRECT_URL")
	}()

	authService := NewAuthService(db)

	if authService == nil {
		t.Fatal("NewAuthService returned nil")
		return
	}

	if authService.config == nil {
		t.Error("AuthService OAuth config not set")
		return
	}

	if authService.config.ClientID != "test_client_id_123" {
		t.Errorf("ClientID = %s, want test_client_id_123", authService.config.ClientID)
	}

	if authService.config.ClientSecret != "test_client_secret_456" {
		t.Errorf("ClientSecret = %s, want test_client_secret_456", authService.config.ClientSecret)
	}
}

func TestGetAuthURL(t *testing.T) {
	db := newMockDBForAuth()

	_ = os.Setenv("GOOGLE_CLIENT_ID", "test_client_id_123")
	_ = os.Setenv("GOOGLE_CLIENT_SECRET", "test_client_secret_456")
	_ = os.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/callback")
	defer func() {
		_ = os.Unsetenv("GOOGLE_CLIENT_ID")
		_ = os.Unsetenv("GOOGLE_CLIENT_SECRET")
		_ = os.Unsetenv("GOOGLE_REDIRECT_URL")
	}()

	authService := NewAuthService(db)

	state := "test_state_123"
	authURL := authService.GetAuthURL(state)

	if authURL == "" {
		t.Error("GetAuthURL returned empty string")
	}

	// Should contain the state parameter
	if !contains(authURL, state) {
		t.Error("Auth URL should contain state parameter")
	}

	// Should contain Google OAuth endpoint
	if !contains(authURL, "accounts.google.com") {
		t.Error("Auth URL should contain Google OAuth endpoint")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		redirectURL  string
		expectError  bool
	}{
		{
			name:         "valid config",
			clientID:     "test_client_id_123",
			clientSecret: "test_client_secret_456",
			redirectURL:  "http://localhost:8080/callback",
			expectError:  false,
		},
		{
			name:         "missing client ID",
			clientID:     "",
			clientSecret: "test_client_secret_456",
			redirectURL:  "http://localhost:8080/callback",
			expectError:  true,
		},
		{
			name:         "missing client secret",
			clientID:     "test_client_id_123",
			clientSecret: "",
			redirectURL:  "http://localhost:8080/callback",
			expectError:  true,
		},
		{
			name:         "missing redirect URL",
			clientID:     "test_client_id_123",
			clientSecret: "test_client_secret_456",
			redirectURL:  "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset secret cache before each test
			secrets.ResetCacheForTesting()

			db := newMockDBForAuth()

			// Save original environment variables
			origClientID := os.Getenv("GOOGLE_CLIENT_ID")
			origClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
			origRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

			// Set test environment variables
			_ = os.Setenv("GOOGLE_CLIENT_ID", tt.clientID)
			_ = os.Setenv("GOOGLE_CLIENT_SECRET", tt.clientSecret)
			_ = os.Setenv("GOOGLE_REDIRECT_URL", tt.redirectURL)

			// Restore original environment variables after test
			defer func() {
				if origClientID != "" {
					_ = os.Setenv("GOOGLE_CLIENT_ID", origClientID)
				} else {
					_ = os.Unsetenv("GOOGLE_CLIENT_ID")
				}
				if origClientSecret != "" {
					_ = os.Setenv("GOOGLE_CLIENT_SECRET", origClientSecret)
				} else {
					_ = os.Unsetenv("GOOGLE_CLIENT_SECRET")
				}
				if origRedirectURL != "" {
					_ = os.Setenv("GOOGLE_REDIRECT_URL", origRedirectURL)
				} else {
					_ = os.Unsetenv("GOOGLE_REDIRECT_URL")
				}
			}()

			authService := NewAuthService(db)
			err := authService.ValidateConfig()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestInitializeAdminUsers(t *testing.T) {
	tests := []struct {
		name          string
		adminEmails   []string
		existingUsers map[string]*database.User
		shouldFailGet bool
		shouldFailSet bool
		expectError   bool
	}{
		{
			name:        "no admin emails configured",
			adminEmails: []string{},
			expectError: false,
		},
		{
			name:        "admin user exists and gets privileges",
			adminEmails: []string{"admin@example.com"},
			existingUsers: map[string]*database.User{
				"admin@example.com": {
					ID:      1,
					Email:   "admin@example.com",
					Name:    "Admin User",
					IsAdmin: false,
				},
			},
			expectError: false,
		},
		{
			name:        "admin user doesn't exist (warning logged but no error)",
			adminEmails: []string{"nonexistent@example.com"},
			expectError: false,
		},
		{
			name:        "admin user already has privileges",
			adminEmails: []string{"admin@example.com"},
			existingUsers: map[string]*database.User{
				"admin@example.com": {
					ID:      1,
					Email:   "admin@example.com",
					Name:    "Admin User",
					IsAdmin: true,
				},
			},
			expectError: false,
		},
		{
			name:          "database error when getting user",
			adminEmails:   []string{"admin@example.com"},
			shouldFailGet: true,
			expectError:   false, // Should continue processing despite error
		},
		{
			name:        "database error when setting admin",
			adminEmails: []string{"admin@example.com"},
			existingUsers: map[string]*database.User{
				"admin@example.com": {
					ID:      1,
					Email:   "admin@example.com",
					Name:    "Admin User",
					IsAdmin: false,
				},
			},
			shouldFailSet: true,
			expectError:   false, // Should continue processing despite error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config with admin emails
			config.ResetForTesting()
			for i, email := range tt.adminEmails {
				envVar := "INITIAL_ADMIN_EMAILS"
				if i == 0 {
					_ = os.Setenv(envVar, email)
				} else {
					_ = os.Setenv(envVar, os.Getenv(envVar)+","+email)
				}
			}
			defer func() {
				_ = os.Unsetenv("INITIAL_ADMIN_EMAILS")
				config.ResetForTesting()
			}()
			config.Load()

			// Setup database mock
			db := newMockDBForAuth()
			db.shouldFailGetByEmail = tt.shouldFailGet
			db.shouldFailSetAdmin = tt.shouldFailSet

			// Add existing users
			for email, user := range tt.existingUsers {
				db.usersByEmail[email] = user
				db.usersByID[user.ID] = user
			}

			// Setup auth service
			_ = os.Setenv("GOOGLE_CLIENT_ID", "test_client_id_123")
			_ = os.Setenv("GOOGLE_CLIENT_SECRET", "test_client_secret_456")
			_ = os.Setenv("GOOGLE_REDIRECT_URL", "http://localhost/callback")
			defer func() {
				_ = os.Unsetenv("GOOGLE_CLIENT_ID")
				_ = os.Unsetenv("GOOGLE_CLIENT_SECRET")
				_ = os.Unsetenv("GOOGLE_REDIRECT_URL")
			}()

			authService := NewAuthService(db)
			err := authService.InitializeAdminUsers()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify admin privileges were set for existing users
			for email, originalUser := range tt.existingUsers {
				if !originalUser.IsAdmin && !tt.shouldFailSet {
					if user, exists := db.usersByEmail[email]; exists {
						if !user.IsAdmin {
							t.Errorf("User %s should have been granted admin privileges", email)
						}
					}
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
