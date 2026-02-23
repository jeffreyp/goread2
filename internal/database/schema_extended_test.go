package database

import (
	"testing"
	"time"
)

// UpdateFeedTracking tests

func TestUpdateFeedTracking(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)

	lastChecked := time.Now()
	lastHadNewContent := time.Now().Add(-1 * time.Hour)
	avgInterval := 3600 // 1 hour in seconds

	err := db.UpdateFeedTracking(feed.ID, lastChecked, lastHadNewContent, avgInterval)
	if err != nil {
		t.Fatalf("UpdateFeedTracking failed: %v", err)
	}

	// Retrieve and verify
	updatedFeed, err := db.GetFeedByURL(feed.URL)
	if err != nil {
		t.Fatalf("GetFeedByURL failed: %v", err)
	}

	if updatedFeed == nil {
		t.Fatal("GetFeedByURL returned nil")
	}

	// Check that last_checked was updated (allowing small time difference)
	if updatedFeed.LastChecked.Sub(lastChecked).Abs() > time.Second {
		t.Errorf("LastChecked not updated correctly: expected %v, got %v", lastChecked, updatedFeed.LastChecked)
	}

	// Check that last_had_new_content was updated
	if updatedFeed.LastHadNewContent.Sub(lastHadNewContent).Abs() > time.Second {
		t.Errorf("LastHadNewContent not updated correctly: expected %v, got %v", lastHadNewContent, updatedFeed.LastHadNewContent)
	}

	// Check average update interval
	if updatedFeed.AverageUpdateInterval != avgInterval {
		t.Errorf("AverageUpdateInterval not updated: expected %d, got %d", avgInterval, updatedFeed.AverageUpdateInterval)
	}
}

func TestUpdateFeedTrackingNonExistent(t *testing.T) {
	db := setupTestDB(t)

	lastChecked := time.Now()
	lastHadNewContent := time.Now()

	// Should not error even if feed doesn't exist (UPDATE without WHERE match)
	err := db.UpdateFeedTracking(99999, lastChecked, lastHadNewContent, 3600)
	if err != nil {
		t.Errorf("UpdateFeedTracking should not error for non-existent feed: %v", err)
	}
}

// UpdateFeedAfterRefresh tests

func TestUpdateFeedAfterRefresh(t *testing.T) {
	db := setupTestDB(t)
	feed := createTestFeed(t, db)

	lastChecked := time.Now()
	lastHadNewContent := time.Now().Add(-30 * time.Minute)
	avgInterval := 7200
	lastFetch := time.Now().Add(-1 * time.Minute)
	etag := `"abc123"`
	lastModified := "Mon, 02 Jan 2006 15:04:05 GMT"

	err := db.UpdateFeedAfterRefresh(feed.ID, lastChecked, lastHadNewContent, avgInterval, lastFetch, etag, lastModified)
	if err != nil {
		t.Fatalf("UpdateFeedAfterRefresh failed: %v", err)
	}

	updated, err := db.GetFeedByURL(feed.URL)
	if err != nil || updated == nil {
		t.Fatalf("GetFeedByURL failed: %v", err)
	}

	if updated.LastChecked.Sub(lastChecked).Abs() > time.Second {
		t.Errorf("LastChecked mismatch: got %v, want %v", updated.LastChecked, lastChecked)
	}
	if updated.LastHadNewContent.Sub(lastHadNewContent).Abs() > time.Second {
		t.Errorf("LastHadNewContent mismatch: got %v, want %v", updated.LastHadNewContent, lastHadNewContent)
	}
	if updated.AverageUpdateInterval != avgInterval {
		t.Errorf("AverageUpdateInterval mismatch: got %d, want %d", updated.AverageUpdateInterval, avgInterval)
	}
	if updated.LastFetch.Sub(lastFetch).Abs() > time.Second {
		t.Errorf("LastFetch mismatch: got %v, want %v", updated.LastFetch, lastFetch)
	}
	if updated.ETag != etag {
		t.Errorf("ETag mismatch: got %q, want %q", updated.ETag, etag)
	}
	if updated.LastModified != lastModified {
		t.Errorf("LastModified mismatch: got %q, want %q", updated.LastModified, lastModified)
	}
}

