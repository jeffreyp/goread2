package database

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Test helpers

func setupTestDB(t *testing.T) *DB {
	t.Helper()

	// Create in-memory database with shared cache for concurrent access
	// The ?cache=shared parameter allows multiple connections to access the same in-memory database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Enable foreign key constraints (required for CASCADE and foreign key tests)
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	dbWrapper := &DB{db}
	if err := dbWrapper.CreateTables(); err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Cleanup database connection after test
	t.Cleanup(func() {
		_ = dbWrapper.Close()
	})

	return dbWrapper
}

func createTestUser(t *testing.T, db *DB) *User {
	t.Helper()

	user := &User{
		GoogleID:            fmt.Sprintf("test_google_id_%d", time.Now().UnixNano()),
		Email:               fmt.Sprintf("test%d@example.com", time.Now().UnixNano()),
		Name:                "Test User",
		Avatar:              "https://example.com/avatar.jpg",
		CreatedAt:           time.Now(),
		SubscriptionStatus:  "trial",
		TrialEndsAt:         time.Now().AddDate(0, 0, 30),
		MaxArticlesOnFeedAdd: 100,
	}

	err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

func createTestFeed(t *testing.T, db *DB) *Feed {
	t.Helper()

	feed := &Feed{
		Title:       "Test Feed",
		URL:         fmt.Sprintf("https://example.com/feed_%d.xml", time.Now().UnixNano()),
		Description: "A test feed",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastFetch:   time.Now(),
	}

	err := db.AddFeed(feed)
	if err != nil {
		t.Fatalf("Failed to create test feed: %v", err)
	}

	return feed
}

func createTestArticle(t *testing.T, db *DB, feedID int) *Article {
	t.Helper()

	article := &Article{
		FeedID:      feedID,
		Title:       "Test Article",
		URL:         fmt.Sprintf("https://example.com/article_%d", time.Now().UnixNano()),
		Content:     "This is a test article content",
		Description: "Test article description",
		Author:      "Test Author",
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	err := db.AddArticle(article)
	if err != nil {
		t.Fatalf("Failed to create test article: %v", err)
	}

	return article
}

// Schema and migration tests

func TestCreateTables(t *testing.T) {
	db := setupTestDB(t)

	// Verify tables exist by querying them
	tables := []string{"users", "feeds", "articles", "user_feeds", "user_articles", "sessions", "admin_tokens", "audit_logs"}

	for _, table := range tables {
		query := fmt.Sprintf("SELECT name FROM sqlite_master WHERE type='table' AND name='%s'", table)
		var name string
		err := db.QueryRow(query).Scan(&name)
		if err == sql.ErrNoRows {
			t.Errorf("Table %s was not created", table)
		} else if err != nil {
			t.Errorf("Error checking table %s: %v", table, err)
		}
	}
}

func TestCreateIndexes(t *testing.T) {
	db := setupTestDB(t)

	// Verify indexes exist
	query := "SELECT name FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%'"
	rows, err := db.Query(query)
	if err != nil {
		t.Fatalf("Failed to query indexes: %v", err)
	}
	defer func() { _ = rows.Close() }()

	indexCount := 0
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Errorf("Failed to scan index name: %v", err)
		}
		indexCount++
	}

	// We expect at least 10 indexes based on CreateIndexes()
	if indexCount < 10 {
		t.Errorf("Expected at least 10 indexes, got %d", indexCount)
	}
}

func TestMigrateDatabase(t *testing.T) {
	db := setupTestDB(t)

	// Run migration (it should be idempotent)
	err := db.migrateDatabase()
	if err != nil {
		t.Errorf("Migration failed: %v", err)
	}

	// Run again to verify idempotency
	err = db.migrateDatabase()
	if err != nil {
		t.Errorf("Second migration failed (not idempotent): %v", err)
	}
}

// User CRUD tests

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)

	user := &User{
		GoogleID:             "test_google_123",
		Email:                "test@example.com",
		Name:                 "Test User",
		Avatar:               "https://example.com/avatar.jpg",
		CreatedAt:            time.Now(),
		MaxArticlesOnFeedAdd: 100,
	}

	err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.ID == 0 {
		t.Error("User ID was not set after creation")
	}

	// Verify default values
	if user.SubscriptionStatus != "trial" {
		t.Errorf("Expected default subscription status 'trial', got '%s'", user.SubscriptionStatus)
	}

	if user.TrialEndsAt.IsZero() {
		t.Error("TrialEndsAt was not set")
	}

	if user.MaxArticlesOnFeedAdd != 100 {
		t.Errorf("Expected MaxArticlesOnFeedAdd 100, got %d", user.MaxArticlesOnFeedAdd)
	}
}

