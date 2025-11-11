package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jeffreyp/goread2/internal/config"
	"github.com/jeffreyp/goread2/internal/services"
	"github.com/jeffreyp/goread2/test/helpers"
)

func TestAPIWithFeatureFlag(t *testing.T) {
	// Clean up test users at start and end
	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	// Clean up environment at the end
	defer func() {
		_ = os.Unsetenv("SUBSCRIPTION_ENABLED")
		config.ResetForTesting()
	}()

	t.Run("SubscriptionEndpoint_SubscriptionDisabled", func(t *testing.T) {
		// Set subscription system to disabled
		_ = os.Setenv("SUBSCRIPTION_ENABLED", "false")
		config.ResetForTesting()
		config.Load()

		// Create test database and services
		db := helpers.CreateTestDB(t)
		subscriptionService := services.NewSubscriptionService(db)

		// Create test user
		user := helpers.CreateTestUser(t, db, "api123", "api@example.com", "API User")

		// Create test server with subscription endpoint
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/subscription" && r.Method == "GET" {
				// Simulate getting subscription info
				info, err := subscriptionService.GetUserSubscriptionInfo(user.ID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(info)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		// Make request to subscription endpoint
		resp, err := http.Get(server.URL + "/api/subscription")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Parse response
		var subscriptionInfo services.SubscriptionInfo
		err = json.NewDecoder(resp.Body).Decode(&subscriptionInfo)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify unlimited status when subscription disabled
		if subscriptionInfo.Status != "unlimited" {
			t.Errorf("Expected status 'unlimited' when subscription disabled, got %s", subscriptionInfo.Status)
		}

		if subscriptionInfo.FeedLimit != -1 {
			t.Errorf("Expected unlimited feed limit (-1), got %d", subscriptionInfo.FeedLimit)
		}

		if !subscriptionInfo.CanAddFeeds {
			t.Error("Expected CanAddFeeds to be true when subscription disabled")
		}
	})

	t.Run("SubscriptionEndpoint_SubscriptionEnabled", func(t *testing.T) {
		// Set subscription system to enabled
		_ = os.Setenv("SUBSCRIPTION_ENABLED", "true")
		config.ResetForTesting()
		config.Load()

		// Create test database and services
		db := helpers.CreateTestDB(t)
		subscriptionService := services.NewSubscriptionService(db)

		// Create test user
		user := helpers.CreateTestUser(t, db, "api456", "api2@example.com", "API User 2")

		// Create test server with subscription endpoint
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/subscription" && r.Method == "GET" {
				// Simulate getting subscription info
				info, err := subscriptionService.GetUserSubscriptionInfo(user.ID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(info)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		// Make request to subscription endpoint
		resp, err := http.Get(server.URL + "/api/subscription")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Parse response
		var subscriptionInfo services.SubscriptionInfo
		err = json.NewDecoder(resp.Body).Decode(&subscriptionInfo)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify trial status when subscription enabled
		if subscriptionInfo.Status != "trial" {
			t.Errorf("Expected status 'trial' when subscription enabled, got %s", subscriptionInfo.Status)
		}

		if subscriptionInfo.FeedLimit != services.FreeTrialFeedLimit {
			t.Errorf("Expected feed limit %d, got %d", services.FreeTrialFeedLimit, subscriptionInfo.FeedLimit)
		}
	})

	t.Run("FeedLimitBehavior_SubscriptionDisabled", func(t *testing.T) {
		// Set subscription system to disabled
		_ = os.Setenv("SUBSCRIPTION_ENABLED", "false")
		config.ResetForTesting()
		config.Load()

		// Create test database and services
		db := helpers.CreateTestDB(t)
		subscriptionService := services.NewSubscriptionService(db)

		// Create test user
		user := helpers.CreateTestUser(t, db, "limit123", "limit@example.com", "Limit User")

		// Test that user can add many feeds when subscription disabled
		for i := 0; i < 50; i++ { // Far above normal limit
			err := subscriptionService.CanUserAddFeed(user.ID)
			if err != nil {
				t.Errorf("Expected user to be able to add feed %d when subscription disabled, got error: %v", i+1, err)
				break
			}
		}
	})

	t.Run("FeedLimitBehavior_SubscriptionEnabled", func(t *testing.T) {
		// Set subscription system to enabled
		_ = os.Setenv("SUBSCRIPTION_ENABLED", "true")
		config.ResetForTesting()
		config.Load()

		// Create test database and services
		db := helpers.CreateTestDB(t)
		subscriptionService := services.NewSubscriptionService(db)

		// Create test user
		user := helpers.CreateTestUser(t, db, "limit456", "limit2@example.com", "Limit User 2")

		// Add feeds up to the limit
		for i := 0; i < services.FreeTrialFeedLimit; i++ {
			feed := helpers.CreateTestFeed(t, db,
				"Test Feed "+string(rune(i+48)), // Simple way to create unique titles
				"http://test"+string(rune(i+48))+".com",
				"Test description")
			err := db.SubscribeUserToFeed(user.ID, feed.ID)
			if err != nil {
				t.Fatalf("Failed to subscribe user to feed: %v", err)
			}
		}

		// Now should hit the limit
		err := subscriptionService.CanUserAddFeed(user.ID)
		if err == nil {
			t.Error("Expected trial user to hit feed limit when subscription enabled")
		}
		if err != services.ErrFeedLimitReached {
			t.Errorf("Expected ErrFeedLimitReached, got: %v", err)
		}
	})
}