func TestUpdateFeedAfterRefreshNonExistent(t *testing.T) {
	db := setupTestDB(t)

	err := db.UpdateFeedAfterRefresh(99999, time.Now(), time.Now(), 3600, time.Now(), "", "")
	if err != nil {
		t.Errorf("UpdateFeedAfterRefresh should not error for non-existent feed: %v", err)
	}
}

// UpdateSessionExpiry tests

func TestUpdateSessionExpiry(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	session := &Session{
		ID:        "test_session_expiry",
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := db.CreateSession(session)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Update expiry to 7 days from now
	newExpiry := time.Now().Add(7 * 24 * time.Hour)
	err = db.UpdateSessionExpiry(session.ID, newExpiry)
	if err != nil {
		t.Fatalf("UpdateSessionExpiry failed: %v", err)
	}

	// Verify update
	updatedSession, err := db.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if updatedSession == nil {
		t.Fatal("GetSession returned nil")
	}

	// Allow for small time differences due to database precision
	if updatedSession.ExpiresAt.Sub(newExpiry).Abs() > time.Second {
		t.Errorf("ExpiresAt not updated correctly: expected %v, got %v", newExpiry, updatedSession.ExpiresAt)
	}
}

func TestUpdateSessionExpiryNonExistent(t *testing.T) {
	db := setupTestDB(t)

	// Should not error even if session doesn't exist
	err := db.UpdateSessionExpiry("nonexistent_session", time.Now().Add(1*time.Hour))
	if err != nil {
		t.Errorf("UpdateSessionExpiry should not error for non-existent session: %v", err)
	}
}

// GetUserArticlesPaginated advanced edge cases

func TestGetUserArticlesPaginatedMultipleFeeds(t *testing.T) {
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

	// Create articles from both feeds
	for i := 0; i < 3; i++ {
		createTestArticle(t, db, feed1.ID)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}
	for i := 0; i < 3; i++ {
		createTestArticle(t, db, feed2.ID)
		time.Sleep(1 * time.Millisecond)
	}

	// Get paginated results
	result, err := db.GetUserArticlesPaginated(user.ID, 10, "", false)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated failed: %v", err)
	}

	if len(result.Articles) != 6 {
		t.Errorf("Expected 6 articles from both feeds, got %d", len(result.Articles))
	}

	// Verify articles are from both feeds
	feed1Count := 0
	feed2Count := 0
	for _, a := range result.Articles {
		switch a.FeedID {
		case feed1.ID:
			feed1Count++
		case feed2.ID:
			feed2Count++
		}
	}

	if feed1Count != 3 || feed2Count != 3 {
		t.Errorf("Expected 3 articles from each feed, got %d from feed1 and %d from feed2", feed1Count, feed2Count)
	}
}

func TestGetUserArticlesPaginatedOrdering(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// Create articles with different published times
	article1 := createTestArticle(t, db, feed.ID)
	time.Sleep(2 * time.Millisecond)
	article2 := createTestArticle(t, db, feed.ID)
	time.Sleep(2 * time.Millisecond)
	article3 := createTestArticle(t, db, feed.ID)

	result, err := db.GetUserArticlesPaginated(user.ID, 10, "", false)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated failed: %v", err)
	}

	if len(result.Articles) != 3 {
		t.Fatalf("Expected 3 articles, got %d", len(result.Articles))
	}

	// Verify articles are ordered by published_at DESC (newest first)
	// article3 should be first (newest)
	if result.Articles[0].ID != article3.ID {
		t.Errorf("Expected newest article (id=%d) first, got id=%d", article3.ID, result.Articles[0].ID)
	}
	if result.Articles[2].ID != article1.ID {
		t.Errorf("Expected oldest article (id=%d) last, got id=%d", article1.ID, result.Articles[2].ID)
	}

	// Use article2 to avoid unused warning
	_ = article2
}

