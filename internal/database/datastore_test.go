package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/datastore"
)

// setupTestDatastoreDB creates a DatastoreDB backed by the local Datastore emulator.
// Skips (rather than fails) when DATASTORE_EMULATOR_HOST is unset, matching
// internal/services/admin_token_datastore_test.go so `go test ./...` stays green without
// the emulator running. Each test gets its own project ID, which the emulator treats as an
// isolated dataset, so tests do not see each other's entities.
func setupTestDatastoreDB(t *testing.T) *DatastoreDB {
	t.Helper()

	if os.Getenv("DATASTORE_EMULATOR_HOST") == "" {
		t.Skip("Datastore emulator not available - set DATASTORE_EMULATOR_HOST to run datastore tests")
	}

	projectID := fmt.Sprintf("test-project-%d", time.Now().UnixNano())
	db, err := NewDatastoreDB(projectID)
	if err != nil {
		t.Fatalf("Failed to create datastore DB: %v", err)
	}

	t.Cleanup(func() {
		cleanupDatastoreEntities(t, db)
		if err := db.Close(); err != nil {
			t.Logf("Failed to close datastore DB: %v", err)
		}
	})

	return db
}

// cleanupDatastoreEntities best-effort deletes all entities created during a test. Each test
// already runs in its own project (see setupTestDatastoreDB), so this mainly bounds emulator
// memory growth across a long test run rather than guarding against cross-test pollution.
func cleanupDatastoreEntities(t *testing.T, db *DatastoreDB) {
	t.Helper()

	ctx := context.Background()
	client := db.GetClient()

	kinds := []string{"User", "Feed", "Article", "UserFeed", "UserArticle", "Session", "AuditLog", "AdminToken"}
	for _, kind := range kinds {
		query := datastore.NewQuery(kind).KeysOnly()
		keys, err := client.GetAll(ctx, query, nil)
		if err != nil {
			t.Logf("Failed to query %s keys for cleanup: %v", kind, err)
			continue
		}
		if len(keys) > 0 {
			if err := client.DeleteMulti(ctx, keys); err != nil {
				t.Logf("Failed to delete %s entities: %v", kind, err)
			}
		}
	}
}

