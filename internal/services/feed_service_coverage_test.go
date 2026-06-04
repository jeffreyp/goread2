package services

import (
	"errors"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

// mockDBFeed is a configurable mock for feed service error-path tests.
type mockDBFeed struct {
	user              *database.User
	articles          []database.Article
	feeds             []database.Feed
	unreadCounts      map[int]int
	shouldFailUser    bool
	shouldFailArticle bool
	shouldFailBatch   bool
	shouldFailStar    bool
	shouldFailMark    bool
	shouldFailUnread  bool
	shouldFailStats   bool
	shouldFailTotal   bool
}

func newMockDBFeed() *mockDBFeed {
	return &mockDBFeed{
		unreadCounts: make(map[int]int),
	}
}

func (m *mockDBFeed) Close() error                                    { return nil }
func (m *mockDBFeed) CreateUser(*database.User) error                 { return nil }
func (m *mockDBFeed) GetUserByGoogleID(string) (*database.User, error) { return nil, nil }
func (m *mockDBFeed) GetUserByID(int) (*database.User, error) {
	if m.shouldFailUser {
		return nil, errors.New("user not found")
	}
	if m.user != nil {
		return m.user, nil
	}
	return &database.User{ID: 1, Email: "t@t.com"}, nil
}
func (m *mockDBFeed) UpdateUserSubscription(int, string, string, time.Time, time.Time) error {
	return nil
}
func (m *mockDBFeed) IsUserSubscriptionActive(int) (bool, error)            { return false, nil }
func (m *mockDBFeed) GetUserFeedCount(int) (int, error)                     { return 0, nil }
func (m *mockDBFeed) UpdateUserMaxArticlesOnFeedAdd(int, int) error         { return nil }
func (m *mockDBFeed) SetUserAdmin(int, bool) error                          { return nil }
func (m *mockDBFeed) SetUserAdminAtomic(int, int, bool) error               { return nil }
func (m *mockDBFeed) GrantFreeMonths(int, int) error                        { return nil }
func (m *mockDBFeed) GetUserByEmail(string) (*database.User, error)         { return nil, nil }
func (m *mockDBFeed) AddFeed(*database.Feed) error                          { return nil }
func (m *mockDBFeed) UpdateFeed(*database.Feed) error                       { return nil }
func (m *mockDBFeed) UpdateFeedTracking(int, time.Time, time.Time, int) error { return nil }
func (m *mockDBFeed) GetFeeds() ([]database.Feed, error)                    { return m.feeds, nil }
func (m *mockDBFeed) GetFeedByURL(string) (*database.Feed, error)           { return nil, nil }
func (m *mockDBFeed) GetUserFeeds(int) ([]database.Feed, error)             { return m.feeds, nil }
func (m *mockDBFeed) GetAllUserFeeds() ([]database.Feed, error)             { return m.feeds, nil }
func (m *mockDBFeed) UpdateFeedCacheHeaders(int, string, string) error      { return nil }
func (m *mockDBFeed) DeleteFeed(int) error                                  { return nil }
func (m *mockDBFeed) SubscribeUserToFeed(int, int) error                    { return nil }
func (m *mockDBFeed) UnsubscribeUserFromFeed(int, int) error                { return nil }
func (m *mockDBFeed) AddArticle(*database.Article) error                    { return nil }
func (m *mockDBFeed) FilterExistingArticleURLs(int, []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (m *mockDBFeed) GetArticles(int) ([]database.Article, error) { return m.articles, nil }
func (m *mockDBFeed) FindArticleByURL(string) (*database.Article, error) {
	return nil, nil
}
func (m *mockDBFeed) GetUserArticles(int) ([]database.Article, error) {
	if m.shouldFailArticle {
		return nil, errors.New("db error")
	}
	return m.articles, nil
}
func (m *mockDBFeed) GetUserArticlesPaginated(int, int, string, bool) (*database.ArticlePaginationResult, error) {
	return &database.ArticlePaginationResult{Articles: m.articles}, nil
}
func (m *mockDBFeed) GetUserFeedArticles(int, int) ([]database.Article, error) {
	return m.articles, nil
}
func (m *mockDBFeed) GetArticleByID(int, int) (*database.Article, error) {
	if len(m.articles) > 0 {
		return &m.articles[0], nil
	}
	return nil, nil
}
func (m *mockDBFeed) GetUserArticleStatus(int, int) (*database.UserArticle, error) {
	return nil, nil
}
func (m *mockDBFeed) SetUserArticleStatus(int, int, bool, bool) error { return nil }
func (m *mockDBFeed) BatchSetUserArticleStatus(int, []database.Article, bool, bool) error {
	if m.shouldFailBatch {
		return errors.New("batch error")
	}
	return nil
}
func (m *mockDBFeed) MarkUserArticleRead(int, int, bool) error {
	if m.shouldFailMark {
		return errors.New("mark error")
	}
	return nil
}
func (m *mockDBFeed) ToggleUserArticleStar(int, int) error {
	if m.shouldFailStar {
		return errors.New("star error")
	}
	return nil
}
func (m *mockDBFeed) GetUserUnreadCounts(int) (map[int]int, error) {
	if m.shouldFailUnread {
		return nil, errors.New("unread error")
	}
	return m.unreadCounts, nil
}
func (m *mockDBFeed) GetTotalArticleCount(int) (int, error) {
	if m.shouldFailTotal {
		return 0, errors.New("total error")
	}
	return len(m.articles), nil
}
func (m *mockDBFeed) GetAccountStats(int) (map[string]interface{}, error) {
	if m.shouldFailStats {
		return nil, errors.New("stats error")
	}
	return map[string]interface{}{"total_articles": 5, "total_unread": 2}, nil
}
func (m *mockDBFeed) CleanupOrphanedUserArticles(int) (int, error)         { return 0, nil }
func (m *mockDBFeed) CreateSession(*database.Session) error                { return nil }
func (m *mockDBFeed) GetSession(string) (*database.Session, error)         { return nil, nil }
func (m *mockDBFeed) UpdateSessionExpiry(string, time.Time) error          { return nil }
func (m *mockDBFeed) DeleteSession(string) error                           { return nil }
func (m *mockDBFeed) DeleteExpiredSessions() error                         { return nil }
func (m *mockDBFeed) CreateAuditLog(*database.AuditLog) error              { return nil }
func (m *mockDBFeed) GetAuditLogs(int, int, map[string]interface{}) ([]database.AuditLog, error) {
	return nil, nil
}
func (m *mockDBFeed) UpdateFeedLastFetch(int, time.Time) error { return nil }
func (m *mockDBFeed) UpdateFeedAfterRefresh(int, time.Time, time.Time, int, time.Time, string, string) error {
	return nil
}

// --- Tests using real SQLite DB ---

func newFeedServiceWithRealDB(t *testing.T) (*FeedService, *database.DB, func()) {
	t.Helper()
	db := setupTestDB(t)
	fs := NewFeedService(db, NewDomainRateLimiter(RateLimiterConfig{RequestsPerMinute: 60, BurstSize: 10}))
	return fs, db, func() { _ = db.Close() }
}

func createTestUser(t *testing.T, db *database.DB) *database.User {
	t.Helper()
	user := &database.User{
		GoogleID:  "google-test-" + t.Name(),
		Email:     t.Name() + "@example.com",
		Name:      "Test User",
		CreatedAt: time.Now(),
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return user
}

func createTestFeed(t *testing.T, db *database.DB) *database.Feed {
	t.Helper()
	now := time.Now()
	feed := &database.Feed{
		Title: "Test Feed", URL: "https://example.com/feed" + t.Name(),
		CreatedAt: now, UpdatedAt: now, LastFetch: now,
		LastChecked: now, LastHadNewContent: now,
	}
	if err := db.AddFeed(feed); err != nil {
		t.Fatalf("AddFeed: %v", err)
	}
	return feed
}

func createTestArticle(t *testing.T, db *database.DB, feedID int) *database.Article {
	t.Helper()
	article := &database.Article{
		FeedID: feedID, Title: "Test Article",
		URL:         "https://example.com/article-" + t.Name(),
		Content:     "content", PublishedAt: time.Now(), CreatedAt: time.Now(),
	}
	if err := db.AddArticle(article); err != nil {
		t.Fatalf("AddArticle: %v", err)
	}
	return article
}

func TestFeedService_GetFeeds(t *testing.T) {
	fs, db, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	createTestFeed(t, db)

	feeds, err := fs.GetFeeds()
	if err != nil {
		t.Fatalf("GetFeeds: %v", err)
	}
	if len(feeds) == 0 {
		t.Error("expected at least one feed")
	}
}

func TestFeedService_GetUserFeeds(t *testing.T) {
	fs, db, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed: %v", err)
	}

	feeds, err := fs.GetUserFeeds(user.ID)
	if err != nil {
		t.Fatalf("GetUserFeeds: %v", err)
	}
	if len(feeds) != 1 {
		t.Errorf("expected 1 feed, got %d", len(feeds))
	}
}

func TestFeedService_DeleteFeed(t *testing.T) {
	fs, db, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	feed := createTestFeed(t, db)

	if err := fs.DeleteFeed(feed.ID); err != nil {
		t.Fatalf("DeleteFeed: %v", err)
	}

	feeds, _ := fs.GetFeeds()
	for _, f := range feeds {
		if f.ID == feed.ID {
			t.Error("feed should have been deleted")
		}
	}
}

func TestFeedService_GetArticles(t *testing.T) {
	fs, db, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	feed := createTestFeed(t, db)
	createTestArticle(t, db, feed.ID)

	articles, err := fs.GetArticles(feed.ID)
	if err != nil {
		t.Fatalf("GetArticles: %v", err)
	}
	if len(articles) == 0 {
		t.Error("expected at least one article")
	}
}

func TestFeedService_GetUserArticles(t *testing.T) {
	fs, db, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed: %v", err)
	}
	article := createTestArticle(t, db, feed.ID)
	if err := db.SetUserArticleStatus(user.ID, article.ID, false, false); err != nil {
		t.Fatalf("SetUserArticleStatus: %v", err)
	}

	articles, err := fs.GetUserArticles(user.ID)
	if err != nil {
		t.Fatalf("GetUserArticles: %v", err)
	}
	if len(articles) == 0 {
		t.Error("expected at least one user article")
	}
}

func TestFeedService_GetArticleByID(t *testing.T) {
	fs, db, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed: %v", err)
	}
	article := createTestArticle(t, db, feed.ID)

	got, err := fs.GetArticleByID(user.ID, article.ID)
	if err != nil {
		t.Fatalf("GetArticleByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected article, got nil")
	}
	if got.ID != article.ID {
		t.Errorf("expected article ID %d, got %d", article.ID, got.ID)
	}
}

