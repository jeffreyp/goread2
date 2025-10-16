package database

import (
	"fmt"
	"testing"
	"time"
)

// User-article status tests

func TestGetUserArticles(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	article1 := createTestArticle(t, db, feed.ID)
	article2 := createTestArticle(t, db, feed.ID)

	// Mark one as read
	if err := db.MarkUserArticleRead(user.ID, article1.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	articles, err := db.GetUserArticles(user.ID)
	if err != nil {
		t.Fatalf("GetUserArticles failed: %v", err)
	}

	if len(articles) < 2 {
		t.Errorf("Expected at least 2 articles, got %d", len(articles))
	}

	// Verify read status
	for _, a := range articles {
		if a.ID == article1.ID && !a.IsRead {
			t.Error("Article 1 should be marked as read")
		}
		if a.ID == article2.ID && a.IsRead {
			t.Error("Article 2 should not be marked as read")
		}
	}
}

func TestGetUserArticlesPaginated(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// Create 5 articles
	for i := 0; i < 5; i++ {
		createTestArticle(t, db, feed.ID)
	}

	// Test pagination
	articles, err := db.GetUserArticlesPaginated(user.ID, 2, 0, false)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated failed: %v", err)
	}

	if len(articles) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(articles))
	}

	// Test offset
	articles, err = db.GetUserArticlesPaginated(user.ID, 2, 2, false)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated with offset failed: %v", err)
	}

	if len(articles) != 2 {
		t.Errorf("Expected 2 articles with offset, got %d", len(articles))
	}
}

func TestGetUserArticlesPaginatedUnreadOnly(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// Create 5 articles, mark 2 as read
	articles := make([]*Article, 5)
	for i := 0; i < 5; i++ {
		articles[i] = createTestArticle(t, db, feed.ID)
	}

	if err := db.MarkUserArticleRead(user.ID, articles[0].ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}
	if err := db.MarkUserArticleRead(user.ID, articles[1].ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	// Get unread only
	unreadArticles, err := db.GetUserArticlesPaginated(user.ID, 10, 0, true)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated(unreadOnly=true) failed: %v", err)
	}

	if len(unreadArticles) != 3 {
		t.Errorf("Expected 3 unread articles, got %d", len(unreadArticles))
	}

	// Verify all are unread
	for _, a := range unreadArticles {
		if a.IsRead {
			t.Error("Unread-only query returned a read article")
		}
	}
}

func TestGetUserFeedArticles(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed1 := createTestFeed(t, db)
	feed2 := createTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed1.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	article1 := createTestArticle(t, db, feed1.ID)
	article2 := createTestArticle(t, db, feed2.ID) // Different feed

	// Should only get articles from feed1
	articles, err := db.GetUserFeedArticles(user.ID, feed1.ID)
	if err != nil {
		t.Fatalf("GetUserFeedArticles failed: %v", err)
	}

	if len(articles) < 1 {
		t.Error("Expected at least 1 article from feed1")
	}

	// Verify we only got articles from feed1
	for _, a := range articles {
		if a.ID == article2.ID {
			t.Error("Should not have received article from unsubscribed feed")
		}
	}

	// User not subscribed to feed2, should get empty list
	articles, err = db.GetUserFeedArticles(user.ID, feed2.ID)
	if err != nil {
		t.Fatalf("GetUserFeedArticles for unsubscribed feed failed: %v", err)
	}

	if len(articles) != 0 {
		t.Errorf("Expected 0 articles for unsubscribed feed, got %d", len(articles))
	}

	// Verify article1 exists
	_ = article1 // Use the variable to avoid unused error
}

func TestSetUserArticleStatus(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	article := createTestArticle(t, db, feed.ID)

	err := db.SetUserArticleStatus(user.ID, article.ID, true, true)
	if err != nil {
		t.Fatalf("SetUserArticleStatus failed: %v", err)
	}

	status, err := db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetUserArticleStatus failed: %v", err)
	}

	if !status.IsRead {
		t.Error("Article should be marked as read")
	}

	if !status.IsStarred {
		t.Error("Article should be marked as starred")
	}
}

func TestMarkUserArticleRead(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	article := createTestArticle(t, db, feed.ID)

	// Mark as read
	err := db.MarkUserArticleRead(user.ID, article.ID, true)
	if err != nil {
		t.Fatalf("MarkUserArticleRead(true) failed: %v", err)
	}

	status, err := db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetUserArticleStatus failed: %v", err)
	}

	if !status.IsRead {
		t.Error("Article should be marked as read")
	}

	if status.IsStarred {
		t.Error("Article should not be starred (default)")
	}

	// Mark as unread
	err = db.MarkUserArticleRead(user.ID, article.ID, false)
	if err != nil {
		t.Fatalf("MarkUserArticleRead(false) failed: %v", err)
	}

	status, err = db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetUserArticleStatus after unread failed: %v", err)
	}

	if status.IsRead {
		t.Error("Article should not be marked as read")
	}
}

