package database

import (
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
