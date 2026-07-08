package database

import (
	"errors"
	"testing"
	"time"
)

// User article status tests

func TestDatastoreGetUserArticleStatus(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	article := createDatastoreTestArticle(t, db, feed.ID)

	_, err := db.GetUserArticleStatus(user.ID, article.ID)
	if err == nil {
		t.Error("Expected error for status that has never been set")
	}

	if err := db.SetUserArticleStatus(user.ID, article.ID, true, true); err != nil {
		t.Fatalf("SetUserArticleStatus failed: %v", err)
	}

	status, err := db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetUserArticleStatus failed: %v", err)
	}
	if !status.IsRead || !status.IsStarred {
		t.Errorf("Expected read+starred status, got %+v", status)
	}
}

func TestDatastoreMarkUserArticleRead(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	article := createDatastoreTestArticle(t, db, feed.ID)

	// Starring first, then marking read, should preserve the starred flag.
	if err := db.ToggleUserArticleStar(user.ID, article.ID); err != nil {
		t.Fatalf("ToggleUserArticleStar failed: %v", err)
	}
	if err := db.MarkUserArticleRead(user.ID, article.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	status, err := db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetUserArticleStatus failed: %v", err)
	}
	if !status.IsRead {
		t.Error("Expected article to be marked read")
	}
	if !status.IsStarred {
		t.Error("Expected starred flag to be preserved by MarkUserArticleRead")
	}

	if err := db.MarkUserArticleRead(user.ID, article.ID, false); err != nil {
		t.Fatalf("MarkUserArticleRead (unread) failed: %v", err)
	}
	status, err = db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetUserArticleStatus failed: %v", err)
	}
	if status.IsRead {
		t.Error("Expected article to be marked unread")
	}
}

func TestDatastoreToggleUserArticleStar(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	article := createDatastoreTestArticle(t, db, feed.ID)

	if err := db.MarkUserArticleRead(user.ID, article.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	if err := db.ToggleUserArticleStar(user.ID, article.ID); err != nil {
		t.Fatalf("ToggleUserArticleStar failed: %v", err)
	}
	status, err := db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetUserArticleStatus failed: %v", err)
	}
	if !status.IsStarred {
		t.Error("Expected article to be starred after first toggle")
	}
	if !status.IsRead {
		t.Error("Expected read flag to be preserved by ToggleUserArticleStar")
	}

	if err := db.ToggleUserArticleStar(user.ID, article.ID); err != nil {
		t.Fatalf("ToggleUserArticleStar (second) failed: %v", err)
	}
	status, err = db.GetUserArticleStatus(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetUserArticleStatus failed: %v", err)
	}
	if status.IsStarred {
		t.Error("Expected article to be unstarred after second toggle")
	}
}

func TestDatastoreBatchSetUserArticleStatus(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	article1 := createDatastoreTestArticle(t, db, feed.ID)
	article2 := createDatastoreTestArticle(t, db, feed.ID)

	if err := db.BatchSetUserArticleStatus(user.ID, []Article{*article1, *article2}, true, false); err != nil {
		t.Fatalf("BatchSetUserArticleStatus failed: %v", err)
	}

	for _, id := range []int{article1.ID, article2.ID} {
		status, err := db.GetUserArticleStatus(user.ID, id)
		if err != nil {
			t.Fatalf("GetUserArticleStatus(%d) failed: %v", id, err)
		}
		if !status.IsRead {
			t.Errorf("Expected article %d to be marked read", id)
		}
	}

	// Empty input should be a no-op, not an error.
	if err := db.BatchSetUserArticleStatus(user.ID, nil, true, true); err != nil {
		t.Fatalf("BatchSetUserArticleStatus (empty) failed: %v", err)
	}
}

func TestDatastoreMarkAllUserArticlesRead(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	article1 := createDatastoreTestArticle(t, db, feed.ID)
	article2 := createDatastoreTestArticle(t, db, feed.ID)

	count, err := db.MarkAllUserArticlesRead(user.ID)
	if err != nil {
		t.Fatalf("MarkAllUserArticlesRead failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 articles marked read, got %d", count)
	}

	for _, id := range []int{article1.ID, article2.ID} {
		status, err := db.GetUserArticleStatus(user.ID, id)
		if err != nil {
			t.Fatalf("GetUserArticleStatus(%d) failed: %v", id, err)
		}
		if !status.IsRead {
			t.Errorf("Expected article %d to be marked read", id)
		}
	}
}

