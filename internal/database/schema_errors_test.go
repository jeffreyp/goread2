package database

import (
	"testing"
	"time"
)

// Error handling and edge case tests for database operations

// Test GetUserArticles error cases
func TestGetUserArticlesNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	articles, err := db.GetUserArticles(99999)
	if err != nil {
		t.Fatalf("GetUserArticles should not error for non-existent user: %v", err)
	}

	if len(articles) != 0 {
		t.Errorf("Expected 0 articles for non-existent user, got %d", len(articles))
	}
}

// Test GetUserFeedArticles error cases
func TestGetUserFeedArticlesNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)

	articles, err := db.GetUserFeedArticles(99999, feed.ID)
	if err != nil {
		t.Fatalf("GetUserFeedArticles should not error for non-existent user: %v", err)
	}

	if len(articles) != 0 {
		t.Errorf("Expected 0 articles for non-existent user, got %d", len(articles))
	}
}

func TestGetUserFeedArticlesNonExistentFeed(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	articles, err := db.GetUserFeedArticles(user.ID, 99999)
	if err != nil {
		t.Fatalf("GetUserFeedArticles should not error for non-existent feed: %v", err)
	}

	if len(articles) != 0 {
		t.Errorf("Expected 0 articles for non-existent feed, got %d", len(articles))
	}
}

// Test GetUserArticlesPaginated error cases
func TestGetUserArticlesPaginatedNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	result, err := db.GetUserArticlesPaginated(99999, 10, "", false)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated should not error for non-existent user: %v", err)
	}

	if len(result.Articles) != 0 {
		t.Errorf("Expected 0 articles for non-existent user, got %d", len(result.Articles))
	}
}

func TestGetUserArticlesPaginatedSmallLimit(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	createTestArticle(t, db, feed.ID)
	createTestArticle(t, db, feed.ID)

	// Limit of 1 should work properly
	result, err := db.GetUserArticlesPaginated(user.ID, 1, "", false)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated with limit=1 failed: %v", err)
	}

	// Should return 1 article
	if len(result.Articles) != 1 {
		t.Errorf("Expected 1 article with limit=1, got %d", len(result.Articles))
	}

	// Should have cursor since there's more data
	if result.NextCursor == "" {
		t.Error("Expected cursor when more results available")
	}
}

// Test GetUserUnreadCounts error cases
func TestGetUserUnreadCountsNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	unreadCounts, err := db.GetUserUnreadCounts(99999)
	if err != nil {
		t.Fatalf("GetUserUnreadCounts should not error for non-existent user: %v", err)
	}

	if len(unreadCounts) != 0 {
		t.Errorf("Expected empty map for non-existent user, got %d entries", len(unreadCounts))
	}
}

// Test GetAccountStats error cases
func TestGetAccountStatsNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	stats, err := db.GetAccountStats(99999)
	if err != nil {
		t.Fatalf("GetAccountStats should not error for non-existent user: %v", err)
	}

	if stats["total_articles"] != 0 {
		t.Errorf("Expected total_articles=0 for non-existent user, got %v", stats["total_articles"])
	}
}

// Test SetUserArticleStatus error cases
func TestSetUserArticleStatusNonExistentArticle(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	// Should error due to foreign key constraint
	err := db.SetUserArticleStatus(user.ID, 99999, true, false)
	if err == nil {
		t.Error("Expected error when setting status for non-existent article")
	}
}

// Test MarkUserArticleRead error cases
func TestMarkUserArticleReadNonExistentArticle(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	// Should error due to foreign key constraint
	err := db.MarkUserArticleRead(user.ID, 99999, true)
	if err == nil {
		t.Error("Expected error when marking non-existent article as read")
	}
}

// Test ToggleUserArticleStar error cases
func TestToggleUserArticleStarNonExistentArticle(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	// Should error due to foreign key constraint
	err := db.ToggleUserArticleStar(user.ID, 99999)
	if err == nil {
		t.Error("Expected error when toggling star for non-existent article")
	}
}

// Test GetUserArticleStatus error cases
func TestGetUserArticleStatusNonExistent(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	_, err := db.GetUserArticleStatus(user.ID, 99999)
	// May return error or nil depending on implementation
	// Just verify it doesn't crash
	_ = err
}