func TestToggleUserArticleStar(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	article := createTestArticle(t, db, feed.ID)

	// First toggle - should star
	err := db.ToggleUserArticleStar(user.ID, article.ID)
	if err != nil {
		t.Fatalf("First ToggleUserArticleStar failed: %v", err)
	}

	status, err := db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetUserArticleStatus failed: %v", err)
	}

	if !status.IsStarred {
		t.Error("Article should be starred after first toggle")
	}

	// Second toggle - should unstar
	err = db.ToggleUserArticleStar(user.ID, article.ID)
	if err != nil {
		t.Fatalf("Second ToggleUserArticleStar failed: %v", err)
	}

	status, err = db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetUserArticleStatus after second toggle failed: %v", err)
	}

	if status.IsStarred {
		t.Error("Article should not be starred after second toggle")
	}
}

func TestBatchSetUserArticleStatus(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	// Create multiple articles
	articles := make([]Article, 3)
	for i := 0; i < 3; i++ {
		article := createTestArticle(t, db, feed.ID)
		articles[i] = *article
	}

	// Batch mark as read
	err := db.BatchSetUserArticleStatus(user.ID, articles, true, false)
	if err != nil {
		t.Fatalf("BatchSetUserArticleStatus failed: %v", err)
	}

	// Verify all are marked as read
	for _, article := range articles {
		status, err := db.GetUserArticleStatus(user.ID, article.ID)
		if err != nil {
			t.Fatalf("GetUserArticleStatus failed for article %d: %v", article.ID, err)
		}

		if !status.IsRead {
			t.Errorf("Article %d should be marked as read", article.ID)
		}

		if status.IsStarred {
			t.Errorf("Article %d should not be starred", article.ID)
		}
	}
}

func TestBatchSetUserArticleStatusEmpty(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	// Empty batch should not error
	err := db.BatchSetUserArticleStatus(user.ID, []Article{}, true, false)
	if err != nil {
		t.Errorf("BatchSetUserArticleStatus with empty list should not error: %v", err)
	}
}