func TestDatastoreGetUserUnreadCounts(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	article1 := createDatastoreTestArticle(t, db, feed.ID)
	_ = createDatastoreTestArticle(t, db, feed.ID)

	if err := db.MarkUserArticleRead(user.ID, article1.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	counts, err := db.GetUserUnreadCounts(user.ID)
	if err != nil {
		t.Fatalf("GetUserUnreadCounts failed: %v", err)
	}
	if counts[feed.ID] != 1 {
		t.Errorf("Expected 1 unread article for feed %d, got %d", feed.ID, counts[feed.ID])
	}
}

func TestDatastoreGetTotalArticleCount(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	createDatastoreTestArticle(t, db, feed.ID)
	createDatastoreTestArticle(t, db, feed.ID)

	count, err := db.GetTotalArticleCount(user.ID)
	if err != nil {
		t.Fatalf("GetTotalArticleCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected total article count 2, got %d", count)
	}
}

func TestDatastoreGetAccountStats(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	article1 := createDatastoreTestArticle(t, db, feed.ID)
	createDatastoreTestArticle(t, db, feed.ID)

	if err := db.MarkUserArticleRead(user.ID, article1.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	stats, err := db.GetAccountStats(user.ID)
	if err != nil {
		t.Fatalf("GetAccountStats failed: %v", err)
	}
	if stats["total_articles"] != 2 {
		t.Errorf("Expected total_articles 2, got %v", stats["total_articles"])
	}
	if stats["total_unread"] != 1 {
		t.Errorf("Expected total_unread 1, got %v", stats["total_unread"])
	}
	if stats["active_feeds"] != 1 {
		t.Errorf("Expected active_feeds 1, got %v", stats["active_feeds"])
	}
}

func TestDatastoreCleanupOrphanedUserArticles(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	article := createDatastoreTestArticle(t, db, feed.ID)

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.MarkUserArticleRead(user.ID, article.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}
	if err := db.UnsubscribeUserFromFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("UnsubscribeUserFromFeed failed: %v", err)
	}

	deleted, err := db.CleanupOrphanedUserArticles(0)
	if err != nil {
		t.Fatalf("CleanupOrphanedUserArticles failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Expected 1 orphaned UserArticle deleted, got %d", deleted)
	}

	_, err = db.GetUserArticleStatus(user.ID, article.ID)
	if err == nil {
		t.Error("Expected orphaned UserArticle status to be gone after cleanup")
	}
}

// Subscription / admin tests

func TestDatastoreUpdateUserSubscription(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)

	lastPayment := time.Now().Add(-24 * time.Hour).Truncate(time.Microsecond)
	nextBilling := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Microsecond)
	err := db.UpdateUserSubscription(user.ID, "active", "sub_12345", lastPayment, nextBilling)
	if err != nil {
		t.Fatalf("UpdateUserSubscription failed: %v", err)
	}

	got, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if got.SubscriptionStatus != "active" {
		t.Errorf("Expected status 'active', got %q", got.SubscriptionStatus)
	}
	if got.SubscriptionID != "sub_12345" {
		t.Errorf("Expected subscription ID 'sub_12345', got %q", got.SubscriptionID)
	}
	if !got.LastPaymentDate.Equal(lastPayment) {
		t.Errorf("Expected LastPaymentDate %v, got %v", lastPayment, got.LastPaymentDate)
	}
	if !got.NextBillingDate.Equal(nextBilling) {
		t.Errorf("Expected NextBillingDate %v, got %v", nextBilling, got.NextBillingDate)
	}
}