func createDatastoreTestUser(t *testing.T, db *DatastoreDB) *User {
	t.Helper()

	user := &User{
		GoogleID:             fmt.Sprintf("test_google_id_%d", time.Now().UnixNano()),
		Email:                fmt.Sprintf("test%d@example.com", time.Now().UnixNano()),
		Name:                 "Test User",
		Avatar:               "https://example.com/avatar.jpg",
		CreatedAt:            time.Now(),
		SubscriptionStatus:   "trial",
		TrialEndsAt:          time.Now().AddDate(0, 0, 30),
		MaxArticlesOnFeedAdd: 100,
	}

	if err := db.CreateUser(user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

func createDatastoreTestFeed(t *testing.T, db *DatastoreDB) *Feed {
	t.Helper()

	now := time.Now()
	feed := &Feed{
		Title:                 "Test Feed",
		URL:                   fmt.Sprintf("https://example.com/feed_%d.xml", now.UnixNano()),
		Description:           "A test feed",
		CreatedAt:             now,
		UpdatedAt:             now,
		LastFetch:             now,
		LastChecked:           now,
		LastHadNewContent:     now,
		AverageUpdateInterval: 0,
	}

	if err := db.AddFeed(feed); err != nil {
		t.Fatalf("Failed to create test feed: %v", err)
	}

	return feed
}

func createDatastoreTestArticle(t *testing.T, db *DatastoreDB, feedID int) *Article {
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

	if err := db.AddArticle(article); err != nil {
		t.Fatalf("Failed to create test article: %v", err)
	}

	return article
}

// Feed tests

func TestDatastoreAddFeed(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)
	if feed.ID <= 0 {
		t.Errorf("Expected positive feed ID, got %d", feed.ID)
	}

	got, err := db.GetFeedByID(feed.ID)
	if err != nil {
		t.Fatalf("GetFeedByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("Expected feed to be found")
	}
	if got.Title != feed.Title || got.URL != feed.URL {
		t.Errorf("Feed mismatch: got %+v, want title=%s url=%s", got, feed.Title, feed.URL)
	}
}

func TestDatastoreUpdateFeed(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)

	feed.Title = "Updated Title"
	feed.Description = "Updated Description"
	if err := db.UpdateFeed(feed); err != nil {
		t.Fatalf("UpdateFeed failed: %v", err)
	}

	got, err := db.GetFeedByID(feed.ID)
	if err != nil {
		t.Fatalf("GetFeedByID failed: %v", err)
	}
	if got.Title != "Updated Title" {
		t.Errorf("Expected updated title, got %s", got.Title)
	}
	if got.Description != "Updated Description" {
		t.Errorf("Expected updated description, got %s", got.Description)
	}
	// URL should be preserved since UpdateFeed doesn't touch it.
	if got.URL != feed.URL {
		t.Errorf("Expected URL to be preserved, got %s", got.URL)
	}
}

func TestDatastoreUpdateFeedTracking(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)

	lastChecked := time.Now().Add(time.Hour).Truncate(time.Microsecond)
	lastHadNewContent := time.Now().Add(2 * time.Hour).Truncate(time.Microsecond)
	if err := db.UpdateFeedTracking(feed.ID, lastChecked, lastHadNewContent, 3600); err != nil {
		t.Fatalf("UpdateFeedTracking failed: %v", err)
	}

	got, err := db.GetFeedByID(feed.ID)
	if err != nil {
		t.Fatalf("GetFeedByID failed: %v", err)
	}
	if !got.LastChecked.Equal(lastChecked) {
		t.Errorf("Expected LastChecked %v, got %v", lastChecked, got.LastChecked)
	}
	if !got.LastHadNewContent.Equal(lastHadNewContent) {
		t.Errorf("Expected LastHadNewContent %v, got %v", lastHadNewContent, got.LastHadNewContent)
	}
	if got.AverageUpdateInterval != 3600 {
		t.Errorf("Expected AverageUpdateInterval 3600, got %d", got.AverageUpdateInterval)
	}

	// A zero lastHadNewContent should not overwrite the previously stored value.
	if err := db.UpdateFeedTracking(feed.ID, lastChecked.Add(time.Hour), time.Time{}, 7200); err != nil {
		t.Fatalf("UpdateFeedTracking (zero lastHadNewContent) failed: %v", err)
	}
	got2, err := db.GetFeedByID(feed.ID)
	if err != nil {
		t.Fatalf("GetFeedByID failed: %v", err)
	}
	if !got2.LastHadNewContent.Equal(lastHadNewContent) {
		t.Errorf("Expected LastHadNewContent to remain %v, got %v", lastHadNewContent, got2.LastHadNewContent)
	}
}

func TestDatastoreUpdateFeedCacheHeaders(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)

	if err := db.UpdateFeedCacheHeaders(feed.ID, `"etag-123"`, "Wed, 21 Oct 2026 07:28:00 GMT"); err != nil {
		t.Fatalf("UpdateFeedCacheHeaders failed: %v", err)
	}

	got, err := db.GetFeedByID(feed.ID)
	if err != nil {
		t.Fatalf("GetFeedByID failed: %v", err)
	}
	if got.ETag != `"etag-123"` {
		t.Errorf("Expected ETag to be set, got %q", got.ETag)
	}
	if got.LastModified != "Wed, 21 Oct 2026 07:28:00 GMT" {
		t.Errorf("Expected LastModified to be set, got %q", got.LastModified)
	}
}

func TestDatastoreUpdateFeedLastFetch(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)

	newFetch := time.Now().Add(time.Hour).Truncate(time.Microsecond)
	if err := db.UpdateFeedLastFetch(feed.ID, newFetch); err != nil {
		t.Fatalf("UpdateFeedLastFetch failed: %v", err)
	}

	got, err := db.GetFeedByID(feed.ID)
	if err != nil {
		t.Fatalf("GetFeedByID failed: %v", err)
	}
	if !got.LastFetch.Equal(newFetch) {
		t.Errorf("Expected LastFetch %v, got %v", newFetch, got.LastFetch)
	}
}

