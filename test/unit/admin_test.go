package unit

import (
	"testing"

	"goread2/internal/services"
	"goread2/test/helpers"
)

func TestUserAdminOperations(t *testing.T) {
	db := helpers.CreateTestDB(t)

	t.Run("GetUserByEmail", func(t *testing.T) {
		// Create a test user
		originalUser := helpers.CreateTestUser(t, db, "google123", "admin@example.com", "Admin User")

		// Retrieve the user by email
		retrievedUser, err := db.GetUserByEmail("admin@example.com")
		if err != nil {
			t.Fatalf("Failed to get user by email: %v", err)
		}

		// Verify all fields
		if retrievedUser.ID != originalUser.ID {
			t.Errorf("Expected ID %d, got %d", originalUser.ID, retrievedUser.ID)
		}
		if retrievedUser.Email != originalUser.Email {
			t.Errorf("Expected email %s, got %s", originalUser.Email, retrievedUser.Email)
		}
		if retrievedUser.Name != originalUser.Name {
			t.Errorf("Expected name %s, got %s", originalUser.Name, retrievedUser.Name)
		}
		if retrievedUser.SubscriptionStatus != "trial" {
			t.Errorf("Expected default subscription status 'trial', got %s", retrievedUser.SubscriptionStatus)
		}
		if retrievedUser.IsAdmin != false {
			t.Errorf("Expected default admin status false, got %v", retrievedUser.IsAdmin)
		}
		if retrievedUser.FreeMonthsRemaining != 0 {
			t.Errorf("Expected default free months 0, got %d", retrievedUser.FreeMonthsRemaining)
		}
	})

	t.Run("SetUserAdmin", func(t *testing.T) {
		// Create a test user
		user := helpers.CreateTestUser(t, db, "google456", "user@example.com", "Regular User")

		// Grant admin access
		err := db.SetUserAdmin(user.ID, true)
		if err != nil {
			t.Fatalf("Failed to set user admin: %v", err)
		}

		// Verify admin status was set
		retrievedUser, err := db.GetUserByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user by ID: %v", err)
		}

		if !retrievedUser.IsAdmin {
			t.Error("Expected user to be admin after SetUserAdmin(true)")
		}

		// Revoke admin access
		err = db.SetUserAdmin(user.ID, false)
		if err != nil {
			t.Fatalf("Failed to revoke user admin: %v", err)
		}

		// Verify admin status was revoked
		retrievedUser, err = db.GetUserByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user by ID after revoke: %v", err)
		}

		if retrievedUser.IsAdmin {
			t.Error("Expected user to not be admin after SetUserAdmin(false)")
		}
	})

	t.Run("GrantFreeMonths", func(t *testing.T) {
		// Create a test user
		user := helpers.CreateTestUser(t, db, "google789", "freemonths@example.com", "Free Months User")

		// Grant 3 free months
		err := db.GrantFreeMonths(user.ID, 3)
		if err != nil {
			t.Fatalf("Failed to grant free months: %v", err)
		}

		// Verify free months were granted
		retrievedUser, err := db.GetUserByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user by ID: %v", err)
		}

		if retrievedUser.FreeMonthsRemaining != 3 {
			t.Errorf("Expected 3 free months, got %d", retrievedUser.FreeMonthsRemaining)
		}

		// Grant 2 more free months (should add to existing)
		err = db.GrantFreeMonths(user.ID, 2)
		if err != nil {
			t.Fatalf("Failed to grant additional free months: %v", err)
		}

		// Verify total free months
		retrievedUser, err = db.GetUserByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user by ID after additional grant: %v", err)
		}

		if retrievedUser.FreeMonthsRemaining != 5 {
			t.Errorf("Expected 5 total free months, got %d", retrievedUser.FreeMonthsRemaining)
		}
	})
}

func TestSubscriptionService(t *testing.T) {
	db := helpers.CreateTestDB(t)
	subscriptionService := services.NewSubscriptionService(db)

	t.Run("GetUserByEmail", func(t *testing.T) {
		// Create a test user
		originalUser := helpers.CreateTestUser(t, db, "service123", "service@example.com", "Service User")

		// Get user through subscription service
		retrievedUser, err := subscriptionService.GetUserByEmail("service@example.com")
		if err != nil {
			t.Fatalf("Failed to get user by email through service: %v", err)
		}

		if retrievedUser.Email != originalUser.Email {
			t.Errorf("Expected email %s, got %s", originalUser.Email, retrievedUser.Email)
		}
	})

	t.Run("SetUserAdmin", func(t *testing.T) {
		// Create a test user
		user := helpers.CreateTestUser(t, db, "service456", "adminservice@example.com", "Admin Service User")

		// Set admin through service
		err := subscriptionService.SetUserAdmin(user.ID, true)
		if err != nil {
			t.Fatalf("Failed to set user admin through service: %v", err)
		}

		// Verify through direct database access
		retrievedUser, err := db.GetUserByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user by ID: %v", err)
		}

		if !retrievedUser.IsAdmin {
			t.Error("Expected user to be admin after service call")
		}
	})

	t.Run("CanUserAddFeed_Admin", func(t *testing.T) {
		// Create admin user
		user := helpers.CreateTestUser(t, db, "admin123", "canadd@example.com", "Can Add User")
		err := db.SetUserAdmin(user.ID, true)
		if err != nil {
			t.Fatalf("Failed to set user admin: %v", err)
		}

		// Admin should always be able to add feeds
		err = subscriptionService.CanUserAddFeed(user.ID)
		if err != nil {
			t.Errorf("Expected admin user to be able to add feeds, got error: %v", err)
		}
	})
}