func TestCreateUserDuplicateGoogleID(t *testing.T) {
	db := setupTestDB(t)

	user1 := &User{
		GoogleID:  "duplicate_google_123",
		Email:     "test1@example.com",
		Name:      "Test User 1",
		CreatedAt: time.Now(),
	}

	err := db.CreateUser(user1)
	if err != nil {
		t.Fatalf("First CreateUser failed: %v", err)
	}

	user2 := &User{
		GoogleID:  "duplicate_google_123",
		Email:     "test2@example.com",
		Name:      "Test User 2",
		CreatedAt: time.Now(),
	}

	err = db.CreateUser(user2)
	if err == nil {
		t.Error("Expected error when creating user with duplicate GoogleID, got nil")
	}
}

func TestGetUserByGoogleID(t *testing.T) {
	db := setupTestDB(t)

	originalUser := createTestUser(t, db)

	retrievedUser, err := db.GetUserByGoogleID(originalUser.GoogleID)
	if err != nil {
		t.Fatalf("GetUserByGoogleID failed: %v", err)
	}

	if retrievedUser.ID != originalUser.ID {
		t.Errorf("Expected user ID %d, got %d", originalUser.ID, retrievedUser.ID)
	}

	if retrievedUser.Email != originalUser.Email {
		t.Errorf("Expected email %s, got %s", originalUser.Email, retrievedUser.Email)
	}

	if retrievedUser.Name != originalUser.Name {
		t.Errorf("Expected name %s, got %s", originalUser.Name, retrievedUser.Name)
	}
}

func TestGetUserByGoogleIDNotFound(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.GetUserByGoogleID("nonexistent_google_id")
	if err == nil {
		t.Error("Expected error for nonexistent user, got nil")
	}
}

func TestGetUserByID(t *testing.T) {
	db := setupTestDB(t)

	originalUser := createTestUser(t, db)

	retrievedUser, err := db.GetUserByID(originalUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	if retrievedUser.GoogleID != originalUser.GoogleID {
		t.Errorf("Expected GoogleID %s, got %s", originalUser.GoogleID, retrievedUser.GoogleID)
	}

	if retrievedUser.Email != originalUser.Email {
		t.Errorf("Expected email %s, got %s", originalUser.Email, retrievedUser.Email)
	}
}

func TestGetUserByEmail(t *testing.T) {
	db := setupTestDB(t)

	originalUser := createTestUser(t, db)

	retrievedUser, err := db.GetUserByEmail(originalUser.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}

	if retrievedUser.ID != originalUser.ID {
		t.Errorf("Expected user ID %d, got %d", originalUser.ID, retrievedUser.ID)
	}
}

func TestUpdateUserSubscription(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	newStatus := "active"
	newSubscriptionID := "sub_123456"
	newLastPayment := time.Now()
	newNextBilling := time.Now().AddDate(0, 1, 0)

	err := db.UpdateUserSubscription(user.ID, newStatus, newSubscriptionID, newLastPayment, newNextBilling)
	if err != nil {
		t.Fatalf("UpdateUserSubscription failed: %v", err)
	}

	// Verify update
	updatedUser, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated user: %v", err)
	}

	if updatedUser.SubscriptionStatus != newStatus {
		t.Errorf("Expected status %s, got %s", newStatus, updatedUser.SubscriptionStatus)
	}

	if updatedUser.SubscriptionID != newSubscriptionID {
		t.Errorf("Expected subscription ID %s, got %s", newSubscriptionID, updatedUser.SubscriptionID)
	}
}