func TestDatastoreUpdateFeedAfterRefresh(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)

	lastChecked := time.Now().Add(time.Hour).Truncate(time.Microsecond)
	lastHadNewContent := time.Now().Add(2 * time.Hour).Truncate(time.Microsecond)
	lastFetch := time.Now().Add(3 * time.Hour).Truncate(time.Microsecond)
	err := db.UpdateFeedAfterRefresh(feed.ID, lastChecked, lastHadNewContent, 1800, lastFetch, `"etag-abc"`, "some-last-modified")
	if err != nil {
		t.Fatalf("UpdateFeedAfterRefresh failed: %v", err)
	}

	got, err := db.GetFeedByID(feed.ID)
	if err != nil {
		t.Fatalf("GetFeedByID failed: %v", err)
	}
	if !got.LastChecked.Equal(lastChecked) {
		t.Errorf("Expected LastChecked %v, got %v", lastChecked, got.LastChecked)
	}
	if !got.LastHadNewContent.Equal(lastHadNewContent) {
		t.Errorf("Expected LastHadNewContent %v, got %v", lastHadNewContent, got.LastHadNewContent)
	}
	if !got.LastFetch.Equal(lastFetch) {
		t.Errorf("Expected LastFetch %v, got %v", lastFetch, got.LastFetch)
	}
	if got.AverageUpdateInterval != 1800 {
		t.Errorf("Expected AverageUpdateInterval 1800, got %d", got.AverageUpdateInterval)
	}
	if got.ETag != `"etag-abc"` {
		t.Errorf("Expected ETag to be set, got %q", got.ETag)
	}
	if got.LastModified != "some-last-modified" {
		t.Errorf("Expected LastModified to be set, got %q", got.LastModified)
	}
}

func TestDatastoreGetFeeds(t *testing.T) {
	db := setupTestDatastoreDB(t)

	now := time.Now()
	titles := []string{"Alpha Feed", "Bravo Feed", "Charlie Feed"}
	for _, title := range titles {
		feed := &Feed{
			Title:     title,
			URL:       fmt.Sprintf("https://example.com/%s_%d", title, now.UnixNano()),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := db.AddFeed(feed); err != nil {
			t.Fatalf("AddFeed failed: %v", err)
		}
	}

	feeds, err := db.GetFeeds()
	if err != nil {
		t.Fatalf("GetFeeds failed: %v", err)
	}
	if len(feeds) != len(titles) {
		t.Fatalf("Expected %d feeds, got %d", len(titles), len(feeds))
	}
	for i, title := range titles {
		if feeds[i].Title != title {
			t.Errorf("Expected feeds ordered by title; feeds[%d].Title = %q, want %q", i, feeds[i].Title, title)
		}
	}
}

func TestDatastoreGetFeedByURL(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)

	got, err := db.GetFeedByURL(feed.URL)
	if err != nil {
		t.Fatalf("GetFeedByURL failed: %v", err)
	}
	if got == nil || got.ID != feed.ID {
		t.Fatalf("Expected to find feed %d, got %+v", feed.ID, got)
	}

	notFound, err := db.GetFeedByURL("https://example.com/does-not-exist.xml")
	if err != nil {
		t.Fatalf("GetFeedByURL (not found) failed: %v", err)
	}
	if notFound != nil {
		t.Errorf("Expected nil for unknown URL, got %+v", notFound)
	}
}

func TestDatastoreGetFeedByID(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)

	got, err := db.GetFeedByID(feed.ID)
	if err != nil {
		t.Fatalf("GetFeedByID failed: %v", err)
	}
	if got == nil || got.ID != feed.ID {
		t.Fatalf("Expected to find feed %d, got %+v", feed.ID, got)
	}

	notFound, err := db.GetFeedByID(feed.ID + 999999)
	if err != nil {
		t.Fatalf("GetFeedByID (not found) failed: %v", err)
	}
	if notFound != nil {
		t.Errorf("Expected nil for unknown feed ID, got %+v", notFound)
	}
}

