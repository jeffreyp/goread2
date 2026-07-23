package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/config"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/secrets"
	"github.com/jeffreyp/goread2/internal/services"
)

// Mock database for testing
type mockDBFeedHandler struct {
	shouldFailUnsubscribe         bool
	shouldFailMarkRead            bool
	shouldFailToggleStar          bool
	shouldFailCleanupOrphaned     bool
	shouldFailGetAccountStats     bool
	shouldFailUpdateMaxArticles   bool
	shouldFailGetArticle          bool
	shouldFailGetUserFeeds        bool
	shouldFailGetUnreadCounts     bool
	mockArticle                   *database.Article
	mockUserFeeds                 []database.Feed
	articlesDeleted               int
	capturedPaginationLimit       int
	mockUser                      *database.User
	shouldFailGetUserByID         bool
	mockFeedCount                 int
	shouldFailGetFeedCount        bool
	shouldFailGetFeeds            bool
	mockFeedArticles              []database.Article
	mockNextCursor                string
	shouldFailGetUserFeedArticles bool
	shouldFailFindArticleByURL    bool
	mockFoundArticle              *database.Article
}

func newMockDBFeedHandler() *mockDBFeedHandler {
	return &mockDBFeedHandler{}
}

func (m *mockDBFeedHandler) CreateUser(*database.User) error                  { return nil }
func (m *mockDBFeedHandler) GetUserByGoogleID(string) (*database.User, error) { return nil, nil }
func (m *mockDBFeedHandler) GetUserByID(userID int) (*database.User, error) {
	if m.shouldFailGetUserByID {
		return nil, errors.New("database error")
	}
	if m.mockUser != nil {
		return m.mockUser, nil
	}
	return &database.User{
		ID:                 userID,
		Email:              "test@example.com",
		Name:               "Test User",
		SubscriptionStatus: "free",
	}, nil
}
func (m *mockDBFeedHandler) UpdateUserSubscription(int, string, string, time.Time, time.Time) error {
	return nil
}
func (m *mockDBFeedHandler) IsUserSubscriptionActive(int) (bool, error) { return false, nil }
func (m *mockDBFeedHandler) GetUserFeedCount(int) (int, error) {
	if m.shouldFailGetFeedCount {
		return 0, errors.New("database error")
	}
	return m.mockFeedCount, nil
}
func (m *mockDBFeedHandler) UpdateUserMaxArticlesOnFeedAdd(userID, maxArticles int) error {
	if m.shouldFailUpdateMaxArticles {
		return errors.New("database error")
	}
	return nil
}
func (m *mockDBFeedHandler) SetUserAdmin(int, bool) error                            { return nil }
func (m *mockDBFeedHandler) SetUserAdminAtomic(int, int, bool) error                 { return nil }
func (m *mockDBFeedHandler) GrantFreeMonths(int, int) error                          { return nil }
func (m *mockDBFeedHandler) GetUserByEmail(string) (*database.User, error)           { return nil, nil }
func (m *mockDBFeedHandler) AddFeed(*database.Feed) error                            { return nil }
func (m *mockDBFeedHandler) UpdateFeed(*database.Feed) error                         { return nil }
func (m *mockDBFeedHandler) UpdateFeedTracking(int, time.Time, time.Time, int) error { return nil }
func (m *mockDBFeedHandler) GetFeeds() ([]database.Feed, error) {
	if m.shouldFailGetFeeds {
		return nil, errors.New("database error")
	}
	return nil, nil
}
func (m *mockDBFeedHandler) GetFeedByURL(string) (*database.Feed, error) { return nil, nil }
func (m *mockDBFeedHandler) GetUserFeeds(int) ([]database.Feed, error) {
	if m.shouldFailGetUserFeeds {
		return nil, errors.New("database error")
	}
	return m.mockUserFeeds, nil
}
func (m *mockDBFeedHandler) GetAllUserFeeds() ([]database.Feed, error) { return nil, nil }
func (m *mockDBFeedHandler) DeleteFeed(int) error                      { return nil }
func (m *mockDBFeedHandler) SubscribeUserToFeed(int, int) error        { return nil }
func (m *mockDBFeedHandler) UnsubscribeUserFromFeed(userID, feedID int) error {
	if m.shouldFailUnsubscribe {
		return errors.New("database error")
	}
	return nil
}
func (m *mockDBFeedHandler) AddArticle(*database.Article) error          { return nil }
func (m *mockDBFeedHandler) GetArticles(int) ([]database.Article, error) { return nil, nil }
func (m *mockDBFeedHandler) FindArticleByURL(string) (*database.Article, error) {
	if m.shouldFailFindArticleByURL {
		return nil, errors.New("database error")
	}
	return m.mockFoundArticle, nil
}
func (m *mockDBFeedHandler) GetUserArticles(int) ([]database.Article, error) { return nil, nil }
func (m *mockDBFeedHandler) GetUserArticlesPaginated(userID, limit int, cursor string, unreadOnly bool) (*database.ArticlePaginationResult, error) {
	m.capturedPaginationLimit = limit
	return &database.ArticlePaginationResult{
		Articles:   []database.Article{},
		NextCursor: "",
	}, nil
}
func (m *mockDBFeedHandler) GetUserFeedArticles(int, int) ([]database.Article, error) {
	if m.shouldFailGetUserFeedArticles {
		return nil, errors.New("database error")
	}
	return m.mockFeedArticles, nil
}
func (m *mockDBFeedHandler) GetUserFeedArticlesPaginated(userID, feedID, limit int, cursor string, unreadOnly bool) (*database.ArticlePaginationResult, error) {
	if m.shouldFailGetUserFeedArticles {
		return nil, errors.New("database error")
	}
	m.capturedPaginationLimit = limit
	return &database.ArticlePaginationResult{
		Articles:   m.mockFeedArticles,
		NextCursor: m.mockNextCursor,
	}, nil
}
func (m *mockDBFeedHandler) GetArticleByID(int, int) (*database.Article, error) {
	if m.shouldFailGetArticle {
		return nil, errors.New("database error")
	}
	return m.mockArticle, nil
}
func (m *mockDBFeedHandler) GetUserArticleStatus(int, int) (*database.UserArticle, error) {
	return nil, nil
}
func (m *mockDBFeedHandler) SetUserArticleStatus(int, int, bool, bool) error { return nil }
func (m *mockDBFeedHandler) MarkAllUserArticlesRead(int) (int, error)        { return 0, nil }
func (m *mockDBFeedHandler) BatchSetUserArticleStatus(int, []database.Article, bool, bool) error {
	return nil
}
func (m *mockDBFeedHandler) MarkUserArticleRead(userID, articleID int, isRead bool) error {
	if m.shouldFailMarkRead {
		return errors.New("database error")
	}
	return nil
}
func (m *mockDBFeedHandler) ToggleUserArticleStar(userID, articleID int) error {
	if m.shouldFailToggleStar {
		return errors.New("database error")
	}
	return nil
}
func (m *mockDBFeedHandler) GetUserUnreadCounts(int) (map[int]int, error) {
	if m.shouldFailGetUnreadCounts {
		return nil, errors.New("database error")
	}
	return map[int]int{}, nil
}
func (m *mockDBFeedHandler) GetTotalArticleCount(int) (int, error) { return 0, nil }
func (m *mockDBFeedHandler) CleanupOrphanedUserArticles(days int) (int, error) {
	if m.shouldFailCleanupOrphaned {
		return 0, errors.New("database error")
	}
	return m.articlesDeleted, nil
}
func (m *mockDBFeedHandler) CreateSession(*database.Session) error        { return nil }
func (m *mockDBFeedHandler) GetSession(string) (*database.Session, error) { return nil, nil }
func (m *mockDBFeedHandler) UpdateSessionExpiry(string, time.Time) error  { return nil }
func (m *mockDBFeedHandler) DeleteSession(string) error                   { return nil }
func (m *mockDBFeedHandler) DeleteExpiredSessions() error                 { return nil }
func (m *mockDBFeedHandler) CreateAuditLog(*database.AuditLog) error      { return nil }
func (m *mockDBFeedHandler) GetAuditLogs(int, int, map[string]interface{}) ([]database.AuditLog, error) {
	return nil, nil
}
func (m *mockDBFeedHandler) GetAccountStats(userID int) (map[string]interface{}, error) {
	if m.shouldFailGetAccountStats {
		return nil, errors.New("database error")
	}
	return map[string]interface{}{
		"total_articles":  100,
		"unread_articles": 25,
		"total_feeds":     10,
	}, nil
}
func (m *mockDBFeedHandler) UpdateFeedLastFetch(int, time.Time) error { return nil }
func (m *mockDBFeedHandler) Close() error                             { return nil }
func (m *mockDBFeedHandler) UpdateFeedCacheHeaders(feedID int, etag, lastModified string) error {
	return nil
}
func (m *mockDBFeedHandler) FilterExistingArticleURLs(int, []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (m *mockDBFeedHandler) UpdateFeedAfterRefresh(int, time.Time, time.Time, int, time.Time, string, string) error {
	return nil
}

func TestNewFeedHandler(t *testing.T) {
	// Create mock services
	mockFeedService := &services.FeedService{}
	mockSubscriptionService := &services.SubscriptionService{}
	mockFeedScheduler := &services.FeedScheduler{}
	mockDB := newMockDBFeedHandler()

	handler := NewFeedHandler(mockFeedService, mockSubscriptionService, mockFeedScheduler, mockDB)

	if handler == nil {
		t.Fatal("NewFeedHandler returned nil")
		return
	}

	if handler.feedService != mockFeedService {
		t.Error("FeedHandler feed service not set correctly")
	}

	if handler.subscriptionService != mockSubscriptionService {
		t.Error("FeedHandler subscription service not set correctly")
	}

	if handler.feedScheduler != mockFeedScheduler {
		t.Error("FeedHandler feed scheduler not set correctly")
	}

	if handler.db != mockDB {
		t.Error("FeedHandler database not set correctly")
	}
}

func TestDeleteFeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful feed deletion", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/feeds/123", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
		c.Set("user", testUser)

		handler.DeleteFeed(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["message"] == nil {
			t.Error("Response should contain message")
		}
	})

	t.Run("no authentication returns 401", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/feeds/123", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}

		handler.DeleteFeed(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("invalid feed ID returns 400", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/feeds/invalid", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "invalid"}}
		c.Set("user", testUser)

		handler.DeleteFeed(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailUnsubscribe = true
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/feeds/123", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
		c.Set("user", testUser)

		handler.DeleteFeed(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

func TestGetArticlesPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("get all articles with pagination", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/all/articles?limit=10&cursor=abc123&unread_only=true", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "all"}}
		c.Set("user", testUser)

		handler.GetArticles(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["articles"] == nil {
			t.Error("Response should contain articles")
		}

		if _, ok := response["next_cursor"]; !ok {
			t.Error("Response should contain next_cursor")
		}
	})

	t.Run("pagination with default limit", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/all/articles", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "all"}}
		c.Set("user", testUser)

		handler.GetArticles(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("pagination with custom limit", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/all/articles?limit=100", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "all"}}
		c.Set("user", testUser)

		handler.GetArticles(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("no authentication returns 401", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/all/articles", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "all"}}

		handler.GetArticles(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("limit above max is clamped to default", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/all/articles?limit=200", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "all"}}
		c.Set("user", testUser)

		handler.GetArticles(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if db.capturedPaginationLimit != 50 {
			t.Errorf("Expected default limit 50 when limit=200 is rejected, got %d", db.capturedPaginationLimit)
		}
	})

	t.Run("limit=0 uses default", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/all/articles?limit=0", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "all"}}
		c.Set("user", testUser)

		handler.GetArticles(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if db.capturedPaginationLimit != 50 {
			t.Errorf("Expected default limit 50 when limit=0, got %d", db.capturedPaginationLimit)
		}
	})

	t.Run("negative limit uses default", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/all/articles?limit=-1", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "all"}}
		c.Set("user", testUser)

		handler.GetArticles(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if db.capturedPaginationLimit != 50 {
			t.Errorf("Expected default limit 50 when limit=-1, got %d", db.capturedPaginationLimit)
		}
	})

	t.Run("non-numeric limit uses default", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/all/articles?limit=abc", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "all"}}
		c.Set("user", testUser)

		handler.GetArticles(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if db.capturedPaginationLimit != 50 {
			t.Errorf("Expected default limit 50 when limit=abc, got %d", db.capturedPaginationLimit)
		}
	})
}

func TestMarkRead(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("mark article as read", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		requestBody := map[string]bool{"is_read": true}
		bodyBytes, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/123/read", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
		c.Set("user", testUser)

		handler.MarkRead(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["message"] == nil {
			t.Error("Response should contain message")
		}
	})

	t.Run("mark article as unread", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		requestBody := map[string]bool{"is_read": false}
		bodyBytes, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/123/read", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
		c.Set("user", testUser)

		handler.MarkRead(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("no authentication returns 401", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		requestBody := map[string]bool{"is_read": true}
		bodyBytes, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/123/read", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}

		handler.MarkRead(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("invalid article ID returns 400", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		requestBody := map[string]bool{"is_read": true}
		bodyBytes, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/invalid/read", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: "invalid"}}
		c.Set("user", testUser)

		handler.MarkRead(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid JSON body returns 400", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/123/read", bytes.NewReader([]byte("invalid json")))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
		c.Set("user", testUser)

		handler.MarkRead(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailMarkRead = true
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		requestBody := map[string]bool{"is_read": true}
		bodyBytes, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/123/read", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
		c.Set("user", testUser)

		handler.MarkRead(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

func TestToggleStar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful star toggle", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/123/star", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
		c.Set("user", testUser)

		handler.ToggleStar(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["message"] == nil {
			t.Error("Response should contain message")
		}
	})

	t.Run("no authentication returns 401", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/123/star", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}

		handler.ToggleStar(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("invalid article ID returns 400", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/invalid/star", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "invalid"}}
		c.Set("user", testUser)

		handler.ToggleStar(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailToggleStar = true
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/123/star", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
		c.Set("user", testUser)

		handler.ToggleStar(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

func TestMarkAllRead(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful mark all read", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/mark-all-read", nil)
		c.Set("user", testUser)

		handler.MarkAllRead(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["message"] == nil {
			t.Error("Response should contain message")
		}

		if response["articles_count"] == nil {
			t.Error("Response should contain articles_count")
		}
	})

	t.Run("no authentication returns 401", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/articles/mark-all-read", nil)

		handler.MarkAllRead(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})
}

func TestCleanupOrphanedUserArticles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful cleanup as admin in local environment", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		db := newMockDBFeedHandler()
		db.articlesDeleted = 42
		handler := NewFeedHandler(nil, nil, nil, db)

		adminUser := &database.User{
			ID:      1,
			Email:   "admin@example.com",
			Name:    "Admin User",
			IsAdmin: true,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-orphaned-articles", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", adminUser)

		handler.CleanupOrphanedUserArticles(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["deleted_count"] != float64(42) {
			t.Errorf("Expected deleted_count 42, got %v", response["deleted_count"])
		}
	})

	t.Run("unauthorized without admin in local environment", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		db := newMockDBFeedHandler()
		handler := NewFeedHandler(nil, nil, nil, db)

		regularUser := &database.User{
			ID:      2,
			Email:   "user@example.com",
			Name:    "Regular User",
			IsAdmin: false,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-orphaned-articles", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", regularUser)

		handler.CleanupOrphanedUserArticles(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("unauthorized without authentication in local environment", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		db := newMockDBFeedHandler()
		handler := NewFeedHandler(nil, nil, nil, db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-orphaned-articles", nil)

		handler.CleanupOrphanedUserArticles(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("unauthorized admin without X-Admin-Token header", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		db := newMockDBFeedHandler()
		handler := NewFeedHandler(nil, nil, nil, db)

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-orphaned-articles", nil)
		c.Set("user", adminUser)

		handler.CleanupOrphanedUserArticles(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("unauthorized admin with wrong X-Admin-Token header", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		db := newMockDBFeedHandler()
		handler := NewFeedHandler(nil, nil, nil, db)

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-orphaned-articles", nil)
		c.Request.Header.Set("X-Admin-Token", "wrong-token")
		c.Set("user", adminUser)

		handler.CleanupOrphanedUserArticles(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("unauthorized when ADMIN_TOKEN not configured", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		db := newMockDBFeedHandler()
		handler := NewFeedHandler(nil, nil, nil, db)

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-orphaned-articles", nil)
		c.Request.Header.Set("X-Admin-Token", "any-token")
		c.Set("user", adminUser)

		handler.CleanupOrphanedUserArticles(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		db := newMockDBFeedHandler()
		db.shouldFailCleanupOrphaned = true
		handler := NewFeedHandler(nil, nil, nil, db)

		adminUser := &database.User{
			ID:      1,
			Email:   "admin@example.com",
			Name:    "Admin User",
			IsAdmin: true,
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-orphaned-articles", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", adminUser)

		handler.CleanupOrphanedUserArticles(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("cron path enqueues via task queue when configured", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		handler := NewFeedHandler(nil, nil, nil, newMockDBFeedHandler())
		tq := &fakeTaskQueue{}
		handler.SetTaskQueue(tq)

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-orphaned-articles", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", adminUser)

		handler.CleanupOrphanedUserArticles(c)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected 202, got %d: %s", w.Code, w.Body.String())
		}
		if len(tq.enqueued) != 1 || tq.enqueued[0] != "/tasks/cleanup-orphaned-articles" {
			t.Errorf("expected /tasks/cleanup-orphaned-articles enqueued once, got %v", tq.enqueued)
		}
	})

	t.Run("cron path returns 500 when enqueue fails", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		handler := NewFeedHandler(nil, nil, nil, newMockDBFeedHandler())
		handler.SetTaskQueue(&fakeTaskQueue{shouldFail: true})

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-orphaned-articles", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", adminUser)

		handler.CleanupOrphanedUserArticles(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestTaskCleanupOrphanedArticles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("unauthorized without queue header or admin auth", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		handler := NewFeedHandler(nil, nil, nil, newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/cleanup-orphaned-articles", nil)
		handler.TaskCleanupOrphanedArticles(c)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("succeeds via admin auth fallback", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		db := newMockDBFeedHandler()
		db.articlesDeleted = 7
		handler := NewFeedHandler(nil, nil, nil, db)

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/cleanup-orphaned-articles", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", adminUser)

		handler.TaskCleanupOrphanedArticles(c)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp["deleted_count"] != float64(7) {
			t.Errorf("expected deleted_count 7, got %v", resp["deleted_count"])
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		db := newMockDBFeedHandler()
		db.shouldFailCleanupOrphaned = true
		handler := NewFeedHandler(nil, nil, nil, db)

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/cleanup-orphaned-articles", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", adminUser)

		handler.TaskCleanupOrphanedArticles(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDebugFeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("debug feed returns feed details", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/debug/feeds/123", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
		c.Set("user", testUser)

		handler.DebugFeed(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["user_id"] == nil {
			t.Error("Response should contain user_id")
		}

		if response["feed_id"] == nil {
			t.Error("Response should contain feed_id")
		}
	})

	t.Run("no authentication returns 401", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/debug/feeds/123", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}

		handler.DebugFeed(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("invalid feed ID returns 400", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/debug/feeds/invalid", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "invalid"}}
		c.Set("user", testUser)

		handler.DebugFeed(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestDebugArticleByURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("debug article by URL finds article", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/debug/articles?url=https://example.com/article", nil)
		c.Set("user", testUser)

		handler.DebugArticleByURL(c)

		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", w.Code)
		}
	})

	t.Run("no authentication returns 401", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/debug/articles?url=https://example.com/article", nil)

		handler.DebugArticleByURL(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("missing URL parameter returns 400", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/debug/articles", nil)
		c.Set("user", testUser)

		handler.DebugArticleByURL(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestGetAccountStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful get account stats", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		subscriptionService := services.NewSubscriptionService(db)
		handler := NewFeedHandler(feedService, subscriptionService, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/account/stats", nil)
		c.Set("user", testUser)

		handler.GetAccountStats(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["total_feeds"] == nil {
			t.Error("Response should contain total_feeds")
		}

		if response["total_articles"] == nil {
			t.Error("Response should contain total_articles")
		}
	})

	t.Run("no authentication returns 401", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		subscriptionService := services.NewSubscriptionService(db)
		handler := NewFeedHandler(feedService, subscriptionService, nil, db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/account/stats", nil)

		handler.GetAccountStats(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailGetAccountStats = true
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		subscriptionService := services.NewSubscriptionService(db)
		handler := NewFeedHandler(feedService, subscriptionService, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/account/stats", nil)
		c.Set("user", testUser)

		handler.GetAccountStats(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

func TestUpdateMaxArticlesOnFeedAdd(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful update max articles", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		requestBody := map[string]int{"max_articles": 100}
		bodyBytes, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/settings/max-articles", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user", testUser)

		handler.UpdateMaxArticlesOnFeedAdd(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["max_articles"] != float64(100) {
			t.Errorf("Expected max_articles 100, got %v", response["max_articles"])
		}
	})

	t.Run("max_articles of 0 (unlimited) succeeds", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		requestBody := map[string]int{"max_articles": 0}
		bodyBytes, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/settings/max-articles", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user", testUser)

		handler.UpdateMaxArticlesOnFeedAdd(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["max_articles"] != float64(0) {
			t.Errorf("Expected max_articles 0, got %v", response["max_articles"])
		}
	})

	t.Run("no authentication returns 401", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		requestBody := map[string]int{"max_articles": 100}
		bodyBytes, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/settings/max-articles", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.UpdateMaxArticlesOnFeedAdd(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("invalid request body returns 400", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/settings/max-articles", bytes.NewReader([]byte("invalid json")))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user", testUser)

		handler.UpdateMaxArticlesOnFeedAdd(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("negative max_articles returns 400", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		requestBody := map[string]int{"max_articles": -1}
		bodyBytes, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/settings/max-articles", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user", testUser)

		handler.UpdateMaxArticlesOnFeedAdd(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("max_articles over 10000 returns 400", func(t *testing.T) {
		db := newMockDBFeedHandler()
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		requestBody := map[string]int{"max_articles": 10001}
		bodyBytes, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/settings/max-articles", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user", testUser)

		handler.UpdateMaxArticlesOnFeedAdd(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailUpdateMaxArticles = true
		rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
			RequestsPerMinute: 10,
			BurstSize:         1,
		})
		feedService := services.NewFeedService(db, rateLimiter)
		handler := NewFeedHandler(feedService, nil, nil, db)

		testUser := &database.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		}

		requestBody := map[string]int{"max_articles": 100}
		bodyBytes, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/settings/max-articles", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user", testUser)

		handler.UpdateMaxArticlesOnFeedAdd(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

// fakeTaskQueue is a TaskQueue test double that records enqueued URIs and
// can be made to fail on demand.
type fakeTaskQueue struct {
	shouldFail bool
	enqueued   []string
}

func (q *fakeTaskQueue) Enqueue(ctx context.Context, relativeURI string) error {
	if q.shouldFail {
		return errors.New("enqueue failed")
	}
	q.enqueued = append(q.enqueued, relativeURI)
	return nil
}

func newFeedHandlerWithSubscription(db *mockDBFeedHandler) *FeedHandler {
	rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{RequestsPerMinute: 60, BurstSize: 10})
	feedService := services.NewFeedService(db, rateLimiter)
	subscriptionService := services.NewSubscriptionService(db)
	return NewFeedHandler(feedService, subscriptionService, nil, db)
}

func TestGetFeeds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds", nil)
		handler.GetFeeds(c)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns empty array when user has no feeds", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds", nil)
		c.Set("user", testUser)
		handler.GetFeeds(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var feeds []database.Feed
		if err := json.Unmarshal(w.Body.Bytes(), &feeds); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if feeds == nil {
			t.Error("response should be an empty array, not null")
		}
	})

	t.Run("returns user feeds", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.mockUserFeeds = []database.Feed{
			{ID: 10, Title: "Feed A", URL: "https://a.example.com/feed"},
			{ID: 11, Title: "Feed B", URL: "https://b.example.com/feed"},
		}
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds", nil)
		c.Set("user", testUser)
		handler.GetFeeds(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var feeds []database.Feed
		if err := json.Unmarshal(w.Body.Bytes(), &feeds); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if len(feeds) != 2 {
			t.Errorf("expected 2 feeds, got %d", len(feeds))
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailGetUserFeeds = true
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds", nil)
		c.Set("user", testUser)
		handler.GetFeeds(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetUnreadCounts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/unread-counts", nil)
		handler.GetUnreadCounts(c)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns unread counts", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/unread-counts", nil)
		c.Set("user", testUser)
		handler.GetUnreadCounts(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var counts map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &counts); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailGetUnreadCounts = true
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/unread-counts", nil)
		c.Set("user", testUser)
		handler.GetUnreadCounts(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetSubscriptionInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User", SubscriptionStatus: "trial"}

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/subscription", nil)
		handler.GetSubscriptionInfo(c)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns subscription info", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/subscription", nil)
		c.Set("user", testUser)
		handler.GetSubscriptionInfo(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var info map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
	})
}

func TestExportOPML(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/export", nil)
		handler.ExportOPML(c)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns OPML with correct headers", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.mockUserFeeds = []database.Feed{
			{ID: 1, Title: "Test Feed", URL: "https://example.com/feed.xml"},
		}
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/export", nil)
		c.Set("user", testUser)
		handler.ExportOPML(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		ct := w.Header().Get("Content-Type")
		if ct != "application/xml; charset=utf-8" {
			t.Errorf("expected Content-Type application/xml, got %q", ct)
		}
		cd := w.Header().Get("Content-Disposition")
		if cd != "attachment; filename=goread2-subscriptions.opml" {
			t.Errorf("unexpected Content-Disposition: %q", cd)
		}
		if w.Body.Len() == 0 {
			t.Error("expected non-empty OPML body")
		}
	})
}

func TestAddFeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

	postFeed := func(body string) *http.Request {
		req := httptest.NewRequest("POST", "/api/feeds", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = postFeed(`{"url":"https://example.com/feed.xml"}`)
		handler.AddFeed(c)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid JSON body returns 400", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = postFeed("not json")
		c.Set("user", testUser)
		handler.AddFeed(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing url returns 400", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = postFeed(`{}`)
		c.Set("user", testUser)
		handler.AddFeed(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("empty url returns 400 invalid URL", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = postFeed(`{"url":""}`)
		c.Set("user", testUser)
		handler.AddFeed(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty URL, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("SSRF-blocked URL returns 400", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = postFeed(`{"url":"http://127.0.0.1/feed.xml"}`)
		c.Set("user", testUser)
		handler.AddFeed(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for SSRF-blocked URL, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("CanUserAddFeed database error returns 500", func(t *testing.T) {
		t.Setenv("SUBSCRIPTION_ENABLED", "true")
		config.ResetForTesting()
		config.Load()
		t.Cleanup(config.ResetForTesting)

		db := newMockDBFeedHandler()
		db.shouldFailGetUserByID = true
		handler := newFeedHandlerWithSubscription(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = postFeed(`{"url":"https://example.com/feed.xml"}`)
		c.Set("user", testUser)
		handler.AddFeed(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("trial expired returns 402", func(t *testing.T) {
		t.Setenv("SUBSCRIPTION_ENABLED", "true")
		config.ResetForTesting()
		config.Load()
		t.Cleanup(config.ResetForTesting)

		db := newMockDBFeedHandler()
		db.mockUser = &database.User{
			ID:                 1,
			SubscriptionStatus: "trial",
			TrialEndsAt:        time.Now().Add(-24 * time.Hour),
		}
		handler := newFeedHandlerWithSubscription(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = postFeed(`{"url":"https://example.com/feed.xml"}`)
		c.Set("user", testUser)
		handler.AddFeed(c)
		if w.Code != http.StatusPaymentRequired {
			t.Errorf("expected 402, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["trial_expired"] != true {
			t.Errorf("expected trial_expired=true in response, got %v", resp)
		}
	})

	t.Run("feed limit reached returns 402", func(t *testing.T) {
		t.Setenv("SUBSCRIPTION_ENABLED", "true")
		config.ResetForTesting()
		config.Load()
		t.Cleanup(config.ResetForTesting)

		db := newMockDBFeedHandler()
		db.mockUser = &database.User{
			ID:                 1,
			SubscriptionStatus: "trial",
			TrialEndsAt:        time.Now().Add(24 * time.Hour),
		}
		db.mockFeedCount = services.FreeTrialFeedLimit
		handler := newFeedHandlerWithSubscription(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = postFeed(`{"url":"https://example.com/feed.xml"}`)
		c.Set("user", testUser)
		handler.AddFeed(c)
		if w.Code != http.StatusPaymentRequired {
			t.Errorf("expected 402, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["limit_reached"] != true {
			t.Errorf("expected limit_reached=true in response, got %v", resp)
		}
	})
}

func TestGetArticlesSingleFeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

	t.Run("invalid feed ID returns 400", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/invalid/articles", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "invalid"}}
		c.Set("user", testUser)
		handler.GetArticles(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("valid feed ID returns paginated articles", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.mockFeedArticles = []database.Article{{ID: 1, Title: "Article 1"}}
		db.mockNextCursor = "next-page-cursor"
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/123/articles?limit=10&cursor=abc&unread_only=true", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
		c.Set("user", testUser)
		handler.GetArticles(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var response struct {
			Articles   []database.Article `json:"articles"`
			NextCursor string             `json:"next_cursor"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if len(response.Articles) != 1 {
			t.Errorf("expected 1 article, got %d", len(response.Articles))
		}
		if response.NextCursor != "next-page-cursor" {
			t.Errorf("expected next_cursor to be passed through, got %q", response.NextCursor)
		}
		if db.capturedPaginationLimit != 10 {
			t.Errorf("expected limit query param 10 to reach GetUserFeedArticlesPaginated, got %d", db.capturedPaginationLimit)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailGetUserFeedArticles = true
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds/123/articles", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
		c.Set("user", testUser)
		handler.GetArticles(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestImportOPML(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

	newOPMLRequest := func(t *testing.T, fieldName, filename string, content []byte) *http.Request {
		t.Helper()
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		part, err := w.CreateFormFile(fieldName, filename)
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("failed to write form file content: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("failed to close multipart writer: %v", err)
		}
		req := httptest.NewRequest("POST", "/api/feeds/import", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		return req
	}

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = newOPMLRequest(t, "opml", "feeds.opml", []byte(`<opml></opml>`))
		handler.ImportOPML(c)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("missing file returns 400", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/feeds/import", bytes.NewReader(nil))
		c.Request.Header.Set("Content-Type", "multipart/form-data; boundary=x")
		c.Set("user", testUser)
		handler.ImportOPML(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("file too large returns 400", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		oversized := bytes.Repeat([]byte("a"), 10*1024*1024+1)
		c.Request = newOPMLRequest(t, "opml", "feeds.opml", oversized)
		c.Set("user", testUser)
		handler.ImportOPML(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("malformed XML returns 500", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = newOPMLRequest(t, "opml", "feeds.opml", []byte(`<opml><body><outline`))
		c.Set("user", testUser)
		handler.ImportOPML(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("empty OPML imports zero feeds", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = newOPMLRequest(t, "opml", "feeds.opml", []byte(`<opml version="1.0"><body></body></opml>`))
		c.Set("user", testUser)
		handler.ImportOPML(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp["imported_count"] != float64(0) {
			t.Errorf("expected imported_count 0, got %v", resp["imported_count"])
		}
	})

	t.Run("trial expired returns 402", func(t *testing.T) {
		t.Setenv("SUBSCRIPTION_ENABLED", "true")
		config.ResetForTesting()
		config.Load()
		t.Cleanup(config.ResetForTesting)

		db := newMockDBFeedHandler()
		db.mockUser = &database.User{
			ID:                 1,
			SubscriptionStatus: "trial",
			TrialEndsAt:        time.Now().Add(-24 * time.Hour),
		}
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		opml := `<opml version="1.0"><body><outline text="Feed" xmlUrl="https://example.com/feed.xml"/></body></opml>`
		c.Request = newOPMLRequest(t, "opml", "feeds.opml", []byte(opml))
		c.Set("user", testUser)
		handler.ImportOPML(c)
		if w.Code != http.StatusPaymentRequired {
			t.Errorf("expected 402, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestRefreshFeeds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("manual refresh success", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/feeds/refresh", nil)
		handler.RefreshFeeds(c)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("manual refresh database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailGetFeeds = true
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/feeds/refresh", nil)
		handler.RefreshFeeds(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("cron path without cron header is forbidden", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/refresh-feeds", nil)
		handler.RefreshFeeds(c)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("cron path enqueues via task queue when configured", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		tq := &fakeTaskQueue{}
		handler.SetTaskQueue(tq)

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/refresh-feeds", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", adminUser)

		handler.RefreshFeeds(c)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected 202, got %d: %s", w.Code, w.Body.String())
		}
		if len(tq.enqueued) != 1 || tq.enqueued[0] != "/tasks/refresh-feeds" {
			t.Errorf("expected /tasks/refresh-feeds enqueued once, got %v", tq.enqueued)
		}
	})

	t.Run("cron path returns 500 when enqueue fails", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		handler.SetTaskQueue(&fakeTaskQueue{shouldFail: true})

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/refresh-feeds", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", adminUser)

		handler.RefreshFeeds(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestTaskRefreshFeeds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("unauthorized without queue header or admin auth", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/refresh-feeds", nil)
		handler.TaskRefreshFeeds(c)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("succeeds via admin auth fallback", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/refresh-feeds", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", adminUser)

		handler.TaskRefreshFeeds(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
		db := newMockDBFeedHandler()
		db.shouldFailGetFeeds = true
		handler := newFeedHandlerWithSubscription(db)

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/refresh-feeds", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", adminUser)

		handler.TaskRefreshFeeds(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDebugAllSubscriptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		handler := newFeedHandlerWithSubscription(newMockDBFeedHandler())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/debug/subscriptions", nil)
		handler.DebugAllSubscriptions(c)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("returns feed statuses", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.mockUserFeeds = []database.Feed{{ID: 1, Title: "Feed A"}}
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/debug/subscriptions", nil)
		c.Set("user", testUser)
		handler.DebugAllSubscriptions(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp["total_feeds"] != float64(1) {
			t.Errorf("expected total_feeds 1, got %v", resp["total_feeds"])
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailGetUserFeeds = true
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/debug/subscriptions", nil)
		c.Set("user", testUser)
		handler.DebugAllSubscriptions(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestDebugArticleByURLFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

	t.Run("article found returns 200", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.mockFoundArticle = &database.Article{ID: 5, Title: "Found Article", URL: "https://example.com/a"}
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/debug/articles?url=https://example.com/a", nil)
		c.Set("user", testUser)
		handler.DebugArticleByURL(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp["found"] != true {
			t.Errorf("expected found=true, got %v", resp)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailFindArticleByURL = true
		handler := newFeedHandlerWithSubscription(db)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/debug/articles?url=https://example.com/a", nil)
		c.Set("user", testUser)
		handler.DebugArticleByURL(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}