func TestIsUserSubscriptionActive(t *testing.T) {
	db := setupTestDB(t)

	tests := []struct {
		name           string
		setupUser      func() *User
		expectedActive bool
	}{
		{
			name: "Active subscription",
			setupUser: func() *User {
				user := createTestUser(t, db)
				err := db.UpdateUserSubscription(user.ID, "active", "sub_123", time.Now(), time.Now().AddDate(0, 1, 0))
				if err != nil {
					t.Fatalf("UpdateUserSubscription failed: %v", err)
				}
				return user
			},
			expectedActive: true,
		},
		{
			name: "Valid trial",
			setupUser: func() *User {
				return createTestUser(t, db) // Default trial is 30 days
			},
			expectedActive: true,
		},
		{
			name: "Expired trial",
			setupUser: func() *User {
				user := createTestUser(t, db)
				// Manually update the trial_ends_at to yesterday
				query := `UPDATE users SET trial_ends_at = ?, subscription_status = 'trial' WHERE id = ?`
				expiredDate := time.Now().AddDate(0, 0, -1)
				_, err := db.Exec(query, expiredDate, user.ID)
				if err != nil {
					t.Fatalf("Failed to set expired trial: %v", err)
				}
				return user
			},
			expectedActive: false,
		},
		{
			name: "Admin user",
			setupUser: func() *User {
				user := createTestUser(t, db)
				err := db.SetUserAdmin(user.ID, true)
				if err != nil {
					t.Fatalf("SetUserAdmin failed: %v", err)
				}
				return user
			},
			expectedActive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := tt.setupUser()

			active, err := db.IsUserSubscriptionActive(user.ID)
			if err != nil {
				t.Fatalf("IsUserSubscriptionActive failed: %v", err)
			}

			if active != tt.expectedActive {
				t.Errorf("Expected active=%v, got %v", tt.expectedActive, active)
			}
		})
	}
}

func TestSetUserAdmin(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	// Set as admin
	err := db.SetUserAdmin(user.ID, true)
	if err != nil {
		t.Fatalf("SetUserAdmin(true) failed: %v", err)
	}

	// Verify
	updatedUser, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if !updatedUser.IsAdmin {
		t.Error("User should be admin")
	}

	// Unset admin
	err = db.SetUserAdmin(user.ID, false)
	if err != nil {
		t.Fatalf("SetUserAdmin(false) failed: %v", err)
	}

	updatedUser, err = db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if updatedUser.IsAdmin {
		t.Error("User should not be admin")
	}
}

func TestGrantFreeMonths(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	// Grant 3 months
	err := db.GrantFreeMonths(user.ID, 3)
	if err != nil {
		t.Fatalf("GrantFreeMonths failed: %v", err)
	}

	updatedUser, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if updatedUser.FreeMonthsRemaining != 3 {
		t.Errorf("Expected 3 free months, got %d", updatedUser.FreeMonthsRemaining)
	}

	// Grant 2 more months
	err = db.GrantFreeMonths(user.ID, 2)
	if err != nil {
		t.Fatalf("Second GrantFreeMonths failed: %v", err)
	}

	updatedUser, err = db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if updatedUser.FreeMonthsRemaining != 5 {
		t.Errorf("Expected 5 free months total, got %d", updatedUser.FreeMonthsRemaining)
	}
}