func TestFeedService_MarkUserArticleRead(t *testing.T) {
	fs, db, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	article := createTestArticle(t, db, feed.ID)
	if err := db.SetUserArticleStatus(user.ID, article.ID, false, false); err != nil {
		t.Fatalf("SetUserArticleStatus: %v", err)
	}

	if err := fs.MarkUserArticleRead(user.ID, article.ID, true, feed.ID, false); err != nil {
		t.Fatalf("MarkUserArticleRead: %v", err)
	}
}

func TestFeedService_ToggleUserArticleStar(t *testing.T) {
	fs, db, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	article := createTestArticle(t, db, feed.ID)
	if err := db.SetUserArticleStatus(user.ID, article.ID, false, false); err != nil {
		t.Fatalf("SetUserArticleStatus: %v", err)
	}

	if err := fs.ToggleUserArticleStar(user.ID, article.ID); err != nil {
		t.Fatalf("ToggleUserArticleStar: %v", err)
	}
}

func TestFeedService_MarkAllArticlesRead_Empty(t *testing.T) {
	fs, _, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	count, err := fs.MarkAllArticlesRead(9999)
	if err != nil {
		t.Fatalf("MarkAllArticlesRead on empty: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 articles marked, got %d", count)
	}
}

func TestFeedService_MarkAllArticlesRead_NonEmpty(t *testing.T) {
	fs, db, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed: %v", err)
	}
	createTestArticle(t, db, feed.ID)

	count, err := fs.MarkAllArticlesRead(user.ID)
	if err != nil {
		t.Fatalf("MarkAllArticlesRead: %v", err)
	}
	if count == 0 {
		t.Error("expected at least 1 article marked")
	}
}