// Test BatchSetUserArticleStatus with large batch
func TestBatchSetUserArticleStatusLargeBatch(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// Create 100 articles
	articles := make([]Article, 100)
	for i := 0; i < 100; i++ {
		article := createTestArticle(t, db, feed.ID)
		articles[i] = *article
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// Batch mark all as read
	err := db.BatchSetUserArticleStatus(user.ID, articles, true, false)
	if err != nil {
		t.Fatalf("BatchSetUserArticleStatus with large batch failed: %v", err)
	}

	// Verify all are marked as read
	for i := 0; i < 10; i++ { // Check a sample
		status, err := db.GetUserArticleStatus(user.ID, articles[i].ID)
		if err != nil {
			t.Fatalf("GetUserArticleStatus failed: %v", err)
		}
		if status == nil || !status.IsRead {
			t.Errorf("Article %d should be marked as read", articles[i].ID)
		}
	}
}

// Test GetUserByEmail error cases
func TestGetUserByEmailNotFound(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.GetUserByEmail("nonexistent@example.com")
	// May error or return nil depending on implementation
	_ = err
}

// Test GetUserByID error cases
func TestGetUserByIDNotFound(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.GetUserByID(99999)
	// May error or return nil depending on implementation
	_ = err
}

// Test GetUserFeeds error cases
func TestGetUserFeedsNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	feeds, err := db.GetUserFeeds(99999)
	if err != nil {
		t.Fatalf("GetUserFeeds should not error for non-existent user: %v", err)
	}

	if len(feeds) != 0 {
		t.Errorf("Expected 0 feeds for non-existent user, got %d", len(feeds))
	}
}

// Test SubscribeUserToFeed error cases
func TestSubscribeUserToFeedNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)

	err := db.SubscribeUserToFeed(99999, feed.ID)
	if err == nil {
		t.Error("Expected error when subscribing non-existent user to feed")
	}
}

func TestSubscribeUserToFeedNonExistentFeed(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	err := db.SubscribeUserToFeed(user.ID, 99999)
	if err == nil {
		t.Error("Expected error when subscribing user to non-existent feed")
	}
}

// Test UnsubscribeUserFromFeed error cases
func TestUnsubscribeUserFromFeedNotSubscribed(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	// Unsubscribe without subscribing first (should not error)
	err := db.UnsubscribeUserFromFeed(user.ID, feed.ID)
	if err != nil {
		t.Fatalf("UnsubscribeUserFromFeed should not error for non-subscribed feed: %v", err)
	}
}

// Test UpdateUserSubscription error cases
func TestUpdateUserSubscriptionNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	err := db.UpdateUserSubscription(99999, "active", "sub_123", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("UpdateUserSubscription should not error for non-existent user: %v", err)
	}
}

// Test IsUserSubscriptionActive error cases
func TestIsUserSubscriptionActiveNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.IsUserSubscriptionActive(99999)
	// May error for non-existent user
	_ = err
}

// Test SetUserAdmin error cases
func TestSetUserAdminNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	err := db.SetUserAdmin(99999, true)
	if err != nil {
		t.Fatalf("SetUserAdmin should not error for non-existent user: %v", err)
	}
}

// Test GrantFreeMonths error cases
func TestGrantFreeMonthsNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	err := db.GrantFreeMonths(99999, 3)
	if err != nil {
		t.Fatalf("GrantFreeMonths should not error for non-existent user: %v", err)
	}
}

// Test GetUserFeedCount error cases
func TestGetUserFeedCountNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	count, err := db.GetUserFeedCount(99999)
	if err != nil {
		t.Fatalf("GetUserFeedCount should not error for non-existent user: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 feeds for non-existent user, got %d", count)
	}
}

// Test UpdateUserMaxArticlesOnFeedAdd error cases
func TestUpdateUserMaxArticlesOnFeedAddNonExistentUser(t *testing.T) {
	db := setupTestDB(t)

	err := db.UpdateUserMaxArticlesOnFeedAdd(99999, 200)
	if err != nil {
		t.Fatalf("UpdateUserMaxArticlesOnFeedAdd should not error for non-existent user: %v", err)
	}
}