func TestGetUserFeedCount(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed1 := createTestFeed(t, db)
	feed2 := createTestFeed(t, db)

	// Initially zero
	count, err := db.GetUserFeedCount(user.ID)
	if err != nil {
		t.Fatalf("GetUserFeedCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 feeds, got %d", count)
	}

	// Subscribe to feeds
	if err := db.SubscribeUserToFeed(user.ID, feed1.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.SubscribeUserToFeed(user.ID, feed2.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	count, err = db.GetUserFeedCount(user.ID)
	if err != nil {
		t.Fatalf("GetUserFeedCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 feeds, got %d", count)
	}
}

func TestUpdateUserMaxArticlesOnFeedAdd(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	err := db.UpdateUserMaxArticlesOnFeedAdd(user.ID, 200)
	if err != nil {
		t.Fatalf("UpdateUserMaxArticlesOnFeedAdd failed: %v", err)
	}

	updatedUser, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if updatedUser.MaxArticlesOnFeedAdd != 200 {
		t.Errorf("Expected max articles 200, got %d", updatedUser.MaxArticlesOnFeedAdd)
	}
}

// Feed CRUD tests

func TestAddFeed(t *testing.T) {
	db := setupTestDB(t)

	feed := &Feed{
		Title:       "Test Feed",
		URL:         "https://example.com/feed.xml",
		Description: "A test feed",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastFetch:   time.Now(),
	}

	err := db.AddFeed(feed)
	if err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}

	if feed.ID == 0 {
		t.Error("Feed ID was not set after creation")
	}
}

func TestAddFeedDuplicateURL(t *testing.T) {
	db := setupTestDB(t)

	feed1 := &Feed{
		Title:       "Test Feed 1",
		URL:         "https://example.com/same-feed.xml",
		Description: "First feed",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastFetch:   time.Now(),
	}

	err := db.AddFeed(feed1)
	if err != nil {
		t.Fatalf("First AddFeed failed: %v", err)
	}

	feed2 := &Feed{
		Title:       "Test Feed 2",
		URL:         "https://example.com/same-feed.xml",
		Description: "Duplicate feed",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastFetch:   time.Now(),
	}

	err = db.AddFeed(feed2)
	if err == nil {
		t.Error("Expected error when adding feed with duplicate URL, got nil")
	}
}

func TestGetFeeds(t *testing.T) {
	db := setupTestDB(t)

	feed1 := createTestFeed(t, db)
	feed2 := createTestFeed(t, db)

	feeds, err := db.GetFeeds()
	if err != nil {
		t.Fatalf("GetFeeds failed: %v", err)
	}

	if len(feeds) < 2 {
		t.Errorf("Expected at least 2 feeds, got %d", len(feeds))
	}

	// Verify our feeds are in the result
	found1, found2 := false, false
	for _, f := range feeds {
		if f.ID == feed1.ID {
			found1 = true
		}
		if f.ID == feed2.ID {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("Not all created feeds were returned by GetFeeds")
	}
}

func TestGetFeedByURL(t *testing.T) {
	db := setupTestDB(t)

	originalFeed := createTestFeed(t, db)

	retrievedFeed, err := db.GetFeedByURL(originalFeed.URL)
	if err != nil {
		t.Fatalf("GetFeedByURL failed: %v", err)
	}

	if retrievedFeed == nil {
		t.Fatal("GetFeedByURL returned nil")
	}

	if retrievedFeed.ID != originalFeed.ID {
		t.Errorf("Expected feed ID %d, got %d", originalFeed.ID, retrievedFeed.ID)
	}

	if retrievedFeed.Title != originalFeed.Title {
		t.Errorf("Expected title %s, got %s", originalFeed.Title, retrievedFeed.Title)
	}
}

func TestGetFeedByURLNotFound(t *testing.T) {
	db := setupTestDB(t)

	feed, err := db.GetFeedByURL("https://nonexistent.com/feed.xml")
	if err != nil {
		t.Fatalf("GetFeedByURL failed with error: %v", err)
	}

	if feed != nil {
		t.Error("Expected nil for nonexistent feed, got a feed")
	}
}

func TestUpdateFeed(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)

	// Update feed
	feed.Title = "Updated Feed Title"
	feed.Description = "Updated description"

	err := db.UpdateFeed(feed)
	if err != nil {
		t.Fatalf("UpdateFeed failed: %v", err)
	}

	// Verify update
	updatedFeed, err := db.GetFeedByURL(feed.URL)
	if err != nil {
		t.Fatalf("Failed to retrieve updated feed: %v", err)
	}

	if updatedFeed.Title != "Updated Feed Title" {
		t.Errorf("Expected title 'Updated Feed Title', got '%s'", updatedFeed.Title)
	}

	if updatedFeed.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", updatedFeed.Description)
	}
}

func TestUpdateFeedLastFetch(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)

	newLastFetch := time.Now().Add(1 * time.Hour)

	err := db.UpdateFeedLastFetch(feed.ID, newLastFetch)
	if err != nil {
		t.Fatalf("UpdateFeedLastFetch failed: %v", err)
	}

	// Verify update
	updatedFeed, err := db.GetFeedByURL(feed.URL)
	if err != nil {
		t.Fatalf("Failed to retrieve updated feed: %v", err)
	}

	// Compare timestamps (allowing for small differences due to database precision)
	if updatedFeed.LastFetch.Sub(newLastFetch).Abs() > time.Second {
		t.Errorf("LastFetch not updated correctly: expected %v, got %v", newLastFetch, updatedFeed.LastFetch)
	}
}

