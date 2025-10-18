package middleware

import (
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"goread2/internal/database"
)

// Mock database for testing
type mockDB struct {
	getUserFeedsCalls int
	feeds             []database.Feed
	err               error
}

func (m *mockDB) GetUserFeeds(userID int) ([]database.Feed, error) {
	m.getUserFeedsCalls++
	return m.feeds, m.err
}

// Implement other database.Database interface methods as no-ops
func (m *mockDB) CreateUser(*database.User) error                                               { return nil }
func (m *mockDB) GetUserByGoogleID(string) (*database.User, error)                              { return nil, nil }
func (m *mockDB) GetUserByID(int) (*database.User, error)                                       { return nil, nil }
func (m *mockDB) UpdateUserSubscription(int, string, string, time.Time, time.Time) error        { return nil }
func (m *mockDB) IsUserSubscriptionActive(int) (bool, error)                                    { return false, nil }
func (m *mockDB) GetUserFeedCount(int) (int, error)                                             { return 0, nil }
func (m *mockDB) UpdateUserMaxArticlesOnFeedAdd(int, int) error                                 { return nil }
func (m *mockDB) SetUserAdmin(int, bool) error                                                  { return nil }
func (m *mockDB) GrantFreeMonths(int, int) error                                                { return nil }
func (m *mockDB) GetUserByEmail(string) (*database.User, error)                                 { return nil, nil }
func (m *mockDB) AddFeed(*database.Feed) error                                                  { return nil }
func (m *mockDB) UpdateFeed(*database.Feed) error                                               { return nil }
func (m *mockDB) GetFeeds() ([]database.Feed, error)                                            { return nil, nil }
func (m *mockDB) GetFeedByURL(string) (*database.Feed, error)                                   { return nil, nil }
func (m *mockDB) GetAllUserFeeds() ([]database.Feed, error)                                     { return nil, nil }
func (m *mockDB) DeleteFeed(int) error                                                          { return nil }
func (m *mockDB) SubscribeUserToFeed(int, int) error                                            { return nil }
func (m *mockDB) UnsubscribeUserFromFeed(int, int) error                                        { return nil }
func (m *mockDB) AddArticle(*database.Article) error                                            { return nil }
func (m *mockDB) GetArticles(int) ([]database.Article, error)                                   { return nil, nil }
func (m *mockDB) FindArticleByURL(string) (*database.Article, error)                            { return nil, nil }
func (m *mockDB) GetUserArticles(int) ([]database.Article, error)                               { return nil, nil }
func (m *mockDB) GetUserArticlesPaginated(int, int, int, bool) ([]database.Article, error)     { return nil, nil }
func (m *mockDB) GetUserFeedArticles(int, int) ([]database.Article, error)                      { return nil, nil }
func (m *mockDB) GetUserArticleStatus(int, int) (*database.UserArticle, error)                  { return nil, nil }
func (m *mockDB) SetUserArticleStatus(int, int, bool, bool) error                               { return nil }
func (m *mockDB) BatchSetUserArticleStatus(int, []database.Article, bool, bool) error           { return nil }
func (m *mockDB) MarkUserArticleRead(int, int, bool) error                                      { return nil }
func (m *mockDB) ToggleUserArticleStar(int, int) error                                          { return nil }
func (m *mockDB) GetUserUnreadCounts(int) (map[int]int, error)                                  { return nil, nil }
func (m *mockDB) CreateSession(*database.Session) error                                         { return nil }
func (m *mockDB) GetSession(string) (*database.Session, error)                                  { return nil, nil }
func (m *mockDB) DeleteSession(string) error                                                    { return nil }
func (m *mockDB) DeleteExpiredSessions() error                                                  { return nil }
func (m *mockDB) CreateAuditLog(*database.AuditLog) error                                       { return nil }
func (m *mockDB) GetAuditLogs(int, int, map[string]interface{}) ([]database.AuditLog, error)    { return nil, nil }
func (m *mockDB) GetAllArticles() ([]database.Article, error)                                   { return nil, nil }
func (m *mockDB) UpdateFeedLastFetch(int, time.Time) error                                      { return nil }
func (m *mockDB) Close() error                                                                  { return nil }

func TestRequestCacheMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("adds cache to context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(nil)

		middleware := RequestCacheMiddleware()
		middleware(c)

		_, exists := c.Get("request_cache")
		if !exists {
			t.Error("Expected request_cache to exist in context")
		}
	})

	t.Run("cache is unique per request", func(t *testing.T) {
		c1, _ := gin.CreateTestContext(nil)
		c2, _ := gin.CreateTestContext(nil)

		middleware := RequestCacheMiddleware()
		middleware(c1)
		middleware(c2)

		cache1, _ := c1.Get("request_cache")
		cache2, _ := c2.Get("request_cache")

		if cache1 == cache2 {
			t.Error("Expected different cache instances for different requests")
		}
	})
}

func TestGetCachedUserFeeds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testFeeds := []database.Feed{
		{ID: 1, Title: "Feed 1", URL: "http://example.com/feed1"},
		{ID: 2, Title: "Feed 2", URL: "http://example.com/feed2"},
	}

	t.Run("fetches from DB when cache doesn't exist", func(t *testing.T) {
		c, _ := gin.CreateTestContext(nil)
		mockDB := &mockDB{feeds: testFeeds}

		feeds, err := GetCachedUserFeeds(c, 123, mockDB)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(feeds) != 2 {
			t.Errorf("Expected 2 feeds, got %d", len(feeds))
		}
		if mockDB.getUserFeedsCalls != 1 {
			t.Errorf("Expected 1 DB call, got %d", mockDB.getUserFeedsCalls)
		}
	})

	t.Run("uses cache on second call", func(t *testing.T) {
		c, _ := gin.CreateTestContext(nil)
		mockDB := &mockDB{feeds: testFeeds}

		// Add middleware to create cache
		middleware := RequestCacheMiddleware()
		middleware(c)

		// First call - should hit DB
		feeds1, err1 := GetCachedUserFeeds(c, 123, mockDB)
		if err1 != nil {
			t.Fatalf("Expected no error on first call, got %v", err1)
		}

		// Second call - should use cache
		feeds2, err2 := GetCachedUserFeeds(c, 123, mockDB)
		if err2 != nil {
			t.Fatalf("Expected no error on second call, got %v", err2)
		}

		if len(feeds1) != 2 || len(feeds2) != 2 {
			t.Errorf("Expected 2 feeds on both calls")
		}
		if mockDB.getUserFeedsCalls != 1 {
			t.Errorf("Expected only 1 DB call (cached), got %d", mockDB.getUserFeedsCalls)
		}
	})

	t.Run("caches are scoped per user", func(t *testing.T) {
		c, _ := gin.CreateTestContext(nil)
		mockDB := &mockDB{feeds: testFeeds}

		middleware := RequestCacheMiddleware()
		middleware(c)

		// Call for user 123
		_, _ = GetCachedUserFeeds(c, 123, mockDB)
		// Call for user 456
		_, _ = GetCachedUserFeeds(c, 456, mockDB)
		// Call for user 123 again
		_, _ = GetCachedUserFeeds(c, 123, mockDB)

		// Should be 2 DB calls (one per unique user)
		if mockDB.getUserFeedsCalls != 2 {
			t.Errorf("Expected 2 DB calls (one per user), got %d", mockDB.getUserFeedsCalls)
		}
	})

	t.Run("returns error from DB", func(t *testing.T) {
		c, _ := gin.CreateTestContext(nil)
		expectedErr := errors.New("database error")
		mockDB := &mockDB{err: expectedErr}

		middleware := RequestCacheMiddleware()
		middleware(c)

		_, err := GetCachedUserFeeds(c, 123, mockDB)

		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("does not cache errors", func(t *testing.T) {
		c, _ := gin.CreateTestContext(nil)
		mockDB := &mockDB{err: errors.New("database error")}

		middleware := RequestCacheMiddleware()
		middleware(c)

		// First call - should return error
		_, err1 := GetCachedUserFeeds(c, 123, mockDB)
		if err1 == nil {
			t.Fatal("Expected error on first call")
		}

		// Fix the error
		mockDB.err = nil
		mockDB.feeds = testFeeds

		// Second call - should retry DB call (not use cached error)
		feeds2, err2 := GetCachedUserFeeds(c, 123, mockDB)
		if err2 != nil {
			t.Fatalf("Expected no error on second call after fix, got %v", err2)
		}
		if len(feeds2) != 2 {
			t.Errorf("Expected 2 feeds after error fixed")
		}
		if mockDB.getUserFeedsCalls != 2 {
			t.Errorf("Expected 2 DB calls (error not cached), got %d", mockDB.getUserFeedsCalls)
		}
	})
}