func TestDatastoreDeleteFeed(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)
	article := createDatastoreTestArticle(t, db, feed.ID)

	if err := db.DeleteFeed(feed.ID); err != nil {
		t.Fatalf("DeleteFeed failed: %v", err)
	}

	got, err := db.GetFeedByID(feed.ID)
	if err != nil {
		t.Fatalf("GetFeedByID failed: %v", err)
	}
	if got != nil {
		t.Errorf("Expected feed to be deleted, got %+v", got)
	}

	articles, err := db.GetArticles(feed.ID)
	if err != nil {
		t.Fatalf("GetArticles failed: %v", err)
	}
	for _, a := range articles {
		if a.ID == article.ID {
			t.Errorf("Expected article %d to be deleted along with its feed", article.ID)
		}
	}
}

// Article tests

func TestDatastoreAddArticle(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)
	article := createDatastoreTestArticle(t, db, feed.ID)
	if article.ID <= 0 {
		t.Fatalf("Expected positive article ID, got %d", article.ID)
	}

	// Adding an article with the same URL again should be idempotent and return the same ID.
	dup := &Article{
		FeedID:      feed.ID,
		Title:       "Duplicate Article",
		URL:         article.URL,
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
	}
	if err := db.AddArticle(dup); err != nil {
		t.Fatalf("AddArticle (duplicate URL) failed: %v", err)
	}
	if dup.ID != article.ID {
		t.Errorf("Expected duplicate URL to resolve to existing article ID %d, got %d", article.ID, dup.ID)
	}
}

func TestDatastoreGetArticles(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)
	older := &Article{
		FeedID:      feed.ID,
		Title:       "Older",
		URL:         fmt.Sprintf("https://example.com/older_%d", time.Now().UnixNano()),
		PublishedAt: time.Now().Add(-time.Hour),
		CreatedAt:   time.Now(),
	}
	newer := &Article{
		FeedID:      feed.ID,
		Title:       "Newer",
		URL:         fmt.Sprintf("https://example.com/newer_%d", time.Now().UnixNano()),
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
	}
	if err := db.AddArticle(older); err != nil {
		t.Fatalf("AddArticle failed: %v", err)
	}
	if err := db.AddArticle(newer); err != nil {
		t.Fatalf("AddArticle failed: %v", err)
	}

	articles, err := db.GetArticles(feed.ID)
	if err != nil {
		t.Fatalf("GetArticles failed: %v", err)
	}
	if len(articles) != 2 {
		t.Fatalf("Expected 2 articles, got %d", len(articles))
	}
	if articles[0].ID != newer.ID || articles[1].ID != older.ID {
		t.Errorf("Expected articles ordered by published_at desc (newer first), got %+v", articles)
	}
}

func TestDatastoreFindArticleByURL(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)
	article := createDatastoreTestArticle(t, db, feed.ID)

	got, err := db.FindArticleByURL(article.URL)
	if err != nil {
		t.Fatalf("FindArticleByURL failed: %v", err)
	}
	if got == nil || got.ID != article.ID {
		t.Fatalf("Expected to find article %d, got %+v", article.ID, got)
	}

	notFound, err := db.FindArticleByURL("https://example.com/does-not-exist")
	if err != nil {
		t.Fatalf("FindArticleByURL (not found) failed: %v", err)
	}
	if notFound != nil {
		t.Errorf("Expected nil for unknown URL, got %+v", notFound)
	}
}

