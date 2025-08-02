package unit

import (
	"testing"

	"goread2/internal/services"
	"goread2/test/helpers"
)

func TestFeedService(t *testing.T) {
	db := helpers.CreateTestDB(t)
	feedService := services.NewFeedService(db)

	// Create test user
	user := helpers.CreateTestUser(t, db, "google123", "test@example.com", "Test User")

	t.Run("GetUserFeeds_Empty", func(t *testing.T) {
		feeds, err := feedService.GetUserFeeds(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user feeds: %v", err)
		}

		if len(feeds) != 0 {
			t.Errorf("Expected 0 feeds, got %d", len(feeds))
		}
	})

	t.Run("AddFeedForUser_NewFeed", func(t *testing.T) {
		// This test would require mocking HTTP client for real RSS feeds
		// For now, we'll test the basic flow with a mock URL

		// First, create a feed manually to simulate successful fetch
		testFeed := helpers.CreateTestFeed(t, db, "Test Feed", "https://test.com/rss", "Test feed")

		// Subscribe user to the feed
		err := db.SubscribeUserToFeed(user.ID, testFeed.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe user to feed: %v", err)
		}

		// Verify user can see the feed
		feeds, err := feedService.GetUserFeeds(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user feeds: %v", err)
		}

		if len(feeds) != 1 {
			t.Errorf("Expected 1 feed, got %d", len(feeds))
		}

		if feeds[0].Title != "Test Feed" {
			t.Errorf("Expected feed title 'Test Feed', got '%s'", feeds[0].Title)
		}
	})

	t.Run("GetUserArticles", func(t *testing.T) {
		// Create a feed and article
		feed := helpers.CreateTestFeed(t, db, "Article Feed", "https://articles.com/rss", "Article feed")
		article := helpers.CreateTestArticle(t, db, feed.ID, "Test Article", "https://articles.com/article1")

		// Subscribe user to feed
		err := db.SubscribeUserToFeed(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe user to feed: %v", err)
		}

		// Get user articles
		articles, err := feedService.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles: %v", err)
		}

		if len(articles) < 1 {
			t.Errorf("Expected at least 1 article, got %d", len(articles))
		}

		// Find our test article
		found := false
		for _, a := range articles {
			if a.ID == article.ID {
				found = true
				if a.Title != "Test Article" {
					t.Errorf("Expected article title 'Test Article', got '%s'", a.Title)
				}
				break
			}
		}

		if !found {
			t.Error("Expected to find test article in user articles")
		}
	})

	t.Run("GetUserFeedArticles", func(t *testing.T) {
		// Create a feed and article
		feed := helpers.CreateTestFeed(t, db, "Specific Feed", "https://specific.com/rss", "Specific feed")
		article := helpers.CreateTestArticle(t, db, feed.ID, "Specific Article", "https://specific.com/article1")

		// Subscribe user to feed
		err := db.SubscribeUserToFeed(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe user to feed: %v", err)
		}

		// Get articles for specific feed
		articles, err := feedService.GetUserFeedArticles(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to get user feed articles: %v", err)
		}

		if len(articles) != 1 {
			t.Errorf("Expected 1 article, got %d", len(articles))
		}

		if articles[0].ID != article.ID {
			t.Errorf("Expected article ID %d, got %d", article.ID, articles[0].ID)
		}
	})

	t.Run("MarkUserArticleRead", func(t *testing.T) {
		// Create a feed and article
		feed := helpers.CreateTestFeed(t, db, "Read Test Feed", "https://readtest.com/rss", "Read test feed")
		article := helpers.CreateTestArticle(t, db, feed.ID, "Read Test Article", "https://readtest.com/article1")

		// Subscribe user to feed
		err := db.SubscribeUserToFeed(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe user to feed: %v", err)
		}

		// Mark article as read
		err = feedService.MarkUserArticleRead(user.ID, article.ID, true)
		if err != nil {
			t.Fatalf("Failed to mark article as read: %v", err)
		}

		// Verify article is marked as read
		articles, err := feedService.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles: %v", err)
		}

		found := false
		for _, a := range articles {
			if a.ID == article.ID {
				found = true
				if !a.IsRead {
					t.Error("Expected article to be marked as read")
				}
				break
			}
		}

		if !found {
			t.Error("Expected to find test article")
		}
	})

	t.Run("ToggleUserArticleStar", func(t *testing.T) {
		// Create a feed and article
		feed := helpers.CreateTestFeed(t, db, "Star Test Feed", "https://startest.com/rss", "Star test feed")
		article := helpers.CreateTestArticle(t, db, feed.ID, "Star Test Article", "https://startest.com/article1")

		// Subscribe user to feed
		err := db.SubscribeUserToFeed(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe user to feed: %v", err)
		}

		// Star the article
		err = feedService.ToggleUserArticleStar(user.ID, article.ID)
		if err != nil {
			t.Fatalf("Failed to star article: %v", err)
		}

		// Verify article is starred
		articles, err := feedService.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles: %v", err)
		}

		found := false
		for _, a := range articles {
			if a.ID == article.ID {
				found = true
				if !a.IsStarred {
					t.Error("Expected article to be starred")
				}
				break
			}
		}

		if !found {
			t.Error("Expected to find test article")
		}

		// Unstar the article
		err = feedService.ToggleUserArticleStar(user.ID, article.ID)
		if err != nil {
			t.Fatalf("Failed to unstar article: %v", err)
		}

		// Verify article is unstarred
		articles, err = feedService.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles after unstar: %v", err)
		}

		for _, a := range articles {
			if a.ID == article.ID {
				if a.IsStarred {
					t.Error("Expected article to be unstarred")
				}
				break
			}
		}
	})
}

