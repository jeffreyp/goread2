package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/services"
)

// Mock database for testing
type mockDBFeedHandler struct {
	shouldFailUnsubscribe       bool
	shouldFailMarkRead          bool
	shouldFailToggleStar        bool
	shouldFailCleanupOrphaned   bool
	shouldFailGetAccountStats   bool
	shouldFailUpdateMaxArticles bool
	articlesDeleted             int
}

func newMockDBFeedHandler() *mockDBFeedHandler {
	return &mockDBFeedHandler{}
}

func (m *mockDBFeedHandler) CreateUser(*database.User) error                  { return nil }
func (m *mockDBFeedHandler) GetUserByGoogleID(string) (*database.User, error) { return nil, nil }
func (m *mockDBFeedHandler) GetUserByID(userID int) (*database.User, error) {
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
func (m *mockDBFeedHandler) GetUserFeedCount(int) (int, error)          { return 0, nil }
func (m *mockDBFeedHandler) UpdateUserMaxArticlesOnFeedAdd(userID, maxArticles int) error {
	if m.shouldFailUpdateMaxArticles {
		return errors.New("database error")
	}
	return nil
}
func (m *mockDBFeedHandler) SetUserAdmin(int, bool) error       { return nil }
func (m *mockDBFeedHandler) GrantFreeMonths(int, int) error     { return nil }
func (m *mockDBFeedHandler) GetUserByEmail(string) (*database.User, error) { return nil, nil }
func (m *mockDBFeedHandler) AddFeed(*database.Feed) error                            { return nil }
func (m *mockDBFeedHandler) UpdateFeed(*database.Feed) error                         { return nil }
func (m *mockDBFeedHandler) UpdateFeedTracking(int, time.Time, time.Time, int) error { return nil }
func (m *mockDBFeedHandler) GetFeeds() ([]database.Feed, error)                      { return nil, nil }
func (m *mockDBFeedHandler) GetFeedByURL(string) (*database.Feed, error)             { return nil, nil }
func (m *mockDBFeedHandler) GetUserFeeds(int) ([]database.Feed, error)               { return nil, nil }
func (m *mockDBFeedHandler) GetAllUserFeeds() ([]database.Feed, error)               { return nil, nil }
func (m *mockDBFeedHandler) DeleteFeed(int) error                                    { return nil }
func (m *mockDBFeedHandler) SubscribeUserToFeed(int, int) error                      { return nil }
func (m *mockDBFeedHandler) UnsubscribeUserFromFeed(userID, feedID int) error {
	if m.shouldFailUnsubscribe {
		return errors.New("database error")
	}
	return nil
}
func (m *mockDBFeedHandler) AddArticle(*database.Article) error                 { return nil }
func (m *mockDBFeedHandler) GetArticles(int) ([]database.Article, error)        { return nil, nil }
func (m *mockDBFeedHandler) FindArticleByURL(string) (*database.Article, error) { return nil, nil }
func (m *mockDBFeedHandler) GetUserArticles(int) ([]database.Article, error)    { return nil, nil }
func (m *mockDBFeedHandler) GetUserArticlesPaginated(userID, limit int, cursor string, unreadOnly bool) (*database.ArticlePaginationResult, error) {
	return &database.ArticlePaginationResult{
		Articles:   []database.Article{},
		NextCursor: "",
	}, nil
}
func (m *mockDBFeedHandler) GetUserFeedArticles(int, int) ([]database.Article, error) {
	return nil, nil
}
func (m *mockDBFeedHandler) GetUserArticleStatus(int, int) (*database.UserArticle, error) {
	return nil, nil
}
func (m *mockDBFeedHandler) SetUserArticleStatus(int, int, bool, bool) error { return nil }
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
func (m *mockDBFeedHandler) GetUserUnreadCounts(int) (map[int]int, error) { return nil, nil }
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
		c.Set("user", regularUser)

		handler.CleanupOrphanedUserArticles(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("unauthorized without authentication in local environment", func(t *testing.T) {
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

	t.Run("database error returns 500", func(t *testing.T) {
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
		c.Set("user", adminUser)

		handler.CleanupOrphanedUserArticles(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
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
