package services

import (
	"errors"
	"os"
	"testing"
	"time"

	"goread2/internal/config"
	"goread2/internal/database"
)

// mockDBForSub implements database.Database interface for subscription service testing
type mockDBForSub struct {
	users               map[int]*database.User
	feedCounts          map[int]int
	subscriptionActive  map[int]bool
	shouldFailGetUser   bool
	shouldFailGetCount  bool
	shouldFailGetActive bool
	shouldFailUpdate    bool
	shouldFailSetAdmin  bool
	shouldFailGrant     bool
}

func newMockDBForSub() *mockDBForSub {
	return &mockDBForSub{
		users:              make(map[int]*database.User),
		feedCounts:         make(map[int]int),
		subscriptionActive: make(map[int]bool),
	}
}

// Add user to mock
func (m *mockDBForSub) addUser(user *database.User) {
	m.users[user.ID] = user
	if m.feedCounts[user.ID] == 0 {
		m.feedCounts[user.ID] = 0
	}
	m.subscriptionActive[user.ID] = user.SubscriptionStatus == "active"
}

// Mock implementations
func (m *mockDBForSub) Close() error                                     { return nil }
func (m *mockDBForSub) CreateUser(*database.User) error                  { return nil }
func (m *mockDBForSub) GetUserByGoogleID(string) (*database.User, error) { return nil, nil }
func (m *mockDBForSub) GetUserByID(userID int) (*database.User, error) {
	if m.shouldFailGetUser {
		return nil, errors.New("failed to get user")
	}
	user, exists := m.users[userID]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}
