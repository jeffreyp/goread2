package database

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// Tests for the per-user queries behind the GDPR data export, run against
// both the SQLite and Datastore implementations.

func TestGetUserArticleHistoryPage(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db)
	other := createTestUser(t, db)
	feed := createTestFeed(t, db)

	var articles []*Article
	for i := 0; i < 5; i++ {
		articles = append(articles, createTestArticle(t, db, feed.ID))
	}
	checkUserArticleHistoryPaging(t, db, user.ID, other.ID, articles)
}

func TestDatastoreGetUserArticleHistoryPage(t *testing.T) {
	db := setupTestDatastoreDB(t)
	user := createDatastoreTestUser(t, db)
	other := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)

	var articles []*Article
	for i := 0; i < 5; i++ {
		articles = append(articles, createDatastoreTestArticle(t, db, feed.ID))
	}
	checkUserArticleHistoryPaging(t, db, user.ID, other.ID, articles)
}

// checkUserArticleHistoryPaging gives userID a status on every article and
// otherID a status on one, then pages through userID's history two records at
// a time and checks that each record appears exactly once with its own flags.
func checkUserArticleHistoryPaging(t *testing.T, db Database, userID, otherID int, articles []*Article) {
	t.Helper()

	for i, article := range articles {
		if err := db.SetUserArticleStatus(userID, article.ID, i%2 == 0, i%2 == 1); err != nil {
			t.Fatalf("SetUserArticleStatus failed: %v", err)
		}
	}
	if err := db.SetUserArticleStatus(otherID, articles[0].ID, false, true); err != nil {
		t.Fatalf("SetUserArticleStatus for other user failed: %v", err)
	}

	got := map[int]UserArticleRecord{}
	cursor := ""
	for pages := 0; ; pages++ {
		if pages > len(articles) {
			t.Fatal("Paging did not terminate")
		}
		records, next, err := db.GetUserArticleHistoryPage(userID, cursor, 2)
		if err != nil {
			t.Fatalf("GetUserArticleHistoryPage failed: %v", err)
		}
		for _, record := range records {
			if _, dup := got[record.ArticleID]; dup {
				t.Errorf("Article %d returned twice", record.ArticleID)
			}
			got[record.ArticleID] = record
		}
		if next == "" {
			break
		}
		cursor = next
	}

	if len(got) != len(articles) {
		t.Fatalf("Expected %d records, got %d", len(articles), len(got))
	}
	for i, article := range articles {
		record := got[article.ID]
		if record.IsRead != (i%2 == 0) || record.IsStarred != (i%2 == 1) {
			t.Errorf("Article %d: unexpected flags %+v", article.ID, record)
		}
		if record.Title != article.Title || record.URL != article.URL || record.FeedID != article.FeedID {
			t.Errorf("Article %d: expected article fields to be joined, got %+v", article.ID, record)
		}
	}
}

func TestGetUserSessions(t *testing.T) {
	db := setupTestDB(t)
	checkGetUserSessions(t, db, createTestUser(t, db).ID, createTestUser(t, db).ID)
}

func TestDatastoreGetUserSessions(t *testing.T) {
	db := setupTestDatastoreDB(t)
	checkGetUserSessions(t, db, createDatastoreTestUser(t, db).ID, createDatastoreTestUser(t, db).ID)
}

func checkGetUserSessions(t *testing.T, db Database, userID, otherID int) {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Second)
	for i, owner := range []int{userID, otherID, userID} {
		session := &Session{
			ID:        fmt.Sprintf("user-sessions-test-%d-%d-%d", userID, otherID, i),
			UserID:    owner,
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
			ExpiresAt: now.Add(7 * 24 * time.Hour),
		}
		if err := db.CreateSession(session); err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}
	}

	sessions, err := db.GetUserSessions(userID)
	if err != nil {
		t.Fatalf("GetUserSessions failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(sessions))
	}
	for _, session := range sessions {
		if session.UserID != userID {
			t.Errorf("Got another user's session: %+v", session)
		}
	}
	if !sessions[0].CreatedAt.Before(sessions[1].CreatedAt) {
		t.Errorf("Expected sessions oldest first, got %v then %v", sessions[0].CreatedAt, sessions[1].CreatedAt)
	}
}

func TestDeleteUserAccount(t *testing.T) {
	db := setupTestDB(t)
	checkDeleteUserAccount(t, db, createTestUser(t, db), createTestUser(t, db), createTestFeed(t, db),
		func(feedID int) *Article { return createTestArticle(t, db, feedID) })
}

func TestDatastoreDeleteUserAccount(t *testing.T) {
	db := setupTestDatastoreDB(t)
	checkDeleteUserAccount(t, db, createDatastoreTestUser(t, db), createDatastoreTestUser(t, db), createDatastoreTestFeed(t, db),
		func(feedID int) *Article { return createDatastoreTestArticle(t, db, feedID) })
}

