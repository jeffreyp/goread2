package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goread2/internal/database"
)

// mockDB implements database.Database interface for testing
type mockDB struct {
	sessions map[string]*database.Session
	users    map[int]*database.User
}

func newMockDB() *mockDB {
	return &mockDB{
		sessions: make(map[string]*database.Session),
		users:    make(map[int]*database.User),
	}
}

func (m *mockDB) Close() error                                                        { return nil }
func (m *mockDB) CreateUser(*database.User) error                                     { return nil }
func (m *mockDB) GetUserByGoogleID(string) (*database.User, error)                    { return nil, nil }
func (m *mockDB) GetUserByID(userID int) (*database.User, error) {
	if user, exists := m.users[userID]; exists {
		return user, nil
	}
	// Return a default user for test purposes
	return &database.User{ID: userID, Email: "test@example.com", Name: "Test User"}, nil
}
func (m *mockDB) UpdateUserSubscription(int, string, string, time.Time, time.Time) error { return nil }
func (m *mockDB) IsUserSubscriptionActive(int) (bool, error)                          { return false, nil }
func (m *mockDB) GetUserFeedCount(int) (int, error)                                   { return 0, nil }
func (m *mockDB) SetUserAdmin(int, bool) error                                        { return nil }
func (m *mockDB) GrantFreeMonths(int, int) error                                      { return nil }
func (m *mockDB) GetUserByEmail(string) (*database.User, error)                       { return nil, nil }
func (m *mockDB) AddFeed(*database.Feed) error                                        { return nil }
func (m *mockDB) UpdateFeed(*database.Feed) error                                     { return nil }
func (m *mockDB) GetFeeds() ([]database.Feed, error)                                  { return nil, nil }
func (m *mockDB) GetFeedByURL(string) (*database.Feed, error)                         { return nil, nil }
func (m *mockDB) GetUserFeeds(int) ([]database.Feed, error)                           { return nil, nil }
func (m *mockDB) GetAllUserFeeds() ([]database.Feed, error)                           { return nil, nil }
func (m *mockDB) DeleteFeed(int) error                                                { return nil }
func (m *mockDB) SubscribeUserToFeed(int, int) error                                  { return nil }
func (m *mockDB) UnsubscribeUserFromFeed(int, int) error                              { return nil }
func (m *mockDB) AddArticle(*database.Article) error                                  { return nil }
func (m *mockDB) GetArticles(int) ([]database.Article, error)                         { return nil, nil }
func (m *mockDB) FindArticleByURL(string) (*database.Article, error)                  { return nil, nil }
func (m *mockDB) GetUserArticles(int) ([]database.Article, error)                     { return nil, nil }
func (m *mockDB) GetUserArticlesPaginated(int, int, int, bool) ([]database.Article, error)  { return nil, nil }
func (m *mockDB) GetUserFeedArticles(int, int) ([]database.Article, error)            { return nil, nil }
func (m *mockDB) GetUserArticleStatus(int, int) (*database.UserArticle, error)        { return nil, nil }
func (m *mockDB) SetUserArticleStatus(int, int, bool, bool) error                     { return nil }
func (m *mockDB) BatchSetUserArticleStatus(int, []database.Article, bool, bool) error { return nil }
func (m *mockDB) MarkUserArticleRead(int, int, bool) error                            { return nil }
func (m *mockDB) ToggleUserArticleStar(int, int) error                                { return nil }
func (m *mockDB) GetUserUnreadCounts(int) (map[int]int, error)                        { return nil, nil }
func (m *mockDB) GetAllArticles() ([]database.Article, error)                         { return nil, nil }
func (m *mockDB) UpdateFeedLastFetch(int, time.Time) error                            { return nil }
func (m *mockDB) UpdateUserMaxArticlesOnFeedAdd(int, int) error                       { return nil }
func (m *mockDB) CreateAuditLog(*database.AuditLog) error                             { return nil }
func (m *mockDB) GetAuditLogs(int, int, map[string]interface{}) ([]database.AuditLog, error) { return nil, nil }

func (m *mockDB) CreateSession(s *database.Session) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockDB) GetSession(id string) (*database.Session, error) {
	if s, exists := m.sessions[id]; exists {
		return s, nil
	}
	return nil, nil
}

func (m *mockDB) DeleteSession(id string) error {
	delete(m.sessions, id)
	return nil
}

func (m *mockDB) DeleteExpiredSessions() error {
	for id, s := range m.sessions {
		if time.Now().After(s.ExpiresAt) {
			delete(m.sessions, id)
		}
	}
	return nil
}