func (m *mockDBForSub) UpdateUserSubscription(userID int, status, subscriptionID string, lastPaymentDate, nextBillingDate time.Time) error {
	if m.shouldFailUpdate {
		return errors.New("failed to update subscription")
	}
	if user, exists := m.users[userID]; exists {
		user.SubscriptionStatus = status
		user.SubscriptionID = subscriptionID
		user.LastPaymentDate = lastPaymentDate
	}
	return nil
}
func (m *mockDBForSub) IsUserSubscriptionActive(userID int) (bool, error) {
	if m.shouldFailGetActive {
		return false, errors.New("failed to check subscription active")
	}
	return m.subscriptionActive[userID], nil
}
func (m *mockDBForSub) GetUserFeedCount(userID int) (int, error) {
	if m.shouldFailGetCount {
		return 0, errors.New("failed to get feed count")
	}
	return m.feedCounts[userID], nil
}
func (m *mockDBForSub) SetUserAdmin(userID int, isAdmin bool) error {
	if m.shouldFailSetAdmin {
		return errors.New("failed to set admin")
	}
	if user, exists := m.users[userID]; exists {
		user.IsAdmin = isAdmin
	}
	return nil
}
func (m *mockDBForSub) GrantFreeMonths(userID int, months int) error {
	if m.shouldFailGrant {
		return errors.New("failed to grant free months")
	}
	if user, exists := m.users[userID]; exists {
		user.FreeMonthsRemaining = months
	}
	return nil
}
func (m *mockDBForSub) GetUserByEmail(email string) (*database.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

// Stub methods to satisfy interface
func (m *mockDBForSub) AddFeed(*database.Feed) error                    { return nil }
func (m *mockDBForSub) UpdateFeed(*database.Feed) error                 { return nil }
func (m *mockDBForSub) GetFeeds() ([]database.Feed, error)              { return nil, nil }
func (m *mockDBForSub) GetFeedByURL(string) (*database.Feed, error)     { return nil, nil }
func (m *mockDBForSub) GetUserFeeds(int) ([]database.Feed, error)       { return nil, nil }
func (m *mockDBForSub) GetAllUserFeeds() ([]database.Feed, error)       { return nil, nil }
func (m *mockDBForSub) DeleteFeed(int) error                            { return nil }
func (m *mockDBForSub) SubscribeUserToFeed(int, int) error              { return nil }
func (m *mockDBForSub) UnsubscribeUserFromFeed(int, int) error          { return nil }
func (m *mockDBForSub) AddArticle(*database.Article) error              { return nil }
func (m *mockDBForSub) GetArticles(int) ([]database.Article, error)     { return nil, nil }
func (m *mockDBForSub) FindArticleByURL(string) (*database.Article, error) { return nil, nil }
func (m *mockDBForSub) GetUserArticles(int) ([]database.Article, error) { return nil, nil }
func (m *mockDBForSub) GetUserArticlesPaginated(int, int, int) ([]database.Article, error) {
	return nil, nil
}
func (m *mockDBForSub) GetUserFeedArticles(int, int) ([]database.Article, error)     { return nil, nil }
func (m *mockDBForSub) GetUserArticleStatus(int, int) (*database.UserArticle, error) { return nil, nil }
func (m *mockDBForSub) SetUserArticleStatus(int, int, bool, bool) error              { return nil }
func (m *mockDBForSub) BatchSetUserArticleStatus(int, []database.Article, bool, bool) error {
	return nil
}
func (m *mockDBForSub) MarkUserArticleRead(int, int, bool) error     { return nil }
func (m *mockDBForSub) ToggleUserArticleStar(int, int) error         { return nil }
func (m *mockDBForSub) GetUserUnreadCounts(int) (map[int]int, error) { return nil, nil }
func (m *mockDBForSub) GetAllArticles() ([]database.Article, error)  { return nil, nil }
func (m *mockDBForSub) UpdateFeedLastFetch(int, time.Time) error     { return nil }
func (m *mockDBForSub) UpdateUserMaxArticlesOnFeedAdd(int, int) error { return nil }

func TestNewSubscriptionService(t *testing.T) {
	db := newMockDBForSub()
	service := NewSubscriptionService(db)

	if service == nil {
		t.Fatal("NewSubscriptionService returned nil")
	}

	if service.db != db {
		t.Error("SubscriptionService database not set correctly")
	}
}

func TestCanUserAddFeed(t *testing.T) {
	tests := []struct {
		name                string
		subscriptionEnabled bool
		user                *database.User
		feedCount           int
		expectError         bool
		expectedError       error
	}{
		{
			name:                "subscription disabled - always allows",
			subscriptionEnabled: false,
			user:                &database.User{ID: 1, SubscriptionStatus: "trial"},
			feedCount:           100,
			expectError:         false,
		},
		{
			name:                "admin user - unlimited feeds",
			subscriptionEnabled: true,
			user:                &database.User{ID: 1, IsAdmin: true},
			feedCount:           100,
			expectError:         false,
		},
		{
			name:                "active subscription - unlimited feeds",
			subscriptionEnabled: true,
			user:                &database.User{ID: 1, SubscriptionStatus: "active"},
			feedCount:           100,
			expectError:         false,
		},
		{
			name:                "user with free months - unlimited feeds",
			subscriptionEnabled: true,
			user:                &database.User{ID: 1, FreeMonthsRemaining: 3},
			feedCount:           100,
			expectError:         false,
		},
		{
			name:                "trial user - under limit",
			subscriptionEnabled: true,
			user:                &database.User{ID: 1, SubscriptionStatus: "trial", TrialEndsAt: time.Now().Add(24 * time.Hour)},
			feedCount:           10,
			expectError:         false,
		},
		{
			name:                "trial user - at limit",
			subscriptionEnabled: true,
			user:                &database.User{ID: 1, SubscriptionStatus: "trial", TrialEndsAt: time.Now().Add(24 * time.Hour)},
			feedCount:           20,
			expectError:         true,
			expectedError:       ErrFeedLimitReached,
		},
		{
			name:                "trial expired",
			subscriptionEnabled: true,
			user:                &database.User{ID: 1, SubscriptionStatus: "trial", TrialEndsAt: time.Now().Add(-24 * time.Hour)},
			feedCount:           10,
			expectError:         true,
			expectedError:       ErrTrialExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			config.ResetForTesting()
			_ = os.Setenv("SUBSCRIPTION_ENABLED", "false")
			if tt.subscriptionEnabled {
				_ = os.Setenv("SUBSCRIPTION_ENABLED", "true")
			}
			defer func() {
				_ = os.Unsetenv("SUBSCRIPTION_ENABLED")
				config.ResetForTesting()
			}()
			config.Load()

			// Setup mock
			db := newMockDBForSub()
			db.addUser(tt.user)
			db.feedCounts[tt.user.ID] = tt.feedCount

			service := NewSubscriptionService(db)
			err := service.CanUserAddFeed(tt.user.ID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.expectedError != nil && err != tt.expectedError {
					t.Errorf("Expected error %v, got %v", tt.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetUserSubscriptionInfo(t *testing.T) {
	tests := []struct {
		name                string
		subscriptionEnabled bool
		user                *database.User
		feedCount           int
		isActive            bool
		expectedStatus      string
		expectedFeedLimit   int
		expectedCanAdd      bool
	}{
		{
			name:                "subscription disabled",
			subscriptionEnabled: false,
			user:                &database.User{ID: 1, SubscriptionStatus: "trial"},
			feedCount:           10,
			isActive:            true,
			expectedStatus:      "unlimited",
			expectedFeedLimit:   -1,
			expectedCanAdd:      true,
		},
		{
			name:                "admin user",
			subscriptionEnabled: true,
			user:                &database.User{ID: 1, IsAdmin: true},
			feedCount:           10,
			isActive:            true,
			expectedStatus:      "admin_trial",
			expectedFeedLimit:   -1,
			expectedCanAdd:      true,
		},
		{
			name:                "active subscriber",
			subscriptionEnabled: true,
			user:                &database.User{ID: 1, SubscriptionStatus: "active"},
			feedCount:           10,
			isActive:            true,
			expectedStatus:      "active",
			expectedFeedLimit:   -1,
			expectedCanAdd:      true,
		},
		{
			name:                "user with free months",
			subscriptionEnabled: true,
			user:                &database.User{ID: 1, SubscriptionStatus: "trial", FreeMonthsRemaining: 3},
			feedCount:           10,
			isActive:            true,
			expectedStatus:      "free_months",
			expectedFeedLimit:   -1,
			expectedCanAdd:      true,
		},
		{
			name:                "trial user under limit",
			subscriptionEnabled: true,
			user:                &database.User{ID: 1, SubscriptionStatus: "trial", TrialEndsAt: time.Now().Add(5 * 24 * time.Hour)},
			feedCount:           10,
			isActive:            true,
			expectedStatus:      "trial",
			expectedFeedLimit:   20,
			expectedCanAdd:      true,
		},
		{
			name:                "trial user at limit",
			subscriptionEnabled: true,
			user:                &database.User{ID: 1, SubscriptionStatus: "trial", TrialEndsAt: time.Now().Add(5 * 24 * time.Hour)},
			feedCount:           20,
			isActive:            true,
			expectedStatus:      "trial",
			expectedFeedLimit:   20,
			expectedCanAdd:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			config.ResetForTesting()
			_ = os.Setenv("SUBSCRIPTION_ENABLED", "false")
			if tt.subscriptionEnabled {
				_ = os.Setenv("SUBSCRIPTION_ENABLED", "true")
			}
			defer func() {
				_ = os.Unsetenv("SUBSCRIPTION_ENABLED")
				config.ResetForTesting()
			}()
			config.Load()

			// Setup mock
			db := newMockDBForSub()
			db.addUser(tt.user)
			db.feedCounts[tt.user.ID] = tt.feedCount
			db.subscriptionActive[tt.user.ID] = tt.isActive

			service := NewSubscriptionService(db)
			info, err := service.GetUserSubscriptionInfo(tt.user.ID)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if info.Status != tt.expectedStatus {
				t.Errorf("Status = %s, want %s", info.Status, tt.expectedStatus)
			}

			if info.FeedLimit != tt.expectedFeedLimit {
				t.Errorf("FeedLimit = %d, want %d", info.FeedLimit, tt.expectedFeedLimit)
			}

			if info.CanAddFeeds != tt.expectedCanAdd {
				t.Errorf("CanAddFeeds = %v, want %v", info.CanAddFeeds, tt.expectedCanAdd)
			}

			if info.CurrentFeeds != tt.feedCount {
				t.Errorf("CurrentFeeds = %d, want %d", info.CurrentFeeds, tt.feedCount)
			}

			if info.IsActive != tt.isActive {
				t.Errorf("IsActive = %v, want %v", info.IsActive, tt.isActive)
			}

			// Test trial days calculation
			if tt.user.SubscriptionStatus == "trial" && tt.subscriptionEnabled {
				expectedDays := int(time.Until(tt.user.TrialEndsAt).Hours() / 24)
				if expectedDays < 0 {
					expectedDays = 0
				}
				if info.TrialDaysRemaining != expectedDays {
					t.Errorf("TrialDaysRemaining = %d, want %d", info.TrialDaysRemaining, expectedDays)
				}
			}
		})
	}
}

func TestUpdateUserSubscription(t *testing.T) {
	db := newMockDBForSub()
	user := &database.User{ID: 1, SubscriptionStatus: "trial"}
	db.addUser(user)

	service := NewSubscriptionService(db)

	testTime := time.Now()
	nextBillingTime := testTime.AddDate(0, 1, 0) // 1 month from now
	err := service.UpdateUserSubscription(1, "active", "sub_123", testTime, nextBillingTime)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify the update
	updatedUser := db.users[1]
	if updatedUser.SubscriptionStatus != "active" {
		t.Errorf("SubscriptionStatus = %s, want active", updatedUser.SubscriptionStatus)
	}
	if updatedUser.SubscriptionID != "sub_123" {
		t.Errorf("SubscriptionID = %s, want sub_123", updatedUser.SubscriptionID)
	}
	if !updatedUser.LastPaymentDate.Equal(testTime) {
		t.Errorf("LastPaymentDate = %v, want %v", updatedUser.LastPaymentDate, testTime)
	}
}

func TestSetUserAdmin(t *testing.T) {
	db := newMockDBForSub()
	user := &database.User{ID: 1, IsAdmin: false}
	db.addUser(user)

	service := NewSubscriptionService(db)

	err := service.SetUserAdmin(1, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify admin status
	if !db.users[1].IsAdmin {
		t.Error("User should be admin")
	}
}

func TestGrantFreeMonths(t *testing.T) {
	db := newMockDBForSub()
	user := &database.User{ID: 1, FreeMonthsRemaining: 0}
	db.addUser(user)

	service := NewSubscriptionService(db)

	err := service.GrantFreeMonths(1, 3)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify free months
	if db.users[1].FreeMonthsRemaining != 3 {
		t.Errorf("FreeMonthsRemaining = %d, want 3", db.users[1].FreeMonthsRemaining)
	}
}

func TestGetUserByEmail(t *testing.T) {
	db := newMockDBForSub()
	user := &database.User{ID: 1, Email: "test@example.com"}
	db.addUser(user)

	service := NewSubscriptionService(db)

	foundUser, err := service.GetUserByEmail("test@example.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if foundUser.ID != 1 {
		t.Errorf("User ID = %d, want 1", foundUser.ID)
	}
	if foundUser.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", foundUser.Email)
	}
}

func TestErrorHandling(t *testing.T) {
	t.Run("CanUserAddFeed with database error", func(t *testing.T) {
		// Enable subscriptions so we hit the database call
		config.ResetForTesting()
		_ = os.Setenv("SUBSCRIPTION_ENABLED", "true")
		defer func() {
			_ = os.Unsetenv("SUBSCRIPTION_ENABLED")
			config.ResetForTesting()
		}()
		config.Load()

		db := newMockDBForSub()
		db.shouldFailGetUser = true

		service := NewSubscriptionService(db)
		err := service.CanUserAddFeed(1)

		if err == nil {
			t.Error("Expected error but got none")
		}
	})

	t.Run("GetUserSubscriptionInfo with get user error", func(t *testing.T) {
		db := newMockDBForSub()
		db.shouldFailGetUser = true

		service := NewSubscriptionService(db)
		_, err := service.GetUserSubscriptionInfo(1)

		if err == nil {
			t.Error("Expected error but got none")
		}
	})

	t.Run("GetUserSubscriptionInfo with get count error", func(t *testing.T) {
		db := newMockDBForSub()
		user := &database.User{ID: 1}
		db.addUser(user)
		db.shouldFailGetCount = true

		service := NewSubscriptionService(db)
		_, err := service.GetUserSubscriptionInfo(1)

		if err == nil {
			t.Error("Expected error but got none")
		}
	})

	t.Run("UpdateUserSubscription with error", func(t *testing.T) {
		db := newMockDBForSub()
		db.shouldFailUpdate = true

		service := NewSubscriptionService(db)
		err := service.UpdateUserSubscription(1, "active", "sub_123", time.Now(), time.Now().AddDate(0, 1, 0))

		if err == nil {
			t.Error("Expected error but got none")
		}
	})
}

func TestNewFeedService(t *testing.T) {
	db := newMockDBForSub()
	rateLimiter := NewDomainRateLimiter(RateLimiterConfig{})
	service := NewFeedService(db, rateLimiter)

	if service == nil {
		t.Error("NewFeedService returned nil")
	}

	// Check if the database was set (we can't directly access private fields,
	// but we can test that the service was created without panicking)
}

func TestNewPaymentService(t *testing.T) {
	db := newMockDBForSub()
	subscriptionService := NewSubscriptionService(db)
	service := NewPaymentService(db, subscriptionService)

	if service == nil {
		t.Error("NewPaymentService returned nil")
	}
}