func TestFeedService_UpdateUserMaxArticlesOnFeedAdd(t *testing.T) {
	fs, db, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	user := createTestUser(t, db)

	if err := fs.UpdateUserMaxArticlesOnFeedAdd(user.ID, 50); err != nil {
		t.Fatalf("UpdateUserMaxArticlesOnFeedAdd: %v", err)
	}
}

func TestFeedService_GetAccountStats(t *testing.T) {
	fs, db, cleanup := newFeedServiceWithRealDB(t)
	defer cleanup()

	user := createTestUser(t, db)

	stats, err := fs.GetAccountStats(user.ID, nil)
	if err != nil {
		t.Fatalf("GetAccountStats: %v", err)
	}
	if stats == nil {
		t.Error("expected non-nil stats")
	}
}

// --- Error path tests using mock DB ---

func TestFeedService_MarkAllArticlesRead_DBError(t *testing.T) {
	mock := newMockDBFeed()
	mock.articles = []database.Article{{ID: 1}}
	mock.shouldFailArticle = true
	fs := NewFeedService(mock, nil)

	_, err := fs.MarkAllArticlesRead(1)
	if err == nil {
		t.Error("expected error when GetUserArticles fails")
	}
}

func TestFeedService_MarkAllArticlesRead_BatchError(t *testing.T) {
	mock := newMockDBFeed()
	mock.articles = []database.Article{{ID: 1}}
	mock.shouldFailBatch = true
	fs := NewFeedService(mock, nil)

	_, err := fs.MarkAllArticlesRead(1)
	if err == nil {
		t.Error("expected error when BatchSetUserArticleStatus fails")
	}
}

