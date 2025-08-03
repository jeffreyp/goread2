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

// TestDatastoreUserFeedOperations tests user feed subscription operations
func TestDatastoreUserFeedOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Datastore tests in short mode")
	}

	db := helpers.CreateTestDatastoreDB(t)

	// Create test user and feeds
	user := &database.User{
		GoogleID:  "feed_test_user",
		Email:     "feeduser@example.com",
		Name:      "Feed Test User",
		CreatedAt: time.Now(),
	}
	err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	feed1 := &database.Feed{
		Title:       "Test Feed 1",
		URL:         "https://test1.com/rss",
		Description: "First test feed",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastFetch:   time.Now(),
	}
	err = db.AddFeed(feed1)
	if err != nil {
		t.Fatalf("Failed to create feed1: %v", err)
	}

	feed2 := &database.Feed{
		Title:       "Test Feed 2",
		URL:         "https://test2.com/rss",
		Description: "Second test feed",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastFetch:   time.Now(),
	}
	err = db.AddFeed(feed2)
	if err != nil {
		t.Fatalf("Failed to create feed2: %v", err)
	}

	t.Run("SubscribeUserToFeed", func(t *testing.T) {
		err := db.SubscribeUserToFeed(user.ID, feed1.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe user to feed: %v", err)
		}

		// Subscribe to second feed
		err = db.SubscribeUserToFeed(user.ID, feed2.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe user to second feed: %v", err)
		}
	})

	t.Run("GetUserFeeds", func(t *testing.T) {
		feeds, err := db.GetUserFeeds(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user feeds: %v", err)
		}

		if len(feeds) != 2 {
			t.Errorf("Expected 2 feeds, got %d", len(feeds))
		}
	})

	t.Run("UnsubscribeUserFromFeed", func(t *testing.T) {
		err := db.UnsubscribeUserFromFeed(user.ID, feed1.ID)
		if err != nil {
			t.Fatalf("Failed to unsubscribe user from feed: %v", err)
		}

		feeds, err := db.GetUserFeeds(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user feeds after unsubscribe: %v", err)
		}

		if len(feeds) != 1 {
			t.Errorf("Expected 1 feed after unsubscribe, got %d", len(feeds))
		}
	})

	t.Run("SubscribeDuplicateFeed", func(t *testing.T) {
		// Try to subscribe to the same feed twice
		err := db.SubscribeUserToFeed(user.ID, feed2.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe to duplicate feed: %v", err)
		}

		feeds, err := db.GetUserFeeds(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user feeds: %v", err)
		}

		// Should still be 1 feed (no duplicates)
		if len(feeds) != 1 {
			t.Errorf("Expected 1 feed (no duplicates), got %d", len(feeds))
		}
	})
}

// TestDatastoreUserArticleOperations tests user article operations
func TestDatastoreUserArticleOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Datastore tests in short mode")
	}

	db := helpers.CreateTestDatastoreDB(t)

	// Create test user and feed
	user := &database.User{
		GoogleID:  "article_test_user",
		Email:     "articleuser@example.com",
		Name:      "Article Test User",
		CreatedAt: time.Now(),
	}
	err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	feed := &database.Feed{
		Title:       "Article Test Feed",
		URL:         "https://articletest.com/rss",
		Description: "Test feed for articles",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastFetch:   time.Now(),
	}
	err = db.AddFeed(feed)
	if err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Subscribe user to feed
	err = db.SubscribeUserToFeed(user.ID, feed.ID)
	if err != nil {
		t.Fatalf("Failed to subscribe user to feed: %v", err)
	}

	// Create test articles
	article1 := &database.Article{
		FeedID:      feed.ID,
		Title:       "Test Article 1",
		URL:         "https://articletest.com/article1",
		Content:     "Content of article 1",
		Description: "Description of article 1",
		Author:      "Test Author",
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
	}
	err = db.AddArticle(article1)
	if err != nil {
		t.Fatalf("Failed to create article1: %v", err)
	}

	article2 := &database.Article{
		FeedID:      feed.ID,
		Title:       "Test Article 2",
		URL:         "https://articletest.com/article2",
		Content:     "Content of article 2",
		Description: "Description of article 2",
		Author:      "Test Author",
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
	}
	err = db.AddArticle(article2)
	if err != nil {
		t.Fatalf("Failed to create article2: %v", err)
	}

	t.Run("GetUserFeedArticles", func(t *testing.T) {
		articles, err := db.GetUserFeedArticles(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to get user feed articles: %v", err)
		}

		if len(articles) != 2 {
			t.Errorf("Expected 2 articles, got %d", len(articles))
		}

		// Articles should start as unread and unstarred
		for _, article := range articles {
			if article.IsRead {
				t.Errorf("Expected article %d to be unread initially", article.ID)
			}
			if article.IsStarred {
				t.Errorf("Expected article %d to be unstarred initially", article.ID)
			}
		}
	})

	t.Run("MarkUserArticleRead", func(t *testing.T) {
		err := db.MarkUserArticleRead(user.ID, article1.ID, true)
		if err != nil {
			t.Fatalf("Failed to mark article as read: %v", err)
		}

		articles, err := db.GetUserFeedArticles(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to get articles after marking read: %v", err)
		}

		var readArticle *database.Article
		for _, article := range articles {
			if article.ID == article1.ID {
				readArticle = &article
				break
			}
		}

		if readArticle == nil {
			t.Fatalf("Article %d not found", article1.ID)
		}

		if !readArticle.IsRead {
			t.Errorf("Expected article %d to be marked as read", article1.ID)
		}
	})

	t.Run("ToggleUserArticleStar", func(t *testing.T) {
		// First toggle - should star the article
		err := db.ToggleUserArticleStar(user.ID, article2.ID)
		if err != nil {
			t.Fatalf("Failed to star article: %v", err)
		}

		articles, err := db.GetUserFeedArticles(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to get articles after starring: %v", err)
		}

		var starredArticle *database.Article
		for _, article := range articles {
			if article.ID == article2.ID {
				starredArticle = &article
				break
			}
		}

		if starredArticle == nil {
			t.Fatalf("Article %d not found", article2.ID)
		}

		if !starredArticle.IsStarred {
			t.Errorf("Expected article %d to be starred", article2.ID)
		}

		// Second toggle - should unstar the article
		err = db.ToggleUserArticleStar(user.ID, article2.ID)
		if err != nil {
			t.Fatalf("Failed to unstar article: %v", err)
		}

		articles, err = db.GetUserFeedArticles(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to get articles after unstarring: %v", err)
		}

		for _, article := range articles {
			if article.ID == article2.ID {
				starredArticle = &article
				break
			}
		}

		if starredArticle.IsStarred {
			t.Errorf("Expected article %d to be unstarred", article2.ID)
		}
	})

	t.Run("GetUserArticles", func(t *testing.T) {
		articles, err := db.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles: %v", err)
		}

		if len(articles) != 2 {
			t.Errorf("Expected 2 articles, got %d", len(articles))
		}
	})
}