package unit

import (
	"testing"
	"time"

	"goread2/internal/auth"
	"goread2/internal/database"
	"goread2/test/helpers"
)

func TestAuthService(t *testing.T) {
	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	db := helpers.CreateTestDB(t)
	authService := auth.NewAuthService(db)

	t.Run("ValidateConfig", func(t *testing.T) {
		err := authService.ValidateConfig()
		if err != nil {
			t.Errorf("Expected config to be valid, got error: %v", err)
		}
	})

	t.Run("GetAuthURL", func(t *testing.T) {
		state := "test_state_123"
		authURL := authService.GetAuthURL(state)

		if authURL == "" {
			t.Error("Expected auth URL to be non-empty")
		}

		// Check that the URL contains the state parameter
		if !contains(authURL, state) {
			t.Errorf("Expected auth URL to contain state %s", state)
		}
	})
}

func TestSessionManager(t *testing.T) {
	db := helpers.CreateTestDB(t)
	sessionManager := auth.NewSessionManager(db)

	// Create a test user
	user := &database.User{
		ID:        1,
		GoogleID:  "google123",
		Email:     "test@example.com",
		Name:      "Test User",
		Avatar:    "https://example.com/avatar.jpg",
		CreatedAt: time.Now(),
	}

	t.Run("CreateSession", func(t *testing.T) {
		session, err := sessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		if session.ID == "" {
			t.Error("Expected session ID to be non-empty")
		}

		if session.UserID != user.ID {
			t.Errorf("Expected session user ID %d, got %d", user.ID, session.UserID)
		}

		if session.ExpiresAt.Before(time.Now()) {
			t.Error("Expected session to expire in the future")
		}
	})

	t.Run("GetSession", func(t *testing.T) {
		// Create a session first
		originalSession, err := sessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Retrieve the session
		retrievedSession, exists := sessionManager.GetSession(originalSession.ID)
		if !exists {
			t.Error("Expected session to exist")
		}

		if retrievedSession.UserID != user.ID {
			t.Errorf("Expected session user ID %d, got %d", user.ID, retrievedSession.UserID)
		}
	})

	t.Run("GetNonexistentSession", func(t *testing.T) {
		_, exists := sessionManager.GetSession("nonexistent_session_id")
		if exists {
			t.Error("Expected nonexistent session to not exist")
		}
	})

	t.Run("DeleteSession", func(t *testing.T) {
		// Create a session first
		session, err := sessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Delete the session
		sessionManager.DeleteSession(session.ID)

		// Try to retrieve the deleted session
		_, exists := sessionManager.GetSession(session.ID)
		if exists {
			t.Error("Expected deleted session to not exist")
		}
	})

	t.Run("ExpiredSession", func(t *testing.T) {
		// Create a session
		session, err := sessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Manually expire the session
		session.ExpiresAt = time.Now().Add(-1 * time.Hour)

		// Try to retrieve the expired session
		_, exists := sessionManager.GetSession(session.ID)
		if exists {
			t.Error("Expected expired session to not exist")
		}
	})
}

func TestMiddleware(t *testing.T) {
	db := helpers.CreateTestDB(t)

	// Create a test user
	user := helpers.CreateTestUser(t, db, "google123", "test@example.com", "Test User")

	t.Run("RequireAuth_ValidSession", func(t *testing.T) {
		testServer := helpers.SetupTestServer(t)

		// Create request with valid session
		req := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds", nil, user)
		rr := testServer.ExecuteRequest(req)

		// Should not return 401
		if rr.Code == 401 {
			t.Error("Expected authenticated request to not return 401")
		}
	})

	t.Run("RequireAuth_NoSession", func(t *testing.T) {
		testServer := helpers.SetupTestServer(t)

		// Create request without session
		req := helpers.CreateUnauthenticatedRequest(t, "GET", "/api/feeds", nil)
		rr := testServer.ExecuteRequest(req)

		// Should return 401
		if rr.Code != 401 {
			t.Errorf("Expected unauthenticated request to return 401, got %d", rr.Code)
		}
	})
}

// TestAuthServiceUserCreation tests user creation and retrieval with both database types
func TestAuthServiceUserCreation(t *testing.T) {
	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testCases := []struct {
		name   string
		dbFunc func(*testing.T) database.Database
	}{
		{
			name:   "SQLite",
			dbFunc: helpers.CreateTestDB,
		},
		{
			name: "Datastore",
			dbFunc: func(t *testing.T) database.Database {
				if testing.Short() {
					t.Skip("Skipping Datastore tests in short mode")
				}
				return helpers.CreateTestDatastoreDB(t)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := tc.dbFunc(t)

			t.Run("HandleCallback_NewUser", func(t *testing.T) {
				// This test simulates what happens in the auth callback
				// when a new user signs in for the first time
				
				// First verify user doesn't exist
				_, err := db.GetUserByGoogleID("new_user_123")
				if err == nil {
					t.Error("Expected new user to not exist initially")
				}

				// Create user (simulating what HandleCallback does)
				user := &database.User{
					GoogleID:  "new_user_123",
					Email:     "newuser@example.com",
					Name:      "New User",
					Avatar:    "https://lh3.googleusercontent.com/avatar",
					CreatedAt: time.Now(),
				}

				err = db.CreateUser(user)
				if err != nil {
					t.Fatalf("Failed to create user: %v", err)
				}

				// Verify user was created with an ID
				if user.ID == 0 {
					t.Error("Expected user ID to be set after creation")
				}

				// Verify user can be retrieved
				retrievedUser, err := db.GetUserByGoogleID("new_user_123")
				if err != nil {
					t.Fatalf("Failed to retrieve created user: %v", err)
				}

				if retrievedUser.Email != user.Email {
					t.Errorf("Expected email %s, got %s", user.Email, retrievedUser.Email)
				}
			})

			t.Run("HandleCallback_ExistingUser", func(t *testing.T) {
				// Create user first
				existingUser := &database.User{
					GoogleID:  "existing_user_456",
					Email:     "existing@example.com",
					Name:      "Existing User",
					Avatar:    "https://lh3.googleusercontent.com/existing",
					CreatedAt: time.Now(),
				}

				err := db.CreateUser(existingUser)
				if err != nil {
					t.Fatalf("Failed to create existing user: %v", err)
				}

				// Simulate HandleCallback finding existing user
				retrievedUser, err := db.GetUserByGoogleID("existing_user_456")
				if err != nil {
					t.Fatalf("Failed to retrieve existing user: %v", err)
				}

				// Verify it's the same user
				if retrievedUser.ID != existingUser.ID {
					t.Errorf("Expected user ID %d, got %d", existingUser.ID, retrievedUser.ID)
				}
				if retrievedUser.Email != existingUser.Email {
					t.Errorf("Expected email %s, got %s", existingUser.Email, retrievedUser.Email)
				}
			})
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			contains(s[1:], substr) ||
			(len(s) > 0 && s[:len(substr)] == substr))
}
