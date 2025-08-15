package unit

import (
	"fmt"
	"os"
	"testing"

	"goread2/internal/config"
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

func TestSubscriptionServiceWithFeatureFlag(t *testing.T) {
	db := helpers.CreateTestDB(t)
	subscriptionService := services.NewSubscriptionService(db)

	// Clean up environment at the end
	defer func() {
		_ = os.Unsetenv("SUBSCRIPTION_ENABLED")
		config.ResetForTesting()
	}()

	t.Run("CanUserAddFeed_SubscriptionDisabled", func(t *testing.T) {
		// Set subscription system to disabled
		_ = os.Setenv("SUBSCRIPTION_ENABLED", "false")
		config.ResetForTesting()
		config.Load()

		// Create a regular trial user
		user := helpers.CreateTestUser(t, db, "trial123", "trial@example.com", "Trial User")

		// Add 25 feeds (above the normal limit of 20)
		for i := 0; i < 25; i++ {
			err := subscriptionService.CanUserAddFeed(user.ID)
			if err != nil {
				t.Errorf("Expected user to be able to add feed %d when subscription disabled, got error: %v", i+1, err)
			}
		}
	})

	t.Run("CanUserAddFeed_SubscriptionEnabled", func(t *testing.T) {
		// Set subscription system to enabled
		_ = os.Setenv("SUBSCRIPTION_ENABLED", "true")
		config.ResetForTesting()
		config.Load()

		// Create a regular trial user
		user := helpers.CreateTestUser(t, db, "trial456", "trial2@example.com", "Trial User 2")

		// Add some feeds (within limit)
		for i := 0; i < 5; i++ {
			feed := helpers.CreateTestFeed(t, db, fmt.Sprintf("Test Feed %d", i), fmt.Sprintf("http://test%d.com", i), "Test description")
			err := db.SubscribeUserToFeed(user.ID, feed.ID)
			if err != nil {
				t.Fatalf("Failed to subscribe user to feed: %v", err)
			}
		}

		// Should still be able to add more feeds (under 20 limit)
		err := subscriptionService.CanUserAddFeed(user.ID)
		if err != nil {
			t.Errorf("Expected trial user to be able to add feeds under limit, got error: %v", err)
		}

		// Add many more feeds to hit the limit
		for i := 5; i < 20; i++ {
			feed := helpers.CreateTestFeed(t, db, fmt.Sprintf("Test Feed %d", i), fmt.Sprintf("http://test%d.com", i), "Test description")
			err := db.SubscribeUserToFeed(user.ID, feed.ID)
			if err != nil {
				t.Fatalf("Failed to subscribe user to feed: %v", err)
			}
		}

		// Now should hit the limit
		err = subscriptionService.CanUserAddFeed(user.ID)
		if err == nil {
			t.Error("Expected trial user to hit feed limit when subscription enabled")
		}
		if err != services.ErrFeedLimitReached {
			t.Errorf("Expected ErrFeedLimitReached, got: %v", err)
		}
	})

	t.Run("GetUserSubscriptionInfo_SubscriptionDisabled", func(t *testing.T) {
		// Set subscription system to disabled
		_ = os.Setenv("SUBSCRIPTION_ENABLED", "false")
		config.ResetForTesting()
		config.Load()

		// Create a regular user
		user := helpers.CreateTestUser(t, db, "info123", "info@example.com", "Info User")

		// Get subscription info
		info, err := subscriptionService.GetUserSubscriptionInfo(user.ID)
		if err != nil {
			t.Fatalf("Failed to get subscription info: %v", err)
		}

		// Should have unlimited status when subscription disabled
		if info.Status != "unlimited" {
			t.Errorf("Expected status 'unlimited' when subscription disabled, got %s", info.Status)
		}

		if info.FeedLimit != -1 {
			t.Errorf("Expected unlimited feed limit (-1), got %d", info.FeedLimit)
		}

		if !info.CanAddFeeds {
			t.Error("Expected CanAddFeeds to be true when subscription disabled")
		}
	})

	t.Run("GetUserSubscriptionInfo_SubscriptionEnabled", func(t *testing.T) {
		// Set subscription system to enabled
		_ = os.Setenv("SUBSCRIPTION_ENABLED", "true")
		config.ResetForTesting()
		config.Load()

		// Create a regular trial user
		user := helpers.CreateTestUser(t, db, "info456", "info2@example.com", "Info User 2")

		// Get subscription info
		info, err := subscriptionService.GetUserSubscriptionInfo(user.ID)
		if err != nil {
			t.Fatalf("Failed to get subscription info: %v", err)
		}

		// Should have trial status when subscription enabled
		if info.Status != "trial" {
			t.Errorf("Expected status 'trial' when subscription enabled, got %s", info.Status)
		}

		if info.FeedLimit != services.FreeTrialFeedLimit {
			t.Errorf("Expected feed limit %d, got %d", services.FreeTrialFeedLimit, info.FeedLimit)
		}
	})

	t.Run("AdminUser_AlwaysUnlimited", func(t *testing.T) {
		// Test both enabled and disabled subscription states
		for _, enabled := range []bool{true, false} {
			t.Run(map[bool]string{true: "enabled", false: "disabled"}[enabled], func(t *testing.T) {
				// Set subscription system state
				_ = os.Setenv("SUBSCRIPTION_ENABLED", map[bool]string{true: "true", false: "false"}[enabled])
				config.ResetForTesting()
				config.Load()

				// Create admin user with unique email for each subtest
				suffix := map[bool]string{true: "enabled", false: "disabled"}[enabled]
				email := fmt.Sprintf("admin3_%s@example.com", suffix)
				googleID := fmt.Sprintf("admin789_%s", suffix)
				user := helpers.CreateTestUser(t, db, googleID, email, "Admin User 3")
				err := db.SetUserAdmin(user.ID, true)
				if err != nil {
					t.Fatalf("Failed to set user admin: %v", err)
				}

				// Admin should always be able to add feeds
				err = subscriptionService.CanUserAddFeed(user.ID)
				if err != nil {
					t.Errorf("Expected admin user to always be able to add feeds (subscription %s), got error: %v", 
						map[bool]string{true: "enabled", false: "disabled"}[enabled], err)
				}

				// Get subscription info
				info, err := subscriptionService.GetUserSubscriptionInfo(user.ID)
				if err != nil {
					t.Fatalf("Failed to get admin subscription info: %v", err)
				}

				expectedStatus := "admin"
				if !enabled {
					expectedStatus = "unlimited" // When subscription disabled, even admins get "unlimited" status
				}

				if info.Status != expectedStatus {
					t.Errorf("Expected admin status '%s' (subscription %s), got %s", 
						expectedStatus, map[bool]string{true: "enabled", false: "disabled"}[enabled], info.Status)
				}

				if info.FeedLimit != -1 {
					t.Errorf("Expected admin to have unlimited feeds (-1), got %d", info.FeedLimit)
				}

				if !info.CanAddFeeds {
					t.Error("Expected admin to always be able to add feeds")
				}
			})
		}
	})
}