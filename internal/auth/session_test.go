package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goread2/internal/database"
)

// mockDB implements database.Database interface for testing
type mockDB struct{}

func (m *mockDB) Close() error { return nil }
func (m *mockDB) CreateUser(*database.User) error { return nil }
func (m *mockDB) GetUserByGoogleID(string) (*database.User, error) { return nil, nil }
func (m *mockDB) GetUserByID(int) (*database.User, error) { return nil, nil }
func (m *mockDB) UpdateUserSubscription(int, string, string, time.Time) error { return nil }
func (m *mockDB) IsUserSubscriptionActive(int) (bool, error) { return false, nil }
func (m *mockDB) GetUserFeedCount(int) (int, error) { return 0, nil }
func (m *mockDB) SetUserAdmin(int, bool) error { return nil }
func (m *mockDB) GrantFreeMonths(int, int) error { return nil }
func (m *mockDB) GetUserByEmail(string) (*database.User, error) { return nil, nil }
func (m *mockDB) AddFeed(*database.Feed) error { return nil }
func (m *mockDB) GetFeeds() ([]database.Feed, error) { return nil, nil }
func (m *mockDB) GetUserFeeds(int) ([]database.Feed, error) { return nil, nil }
func (m *mockDB) GetAllUserFeeds() ([]database.Feed, error) { return nil, nil }
func (m *mockDB) DeleteFeed(int) error { return nil }
func (m *mockDB) SubscribeUserToFeed(int, int) error { return nil }
func (m *mockDB) UnsubscribeUserFromFeed(int, int) error { return nil }
func (m *mockDB) AddArticle(*database.Article) error { return nil }
func (m *mockDB) GetArticles(int) ([]database.Article, error) { return nil, nil }
func (m *mockDB) GetUserArticles(int) ([]database.Article, error) { return nil, nil }
func (m *mockDB) GetUserArticlesPaginated(int, int, int) ([]database.Article, error) { return nil, nil }
func (m *mockDB) GetUserFeedArticles(int, int) ([]database.Article, error) { return nil, nil }
func (m *mockDB) GetUserArticleStatus(int, int) (*database.UserArticle, error) { return nil, nil }
func (m *mockDB) SetUserArticleStatus(int, int, bool, bool) error { return nil }
func (m *mockDB) BatchSetUserArticleStatus(int, []database.Article, bool, bool) error { return nil }
func (m *mockDB) MarkUserArticleRead(int, int, bool) error { return nil }
func (m *mockDB) ToggleUserArticleStar(int, int) error { return nil }
func (m *mockDB) GetUserUnreadCounts(int) (map[int]int, error) { return nil, nil }
func (m *mockDB) GetAllArticles() ([]database.Article, error) { return nil, nil }
func (m *mockDB) UpdateFeedLastFetch(int, time.Time) error { return nil }

func TestNewSessionManager(t *testing.T) {
	db := &mockDB{}
	defer func() { _ = db.Close() }()

	sm := NewSessionManager(db)
	if sm == nil {
		t.Fatal("NewSessionManager returned nil")
	}

	if sm.sessions == nil {
		t.Error("sessions map not initialized")
	}

	if sm.db == nil {
		t.Error("database not set")
	}
}

func TestCreateSession(t *testing.T) {
	db := &mockDB{}
	defer func() { _ = db.Close() }()

	sm := NewSessionManager(db)
	
	user := &database.User{
		ID:       1,
		GoogleID: "test123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	session, err := sm.CreateSession(user)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session == nil {
		t.Fatal("CreateSession returned nil session")
	}

	if session.ID == "" {
		t.Error("Session ID is empty")
	}

	if session.UserID != user.ID {
		t.Errorf("Session UserID = %d, want %d", session.UserID, user.ID)
	}

	if session.User != user {
		t.Error("Session User not set correctly")
	}

	if session.CreatedAt.IsZero() {
		t.Error("Session CreatedAt not set")
	}

	if session.ExpiresAt.IsZero() {
		t.Error("Session ExpiresAt not set")
	}

	expectedExpiration := session.CreatedAt.Add(7 * 24 * time.Hour)
	if session.ExpiresAt.Before(expectedExpiration.Add(-time.Minute)) ||
		session.ExpiresAt.After(expectedExpiration.Add(time.Minute)) {
		t.Error("Session expiration time not set correctly (should be ~7 days)")
	}
}

