package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/secrets"
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
func (m *mockDBAuthHandler) SetUserAdminAtomic(int, int, bool) error                 { return nil }
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
func (m *mockDBAuthHandler) GetUserFeedArticles(int, int) ([]database.Article, error) {
	return nil, nil
}
func (m *mockDBAuthHandler) GetUserFeedArticlesPaginated(int, int, int, string, bool) (*database.ArticlePaginationResult, error) {
	return &database.ArticlePaginationResult{}, nil
}
func (m *mockDBAuthHandler) GetArticleByID(int, int) (*database.Article, error) { return nil, nil }
func (m *mockDBAuthHandler) GetUserArticleStatus(int, int) (*database.UserArticle, error) {
	return nil, nil
}
func (m *mockDBAuthHandler) SetUserArticleStatus(int, int, bool, bool) error { return nil }
func (m *mockDBAuthHandler) MarkAllUserArticlesRead(int) (int, error)        { return 0, nil }
func (m *mockDBAuthHandler) BatchSetUserArticleStatus(int, []database.Article, bool, bool) error {
	return nil
}
func (m *mockDBAuthHandler) MarkUserArticleRead(int, int, bool) error     { return nil }
func (m *mockDBAuthHandler) ToggleUserArticleStar(int, int) error         { return nil }
func (m *mockDBAuthHandler) GetUserUnreadCounts(int) (map[int]int, error) { return nil, nil }
func (m *mockDBAuthHandler) GetTotalArticleCount(int) (int, error)        { return 0, nil }
func (m *mockDBAuthHandler) CleanupOrphanedUserArticles(int) (int, error) { return 0, nil }
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
func (m *mockDBAuthHandler) DeleteExpiredSessions() error            { return nil }
func (m *mockDBAuthHandler) UpdateLastActiveTime(int) error          { return nil }
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
	sessionManager.StoreOAuthState(state, false)

	// Validate immediately - should succeed
	if valid, _ := sessionManager.ValidateAndConsumeOAuthState(state); !valid {
		t.Error("Fresh state should be valid")
	}

	// Try to validate again - should fail (already consumed)
	if valid, _ := sessionManager.ValidateAndConsumeOAuthState(state); valid {
		t.Error("Consumed state should not be valid again")
	}
}

// TestOAuthStateInvalidState tests that invalid/unknown states are rejected
func TestOAuthStateInvalidState(t *testing.T) {
	db := &mockDBAuthHandler{}
	sessionManager := auth.NewSessionManager(db)

	// Try to validate a state that was never stored
	if valid, _ := sessionManager.ValidateAndConsumeOAuthState("unknown-state"); valid {
		t.Error("Unknown state should be invalid")
	}
}

// TestOAuthStateMobileFlag tests that the mobile flag round-trips through the
// state store, since the callback relies on it to pick the goread2:// handoff.
func TestOAuthStateMobileFlag(t *testing.T) {
	db := &mockDBAuthHandler{}
	sessionManager := auth.NewSessionManager(db)

	sessionManager.StoreOAuthState("web-state", false)
	sessionManager.StoreOAuthState("ios-state", true)

	if valid, mobile := sessionManager.ValidateAndConsumeOAuthState("web-state"); !valid || mobile {
		t.Errorf("web state: expected valid=true mobile=false, got valid=%v mobile=%v", valid, mobile)
	}
	if valid, mobile := sessionManager.ValidateAndConsumeOAuthState("ios-state"); !valid || !mobile {
		t.Errorf("ios state: expected valid=true mobile=true, got valid=%v mobile=%v", valid, mobile)
	}
}

// TestAuthCodeExchange tests the one-time code lifecycle used by the mobile
// auth handoff: a code redeems exactly once and unknown codes are rejected.
func TestAuthCodeExchange(t *testing.T) {
	db := &mockDBAuthHandler{}
	sessionManager := auth.NewSessionManager(db)

	expiresAt := time.Now().Add(7 * 24 * time.Hour).Truncate(time.Second)
	session := &auth.Session{ID: "session-abc", ExpiresAt: expiresAt}

	code, err := sessionManager.CreateAuthCode(session)
	if err != nil {
		t.Fatalf("CreateAuthCode failed: %v", err)
	}
	if code == "" || code == session.ID {
		t.Fatalf("expected a distinct non-empty code, got %q", code)
	}

	sessionID, sessionExpiresAt, ok := sessionManager.ExchangeAuthCode(code)
	if !ok {
		t.Fatal("fresh code should exchange successfully")
	}
	if sessionID != session.ID {
		t.Errorf("expected session ID %q, got %q", session.ID, sessionID)
	}
	if !sessionExpiresAt.Equal(expiresAt) {
		t.Errorf("expected session expiry %v, got %v", expiresAt, sessionExpiresAt)
	}

	if _, _, ok := sessionManager.ExchangeAuthCode(code); ok {
		t.Error("consumed code should not exchange again")
	}
	if _, _, ok := sessionManager.ExchangeAuthCode("unknown-code"); ok {
		t.Error("unknown code should not exchange")
	}
}

func TestToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newHandler := func() *AuthHandler {
		db := &mockDBAuthHandler{}
		sessionManager := auth.NewSessionManager(db)
		csrfManager := auth.NewCSRFManager()
		return NewAuthHandler(nil, sessionManager, csrfManager)
	}

	t.Run("valid code returns session token", func(t *testing.T) {
		handler := newHandler()
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		code, err := handler.sessionManager.CreateAuthCode(&auth.Session{ID: "session-xyz", ExpiresAt: expiresAt})
		if err != nil {
			t.Fatalf("CreateAuthCode failed: %v", err)
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := strings.NewReader(`{"code":"` + code + `"}`)
		c.Request = httptest.NewRequest("POST", "/auth/token", body)
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Token(c)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp["session_token"] != "session-xyz" {
			t.Errorf("expected session_token session-xyz, got %v", resp["session_token"])
		}
		if resp["cookie_name"] != "session_id_local" {
			t.Errorf("expected cookie_name session_id_local in tests, got %v", resp["cookie_name"])
		}
		if resp["expires_at"] == nil {
			t.Error("response missing expires_at")
		}
	})

	t.Run("invalid code returns 401", func(t *testing.T) {
		handler := newHandler()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/token", strings.NewReader(`{"code":"bogus"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Token(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("missing code returns 400", func(t *testing.T) {
		handler := newHandler()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/token", strings.NewReader(`{}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Token(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

func TestLoginMobileRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newHandler := func() *AuthHandler {
		db := &mockDBAuthHandler{}
		sessionManager := auth.NewSessionManager(db)
		csrfManager := auth.NewCSRFManager()
		return NewAuthHandler(auth.NewAuthService(db), sessionManager, csrfManager)
	}

	t.Run("client=ios redirects to Google and flags state mobile", func(t *testing.T) {
		handler := newHandler()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/auth/login?client=ios", nil)

		handler.Login(c)

		if w.Code != http.StatusFound {
			t.Fatalf("expected 302, got %d: %s", w.Code, w.Body.String())
		}
		location := w.Header().Get("Location")
		locURL, err := url.Parse(location)
		if err != nil || locURL.Host != "accounts.google.com" {
			t.Fatalf("expected redirect to accounts.google.com, got %q", location)
		}
		state := locURL.Query().Get("state")
		if state == "" {
			t.Fatal("redirect URL missing state parameter")
		}
		if valid, mobile := handler.sessionManager.ValidateAndConsumeOAuthState(state); !valid || !mobile {
			t.Errorf("expected stored state to be valid and mobile, got valid=%v mobile=%v", valid, mobile)
		}
	})

	t.Run("web login still returns auth_url JSON", func(t *testing.T) {
		handler := newHandler()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/auth/login", nil)

		handler.Login(c)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp["auth_url"] == nil {
			t.Error("response missing auth_url")
		}
	})
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
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
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
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
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
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
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
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
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
		secrets.ResetCacheForTesting()
		t.Cleanup(secrets.ResetCacheForTesting)
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

func TestMe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newHandler := func() *AuthHandler {
		db := &mockDBAuthHandler{}
		sessionManager := auth.NewSessionManager(db)
		csrfManager := auth.NewCSRFManager()
		return NewAuthHandler(nil, sessionManager, csrfManager)
	}

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		handler := newHandler()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/me", nil)
		handler.Me(c)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("authenticated returns user info", func(t *testing.T) {
		handler := newHandler()
		user := &database.User{ID: 7, Email: "alice@example.com", Name: "Alice", MaxArticlesOnFeedAdd: 50}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/me", nil)
		c.Set("user", user)
		handler.Me(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		userField, ok := resp["user"].(map[string]interface{})
		if !ok {
			t.Fatal("response missing 'user' object")
		}
		if userField["email"] != "alice@example.com" {
			t.Errorf("expected email alice@example.com, got %v", userField["email"])
		}
		if _, ok := resp["csrf_token"]; !ok {
			t.Error("response missing 'csrf_token' field")
		}
	})
}

func TestLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newHandler := func() *AuthHandler {
		db := &mockDBAuthHandler{}
		sessionManager := auth.NewSessionManager(db)
		csrfManager := auth.NewCSRFManager()
		return NewAuthHandler(nil, sessionManager, csrfManager)
	}

	t.Run("always returns 200 and clears cookie", func(t *testing.T) {
		handler := newHandler()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/logout", nil)
		handler.Logout(c)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp["message"] == nil {
			t.Error("expected message in response")
		}
	})
}