func TestDeleteFeed(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)

	err := db.DeleteFeed(feed.ID)
	if err != nil {
		t.Fatalf("DeleteFeed failed: %v", err)
	}

	// Verify deletion
	deletedFeed, err := db.GetFeedByURL(feed.URL)
	if err != nil {
		t.Fatalf("Error checking deleted feed: %v", err)
	}

	if deletedFeed != nil {
		t.Error("Feed should have been deleted")
	}
}

// Article CRUD tests

func TestAddArticle(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)

	article := &Article{
		FeedID:      feed.ID,
		Title:       "Test Article",
		URL:         "https://example.com/article1",
		Content:     "Article content here",
		Description: "Article description",
		Author:      "Test Author",
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	err := db.AddArticle(article)
	if err != nil {
		t.Fatalf("AddArticle failed: %v", err)
	}

	if article.ID == 0 {
		t.Error("Article ID was not set after creation")
	}
}

func TestAddArticleDuplicateURL(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)

	article1 := &Article{
		FeedID:      feed.ID,
		Title:       "Article 1",
		URL:         "https://example.com/same-article",
		Content:     "Content 1",
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	err := db.AddArticle(article1)
	if err != nil {
		t.Fatalf("First AddArticle failed: %v", err)
	}

	article2 := &Article{
		FeedID:      feed.ID,
		Title:       "Article 2",
		URL:         "https://example.com/same-article",
		Content:     "Content 2",
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	err = db.AddArticle(article2)
	// Should not error due to INSERT OR IGNORE, but ID should be set to existing article
	if err != nil {
		t.Fatalf("Second AddArticle failed: %v", err)
	}

	if article2.ID == 0 {
		t.Error("Duplicate article ID should still be set to existing article ID")
	}
}

func TestGetArticles(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)
	article1 := createTestArticle(t, db, feed.ID)
	article2 := createTestArticle(t, db, feed.ID)

	articles, err := db.GetArticles(feed.ID)
	if err != nil {
		t.Fatalf("GetArticles failed: %v", err)
	}

	if len(articles) < 2 {
		t.Errorf("Expected at least 2 articles, got %d", len(articles))
	}

	// Verify our articles are in the result
	found1, found2 := false, false
	for _, a := range articles {
		if a.ID == article1.ID {
			found1 = true
		}
		if a.ID == article2.ID {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("Not all created articles were returned by GetArticles")
	}
}

func TestFindArticleByURL(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)
	originalArticle := createTestArticle(t, db, feed.ID)

	foundArticle, err := db.FindArticleByURL(originalArticle.URL)
	if err != nil {
		t.Fatalf("FindArticleByURL failed: %v", err)
	}

	if foundArticle == nil {
		t.Fatal("FindArticleByURL returned nil")
	}

	if foundArticle.ID != originalArticle.ID {
		t.Errorf("Expected article ID %d, got %d", originalArticle.ID, foundArticle.ID)
	}

	if foundArticle.Title != originalArticle.Title {
		t.Errorf("Expected title %s, got %s", originalArticle.Title, foundArticle.Title)
	}
}

func TestFindArticleByURLNotFound(t *testing.T) {
	db := setupTestDB(t)

	article, err := db.FindArticleByURL("https://nonexistent.com/article")
	if err != nil {
		t.Fatalf("FindArticleByURL failed with error: %v", err)
	}

	if article != nil {
		t.Error("Expected nil for nonexistent article, got an article")
	}
}

func TestGetAllArticles(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)
	createTestArticle(t, db, feed.ID)
	createTestArticle(t, db, feed.ID)

	articles, err := db.GetAllArticles()
	if err != nil {
		t.Fatalf("GetAllArticles failed: %v", err)
	}

	if len(articles) < 2 {
		t.Errorf("Expected at least 2 articles, got %d", len(articles))
	}
}

// User-feed subscription tests

func TestSubscribeUserToFeed(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	err := db.SubscribeUserToFeed(user.ID, feed.ID)
	if err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// Verify subscription
	userFeeds, err := db.GetUserFeeds(user.ID)
	if err != nil {
		t.Fatalf("GetUserFeeds failed: %v", err)
	}

	if len(userFeeds) != 1 {
		t.Errorf("Expected 1 feed, got %d", len(userFeeds))
	}

	if userFeeds[0].ID != feed.ID {
		t.Errorf("Expected feed ID %d, got %d", feed.ID, userFeeds[0].ID)
	}
}

