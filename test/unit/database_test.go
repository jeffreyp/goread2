package unit

import (
	"testing"
	"time"

	"goread2/internal/database"
	"goread2/test/helpers"
)

func TestUserOperations(t *testing.T) {
	db := helpers.CreateTestDB(t)

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
		// Create a user first
		originalUser := helpers.CreateTestUser(t, db, "google456", "test2@example.com", "Test User 2")

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
	})

	t.Run("GetUserByID", func(t *testing.T) {
		// Create a user first
		originalUser := helpers.CreateTestUser(t, db, "google789", "test3@example.com", "Test User 3")

		// Retrieve the user
		retrievedUser, err := db.GetUserByID(originalUser.ID)
		if err != nil {
			t.Fatalf("Failed to get user by ID: %v", err)
		}

		if retrievedUser.GoogleID != originalUser.GoogleID {
			t.Errorf("Expected Google ID %s, got %s", originalUser.GoogleID, retrievedUser.GoogleID)
		}
	})
}

func TestFeedOperations(t *testing.T) {
	db := helpers.CreateTestDB(t)

	t.Run("AddFeed", func(t *testing.T) {
		feed := &database.Feed{
			Title:       "Test Feed",
			URL:         "https://example.com/feed.xml",
			Description: "A test RSS feed",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			LastFetch:   time.Now(),
		}

		err := db.AddFeed(feed)
		if err != nil {
			t.Fatalf("Failed to add feed: %v", err)
		}

		if feed.ID == 0 {
			t.Error("Expected feed ID to be set after creation")
		}
	})

	t.Run("GetFeeds", func(t *testing.T) {
		// Create a few feeds
		helpers.CreateTestFeed(t, db, "Feed 1", "https://feed1.com/rss", "First feed")
		helpers.CreateTestFeed(t, db, "Feed 2", "https://feed2.com/rss", "Second feed")

		feeds, err := db.GetFeeds()
		if err != nil {
			t.Fatalf("Failed to get feeds: %v", err)
		}

		if len(feeds) < 2 {
			t.Errorf("Expected at least 2 feeds, got %d", len(feeds))
		}
	})
}

func TestUserFeedOperations(t *testing.T) {
	db := helpers.CreateTestDB(t)

	// Create test data
	user := helpers.CreateTestUser(t, db, "user123", "user@example.com", "Test User")
	feed1 := helpers.CreateTestFeed(t, db, "Feed 1", "https://feed1.com/rss", "First feed")
	feed2 := helpers.CreateTestFeed(t, db, "Feed 2", "https://feed2.com/rss", "Second feed")

	t.Run("SubscribeUserToFeed", func(t *testing.T) {
		err := db.SubscribeUserToFeed(user.ID, feed1.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe user to feed: %v", err)
		}

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
}

func TestArticleOperations(t *testing.T) {
	db := helpers.CreateTestDB(t)

	// Create test data
	user := helpers.CreateTestUser(t, db, "user456", "user2@example.com", "Test User 2")
	feed := helpers.CreateTestFeed(t, db, "Test Feed", "https://test.com/rss", "Test feed")

	// Subscribe user to feed
	err := db.SubscribeUserToFeed(user.ID, feed.ID)
	if err != nil {
		t.Fatalf("Failed to subscribe user to feed: %v", err)
	}

	// Create articles
	article1 := helpers.CreateTestArticle(t, db, feed.ID, "Article 1", "https://test.com/article1")
	article2 := helpers.CreateTestArticle(t, db, feed.ID, "Article 2", "https://test.com/article2")

	t.Run("GetUserArticles", func(t *testing.T) {
		articles, err := db.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles: %v", err)
		}

		if len(articles) != 2 {
			t.Errorf("Expected 2 articles, got %d", len(articles))
		}
	})

	t.Run("GetUserFeedArticles", func(t *testing.T) {
		articles, err := db.GetUserFeedArticles(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to get user feed articles: %v", err)
		}

		if len(articles) != 2 {
			t.Errorf("Expected 2 articles, got %d", len(articles))
		}
	})

	t.Run("MarkUserArticleRead", func(t *testing.T) {
		err := db.MarkUserArticleRead(user.ID, article1.ID, true)
		if err != nil {
			t.Fatalf("Failed to mark article as read: %v", err)
		}

		articles, err := db.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles: %v", err)
		}

		var readArticle *database.Article
		for _, article := range articles {
			if article.ID == article1.ID {
				readArticle = &article
				break
			}
		}

		if readArticle == nil {
			t.Fatalf("Article with ID %d not found in user articles", article1.ID)
		}

		if !readArticle.IsRead {
			t.Errorf("Expected article %d to be marked as read, but IsRead = %v", article1.ID, readArticle.IsRead)
		}
	})

	t.Run("ToggleUserArticleStar", func(t *testing.T) {
		// First toggle - should star the article
		err := db.ToggleUserArticleStar(user.ID, article2.ID)
		if err != nil {
			t.Fatalf("Failed to star article: %v", err)
		}

		articles, err := db.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles: %v", err)
		}

		var starredArticle *database.Article
		for _, article := range articles {
			if article.ID == article2.ID {
				starredArticle = &article
				break
			}
		}

		if starredArticle == nil {
			t.Fatalf("Article with ID %d not found in user articles", article2.ID)
		}

		if !starredArticle.IsStarred {
			t.Errorf("Expected article %d to be starred, but IsStarred = %v", article2.ID, starredArticle.IsStarred)
		}

		// Second toggle - should unstar the article
		err = db.ToggleUserArticleStar(user.ID, article2.ID)
		if err != nil {
			t.Fatalf("Failed to unstar article: %v", err)
		}

		articles, err = db.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles after unstar: %v", err)
		}

		for _, article := range articles {
			if article.ID == article2.ID {
				starredArticle = &article
				break
			}
		}

		if starredArticle.IsStarred {
			t.Errorf("Expected article %d to be unstarred, but IsStarred = %v", article2.ID, starredArticle.IsStarred)
		}
	})
}

func TestDataIsolation(t *testing.T) {
	db := helpers.CreateTestDB(t)

	// Create two users
	user1 := helpers.CreateTestUser(t, db, "user1", "user1@example.com", "User 1")
	user2 := helpers.CreateTestUser(t, db, "user2", "user2@example.com", "User 2")

	// Create a feed
	feed := helpers.CreateTestFeed(t, db, "Shared Feed", "https://shared.com/rss", "Shared feed")
	article := helpers.CreateTestArticle(t, db, feed.ID, "Shared Article", "https://shared.com/article1")

	// Subscribe both users to the feed
	_ = db.SubscribeUserToFeed(user1.ID, feed.ID)
	_ = db.SubscribeUserToFeed(user2.ID, feed.ID)

	// User 1 marks article as read
	err := db.MarkUserArticleRead(user1.ID, article.ID, true)
	if err != nil {
		t.Fatalf("Failed to mark article as read for user 1: %v", err)
	}

	// Check that only user 1 sees the article as read
	user1Articles, err := db.GetUserArticles(user1.ID)
	if err != nil {
		t.Fatalf("Failed to get articles for user 1: %v", err)
	}

	user2Articles, err := db.GetUserArticles(user2.ID)
	if err != nil {
		t.Fatalf("Failed to get articles for user 2: %v", err)
	}

	if len(user1Articles) != 1 || !user1Articles[0].IsRead {
		t.Error("Expected user 1 to see article as read")
	}

	if len(user2Articles) != 1 || user2Articles[0].IsRead {
		t.Error("Expected user 2 to see article as unread")
	}
}