func TestGetUserArticlesPaginatedCursorBoundary(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// Create exactly 5 articles
	for i := 0; i < 5; i++ {
		createTestArticle(t, db, feed.ID)
		time.Sleep(1 * time.Millisecond)
	}

	// Test with limit=5 (should get all, no cursor)
	result, err := db.GetUserArticlesPaginated(user.ID, 5, "", false)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated failed: %v", err)
	}
	if len(result.Articles) != 5 {
		t.Errorf("Expected 5 articles, got %d", len(result.Articles))
	}
	if result.NextCursor != "" {
		t.Error("Expected no cursor when all results fit in one page")
	}

	// Test with limit=4 (should get 4, with cursor)
	result, err = db.GetUserArticlesPaginated(user.ID, 4, "", false)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated failed: %v", err)
	}
	if len(result.Articles) != 4 {
		t.Errorf("Expected 4 articles, got %d", len(result.Articles))
	}
	if result.NextCursor == "" {
		t.Error("Expected cursor when more results available")
	}

	// Use cursor to get remaining article
	result, err = db.GetUserArticlesPaginated(user.ID, 4, result.NextCursor, false)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated with cursor failed: %v", err)
	}
	if len(result.Articles) != 1 {
		t.Errorf("Expected 1 remaining article, got %d", len(result.Articles))
	}
	if result.NextCursor != "" {
		t.Error("Expected no cursor after getting last page")
	}
}

// CleanupOrphanedUserArticles edge cases

func TestCleanupOrphanedUserArticlesWithAge(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	// Subscribe and then unsubscribe
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

	// Try cleanup with 1 day age requirement (orphaned record is fresh, should not be deleted)
	deletedCount, err := db.CleanupOrphanedUserArticles(1)
	if err != nil {
		t.Fatalf("CleanupOrphanedUserArticles failed: %v", err)
	}

	if deletedCount != 0 {
		t.Errorf("Expected 0 deletions (record too fresh), got %d", deletedCount)
	}

	// Verify record still exists
	status, err := db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil || status == nil {
		t.Error("User article should still exist (too fresh to clean)")
	}

	// Try cleanup with 0 days (should delete immediately)
	deletedCount, err = db.CleanupOrphanedUserArticles(0)
	if err != nil {
		t.Fatalf("CleanupOrphanedUserArticles failed: %v", err)
	}

	if deletedCount != 1 {
		t.Errorf("Expected 1 deletion, got %d", deletedCount)
	}

	// Verify record is now deleted
	status, err = db.GetUserArticleStatus(user.ID, article.ID)
	if err == nil && status != nil {
		t.Error("User article should have been deleted")
	}
}