func TestNewSessionManager(t *testing.T) {
	db := newMockDB()
	defer func() { _ = db.Close() }()

	sm := NewSessionManager(db)
	if sm == nil {
		t.Fatal("NewSessionManager returned nil")
	}

	if sm.db == nil {
		t.Error("database not set")
	}
}

func TestCreateSession(t *testing.T) {
	db := newMockDB()
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
	db := newMockDB()
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
	db := newMockDB()
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
	db := newMockDB()
	defer func() { _ = db.Close() }()

	sm := NewSessionManager(db)

	user := &database.User{
		ID:       1,
		GoogleID: "test123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	// Create an expired session directly in the database
	expiredSessionID := "expired_session_123"
	expiredSession := &database.Session{
		ID:        expiredSessionID,
		UserID:    user.ID,
		CreatedAt: time.Now().Add(-8 * 24 * time.Hour), // Created 8 days ago
		ExpiresAt: time.Now().Add(-1 * time.Hour),      // Expired 1 hour ago
	}

	err := db.CreateSession(expiredSession)
	if err != nil {
		t.Fatalf("Failed to create expired session: %v", err)
	}

	// Verify the expired session was created in the database
	dbSession, err := db.GetSession(expiredSessionID)
	if err != nil || dbSession == nil {
		t.Fatal("Expired session should exist in database before GetSession call")
	}

	// GetSession should return nil for expired sessions
	session, exists := sm.GetSession(expiredSessionID)
	if exists {
		t.Error("GetSession should return false for expired session")
	}
	if session != nil {
		t.Error("GetSession should return nil for expired session")
	}

	// Verify the expired session was deleted from the database
	dbSession, err = db.GetSession(expiredSessionID)
	if err != nil {
		t.Fatalf("Failed to check if session was deleted: %v", err)
	}
	if dbSession != nil {
		t.Error("Expired session should be deleted from database after GetSession call")
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	db := newMockDB()
	defer func() { _ = db.Close() }()

	sm := NewSessionManager(db)

	user := &database.User{
		ID:       1,
		GoogleID: "test123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	// Create a valid session
	validSession, err := sm.CreateSession(user)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Create multiple expired sessions directly in the database
	expiredSession1 := &database.Session{
		ID:        "expired_1",
		UserID:    user.ID,
		CreatedAt: time.Now().Add(-8 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	expiredSession2 := &database.Session{
		ID:        "expired_2",
		UserID:    user.ID,
		CreatedAt: time.Now().Add(-10 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(-3 * 24 * time.Hour),
	}

	if err := db.CreateSession(expiredSession1); err != nil {
		t.Fatalf("Failed to create expired session 1: %v", err)
	}
	if err := db.CreateSession(expiredSession2); err != nil {
		t.Fatalf("Failed to create expired session 2: %v", err)
	}

	// Verify all sessions exist
	if s, _ := db.GetSession(validSession.ID); s == nil {
		t.Error("Valid session should exist before cleanup")
	}
	if s, _ := db.GetSession("expired_1"); s == nil {
		t.Error("Expired session 1 should exist before cleanup")
	}
	if s, _ := db.GetSession("expired_2"); s == nil {
		t.Error("Expired session 2 should exist before cleanup")
	}

	// Run cleanup
	err = db.DeleteExpiredSessions()
	if err != nil {
		t.Fatalf("DeleteExpiredSessions failed: %v", err)
	}

	// Verify valid session still exists
	validDBSession, err := db.GetSession(validSession.ID)
	if err != nil {
		t.Fatalf("Error getting valid session: %v", err)
	}
	if validDBSession == nil {
		t.Error("Valid session should still exist after cleanup")
	}

	// Verify expired sessions were deleted
	expiredDBSession1, err := db.GetSession("expired_1")
	if err != nil {
		t.Fatalf("Error checking expired session 1: %v", err)
	}
	if expiredDBSession1 != nil {
		t.Error("Expired session 1 should be deleted after cleanup")
	}

	expiredDBSession2, err := db.GetSession("expired_2")
	if err != nil {
		t.Fatalf("Error checking expired session 2: %v", err)
	}
	if expiredDBSession2 != nil {
		t.Error("Expired session 2 should be deleted after cleanup")
	}
}

func TestSetSessionCookie(t *testing.T) {
	db := newMockDB()
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
	expectedName := "session_id_local" // Tests run in local mode
	if cookie.Name != expectedName {
		t.Errorf("Cookie name = %s, want %s", cookie.Name, expectedName)
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
	db := newMockDB()
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
	expectedName := "session_id_local" // Tests run in local mode
	if cookie.Name != expectedName {
		t.Errorf("Cookie name = %s, want %s", cookie.Name, expectedName)
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
	db := newMockDB()
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
		Name:  "session_id_local", // Tests run in local mode
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
		Name:  "session_id_local", // Tests run in local mode
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