// checkDeleteUserAccount gives user and other the same feed, article states,
// sessions, and audit entries, deletes user, and checks that only user's data
// is gone, that audit entries are pseudonymized rather than deleted, and that
// a second deletion succeeds.
func checkDeleteUserAccount(t *testing.T, db Database, user, other *User, feed *Feed, newArticle func(feedID int) *Article) {
	t.Helper()

	article := newArticle(feed.ID)
	for _, u := range []*User{user, other} {
		if err := db.SubscribeUserToFeed(u.ID, feed.ID); err != nil {
			t.Fatalf("SubscribeUserToFeed failed: %v", err)
		}
		if err := db.SetUserArticleStatus(u.ID, article.ID, true, true); err != nil {
			t.Fatalf("SetUserArticleStatus failed: %v", err)
		}
		if err := db.CreateSession(&Session{
			ID:        fmt.Sprintf("delete-account-test-%d", u.ID),
			UserID:    u.ID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}); err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}
	}
	// One entry with user as the target of another admin's action, and one
	// with user as the admin acting on other.
	for _, entry := range []*AuditLog{
		{AdminUserID: other.ID, AdminEmail: other.Email, TargetUserID: user.ID, TargetUserEmail: user.Email, IPAddress: "198.51.100.1"},
		{AdminUserID: user.ID, AdminEmail: user.Email, TargetUserID: other.ID, TargetUserEmail: other.Email, IPAddress: "198.51.100.2"},
	} {
		entry.Timestamp = time.Now()
		entry.OperationType = "grant_free_months"
		entry.Result = "success"
		if err := db.CreateAuditLog(entry); err != nil {
			t.Fatalf("CreateAuditLog failed: %v", err)
		}
	}

	if err := db.DeleteUserAccount(user.ID); err != nil {
		t.Fatalf("DeleteUserAccount failed: %v", err)
	}

	if _, err := db.GetUserByID(user.ID); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound for deleted user, got %v", err)
	}
	if feeds, err := db.GetUserFeeds(user.ID); err != nil || len(feeds) != 0 {
		t.Errorf("Expected no feeds for deleted user, got %d (err %v)", len(feeds), err)
	}
	if sessions, err := db.GetUserSessions(user.ID); err != nil || len(sessions) != 0 {
		t.Errorf("Expected no sessions for deleted user, got %d (err %v)", len(sessions), err)
	}
	if records, _, err := db.GetUserArticleHistoryPage(user.ID, "", 10); err != nil || len(records) != 0 {
		t.Errorf("Expected no article history for deleted user, got %d (err %v)", len(records), err)
	}

	// The other user's data and the shared feed are untouched.
	if _, err := db.GetUserByID(other.ID); err != nil {
		t.Errorf("Other user should still exist: %v", err)
	}
	if feeds, err := db.GetUserFeeds(other.ID); err != nil || len(feeds) != 1 {
		t.Errorf("Expected other user's feed to remain, got %d (err %v)", len(feeds), err)
	}
	if records, _, err := db.GetUserArticleHistoryPage(other.ID, "", 10); err != nil || len(records) != 1 {
		t.Errorf("Expected other user's article history to remain, got %d (err %v)", len(records), err)
	}
	if sessions, err := db.GetUserSessions(other.ID); err != nil || len(sessions) != 1 {
		t.Errorf("Expected other user's session to remain, got %d (err %v)", len(sessions), err)
	}

	pseudonym := DeletedUserPseudonym(user.ID)
	asTarget, err := db.GetAuditLogs(10, 0, map[string]interface{}{"target_user_id": user.ID})
	if err != nil || len(asTarget) != 1 {
		t.Fatalf("Expected 1 retained audit entry targeting user, got %d (err %v)", len(asTarget), err)
	}
	if asTarget[0].TargetUserEmail != pseudonym || asTarget[0].IPAddress != "198.51.100.1" || asTarget[0].AdminEmail != other.Email {
		t.Errorf("Target entry should replace only the deleted user's email, got %+v", asTarget[0])
	}
	asActor, err := db.GetAuditLogs(10, 0, map[string]interface{}{"admin_user_id": user.ID})
	if err != nil || len(asActor) != 1 {
		t.Fatalf("Expected 1 retained audit entry by user, got %d (err %v)", len(asActor), err)
	}
	if asActor[0].AdminEmail != pseudonym || asActor[0].IPAddress != "" || asActor[0].TargetUserEmail != other.Email {
		t.Errorf("Actor entry should replace the deleted user's email and IP, got %+v", asActor[0])
	}

	if err := db.DeleteUserAccount(user.ID); err != nil {
		t.Errorf("Deleting an already deleted account should succeed, got %v", err)
	}
}

func TestAccountTombstone(t *testing.T) {
	checkAccountTombstone(t, setupTestDB(t))
}

func TestDatastoreAccountTombstone(t *testing.T) {
	checkAccountTombstone(t, setupTestDatastoreDB(t))
}

func checkAccountTombstone(t *testing.T, db Database) {
	t.Helper()

	id := fmt.Sprintf("tombstone-test-%d", time.Now().UnixNano())
	if found, err := db.HasAccountTombstone(id); err != nil || found {
		t.Fatalf("Expected no tombstone before creation, got %v (err %v)", found, err)
	}
	for i := 0; i < 2; i++ { // creating twice is not an error
		if err := db.CreateAccountTombstone(id); err != nil {
			t.Fatalf("CreateAccountTombstone failed: %v", err)
		}
	}
	if found, err := db.HasAccountTombstone(id); err != nil || !found {
		t.Errorf("Expected tombstone after creation, got %v (err %v)", found, err)
	}
}