func TestGetSession(t *testing.T) {
	db := &mockDB{}
	defer func() { _ = db.Close() }()

	sm := NewSessionManager(db)
	
	user := &database.User{
		ID:       1,
		GoogleID: "test123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	// Create a session
	session, err := sm.CreateSession(user)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Get the session
	retrievedSession, exists := sm.GetSession(session.ID)
	if !exists {
		t.Error("GetSession should return true for existing session")
	}
	if retrievedSession == nil {
		t.Fatal("GetSession returned nil")
	}

	if retrievedSession.ID != session.ID {
		t.Error("Retrieved session ID doesn't match")
	}

	// Test non-existent session
	nonExistentSession, exists := sm.GetSession("nonexistent")
	if exists {
		t.Error("GetSession should return false for non-existent session")
	}
	if nonExistentSession != nil {
		t.Error("GetSession should return nil for non-existent session")
	}
}

func TestDeleteSession(t *testing.T) {
	db := &mockDB{}
	defer func() { _ = db.Close() }()

	sm := NewSessionManager(db)
	
	user := &database.User{
		ID:       1,
		GoogleID: "test123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	// Create a session
	session, err := sm.CreateSession(user)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify session exists
	_, exists := sm.GetSession(session.ID)
	if !exists {
		t.Error("Session should exist before deletion")
	}

	// Delete the session
	sm.DeleteSession(session.ID)

	// Verify session is deleted
	_, exists = sm.GetSession(session.ID)
	if exists {
		t.Error("Session should not exist after deletion")
	}

	// Test deleting non-existent session (should not panic)
	sm.DeleteSession("nonexistent")
}

func TestGenerateSessionID(t *testing.T) {
	id1, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID failed: %v", err)
	}

	if id1 == "" {
		t.Error("generateSessionID returned empty string")
	}

	id2, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID failed on second call: %v", err)
	}

	if id1 == id2 {
		t.Error("generateSessionID should return unique IDs")
	}

	// Check that ID has reasonable length (base64 encoded 32 bytes should be ~44 chars)
	if len(id1) < 40 {
		t.Errorf("Session ID seems too short: %d characters", len(id1))
	}
}

func TestSessionExpiration(t *testing.T) {
	db := &mockDB{}
	defer func() { _ = db.Close() }()

	sm := NewSessionManager(db)
	
	user := &database.User{
		ID:       1,
		GoogleID: "test123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	// Create a session
	session, err := sm.CreateSession(user)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Manually expire the session
	sm.mu.Lock()
	sm.sessions[session.ID].ExpiresAt = time.Now().Add(-time.Hour)
	sm.mu.Unlock()

	// GetSession should return nil for expired sessions
	retrievedSession, exists := sm.GetSession(session.ID)
	if exists {
		t.Error("GetSession should return false for expired session")
	}
	if retrievedSession != nil {
		t.Error("GetSession should return nil for expired session")
	}

	// Session should be removed from memory
	sm.mu.RLock()
	_, stillExists := sm.sessions[session.ID]
	sm.mu.RUnlock()

	if stillExists {
		t.Error("Expired session should be removed from memory")
	}
}

func TestSetSessionCookie(t *testing.T) {
	db := &mockDB{}
	defer func() { _ = db.Close() }()

	sm := NewSessionManager(db)
	
	user := &database.User{
		ID:       1,
		GoogleID: "test123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	session, err := sm.CreateSession(user)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	w := httptest.NewRecorder()
	sm.SetSessionCookie(w, session)

	response := w.Result()
	cookies := response.Cookies()
	
	if len(cookies) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(cookies))
		return
	}

	cookie := cookies[0]
	if cookie.Name != "session_id" {
		t.Errorf("Cookie name = %s, want session_id", cookie.Name)
	}

	if cookie.Value != session.ID {
		t.Errorf("Cookie value = %s, want %s", cookie.Value, session.ID)
	}

	if !cookie.HttpOnly {
		t.Error("Cookie should be HttpOnly")
	}

	if cookie.Path != "/" {
		t.Errorf("Cookie path = %s, want /", cookie.Path)
	}
}

func TestClearSessionCookie(t *testing.T) {
	db := &mockDB{}
	defer func() { _ = db.Close() }()

	sm := NewSessionManager(db)

	w := httptest.NewRecorder()
	sm.ClearSessionCookie(w)

	response := w.Result()
	cookies := response.Cookies()
	
	if len(cookies) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(cookies))
		return
	}

	cookie := cookies[0]
	if cookie.Name != "session_id" {
		t.Errorf("Cookie name = %s, want session_id", cookie.Name)
	}

	if cookie.Value != "" {
		t.Errorf("Cookie value should be empty, got %s", cookie.Value)
	}

	if !cookie.HttpOnly {
		t.Error("Cookie should be HttpOnly")
	}

	if cookie.Expires.Unix() != 0 {
		t.Error("Cookie should be expired (Unix timestamp 0)")
	}
}

func TestGetSessionFromRequest(t *testing.T) {
	db := &mockDB{}
	defer func() { _ = db.Close() }()

	sm := NewSessionManager(db)
	
	user := &database.User{
		ID:       1,
		GoogleID: "test123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	session, err := sm.CreateSession(user)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Test with valid session cookie
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: session.ID,
	})

	retrievedSession, exists := sm.GetSessionFromRequest(req)
	if !exists {
		t.Error("GetSessionFromRequest should return true for valid cookie")
	}
	if retrievedSession == nil {
		t.Fatal("GetSessionFromRequest returned nil session")
	}
	if retrievedSession.ID != session.ID {
		t.Error("Retrieved session ID doesn't match")
	}

	// Test with no cookie
	reqNoCookie := httptest.NewRequest("GET", "/", nil)
	retrievedSession, exists = sm.GetSessionFromRequest(reqNoCookie)
	if exists {
		t.Error("GetSessionFromRequest should return false when no cookie present")
	}
	if retrievedSession != nil {
		t.Error("GetSessionFromRequest should return nil when no cookie present")
	}

	// Test with invalid session ID in cookie
	reqInvalidCookie := httptest.NewRequest("GET", "/", nil)
	reqInvalidCookie.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "invalid_session_id",
	})

	retrievedSession, exists = sm.GetSessionFromRequest(reqInvalidCookie)
	if exists {
		t.Error("GetSessionFromRequest should return false for invalid session ID")
	}
	if retrievedSession != nil {
		t.Error("GetSessionFromRequest should return nil for invalid session ID")
	}
}