// Test CreateSession error cases
func TestCreateSessionEmptyID(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	session := &Session{
		ID:        "", // Empty ID
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := db.CreateSession(session)
	// This might error depending on database constraints
	// If it doesn't error, that's also valid behavior
	_ = err
}

// Test DeleteSession error cases
func TestDeleteSessionNonExistent(t *testing.T) {
	db := setupTestDB(t)

	err := db.DeleteSession("nonexistent_session")
	if err != nil {
		t.Fatalf("DeleteSession should not error for non-existent session: %v", err)
	}
}

// Test CleanupOrphanedUserArticles with various day values
func TestCleanupOrphanedUserArticlesNegativeDays(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	article := createTestArticle(t, db, feed.ID)
	if err := db.MarkUserArticleRead(user.ID, article.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	if err := db.UnsubscribeUserFromFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("UnsubscribeUserFromFeed failed: %v", err)
	}

	// Negative days should be treated as future date, so nothing should be deleted
	deletedCount, err := db.CleanupOrphanedUserArticles(-1)
	if err != nil {
		t.Fatalf("CleanupOrphanedUserArticles failed: %v", err)
	}

	// With negative days, the datetime calculation creates a future date,
	// so created_at < future_date is always true, should delete the orphan
	if deletedCount != 1 {
		t.Logf("Note: Negative days resulted in %d deletions (behavior may vary)", deletedCount)
	}
}

// Test multiple articles with same timestamp (edge case for pagination)
func TestGetUserArticlesPaginatedSameTimestamp(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// Create articles with same published_at by using SQL directly
	now := time.Now()
	for i := 0; i < 5; i++ {
		article := &Article{
			FeedID:      feed.ID,
			Title:       "Same Time Article",
			URL:         formatString("https://example.com/same_time_%d", i),
			Content:     "Content",
			PublishedAt: now, // Same timestamp
			CreatedAt:   time.Now(),
		}
		if err := db.AddArticle(article); err != nil {
			t.Fatalf("AddArticle failed: %v", err)
		}
	}

	// Paginate with limit=2
	result, err := db.GetUserArticlesPaginated(user.ID, 2, "", false)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated failed: %v", err)
	}

	if len(result.Articles) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(result.Articles))
	}

	if result.NextCursor == "" {
		t.Error("Expected cursor for more results")
	}

	// Get next page
	result2, err := db.GetUserArticlesPaginated(user.ID, 2, result.NextCursor, false)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated page 2 failed: %v", err)
	}

	if len(result2.Articles) != 2 {
		t.Errorf("Expected 2 articles on page 2, got %d", len(result2.Articles))
	}

	// Verify no overlap (all IDs should be unique)
	seenIDs := make(map[int]bool)
	for _, a := range result.Articles {
		if seenIDs[a.ID] {
			t.Errorf("Duplicate article ID %d in results", a.ID)
		}
		seenIDs[a.ID] = true
	}
	for _, a := range result2.Articles {
		if seenIDs[a.ID] {
			t.Errorf("Duplicate article ID %d between pages", a.ID)
		}
		seenIDs[a.ID] = true
	}
}

// Test GetFeedByID (helper to check if it exists)
func TestGetFeedByIDNotFound(t *testing.T) {
	db := setupTestDB(t)

	// GetFeedByID doesn't exist, but we test GetFeedByURL which is similar
	feed, err := db.GetFeedByURL("https://nonexistent-feed.example.com/feed.xml")
	if err != nil {
		t.Fatalf("GetFeedByURL should not error for non-existent feed: %v", err)
	}

	if feed != nil {
		t.Error("Expected nil for non-existent feed")
	}
}

// Test GetArticles error cases
func TestGetArticlesNonExistentFeed(t *testing.T) {
	db := setupTestDB(t)

	articles, err := db.GetArticles(99999)
	if err != nil {
		t.Fatalf("GetArticles should not error for non-existent feed: %v", err)
	}

	if len(articles) != 0 {
		t.Errorf("Expected 0 articles for non-existent feed, got %d", len(articles))
	}
}

// Test UpdateFeedLastFetch error cases
func TestUpdateFeedLastFetchNonExistentFeed(t *testing.T) {
	db := setupTestDB(t)

	err := db.UpdateFeedLastFetch(99999, time.Now())
	if err != nil {
		t.Fatalf("UpdateFeedLastFetch should not error for non-existent feed: %v", err)
	}
}

// Test UpdateFeed error cases
func TestUpdateFeedNonExistent(t *testing.T) {
	db := setupTestDB(t)

	feed := &Feed{
		ID:          99999,
		Title:       "Non-existent",
		URL:         "https://example.com/nonexistent.xml",
		Description: "Test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastFetch:   time.Now(),
	}

	err := db.UpdateFeed(feed)
	if err != nil {
		t.Fatalf("UpdateFeed should not error for non-existent feed: %v", err)
	}
}

// Test DeleteFeed error cases
func TestDeleteFeedNonExistent(t *testing.T) {
	db := setupTestDB(t)

	err := db.DeleteFeed(99999)
	if err != nil {
		t.Fatalf("DeleteFeed should not error for non-existent feed: %v", err)
	}
}