func TestDatastoreFilterExistingArticleURLs(t *testing.T) {
	db := setupTestDatastoreDB(t)

	feed := createDatastoreTestFeed(t, db)
	existing := createDatastoreTestArticle(t, db, feed.ID)
	missingURL := "https://example.com/never-added"

	result, err := db.FilterExistingArticleURLs(feed.ID, []string{existing.URL, missingURL})
	if err != nil {
		t.Fatalf("FilterExistingArticleURLs failed: %v", err)
	}
	if !result[existing.URL] {
		t.Errorf("Expected %s to be reported as existing", existing.URL)
	}
	if result[missingURL] {
		t.Errorf("Expected %s to not be reported as existing", missingURL)
	}

	empty, err := db.FilterExistingArticleURLs(feed.ID, nil)
	if err != nil {
		t.Fatalf("FilterExistingArticleURLs (empty) failed: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("Expected empty map for empty input, got %v", empty)
	}
}

// User tests

func TestDatastoreCreateUser(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := &User{
		GoogleID:  fmt.Sprintf("google_%d", time.Now().UnixNano()),
		Email:     fmt.Sprintf("newuser%d@example.com", time.Now().UnixNano()),
		Name:      "New User",
		CreatedAt: time.Now(),
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.ID <= 0 {
		t.Fatalf("Expected positive user ID, got %d", user.ID)
	}
	if user.SubscriptionStatus != "trial" {
		t.Errorf("Expected default SubscriptionStatus 'trial', got %q", user.SubscriptionStatus)
	}
	if user.MaxArticlesOnFeedAdd != 100 {
		t.Errorf("Expected default MaxArticlesOnFeedAdd 100, got %d", user.MaxArticlesOnFeedAdd)
	}
	wantTrialEnd := user.CreatedAt.AddDate(0, 0, 30)
	if !user.TrialEndsAt.Equal(wantTrialEnd) {
		t.Errorf("Expected TrialEndsAt %v, got %v", wantTrialEnd, user.TrialEndsAt)
	}
}

func TestDatastoreGetUserByGoogleID(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)

	got, err := db.GetUserByGoogleID(user.GoogleID)
	if err != nil {
		t.Fatalf("GetUserByGoogleID failed: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("Expected user %d, got %d", user.ID, got.ID)
	}

	_, err = db.GetUserByGoogleID("nonexistent-google-id")
	if err == nil {
		t.Error("Expected error for nonexistent Google ID")
	}
}

func TestDatastoreGetUserByID(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)

	got, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if got.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, got.Email)
	}

	_, err = db.GetUserByID(user.ID + 999999)
	if err == nil {
		t.Error("Expected error for nonexistent user ID")
	}
}

func TestDatastoreGetUserByEmail(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)

	got, err := db.GetUserByEmail(user.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("Expected user %d, got %d", user.ID, got.ID)
	}

	_, err = db.GetUserByEmail("nobody@example.com")
	if err == nil {
		t.Error("Expected error for nonexistent email")
	}
}

func TestDatastoreGetUserFeeds(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed1 := createDatastoreTestFeed(t, db)
	feed2 := createDatastoreTestFeed(t, db)
	otherFeed := createDatastoreTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed1.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.SubscribeUserToFeed(user.ID, feed2.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	feeds, err := db.GetUserFeeds(user.ID)
	if err != nil {
		t.Fatalf("GetUserFeeds failed: %v", err)
	}
	if len(feeds) != 2 {
		t.Fatalf("Expected 2 feeds, got %d", len(feeds))
	}
	seen := map[int]bool{}
	for _, f := range feeds {
		seen[f.ID] = true
	}
	if !seen[feed1.ID] || !seen[feed2.ID] {
		t.Errorf("Expected feeds %d and %d, got %+v", feed1.ID, feed2.ID, feeds)
	}
	if seen[otherFeed.ID] {
		t.Errorf("Did not expect unsubscribed feed %d", otherFeed.ID)
	}
}

func TestDatastoreGetAllUserFeeds(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user1 := createDatastoreTestUser(t, db)
	user2 := createDatastoreTestUser(t, db)
	sharedFeed := createDatastoreTestFeed(t, db)
	unsubscribedFeed := createDatastoreTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user1.ID, sharedFeed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.SubscribeUserToFeed(user2.ID, sharedFeed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	feeds, err := db.GetAllUserFeeds()
	if err != nil {
		t.Fatalf("GetAllUserFeeds failed: %v", err)
	}
	if len(feeds) != 1 {
		t.Fatalf("Expected 1 unique feed across both users, got %d: %+v", len(feeds), feeds)
	}
	if feeds[0].ID != sharedFeed.ID {
		t.Errorf("Expected feed %d, got %d", sharedFeed.ID, feeds[0].ID)
	}
	_ = unsubscribedFeed
}

func TestDatastoreSubscribeUserToFeed(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	// Subscribing again should be a no-op, not create a duplicate.
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed (duplicate) failed: %v", err)
	}

	count, err := db.GetUserFeedCount(user.ID)
	if err != nil {
		t.Fatalf("GetUserFeedCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected feed count 1 after duplicate subscribe, got %d", count)
	}
}

func TestDatastoreUnsubscribeUserFromFeed(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.UnsubscribeUserFromFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("UnsubscribeUserFromFeed failed: %v", err)
	}

	feeds, err := db.GetUserFeeds(user.ID)
	if err != nil {
		t.Fatalf("GetUserFeeds failed: %v", err)
	}
	if len(feeds) != 0 {
		t.Errorf("Expected no feeds after unsubscribe, got %+v", feeds)
	}
}