func TestCleanupOrphanedUserArticlesMultipleUsers(t *testing.T) {
	db := setupTestDB(t)

	user1 := createTestUser(t, db)
	user2 := createTestUser(t, db)
	feed := createTestFeed(t, db)

	// Both users subscribe
	if err := db.SubscribeUserToFeed(user1.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.SubscribeUserToFeed(user2.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	article := createTestArticle(t, db, feed.ID)

	// Both users read the article
	if err := db.MarkUserArticleRead(user1.ID, article.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}
	if err := db.MarkUserArticleRead(user2.ID, article.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	// Only user1 unsubscribes
	if err := db.UnsubscribeUserFromFeed(user1.ID, feed.ID); err != nil {
		t.Fatalf("UnsubscribeUserFromFeed failed: %v", err)
	}

	// Cleanup should only delete user1's orphaned record
	deletedCount, err := db.CleanupOrphanedUserArticles(0)
	if err != nil {
		t.Fatalf("CleanupOrphanedUserArticles failed: %v", err)
	}

	if deletedCount != 1 {
		t.Errorf("Expected 1 deletion (user1's orphaned record), got %d", deletedCount)
	}

	// Verify user1's record is deleted
	status1, err := db.GetUserArticleStatus(user1.ID, article.ID)
	if err == nil && status1 != nil {
		t.Error("User1's article status should have been deleted")
	}

	// Verify user2's record still exists
	status2, err := db.GetUserArticleStatus(user2.ID, article.ID)
	if err != nil || status2 == nil {
		t.Error("User2's article status should still exist")
	}
}

func TestCleanupOrphanedUserArticlesNoOrphans(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	// Subscribe and create article, but don't unsubscribe
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	article := createTestArticle(t, db, feed.ID)
	if err := db.MarkUserArticleRead(user.ID, article.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	// Cleanup should find no orphans
	deletedCount, err := db.CleanupOrphanedUserArticles(0)
	if err != nil {
		t.Fatalf("CleanupOrphanedUserArticles failed: %v", err)
	}

	if deletedCount != 0 {
		t.Errorf("Expected 0 deletions (no orphans), got %d", deletedCount)
	}

	// Verify record still exists
	status, err := db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil || status == nil {
		t.Error("User article should still exist (not orphaned)")
	}
}

// GetAccountStats edge cases

func TestGetAccountStatsNoFeeds(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	stats, err := db.GetAccountStats(user.ID)
	if err != nil {
		t.Fatalf("GetAccountStats failed: %v", err)
	}

	// Should return zero values
	if stats["total_articles"] != 0 {
		t.Errorf("Expected total_articles=0, got %v", stats["total_articles"])
	}
	if stats["total_unread"] != 0 {
		t.Errorf("Expected total_unread=0, got %v", stats["total_unread"])
	}
	if stats["active_feeds"] != 0 {
		t.Errorf("Expected active_feeds=0, got %v", stats["active_feeds"])
	}
}

func TestGetAccountStatsOnlyReadArticles(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// Create articles and mark all as read
	for i := 0; i < 5; i++ {
		article := createTestArticle(t, db, feed.ID)
		if err := db.MarkUserArticleRead(user.ID, article.ID, true); err != nil {
			t.Fatalf("MarkUserArticleRead failed: %v", err)
		}
	}

	stats, err := db.GetAccountStats(user.ID)
	if err != nil {
		t.Fatalf("GetAccountStats failed: %v", err)
	}

	if stats["total_articles"] != 5 {
		t.Errorf("Expected total_articles=5, got %v", stats["total_articles"])
	}
	if stats["total_unread"] != 0 {
		t.Errorf("Expected total_unread=0 (all read), got %v", stats["total_unread"])
	}
	if stats["active_feeds"] != 0 {
		t.Errorf("Expected active_feeds=0 (no unread), got %v", stats["active_feeds"])
	}
}

func TestGetAccountStatsMultipleFeeds(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed1 := createTestFeed(t, db)
	feed2 := createTestFeed(t, db)
	feed3 := createTestFeed(t, db)

	// Subscribe to all feeds
	if err := db.SubscribeUserToFeed(user.ID, feed1.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.SubscribeUserToFeed(user.ID, feed2.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.SubscribeUserToFeed(user.ID, feed3.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// feed1: 3 unread articles
	for i := 0; i < 3; i++ {
		createTestArticle(t, db, feed1.ID)
	}

	// feed2: 2 unread + 1 read
	article := createTestArticle(t, db, feed2.ID)
	if err := db.MarkUserArticleRead(user.ID, article.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}
	createTestArticle(t, db, feed2.ID)
	createTestArticle(t, db, feed2.ID)

	// feed3: all read
	for i := 0; i < 2; i++ {
		article := createTestArticle(t, db, feed3.ID)
		if err := db.MarkUserArticleRead(user.ID, article.ID, true); err != nil {
			t.Fatalf("MarkUserArticleRead failed: %v", err)
		}
	}

	stats, err := db.GetAccountStats(user.ID)
	if err != nil {
		t.Fatalf("GetAccountStats failed: %v", err)
	}

	// Total: 3 + 3 + 2 = 8 articles
	if stats["total_articles"] != 8 {
		t.Errorf("Expected total_articles=8, got %v", stats["total_articles"])
	}

	// Unread: 3 + 2 + 0 = 5
	if stats["total_unread"] != 5 {
		t.Errorf("Expected total_unread=5, got %v", stats["total_unread"])
	}

	// Active feeds (with unread): feed1 and feed2 = 2
	if stats["active_feeds"] != 2 {
		t.Errorf("Expected active_feeds=2, got %v", stats["active_feeds"])
	}
}

// Session management edge cases

func TestSessionExpiryValidation(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	// Create session that expires in past
	expiredSession := &Session{
		ID:        "already_expired",
		UserID:    user.ID,
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	err := db.CreateSession(expiredSession)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Should still be able to retrieve it (expiry is just metadata)
	retrieved, err := db.GetSession(expiredSession.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if retrieved == nil {
		t.Error("Should be able to retrieve expired session")
	}

	// DeleteExpiredSessions should remove it
	err = db.DeleteExpiredSessions()
	if err != nil {
		t.Fatalf("DeleteExpiredSessions failed: %v", err)
	}

	// Now it should be gone
	retrieved, err = db.GetSession(expiredSession.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Expired session should have been deleted")
	}
}

func TestSessionDuplicateID(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	session1 := &Session{
		ID:        "duplicate_id",
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := db.CreateSession(session1)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Try to create another session with same ID
	session2 := &Session{
		ID:        "duplicate_id",
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}

	err = db.CreateSession(session2)
	if err == nil {
		t.Error("Expected error when creating session with duplicate ID")
	}
}

// Audit log edge cases

func TestGetAuditLogsEmptyDatabase(t *testing.T) {
	db := setupTestDB(t)

	logs, err := db.GetAuditLogs(50, 0, nil)
	if err != nil {
		t.Fatalf("GetAuditLogs failed: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected 0 logs from empty database, got %d", len(logs))
	}
}

func TestGetAuditLogsLargeOffset(t *testing.T) {
	db := setupTestDB(t)

	// Create one log
	log := &AuditLog{
		Timestamp:        time.Now(),
		AdminUserID:      1,
		AdminEmail:       "admin@example.com",
		OperationType:    "test",
		TargetUserID:     2,
		TargetUserEmail:  "user@example.com",
		OperationDetails: "{}",
		IPAddress:        "127.0.0.1",
		Result:           "success",
	}

	if err := db.CreateAuditLog(log); err != nil {
		t.Fatalf("CreateAuditLog failed: %v", err)
	}

	// Query with offset beyond available results
	logs, err := db.GetAuditLogs(10, 100, nil)
	if err != nil {
		t.Fatalf("GetAuditLogs failed: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected 0 logs with large offset, got %d", len(logs))
	}
}

func TestGetAuditLogsFilterCombinations(t *testing.T) {
	db := setupTestDB(t)

	// Create logs with various combinations
	logs := []AuditLog{
		{
			Timestamp:        time.Now().Add(-3 * time.Hour),
			AdminUserID:      1,
			AdminEmail:       "admin1@example.com",
			OperationType:    "grant_admin",
			TargetUserID:     10,
			TargetUserEmail:  "target1@example.com",
			OperationDetails: "{}",
			IPAddress:        "192.168.1.1",
			Result:           "success",
		},
		{
			Timestamp:        time.Now().Add(-2 * time.Hour),
			AdminUserID:      1,
			AdminEmail:       "admin1@example.com",
			OperationType:    "grant_free_months",
			TargetUserID:     11,
			TargetUserEmail:  "target2@example.com",
			OperationDetails: "{}",
			IPAddress:        "192.168.1.1",
			Result:           "success",
		},
		{
			Timestamp:        time.Now().Add(-1 * time.Hour),
			AdminUserID:      2,
			AdminEmail:       "admin2@example.com",
			OperationType:    "grant_admin",
			TargetUserID:     10,
			TargetUserEmail:  "target1@example.com",
			OperationDetails: "{}",
			IPAddress:        "192.168.1.2",
			Result:           "failure",
		},
	}

	for i := range logs {
		if err := db.CreateAuditLog(&logs[i]); err != nil {
			t.Fatalf("CreateAuditLog failed: %v", err)
		}
	}

	// Test filter: admin_user_id=1 AND operation_type=grant_admin
	filters := map[string]interface{}{
		"admin_user_id":  1,
		"operation_type": "grant_admin",
	}
	results, err := db.GetAuditLogs(50, 0, filters)
	if err != nil {
		t.Fatalf("GetAuditLogs failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for admin1+grant_admin filter, got %d", len(results))
	}

	// Test filter: target_user_id=10 (should match 2 logs)
	filters = map[string]interface{}{
		"target_user_id": 10,
	}
	results, err = db.GetAuditLogs(50, 0, filters)
	if err != nil {
		t.Fatalf("GetAuditLogs failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for target_user_id=10, got %d", len(results))
	}

	// Test filter: admin_user_id=2 AND target_user_id=10
	filters = map[string]interface{}{
		"admin_user_id":  2,
		"target_user_id": 10,
	}
	results, err = db.GetAuditLogs(50, 0, filters)
	if err != nil {
		t.Fatalf("GetAuditLogs failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for admin2+target10, got %d", len(results))
	}
}
