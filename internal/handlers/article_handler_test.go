package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/services"
)

func newArticleHandler(db *mockDBFeedHandler) *ArticleHandler {
	rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
	})
	feedService := services.NewFeedService(db, rateLimiter)
	return NewArticleHandler(feedService)
}

func TestGetArticle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		db := newMockDBFeedHandler()
		handler := newArticleHandler(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/articles/1", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

		handler.GetArticle(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid article ID returns 400", func(t *testing.T) {
		db := newMockDBFeedHandler()
		handler := newArticleHandler(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/articles/abc", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "abc"}}
		c.Set("user", testUser)

		handler.GetArticle(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("database error returns 500", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.shouldFailGetArticle = true
		handler := newArticleHandler(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/articles/42", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "42"}}
		c.Set("user", testUser)

		handler.GetArticle(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})

	t.Run("article not found returns 404", func(t *testing.T) {
		db := newMockDBFeedHandler()
		// mockArticle is nil by default — GetArticleByID returns nil, nil
		handler := newArticleHandler(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/articles/99", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "99"}}
		c.Set("user", testUser)

		handler.GetArticle(c)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})

	t.Run("happy path returns 200 with article", func(t *testing.T) {
		db := newMockDBFeedHandler()
		db.mockArticle = &database.Article{
			ID:          42,
			FeedID:      1,
			Title:       "Test Article",
			URL:         "https://example.com/article",
			Content:     "<p>Test content</p>",
			PublishedAt: time.Now(),
		}
		handler := newArticleHandler(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/articles/42", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "42"}}
		c.Set("user", testUser)

		handler.GetArticle(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var article database.Article
		if err := json.Unmarshal(w.Body.Bytes(), &article); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if article.ID != 42 {
			t.Errorf("expected article ID 42, got %d", article.ID)
		}
		if article.Title != "Test Article" {
			t.Errorf("expected title 'Test Article', got %q", article.Title)
		}
	})
}