func TestDatastoreGetUserFeedCount(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed1 := createDatastoreTestFeed(t, db)
	feed2 := createDatastoreTestFeed(t, db)

	if err := db.SubscribeUserToFeed(user.ID, feed1.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.SubscribeUserToFeed(user.ID, feed2.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	count, err := db.GetUserFeedCount(user.ID)
	if err != nil {
		t.Fatalf("GetUserFeedCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected feed count 2, got %d", count)
	}
}

func TestDatastoreGetUserFeedArticles(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	article := createDatastoreTestArticle(t, db, feed.ID)

	// Not subscribed yet: should return an empty slice, not an error.
	articles, err := db.GetUserFeedArticles(user.ID, feed.ID)
	if err != nil {
		t.Fatalf("GetUserFeedArticles (unsubscribed) failed: %v", err)
	}
	if len(articles) != 0 {
		t.Errorf("Expected no articles before subscribing, got %+v", articles)
	}

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.MarkUserArticleRead(user.ID, article.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	articles, err = db.GetUserFeedArticles(user.ID, feed.ID)
	if err != nil {
		t.Fatalf("GetUserFeedArticles failed: %v", err)
	}
	if len(articles) != 1 {
		t.Fatalf("Expected 1 article, got %d", len(articles))
	}
	if articles[0].ID != article.ID {
		t.Errorf("Expected article %d, got %d", article.ID, articles[0].ID)
	}
	if !articles[0].IsRead {
		t.Error("Expected article to be marked read")
	}
	if articles[0].FeedTitle != feed.Title {
		t.Errorf("Expected FeedTitle %q, got %q", feed.Title, articles[0].FeedTitle)
	}
}

func TestDatastoreGetArticleByID(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	article := createDatastoreTestArticle(t, db, feed.ID)

	// Not subscribed: should return nil, nil.
	got, err := db.GetArticleByID(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetArticleByID (unsubscribed) failed: %v", err)
	}
	if got != nil {
		t.Errorf("Expected nil for unsubscribed user, got %+v", got)
	}

	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	if err := db.SetUserArticleStatus(user.ID, article.ID, false, true); err != nil {
		t.Fatalf("SetUserArticleStatus failed: %v", err)
	}

	got, err = db.GetArticleByID(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetArticleByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("Expected article to be found")
	}
	if got.ID != article.ID {
		t.Errorf("Expected article %d, got %d", article.ID, got.ID)
	}
	if !got.IsStarred {
		t.Error("Expected article to be starred")
	}
	if got.FeedTitle != feed.Title {
		t.Errorf("Expected FeedTitle %q, got %q", feed.Title, got.FeedTitle)
	}

	notFound, err := db.GetArticleByID(user.ID, article.ID+999999)
	if err != nil {
		t.Fatalf("GetArticleByID (nonexistent article) failed: %v", err)
	}
	if notFound != nil {
		t.Errorf("Expected nil for nonexistent article, got %+v", notFound)
	}
}

func TestDatastoreGetUserArticles(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}
	article1 := createDatastoreTestArticle(t, db, feed.ID)
	article2 := createDatastoreTestArticle(t, db, feed.ID)

	if err := db.MarkUserArticleRead(user.ID, article1.ID, true); err != nil {
		t.Fatalf("MarkUserArticleRead failed: %v", err)
	}

	articles, err := db.GetUserArticles(user.ID)
	if err != nil {
		t.Fatalf("GetUserArticles failed: %v", err)
	}
	if len(articles) != 2 {
		t.Fatalf("Expected 2 articles, got %d", len(articles))
	}
	for _, a := range articles {
		if a.ID == article1.ID && !a.IsRead {
			t.Error("Expected article1 to be marked read")
		}
		if a.ID == article2.ID && a.IsRead {
			t.Error("Expected article2 to be unread")
		}
	}
}

func TestDatastoreGetUserArticlesPaginated(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed := createDatastoreTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// GetUserArticlesPaginated re-runs a per-feed projection query capped at limit*2 articles
	// on every call (see articlesPerFeed in datastore.go), regardless of cursor. Keep total
	// within that window (limit*2) so every article is visible to the projection on each page.
	const limit = 2
	const total = limit * 2
	for i := 0; i < total; i++ {
		a := &Article{
			FeedID:      feed.ID,
			Title:       fmt.Sprintf("Article %d", i),
			URL:         fmt.Sprintf("https://example.com/paginated_%d_%d", time.Now().UnixNano(), i),
			PublishedAt: time.Now().Add(time.Duration(i) * time.Second),
			CreatedAt:   time.Now(),
		}
		if err := db.AddArticle(a); err != nil {
			t.Fatalf("AddArticle failed: %v", err)
		}
	}

	seen := map[int]bool{}
	cursor := ""
	pages := 0
	for {
		result, err := db.GetUserArticlesPaginated(user.ID, limit, cursor, false)
		if err != nil {
			t.Fatalf("GetUserArticlesPaginated failed: %v", err)
		}
		for _, a := range result.Articles {
			seen[a.ID] = true
		}
		pages++
		if result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
		if pages > total {
			t.Fatalf("Paginated more times (%d) than articles exist (%d); possible infinite loop", pages, total)
		}
	}

	if len(seen) != total {
		t.Errorf("Expected to see all %d articles across pages, got %d", total, len(seen))
	}
}

func TestDatastoreGetUserFeedArticlesPaginated(t *testing.T) {
	db := setupTestDatastoreDB(t)

	user := createDatastoreTestUser(t, db)
	feed1 := createDatastoreTestFeed(t, db)
	feed2 := createDatastoreTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed1.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	article1 := createDatastoreTestArticle(t, db, feed1.ID)
	article2 := createDatastoreTestArticle(t, db, feed1.ID)
	otherFeedArticle := createDatastoreTestArticle(t, db, feed2.ID)

	result, err := db.GetUserFeedArticlesPaginated(user.ID, feed1.ID, 10, "", false)
	if err != nil {
		t.Fatalf("GetUserFeedArticlesPaginated failed: %v", err)
	}
	if len(result.Articles) != 2 {
		t.Fatalf("Expected 2 articles from feed1, got %d", len(result.Articles))
	}
	seen := map[int]bool{}
	for _, a := range result.Articles {
		seen[a.ID] = true
		if a.FeedID != feed1.ID {
			t.Errorf("Expected article from feed %d, got feed %d", feed1.ID, a.FeedID)
		}
	}
	if !seen[article1.ID] || !seen[article2.ID] {
		t.Errorf("Expected to see both feed1 articles, got %+v", result.Articles)
	}
	if seen[otherFeedArticle.ID] {
		t.Error("Should not have received article from a different feed")
	}

	// User is not subscribed to feed2: should get an empty, non-error result.
	unsubscribed, err := db.GetUserFeedArticlesPaginated(user.ID, feed2.ID, 10, "", false)
	if err != nil {
		t.Fatalf("GetUserFeedArticlesPaginated for unsubscribed feed failed: %v", err)
	}
	if len(unsubscribed.Articles) != 0 {
		t.Errorf("Expected 0 articles for unsubscribed feed, got %d", len(unsubscribed.Articles))
	}
}
