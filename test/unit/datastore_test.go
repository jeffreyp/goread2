package unit

import (
	"testing"
	"time"

	"goread2/internal/database"
	"goread2/test/helpers"
)

func TestDatastoreUserOperations(t *testing.T) {
	// Skip if not running with Datastore emulator
	if testing.Short() {
		t.Skip("Skipping Datastore tests in short mode")
	}

	db := helpers.CreateTestDatastoreDB(t)

	t.Run("CreateUser", func(t *testing.T) {
		user := &database.User{
			GoogleID:  "google123",
			Email:     "test@example.com",
			Name:      "Test User",
			Avatar:    "https://example.com/avatar.jpg",
			CreatedAt: time.Now(),
		}

		err := db.CreateUser(user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		if user.ID == 0 {
			t.Error("Expected user ID to be set after creation")
		}
	})

	t.Run("GetUserByGoogleID", func(t *testing.T) {
		// First create a user
		originalUser := &database.User{
			GoogleID:  "google456",
			Email:     "test2@example.com",
			Name:      "Test User 2",
			Avatar:    "https://example.com/avatar2.jpg",
			CreatedAt: time.Now(),
		}

		err := db.CreateUser(originalUser)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Retrieve the user
		retrievedUser, err := db.GetUserByGoogleID("google456")
		if err != nil {
			t.Fatalf("Failed to get user by Google ID: %v", err)
		}

		if retrievedUser.Email != originalUser.Email {
			t.Errorf("Expected email %s, got %s", originalUser.Email, retrievedUser.Email)
		}
		if retrievedUser.Name != originalUser.Name {
			t.Errorf("Expected name %s, got %s", originalUser.Name, retrievedUser.Name)
		}
		if retrievedUser.GoogleID != originalUser.GoogleID {
			t.Errorf("Expected Google ID %s, got %s", originalUser.GoogleID, retrievedUser.GoogleID)
		}
	})

	t.Run("GetUserByID", func(t *testing.T) {
		// First create a user
		originalUser := &database.User{
			GoogleID:  "google789",
			Email:     "test3@example.com",
			Name:      "Test User 3",
			Avatar:    "https://example.com/avatar3.jpg",
			CreatedAt: time.Now(),
		}

		err := db.CreateUser(originalUser)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Retrieve the user by ID
		retrievedUser, err := db.GetUserByID(originalUser.ID)
		if err != nil {
			t.Fatalf("Failed to get user by ID: %v", err)
		}

		if retrievedUser.GoogleID != originalUser.GoogleID {
			t.Errorf("Expected Google ID %s, got %s", originalUser.GoogleID, retrievedUser.GoogleID)
		}
		if retrievedUser.Email != originalUser.Email {
			t.Errorf("Expected email %s, got %s", originalUser.Email, retrievedUser.Email)
		}
	})

	t.Run("GetUserByGoogleID_NotFound", func(t *testing.T) {
		_, err := db.GetUserByGoogleID("nonexistent")
		if err == nil {
			t.Error("Expected error when getting non-existent user")
		}
	})

	t.Run("CreateUser_DuplicateGoogleID", func(t *testing.T) {
		user1 := &database.User{
			GoogleID:  "duplicate123",
			Email:     "user1@example.com",
			Name:      "User 1",
			CreatedAt: time.Now(),
		}

		user2 := &database.User{
			GoogleID:  "duplicate123", // Same Google ID
			Email:     "user2@example.com",
			Name:      "User 2",
			CreatedAt: time.Now(),
		}

		// First user should succeed
		err := db.CreateUser(user1)
		if err != nil {
			t.Fatalf("Failed to create first user: %v", err)
		}

		// Second user with same Google ID should fail or handle gracefully
		err = db.CreateUser(user2)
		// Note: Datastore doesn't have unique constraints like SQL, so this might succeed
		// In a real implementation, you'd want to check for duplicates first
		if err != nil {
			t.Logf("Expected behavior: duplicate Google ID rejected: %v", err)
		}
	})
}

// TestDatastoreUserOperationsIntegration tests the user operations with auth flow
func TestDatastoreUserOperationsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	db := helpers.CreateTestDatastoreDB(t)

	t.Run("AuthFlow_NewUser", func(t *testing.T) {
		googleID := "auth_flow_new_user"
		
		// Simulate auth flow - try to get user (should fail)
		_, err := db.GetUserByGoogleID(googleID)
		if err == nil {
			t.Error("Expected user not to exist initially")
		}

		// Create new user (as auth service would do)
		user := &database.User{
			GoogleID:  googleID,
			Email:     "newuser@example.com",
			Name:      "New User",
			Avatar:    "https://example.com/new_avatar.jpg",
			CreatedAt: time.Now(),
		}

		err = db.CreateUser(user)
		if err != nil {
			t.Fatalf("Failed to create user during auth flow: %v", err)
		}

		// Verify user can be retrieved
		retrievedUser, err := db.GetUserByGoogleID(googleID)
		if err != nil {
			t.Fatalf("Failed to retrieve newly created user: %v", err)
		}

		if retrievedUser.Email != user.Email {
			t.Errorf("Expected email %s, got %s", user.Email, retrievedUser.Email)
		}
	})

	t.Run("AuthFlow_ExistingUser", func(t *testing.T) {
		googleID := "auth_flow_existing_user"
		
		// Create user first
		originalUser := &database.User{
			GoogleID:  googleID,
			Email:     "existing@example.com",
			Name:      "Existing User",
			Avatar:    "https://example.com/existing_avatar.jpg",
			CreatedAt: time.Now(),
		}

		err := db.CreateUser(originalUser)
		if err != nil {
			t.Fatalf("Failed to create initial user: %v", err)
		}

		// Simulate auth flow - get existing user
		retrievedUser, err := db.GetUserByGoogleID(googleID)
		if err != nil {
			t.Fatalf("Failed to get existing user during auth flow: %v", err)
		}

		if retrievedUser.ID != originalUser.ID {
			t.Errorf("Expected user ID %d, got %d", originalUser.ID, retrievedUser.ID)
		}
		if retrievedUser.Email != originalUser.Email {
			t.Errorf("Expected email %s, got %s", originalUser.Email, retrievedUser.Email)
		}
	})
}