func TestDatastoreIsUserSubscriptionActive(t *testing.T) {
	db := setupTestDatastoreDB(t)

	t.Run("AdminBypassesSubscription", func(t *testing.T) {
		user := createDatastoreTestUser(t, db)
		if err := db.SetUserAdmin(user.ID, true); err != nil {
			t.Fatalf("SetUserAdmin failed: %v", err)
		}
		active, err := db.IsUserSubscriptionActive(user.ID)
		if err != nil {
			t.Fatalf("IsUserSubscriptionActive failed: %v", err)
		}
		if !active {
			t.Error("Expected admin user to be active")
		}
	})

	t.Run("ActivePaidSubscription", func(t *testing.T) {
		user := createDatastoreTestUser(t, db)
		if err := db.UpdateUserSubscription(user.ID, "active", "sub_x", time.Now(), time.Now()); err != nil {
			t.Fatalf("UpdateUserSubscription failed: %v", err)
		}
		active, err := db.IsUserSubscriptionActive(user.ID)
		if err != nil {
			t.Fatalf("IsUserSubscriptionActive failed: %v", err)
		}
		if !active {
			t.Error("Expected active subscription user to be active")
		}
	})

	t.Run("UnexpiredTrial", func(t *testing.T) {
		user := createDatastoreTestUser(t, db)
		active, err := db.IsUserSubscriptionActive(user.ID)
		if err != nil {
			t.Fatalf("IsUserSubscriptionActive failed: %v", err)
		}
		if !active {
			t.Error("Expected user on unexpired trial to be active")
		}
	})

	t.Run("ExpiredTrialNoFreeMonths", func(t *testing.T) {
		user := &User{
			GoogleID:           "expired_trial_google_id",
			Email:              "expired_trial@example.com",
			Name:               "Expired Trial User",
			CreatedAt:          time.Now(),
			SubscriptionStatus: "trial",
			TrialEndsAt:        time.Now().Add(-24 * time.Hour),
		}
		if err := db.CreateUser(user); err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		active, err := db.IsUserSubscriptionActive(user.ID)
		if err != nil {
			t.Fatalf("IsUserSubscriptionActive failed: %v", err)
		}
		if active {
			t.Error("Expected user with expired trial and no free months to be inactive")
		}
	})

	t.Run("FreeMonthsRemaining", func(t *testing.T) {
		user := &User{
			GoogleID:           "free_months_google_id",
			Email:              "free_months@example.com",
			Name:               "Free Months User",
			CreatedAt:          time.Now(),
			SubscriptionStatus: "expired",
			TrialEndsAt:        time.Now().Add(-24 * time.Hour),
		}
		if err := db.CreateUser(user); err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		if err := db.GrantFreeMonths(user.ID, 1); err != nil {
			t.Fatalf("GrantFreeMonths failed: %v", err)
		}
		active, err := db.IsUserSubscriptionActive(user.ID)
		if err != nil {
			t.Fatalf("IsUserSubscriptionActive failed: %v", err)
		}
		if !active {
			t.Error("Expected user with free months remaining to be active")
		}
	})
}

func TestDatastoreUpdateUserMaxArticlesOnFeedAdd(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)

	if err := db.UpdateUserMaxArticlesOnFeedAdd(user.ID, 250); err != nil {
		t.Fatalf("UpdateUserMaxArticlesOnFeedAdd failed: %v", err)
	}
	got, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if got.MaxArticlesOnFeedAdd != 250 {
		t.Errorf("Expected MaxArticlesOnFeedAdd 250, got %d", got.MaxArticlesOnFeedAdd)
	}

	if err := db.UpdateUserMaxArticlesOnFeedAdd(user.ID, 0); err == nil {
		t.Error("Expected error for MaxArticlesOnFeedAdd of 0")
	}
	if err := db.UpdateUserMaxArticlesOnFeedAdd(user.ID, 10001); err == nil {
		t.Error("Expected error for MaxArticlesOnFeedAdd over 10000")
	}
}

func TestDatastoreSetUserAdmin(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)

	if err := db.SetUserAdmin(user.ID, true); err != nil {
		t.Fatalf("SetUserAdmin failed: %v", err)
	}
	got, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if !got.IsAdmin {
		t.Error("Expected user to be admin")
	}

	if err := db.SetUserAdmin(user.ID, false); err != nil {
		t.Fatalf("SetUserAdmin (revoke) failed: %v", err)
	}
	got, err = db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if got.IsAdmin {
		t.Error("Expected user to no longer be admin")
	}
}

func TestDatastoreSetUserAdminAtomic(t *testing.T) {
	db := setupTestDatastoreDB(t)

	admin := createDatastoreTestUser(t, db)
	target := createDatastoreTestUser(t, db)

	if err := db.SetUserAdminAtomic(target.ID, admin.ID, true); err != nil {
		t.Fatalf("SetUserAdminAtomic failed: %v", err)
	}
	got, err := db.GetUserByID(target.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if !got.IsAdmin {
		t.Error("Expected target user to be admin")
	}

	err = db.SetUserAdminAtomic(admin.ID, admin.ID, false)
	if !errors.Is(err, ErrSelfDemotion) {
		t.Errorf("Expected ErrSelfDemotion for self-demotion, got %v", err)
	}
}

func TestDatastoreGrantFreeMonths(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)

	if err := db.GrantFreeMonths(user.ID, 2); err != nil {
		t.Fatalf("GrantFreeMonths failed: %v", err)
	}
	if err := db.GrantFreeMonths(user.ID, 3); err != nil {
		t.Fatalf("GrantFreeMonths (second grant) failed: %v", err)
	}

	got, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if got.FreeMonthsRemaining != 5 {
		t.Errorf("Expected FreeMonthsRemaining 5, got %d", got.FreeMonthsRemaining)
	}
}

// Session tests

