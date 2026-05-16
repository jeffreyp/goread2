package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/database"
)

// mockDBAuthHandler is a minimal mock database for auth handler tests
type mockDBAuthHandler struct{}

func (m *mockDBAuthHandler) CreateUser(*database.User) error                  { return nil }
func (m *mockDBAuthHandler) GetUserByGoogleID(string) (*database.User, error) { return nil, nil }
func (m *mockDBAuthHandler) GetUserByID(int) (*database.User, error)          { return nil, nil }
func (m *mockDBAuthHandler) UpdateUserSubscription(int, string, string, time.Time, time.Time) error {
	return nil
}
func (m *mockDBAuthHandler) IsUserSubscriptionActive(int) (bool, error)              { return false, nil }
func (m *mockDBAuthHandler) GetUserFeedCount(int) (int, error)                       { return 0, nil }
func (m *mockDBAuthHandler) UpdateUserMaxArticlesOnFeedAdd(int, int) error           { return nil }
func (m *mockDBAuthHandler) SetUserAdmin(int, bool) error                            { return nil }
func (m *mockDBAuthHandler) GrantFreeMonths(int, int) error                          { return nil }
func (m *mockDBAuthHandler) GetUserByEmail(string) (*database.User, error)           { return nil, nil }
func (m *mockDBAuthHandler) AddFeed(*database.Feed) error                            { return nil }
func (m *mockDBAuthHandler) UpdateFeed(*database.Feed) error                         { return nil }
func (m *mockDBAuthHandler) UpdateFeedTracking(int, time.Time, time.Time, int) error { return nil }
func (m *mockDBAuthHandler) GetFeeds() ([]database.Feed, error)                      { return nil, nil }
func (m *mockDBAuthHandler) GetFeedByURL(string) (*database.Feed, error)             { return nil, nil }
func (m *mockDBAuthHandler) GetUserFeeds(int) ([]database.Feed, error)               { return nil, nil }
func (m *mockDBAuthHandler) GetAllUserFeeds() ([]database.Feed, error)               { return nil, nil }
func (m *mockDBAuthHandler) DeleteFeed(int) error                                    { return nil }
func (m *mockDBAuthHandler) SubscribeUserToFeed(int, int) error                      { return nil }
func (m *mockDBAuthHandler) UnsubscribeUserFromFeed(int, int) error                  { return nil }
func (m *mockDBAuthHandler) AddArticle(*database.Article) error                      { return nil }
func (m *mockDBAuthHandler) GetArticles(int) ([]database.Article, error)             { return nil, nil }
func (m *mockDBAuthHandler) FindArticleByURL(string) (*database.Article, error)      { return nil, nil }
func (m *mockDBAuthHandler) GetUserArticles(int) ([]database.Article, error)         { return nil, nil }
func (m *mockDBAuthHandler) GetUserArticlesPaginated(int, int, string, bool) (*database.ArticlePaginationResult, error) {
	return &database.ArticlePaginationResult{}, nil
}
func (m *mockDBAuthHandler) GetUserFeedArticles(int, int) ([]database.Article, error) { return nil, nil }
func (m *mockDBAuthHandler) GetUserArticleStatus(int, int) (*database.UserArticle, error) {
	return nil, nil
}
func (m *mockDBAuthHandler) SetUserArticleStatus(int, int, bool, bool) error { return nil }
func (m *mockDBAuthHandler) BatchSetUserArticleStatus(int, []database.Article, bool, bool) error {
	return nil
}
func (m *mockDBAuthHandler) MarkUserArticleRead(int, int, bool) error         { return nil }
func (m *mockDBAuthHandler) ToggleUserArticleStar(int, int) error             { return nil }
func (m *mockDBAuthHandler) GetUserUnreadCounts(int) (map[int]int, error)     { return nil, nil }
func (m *mockDBAuthHandler) CleanupOrphanedUserArticles(int) (int, error)     { return 0, nil }
func (m *mockDBAuthHandler) GetUserArticlesByIDs(int, []int) ([]database.Article, error) {
	return nil, nil
}
func (m *mockDBAuthHandler) GetSession(string) (*database.Session, error) { return nil, nil }
func (m *mockDBAuthHandler) CreateSession(*database.Session) error        { return nil }
func (m *mockDBAuthHandler) UpdateSessionExpiry(string, time.Time) error  { return nil }
func (m *mockDBAuthHandler) DeleteSession(string) error                   { return nil }
func (m *mockDBAuthHandler) GetExpiredSessions() ([]database.Session, error) {
	return nil, nil
}
func (m *mockDBAuthHandler) DeleteExpiredSessions() error { return nil }
func (m *mockDBAuthHandler) UpdateLastActiveTime(int) error { return nil }
func (m *mockDBAuthHandler) CreateAuditLog(*database.AuditLog) error { return nil }
func (m *mockDBAuthHandler) GetAuditLogs(int, int, map[string]interface{}) ([]database.AuditLog, error) {
	return nil, nil
}
func (m *mockDBAuthHandler) GetAccountStats(int) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (m *mockDBAuthHandler) UpdateFeedLastFetch(int, time.Time) error { return nil }
func (m *mockDBAuthHandler) Close() error                             { return nil }
func (m *mockDBAuthHandler) UpdateFeedCacheHeaders(feedID int, etag, lastModified string) error {
	return nil
}
func (m *mockDBAuthHandler) FilterExistingArticleURLs(int, []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (m *mockDBAuthHandler) UpdateFeedAfterRefresh(int, time.Time, time.Time, int, time.Time, string, string) error {
	return nil
}

func TestNewAuthHandler(t *testing.T) {
	// Create mock services
	mockAuthService := &auth.AuthService{}
	mockSessionManager := &auth.SessionManager{}
	mockCSRFManager := auth.NewCSRFManager()

	handler := NewAuthHandler(mockAuthService, mockSessionManager, mockCSRFManager)

	if handler == nil {
		t.Fatal("NewAuthHandler returned nil")
		return
	}

	if handler.authService != mockAuthService {
		t.Error("AuthHandler auth service not set correctly")
	}

	if handler.sessionManager != mockSessionManager {
		t.Error("AuthHandler session manager not set correctly")
	}

	if handler.csrfManager != mockCSRFManager {
		t.Error("AuthHandler CSRF manager not set correctly")
	}
}


// TestOAuthStateExpiration tests that expired states are rejected
func TestOAuthStateExpiration(t *testing.T) {
	db := &mockDBAuthHandler{}
	sessionManager := auth.NewSessionManager(db)

	// Store a state
	state := "test-state-12345"
	sessionManager.StoreOAuthState(state)

	// Validate immediately - should succeed
	if !sessionManager.ValidateAndConsumeOAuthState(state) {
		t.Error("Fresh state should be valid")
	}

	// Try to validate again - should fail (already consumed)
	if sessionManager.ValidateAndConsumeOAuthState(state) {
		t.Error("Consumed state should not be valid again")
	}
}

// TestOAuthStateInvalidState tests that invalid/unknown states are rejected
func TestOAuthStateInvalidState(t *testing.T) {
	db := &mockDBAuthHandler{}
	sessionManager := auth.NewSessionManager(db)

	// Try to validate a state that was never stored
	if sessionManager.ValidateAndConsumeOAuthState("unknown-state") {
		t.Error("Unknown state should be invalid")
	}
}

func TestCleanupExpiredSessions_CronAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newHandler := func() *AuthHandler {
		db := &mockDBAuthHandler{}
		sessionManager := auth.NewSessionManager(db)
		csrfManager := auth.NewCSRFManager()
		return NewAuthHandler(nil, sessionManager, csrfManager)
	}

	t.Run("admin with valid token succeeds", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-cron-token")
		handler := newHandler()

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-sessions", nil)
		c.Request.Header.Set("X-Admin-Token", "test-cron-token")
		c.Set("user", adminUser)

		handler.CleanupExpiredSessions(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("admin without X-Admin-Token header is rejected", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-cron-token")
		handler := newHandler()

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-sessions", nil)
		c.Set("user", adminUser)

		handler.CleanupExpiredSessions(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
	})

	t.Run("admin with wrong X-Admin-Token is rejected", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-cron-token")
		handler := newHandler()

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-sessions", nil)
		c.Request.Header.Set("X-Admin-Token", "wrong-token")
		c.Set("user", adminUser)

		handler.CleanupExpiredSessions(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
	})

	t.Run("non-admin is rejected even with valid token", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "test-cron-token")
		handler := newHandler()

		regularUser := &database.User{ID: 2, Email: "user@example.com", IsAdmin: false}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-sessions", nil)
		c.Request.Header.Set("X-Admin-Token", "test-cron-token")
		c.Set("user", regularUser)

		handler.CleanupExpiredSessions(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
	})

	t.Run("blocked when ADMIN_TOKEN not configured", func(t *testing.T) {
		t.Setenv("ADMIN_TOKEN", "")
		handler := newHandler()

		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/cron/cleanup-sessions", nil)
		c.Request.Header.Set("X-Admin-Token", "any-token")
		c.Set("user", adminUser)

		handler.CleanupExpiredSessions(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
	})
}