func TestSubscribeUserToFeedDuplicate(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	// Subscribe twice
	err := db.SubscribeUserToFeed(user.ID, feed.ID)
	if err != nil {
		t.Fatalf("First SubscribeUserToFeed failed: %v", err)
	}

	err = db.SubscribeUserToFeed(user.ID, feed.ID)
	if err != nil {
		t.Fatalf("Second SubscribeUserToFeed failed (should be ignored): %v", err)
	}

	// Should still only have one subscription
	userFeeds, err := db.GetUserFeeds(user.ID)
	if err != nil {
		t.Fatalf("GetUserFeeds failed: %v", err)
	}

	if len(userFeeds) != 1 {
		t.Errorf("Expected 1 feed, got %d (duplicate not ignored)", len(userFeeds))
	}
}

func TestUnsubscribeUserFromFeed(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	// Subscribe first
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// Create an article and mark it read
	article := createTestArticle(t, db, feed.ID)
	if err := db.MarkUserArticleRead(user.ID, article.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	// Unsubscribe
	err := db.UnsubscribeUserFromFeed(user.ID, feed.ID)
	if err != nil {
		t.Fatalf("UnsubscribeUserFromFeed failed: %v", err)
	}

	// Verify unsubscription
	userFeeds, err := db.GetUserFeeds(user.ID)
	if err != nil {
		t.Fatalf("GetUserFeeds failed: %v", err)
	}

	if len(userFeeds) != 0 {
		t.Errorf("Expected 0 feeds, got %d", len(userFeeds))
	}

	// Verify user article status was also cleaned up
	status, err := db.GetUserArticleStatus(user.ID, article.ID)
	if err == nil || status != nil {
		t.Error("User article status should have been deleted with unsubscription")
	}
}

func TestGetUserFeeds(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed1 := createTestFeed(t, db)
	feed2 := createTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed1.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.SubscribeUserToFeed(user.ID, feed2.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	userFeeds, err := db.GetUserFeeds(user.ID)
	if err != nil {
		t.Fatalf("GetUserFeeds failed: %v", err)
	}

	if len(userFeeds) != 2 {
		t.Errorf("Expected 2 feeds, got %d", len(userFeeds))
	}
}

func TestGetAllUserFeeds(t *testing.T) {
	db := setupTestDB(t)

	user1 := createTestUser(t, db)
	user2 := createTestUser(t, db)
	feed1 := createTestFeed(t, db)
	feed2 := createTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user1.ID, feed1.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.SubscribeUserToFeed(user2.ID, feed2.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	allFeeds, err := db.GetAllUserFeeds()
	if err != nil {
		t.Fatalf("GetAllUserFeeds failed: %v", err)
	}

	if len(allFeeds) < 2 {
		t.Errorf("Expected at least 2 feeds, got %d", len(allFeeds))
	}
}

// Audit Log tests

func TestCreateAuditLog(t *testing.T) {
	db := setupTestDB(t)

	log := &AuditLog{
		Timestamp:        time.Now(),
		AdminUserID:      1,
		AdminEmail:       "admin@example.com",
		OperationType:    "grant_admin",
		TargetUserID:     2,
		TargetUserEmail:  "user@example.com",
		OperationDetails: `{"is_admin":true,"user_name":"Test User"}`,
		IPAddress:        "192.168.1.1",
		Result:           "success",
		ErrorMessage:     "",
	}

	err := db.CreateAuditLog(log)
	if err != nil {
		t.Fatalf("CreateAuditLog failed: %v", err)
	}

	if log.ID == 0 {
		t.Error("Audit log ID was not set after creation")
	}
}

func TestCreateAuditLogCLI(t *testing.T) {
	db := setupTestDB(t)

	log := &AuditLog{
		Timestamp:        time.Now(),
		AdminUserID:      0, // CLI uses ID 0
		AdminEmail:       "CLI_ADMIN",
		OperationType:    "set_admin",
		TargetUserID:     5,
		TargetUserEmail:  "user@example.com",
		OperationDetails: `{}`,
		IPAddress:        "CLI",
		Result:           "success",
		ErrorMessage:     "",
	}

	err := db.CreateAuditLog(log)
	if err != nil {
		t.Fatalf("CreateAuditLog for CLI failed: %v", err)
	}

	if log.ID == 0 {
		t.Error("CLI audit log ID was not set after creation")
	}
}

func TestCreateAuditLogFailure(t *testing.T) {
	db := setupTestDB(t)

	log := &AuditLog{
		Timestamp:        time.Now(),
		AdminUserID:      1,
		AdminEmail:       "admin@example.com",
		OperationType:    "grant_free_months",
		TargetUserID:     0,
		TargetUserEmail:  "unknown@example.com",
		OperationDetails: `{"months_granted":6}`,
		IPAddress:        "192.168.1.1",
		Result:           "failure",
		ErrorMessage:     "user not found",
	}

	err := db.CreateAuditLog(log)
	if err != nil {
		t.Fatalf("CreateAuditLog for failure failed: %v", err)
	}

	if log.ID == 0 {
		t.Error("Failure audit log ID was not set after creation")
	}
}

func TestGetAuditLogs(t *testing.T) {
	db := setupTestDB(t)

	// Create multiple test audit logs
	logs := []AuditLog{
		{
			Timestamp:        time.Now().Add(-3 * time.Hour),
			AdminUserID:      1,
			AdminEmail:       "admin@example.com",
			OperationType:    "grant_admin",
			TargetUserID:     2,
			TargetUserEmail:  "user1@example.com",
			OperationDetails: `{}`,
			IPAddress:        "192.168.1.1",
			Result:           "success",
		},
		{
			Timestamp:        time.Now().Add(-2 * time.Hour),
			AdminUserID:      1,
			AdminEmail:       "admin@example.com",
			OperationType:    "grant_free_months",
			TargetUserID:     3,
			TargetUserEmail:  "user2@example.com",
			OperationDetails: `{"months_granted":6}`,
			IPAddress:        "192.168.1.1",
			Result:           "success",
		},
		{
			Timestamp:        time.Now().Add(-1 * time.Hour),
			AdminUserID:      2,
			AdminEmail:       "admin2@example.com",
			OperationType:    "revoke_admin",
			TargetUserID:     4,
			TargetUserEmail:  "user3@example.com",
			OperationDetails: `{"is_admin":false}`,
			IPAddress:        "192.168.1.2",
			Result:           "success",
		},
	}

	for i := range logs {
		if err := db.CreateAuditLog(&logs[i]); err != nil {
			t.Fatalf("Failed to create test audit log: %v", err)
		}
	}

	t.Run("get all logs", func(t *testing.T) {
		retrievedLogs, err := db.GetAuditLogs(50, 0, nil)
		if err != nil {
			t.Fatalf("GetAuditLogs failed: %v", err)
		}

		if len(retrievedLogs) < 3 {
			t.Errorf("Expected at least 3 logs, got %d", len(retrievedLogs))
		}

		// Verify logs are ordered by timestamp DESC (newest first)
		for i := 0; i < len(retrievedLogs)-1; i++ {
			if retrievedLogs[i].Timestamp.Before(retrievedLogs[i+1].Timestamp) {
				t.Error("Logs should be ordered by timestamp DESC (newest first)")
				break
			}
		}
	})

	t.Run("pagination", func(t *testing.T) {
		// Get first 2 logs
		page1, err := db.GetAuditLogs(2, 0, nil)
		if err != nil {
			t.Fatalf("GetAuditLogs page 1 failed: %v", err)
		}

		if len(page1) != 2 {
			t.Errorf("Expected 2 logs in page 1, got %d", len(page1))
		}

		// Get next log
		page2, err := db.GetAuditLogs(2, 2, nil)
		if err != nil {
			t.Fatalf("GetAuditLogs page 2 failed: %v", err)
		}

		if len(page2) < 1 {
			t.Errorf("Expected at least 1 log in page 2, got %d", len(page2))
		}

		// Verify no overlap
		if len(page1) > 0 && len(page2) > 0 {
			if page1[0].ID == page2[0].ID {
				t.Error("Pages should not overlap")
			}
		}
	})

	t.Run("filter by admin_user_id", func(t *testing.T) {
		filters := map[string]interface{}{
			"admin_user_id": 1,
		}

		filteredLogs, err := db.GetAuditLogs(50, 0, filters)
		if err != nil {
			t.Fatalf("GetAuditLogs with filter failed: %v", err)
		}

		if len(filteredLogs) != 2 {
			t.Errorf("Expected 2 logs for admin_user_id=1, got %d", len(filteredLogs))
		}

		for _, log := range filteredLogs {
			if log.AdminUserID != 1 {
				t.Errorf("Expected AdminUserID=1, got %d", log.AdminUserID)
			}
		}
	})

	t.Run("filter by target_user_id", func(t *testing.T) {
		filters := map[string]interface{}{
			"target_user_id": 2,
		}

		filteredLogs, err := db.GetAuditLogs(50, 0, filters)
		if err != nil {
			t.Fatalf("GetAuditLogs with filter failed: %v", err)
		}

		if len(filteredLogs) != 1 {
			t.Errorf("Expected 1 log for target_user_id=2, got %d", len(filteredLogs))
		}

		if len(filteredLogs) > 0 && filteredLogs[0].TargetUserID != 2 {
			t.Errorf("Expected TargetUserID=2, got %d", filteredLogs[0].TargetUserID)
		}
	})

	t.Run("filter by operation_type", func(t *testing.T) {
		filters := map[string]interface{}{
			"operation_type": "grant_free_months",
		}

		filteredLogs, err := db.GetAuditLogs(50, 0, filters)
		if err != nil {
			t.Fatalf("GetAuditLogs with filter failed: %v", err)
		}

		if len(filteredLogs) != 1 {
			t.Errorf("Expected 1 log for operation_type=grant_free_months, got %d", len(filteredLogs))
		}

		if len(filteredLogs) > 0 && filteredLogs[0].OperationType != "grant_free_months" {
			t.Errorf("Expected OperationType=grant_free_months, got %s", filteredLogs[0].OperationType)
		}
	})

	t.Run("filter with multiple criteria", func(t *testing.T) {
		filters := map[string]interface{}{
			"admin_user_id":  1,
			"operation_type": "grant_admin",
		}

		filteredLogs, err := db.GetAuditLogs(50, 0, filters)
		if err != nil {
			t.Fatalf("GetAuditLogs with multiple filters failed: %v", err)
		}

		if len(filteredLogs) != 1 {
			t.Errorf("Expected 1 log with multiple filters, got %d", len(filteredLogs))
		}

		if len(filteredLogs) > 0 {
			log := filteredLogs[0]
			if log.AdminUserID != 1 || log.OperationType != "grant_admin" {
				t.Error("Log doesn't match all filter criteria")
			}
		}
	})

	t.Run("no results for non-matching filter", func(t *testing.T) {
		filters := map[string]interface{}{
			"admin_user_id": 999,
		}

		filteredLogs, err := db.GetAuditLogs(50, 0, filters)
		if err != nil {
			t.Fatalf("GetAuditLogs with non-matching filter failed: %v", err)
		}

		if len(filteredLogs) != 0 {
			t.Errorf("Expected 0 logs for non-matching filter, got %d", len(filteredLogs))
		}
	})
}

func TestAuditLogIndexes(t *testing.T) {
	db := setupTestDB(t)

	// Verify audit log indexes exist
	expectedIndexes := []string{
		"idx_audit_logs_timestamp",
		"idx_audit_logs_admin_user",
		"idx_audit_logs_target_user",
		"idx_audit_logs_operation",
	}

	for _, indexName := range expectedIndexes {
		query := "SELECT name FROM sqlite_master WHERE type='index' AND name=?"
		var name string
		err := db.QueryRow(query, indexName).Scan(&name)
		if err == sql.ErrNoRows {
			t.Errorf("Expected index %s was not created", indexName)
		} else if err != nil {
			t.Errorf("Error checking index %s: %v", indexName, err)
		}
	}
}

func TestAuditLogTableExists(t *testing.T) {
	db := setupTestDB(t)

	// Verify audit_logs table exists
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name='audit_logs'"
	var name string
	err := db.QueryRow(query).Scan(&name)
	if err == sql.ErrNoRows {
		t.Error("audit_logs table was not created")
	} else if err != nil {
		t.Errorf("Error checking audit_logs table: %v", err)
	}
}