func TestFeedService_MarkUserArticleRead_DBError(t *testing.T) {
	mock := newMockDBFeed()
	mock.shouldFailMark = true
	fs := NewFeedService(mock, nil)

	err := fs.MarkUserArticleRead(1, 1, true, 1, false)
	if err == nil {
		t.Error("expected error when MarkUserArticleRead fails")
	}
}

func TestFeedService_ToggleUserArticleStar_DBError(t *testing.T) {
	mock := newMockDBFeed()
	mock.shouldFailStar = true
	fs := NewFeedService(mock, nil)

	err := fs.ToggleUserArticleStar(1, 1)
	if err == nil {
		t.Error("expected error when ToggleUserArticleStar fails")
	}
}

func TestFeedService_GetUserUnreadCounts_DBError(t *testing.T) {
	mock := newMockDBFeed()
	mock.shouldFailUnread = true
	fs := NewFeedService(mock, nil)

	_, err := fs.GetUserUnreadCounts(1, nil)
	if err == nil {
		t.Error("expected error when GetUserUnreadCounts fails")
	}
}

func TestFeedService_GetAccountStats_DBError(t *testing.T) {
	mock := newMockDBFeed()
	mock.shouldFailStats = true
	fs := NewFeedService(mock, nil)

	_, err := fs.GetAccountStats(1, nil)
	if err == nil {
		t.Error("expected error when GetAccountStats fails")
	}
}