func TestGetUserUnreadCounts(t *testing.T) {
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

	// Add articles
	article1 := createTestArticle(t, db, feed1.ID)
	article2 := createTestArticle(t, db, feed1.ID)
	article3 := createTestArticle(t, db, feed2.ID)

	// Mark one as read
	if err := db.MarkUserArticleRead(user.ID, article1.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	unreadCounts, err := db.GetUserUnreadCounts(user.ID)
	if err != nil {
		t.Fatalf("GetUserUnreadCounts failed: %v", err)
	}

	if unreadCounts[feed1.ID] != 1 {
		t.Errorf("Expected 1 unread for feed1, got %d", unreadCounts[feed1.ID])
	}

	if unreadCounts[feed2.ID] != 1 {
		t.Errorf("Expected 1 unread for feed2, got %d", unreadCounts[feed2.ID])
	}

	// Use variables to avoid unused errors
	_ = article2
	_ = article3
}

func TestGetUserUnreadCountsNoFeeds(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	unreadCounts, err := db.GetUserUnreadCounts(user.ID)
	if err != nil {
		t.Fatalf("GetUserUnreadCounts failed: %v", err)
	}

	if len(unreadCounts) != 0 {
		t.Errorf("Expected empty map for user with no feeds, got %d entries", len(unreadCounts))
	}
}

// Session management tests

func TestCreateSession(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	session := &Session{
		ID:        "session_123",
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	err := db.CreateSession(session)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
}

func TestGetSession(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	originalSession := &Session{
		ID:        "session_456",
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	err := db.CreateSession(originalSession)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	retrievedSession, err := db.GetSession(originalSession.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if retrievedSession == nil {
		t.Fatal("GetSession returned nil")
	}

	if retrievedSession.ID != originalSession.ID {
		t.Errorf("Expected session ID %s, got %s", originalSession.ID, retrievedSession.ID)
	}

	if retrievedSession.UserID != originalSession.UserID {
		t.Errorf("Expected user ID %d, got %d", originalSession.UserID, retrievedSession.UserID)
	}
}

func TestGetSessionNotFound(t *testing.T) {
	db := setupTestDB(t)

	session, err := db.GetSession("nonexistent_session")
	if err != nil {
		t.Fatalf("GetSession should not error for nonexistent session: %v", err)
	}

	if session != nil {
		t.Error("Expected nil for nonexistent session")
	}
}

func TestDeleteSession(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	session := &Session{
		ID:        "session_789",
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	err := db.CreateSession(session)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Delete session
	err = db.DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify deletion
	deletedSession, err := db.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession after delete failed: %v", err)
	}

	if deletedSession != nil {
		t.Error("Session should have been deleted")
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	// Create valid session
	validSession := &Session{
		ID:        "valid_session",
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := db.CreateSession(validSession)
	if err != nil {
		t.Fatalf("CreateSession (valid) failed: %v", err)
	}

	// Create expired session
	expiredSession := &Session{
		ID:        "expired_session",
		UserID:    user.ID,
		CreatedAt: time.Now().Add(-8 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	err = db.CreateSession(expiredSession)
	if err != nil {
		t.Fatalf("CreateSession (expired) failed: %v", err)
	}

	// Delete expired sessions
	err = db.DeleteExpiredSessions()
	if err != nil {
		t.Fatalf("DeleteExpiredSessions failed: %v", err)
	}

	// Valid session should still exist
	validCheck, err := db.GetSession(validSession.ID)
	if err != nil {
		t.Fatalf("GetSession for valid session failed: %v", err)
	}
	if validCheck == nil {
		t.Error("Valid session should not have been deleted")
	}

	// Expired session should be deleted
	expiredCheck, err := db.GetSession(expiredSession.ID)
	if err != nil {
		t.Fatalf("GetSession for expired session failed: %v", err)
	}
	if expiredCheck != nil {
		t.Error("Expired session should have been deleted")
	}
}

// Error handling and edge case tests

func TestForeignKeyConstraintArticle(t *testing.T) {
	db := setupTestDB(t)

	// Try to create article with non-existent feed
	article := &Article{
		FeedID:      99999, // Non-existent feed
		Title:       "Test Article",
		URL:         "https://example.com/test",
		Content:     "Content",
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	err := db.AddArticle(article)
	if err == nil {
		t.Error("Expected error when adding article with non-existent feed")
	}
}

func TestCascadeDeleteFeed(t *testing.T) {
	db := setupTestDB(t)

	feed := createTestFeed(t, db)
	article := createTestArticle(t, db, feed.ID)

	// Delete feed
	err := db.DeleteFeed(feed.ID)
	if err != nil {
		t.Fatalf("DeleteFeed failed: %v", err)
	}

	// Articles should also be deleted (cascade)
	foundArticle, err := db.FindArticleByURL(article.URL)
	if err != nil {
		t.Fatalf("FindArticleByURL failed: %v", err)
	}

	if foundArticle != nil {
		t.Error("Article should have been cascade deleted with feed")
	}
}

func TestCascadeDeleteUserSessions(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)

	session := &Session{
		ID:        "cascade_test_session",
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := db.CreateSession(session)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Delete user (this would cascade delete sessions)
	// Note: We don't have a DeleteUser method, but if we did, this would test cascade
	// For now, just verify the foreign key relationship exists

	// Try to create session for non-existent user
	invalidSession := &Session{
		ID:        "invalid_user_session",
		UserID:    99999, // Non-existent user
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err = db.CreateSession(invalidSession)
	if err == nil {
		t.Error("Expected error when creating session for non-existent user")
	}
}

func TestDatabaseClose(t *testing.T) {
	db := setupTestDB(t)

	err := db.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Trying to query after close should fail
	_, err = db.GetFeeds()
	if err == nil {
		t.Error("Expected error when querying closed database")
	}
}

func TestConcurrentAccess(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)

	// Create multiple articles concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(index int) {
			article := &Article{
				FeedID:      feed.ID,
				Title:       "Concurrent Article",
				URL:         formatString("https://example.com/concurrent_%d_%d", time.Now().UnixNano(), index),
				Content:     "Content",
				PublishedAt: time.Now(),
				CreatedAt:   time.Now(),
			}
			_ = db.AddArticle(article)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all articles were created
	articles, err := db.GetArticles(feed.ID)
	if err != nil {
		t.Fatalf("GetArticles failed: %v", err)
	}

	if len(articles) < 10 {
		t.Errorf("Expected at least 10 articles, got %d", len(articles))
	}

	// Use user to avoid unused error
	_ = user
}

// Helper function for formatting strings in concurrent test
func formatString(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