func TestFeedServiceMultiUser(t *testing.T) {
	db := helpers.CreateTestDB(t)
	feedService := services.NewFeedService(db)

	// Create two users
	user1 := helpers.CreateTestUser(t, db, "user1", "user1@example.com", "User 1")
	user2 := helpers.CreateTestUser(t, db, "user2", "user2@example.com", "User 2")

	// Create a shared feed
	feed := helpers.CreateTestFeed(t, db, "Shared Feed", "https://shared.com/rss", "Shared feed")
	article := helpers.CreateTestArticle(t, db, feed.ID, "Shared Article", "https://shared.com/article1")

	// Subscribe both users to the feed
	err := db.SubscribeUserToFeed(user1.ID, feed.ID)
	if err != nil {
		t.Fatalf("Failed to subscribe user1 to feed: %v", err)
	}

	err = db.SubscribeUserToFeed(user2.ID, feed.ID)
	if err != nil {
		t.Fatalf("Failed to subscribe user2 to feed: %v", err)
	}

	t.Run("UserSpecificReadStatus", func(t *testing.T) {
		// User 1 marks article as read
		err = feedService.MarkUserArticleRead(user1.ID, article.ID, true)
		if err != nil {
			t.Fatalf("Failed to mark article as read for user1: %v", err)
		}

		// Check user 1 sees article as read
		user1Articles, err := feedService.GetUserArticles(user1.ID)
		if err != nil {
			t.Fatalf("Failed to get articles for user1: %v", err)
		}

		user1Read := false
		for _, a := range user1Articles {
			if a.ID == article.ID && a.IsRead {
				user1Read = true
				break
			}
		}

		if !user1Read {
			t.Error("Expected user1 to see article as read")
		}

		// Check user 2 sees article as unread
		user2Articles, err := feedService.GetUserArticles(user2.ID)
		if err != nil {
			t.Fatalf("Failed to get articles for user2: %v", err)
		}

		user2Read := false
		for _, a := range user2Articles {
			if a.ID == article.ID && a.IsRead {
				user2Read = true
				break
			}
		}

		if user2Read {
			t.Error("Expected user2 to see article as unread")
		}
	})

	t.Run("UserSpecificStarStatus", func(t *testing.T) {
		// User 2 stars the article
		err = feedService.ToggleUserArticleStar(user2.ID, article.ID)
		if err != nil {
			t.Fatalf("Failed to star article for user2: %v", err)
		}

		// Check user 2 sees article as starred
		user2Articles, err := feedService.GetUserArticles(user2.ID)
		if err != nil {
			t.Fatalf("Failed to get articles for user2: %v", err)
		}

		user2Starred := false
		for _, a := range user2Articles {
			if a.ID == article.ID && a.IsStarred {
				user2Starred = true
				break
			}
		}

		if !user2Starred {
			t.Error("Expected user2 to see article as starred")
		}

		// Check user 1 sees article as unstarred
		user1Articles, err := feedService.GetUserArticles(user1.ID)
		if err != nil {
			t.Fatalf("Failed to get articles for user1: %v", err)
		}

		user1Starred := false
		for _, a := range user1Articles {
			if a.ID == article.ID && a.IsStarred {
				user1Starred = true
				break
			}
		}

		if user1Starred {
			t.Error("Expected user1 to see article as unstarred")
		}
	})
}