func TestDatastoreCreateAndGetSession(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	session := &Session{
		ID:        "session-token-1",
		UserID:    user.ID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
		ExpiresAt: time.Now().Add(24 * time.Hour).Truncate(time.Microsecond),
	}
	if err := db.CreateSession(session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	got, err := db.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if got == nil {
		t.Fatal("Expected session to be found")
	}
	if got.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, got.UserID)
	}
	if !got.ExpiresAt.Equal(session.ExpiresAt) {
		t.Errorf("Expected ExpiresAt %v, got %v", session.ExpiresAt, got.ExpiresAt)
	}

	notFound, err := db.GetSession("nonexistent-session")
	if err != nil {
		t.Fatalf("GetSession (not found) failed: %v", err)
	}
	if notFound != nil {
		t.Errorf("Expected nil for nonexistent session, got %+v", notFound)
	}
}

func TestDatastoreUpdateSessionExpiry(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	session := &Session{
		ID:        "session-token-2",
		UserID:    user.ID,
		CreatedAt: time.Now().Truncate(time.Microsecond),
		ExpiresAt: time.Now().Add(time.Hour).Truncate(time.Microsecond),
	}
	if err := db.CreateSession(session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	newExpiry := time.Now().Add(48 * time.Hour).Truncate(time.Microsecond)
	if err := db.UpdateSessionExpiry(session.ID, newExpiry); err != nil {
		t.Fatalf("UpdateSessionExpiry failed: %v", err)
	}

	got, err := db.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if !got.ExpiresAt.Equal(newExpiry) {
		t.Errorf("Expected ExpiresAt %v, got %v", newExpiry, got.ExpiresAt)
	}
	if got.UserID != user.ID {
		t.Errorf("Expected UserID %d to be preserved, got %d", user.ID, got.UserID)
	}
}

func TestDatastoreDeleteSession(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	session := &Session{
		ID:        "session-token-3",
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.CreateSession(session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if err := db.DeleteSession(session.ID); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	got, err := db.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if got != nil {
		t.Errorf("Expected session to be deleted, got %+v", got)
	}
}

func TestDatastoreDeleteExpiredSessions(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	expired := &Session{
		ID:        "session-expired",
		UserID:    user.ID,
		CreatedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour),
	}
	active := &Session{
		ID:        "session-active",
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := db.CreateSession(expired); err != nil {
		t.Fatalf("CreateSession (expired) failed: %v", err)
	}
	if err := db.CreateSession(active); err != nil {
		t.Fatalf("CreateSession (active) failed: %v", err)
	}

	if err := db.DeleteExpiredSessions(); err != nil {
		t.Fatalf("DeleteExpiredSessions failed: %v", err)
	}

	gotExpired, err := db.GetSession(expired.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if gotExpired != nil {
		t.Errorf("Expected expired session to be deleted, got %+v", gotExpired)
	}

	gotActive, err := db.GetSession(active.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if gotActive == nil {
		t.Error("Expected active session to remain")
	}
}

// Audit log tests

func TestDatastoreCreateAndGetAuditLogs(t *testing.T) {
	db := setupTestDatastoreDB(t)

	admin := createDatastoreTestUser(t, db)
	target := createDatastoreTestUser(t, db)

	log1 := &AuditLog{
		Timestamp:     time.Now().Add(-time.Hour),
		AdminUserID:   admin.ID,
		AdminEmail:    admin.Email,
		OperationType: "grant_admin",
		TargetUserID:  target.ID,
		Result:        "success",
	}
	log2 := &AuditLog{
		Timestamp:     time.Now(),
		AdminUserID:   admin.ID,
		AdminEmail:    admin.Email,
		OperationType: "revoke_admin",
		TargetUserID:  target.ID,
		Result:        "success",
	}
	if err := db.CreateAuditLog(log1); err != nil {
		t.Fatalf("CreateAuditLog failed: %v", err)
	}
	if log1.ID <= 0 {
		t.Errorf("Expected positive audit log ID, got %d", log1.ID)
	}
	if err := db.CreateAuditLog(log2); err != nil {
		t.Fatalf("CreateAuditLog failed: %v", err)
	}

	logs, err := db.GetAuditLogs(10, 0, nil)
	if err != nil {
		t.Fatalf("GetAuditLogs failed: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("Expected 2 audit logs, got %d", len(logs))
	}
	// Ordered by timestamp DESC (newest first).
	if logs[0].OperationType != "revoke_admin" {
		t.Errorf("Expected newest log first (revoke_admin), got %q", logs[0].OperationType)
	}

	filtered, err := db.GetAuditLogs(10, 0, map[string]interface{}{"operation_type": "grant_admin"})
	if err != nil {
		t.Fatalf("GetAuditLogs (filtered) failed: %v", err)
	}
	if len(filtered) != 1 || filtered[0].OperationType != "grant_admin" {
		t.Errorf("Expected 1 filtered log for grant_admin, got %+v", filtered)
	}
}
