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

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			contains(s[1:], substr) ||
			(len(s) > 0 && s[:len(substr)] == substr))
}
