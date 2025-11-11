package auth

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

// CachedSession represents a session stored in the in-memory cache
type CachedSession struct {
	Session      *Session
	CachedAt     time.Time
	CacheExpires time.Time
}

type SessionManager struct {
	db       database.Database
	cache    map[string]*CachedSession // sessionID -> cached session
	cacheMu  sync.RWMutex
	cacheTTL time.Duration // How long to cache sessions (default: 10 minutes)
}

type Session struct {
	ID        string
	UserID    int
	User      *database.User
	CreatedAt time.Time
	ExpiresAt time.Time
}

func NewSessionManager(db database.Database) *SessionManager {
	sm := &SessionManager{
		db:       db,
		cache:    make(map[string]*CachedSession),
		cacheTTL: 10 * time.Minute, // Cache sessions for 10 minutes
	}

	// Start cleanup goroutine for database sessions
	go sm.cleanupExpiredSessions()

	// Start cleanup goroutine for cache
	go sm.cleanupExpiredCache()

	return sm
}

func (sm *SessionManager) CreateSession(user *database.User) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        sessionID,
		UserID:    user.ID,
		User:      user,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
	}

	// Save to database
	dbSession := &database.Session{
		ID:        session.ID,
		UserID:    session.UserID,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}

	if err := sm.db.CreateSession(dbSession); err != nil {
		return nil, err
	}

	return session, nil
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	// Check cache first (read lock)
	sm.cacheMu.RLock()
	if cached, exists := sm.cache[sessionID]; exists {
		// Check if cache entry is still valid
		if time.Now().Before(cached.CacheExpires) {
			sm.cacheMu.RUnlock()
			// Cache hit! Return cached session without database read
			return cached.Session, true
		}
	}
	sm.cacheMu.RUnlock()

	// Cache miss or expired - fetch from database
	dbSession, err := sm.db.GetSession(sessionID)
	if err != nil || dbSession == nil {
		return nil, false
	}

	// Check if session is expired
	if time.Now().After(dbSession.ExpiresAt) {
		sm.DeleteSession(sessionID)
		return nil, false
	}

	// Load user from database
	user, err := sm.db.GetUserByID(dbSession.UserID)
	if err != nil {
		return nil, false
	}

	session := &Session{
		ID:        dbSession.ID,
		UserID:    dbSession.UserID,
		User:      user,
		CreatedAt: dbSession.CreatedAt,
		ExpiresAt: dbSession.ExpiresAt,
	}

	// Store in cache for future requests (write lock)
	sm.cacheMu.Lock()
	sm.cache[sessionID] = &CachedSession{
		Session:      session,
		CachedAt:     time.Now(),
		CacheExpires: time.Now().Add(sm.cacheTTL),
	}
	sm.cacheMu.Unlock()

	return session, true
}

func (sm *SessionManager) DeleteSession(sessionID string) {
	// Delete from database
	if err := sm.db.DeleteSession(sessionID); err != nil {
		log.Printf("Error deleting session %s: %v", sessionID, err)
	}

	// Invalidate cache entry (write lock)
	sm.cacheMu.Lock()
	delete(sm.cache, sessionID)
	sm.cacheMu.Unlock()
}

// getCookieName returns an environment-specific cookie name to prevent
// local and production authentication states from conflicting
func (sm *SessionManager) getCookieName() string {
	isProduction := os.Getenv("GAE_ENV") == "standard" || os.Getenv("ENVIRONMENT") == "production"
	if isProduction {
		return "session_id"
	}
	return "session_id_local"
}

func (sm *SessionManager) SetSessionCookie(w http.ResponseWriter, session *Session) {
	// Enable Secure flag in production environments
	isProduction := os.Getenv("GAE_ENV") == "standard" || os.Getenv("ENVIRONMENT") == "production"

	cookie := &http.Cookie{
		Name:     sm.getCookieName(),
		Value:    session.ID,
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   isProduction, // Secure cookies in production only
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
	http.SetCookie(w, cookie)
}

func (sm *SessionManager) ClearSessionCookie(w http.ResponseWriter) {
	// Enable Secure flag in production environments
	isProduction := os.Getenv("GAE_ENV") == "standard" || os.Getenv("ENVIRONMENT") == "production"

	cookie := &http.Cookie{
		Name:     sm.getCookieName(),
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   isProduction, // Secure cookies in production only
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
	http.SetCookie(w, cookie)
}

func (sm *SessionManager) GetSessionFromRequest(r *http.Request) (*Session, bool) {
	cookie, err := r.Cookie(sm.getCookieName())
	if err != nil {
		return nil, false
	}

	return sm.GetSession(cookie.Value)
}

func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if err := sm.db.DeleteExpiredSessions(); err != nil {
			log.Printf("Error cleaning up expired sessions: %v", err)
		}
	}
}

// cleanupExpiredCache periodically removes expired entries from the session cache
// This prevents memory from growing unbounded and ensures cache stays fresh
func (sm *SessionManager) cleanupExpiredCache() {
	ticker := time.NewTicker(5 * time.Minute) // Clean up every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		sm.cacheMu.Lock()

		// Remove expired cache entries
		for sessionID, cached := range sm.cache {
			if now.After(cached.CacheExpires) {
				delete(sm.cache, sessionID)
			}
		}

		sm.cacheMu.Unlock()
	}
}

// GetCacheStats returns statistics about the session cache
// Useful for monitoring cache hit rates and memory usage
func (sm *SessionManager) GetCacheStats() map[string]int {
	sm.cacheMu.RLock()
	defer sm.cacheMu.RUnlock()

	expired := 0
	now := time.Now()
	for _, cached := range sm.cache {
		if now.After(cached.CacheExpires) {
			expired++
		}
	}

	return map[string]int{
		"total":   len(sm.cache),
		"expired": expired,
		"active":  len(sm.cache) - expired,
	}
}

// InvalidateCache clears all cached sessions
// Useful for testing or when user data changes that require cache invalidation
func (sm *SessionManager) InvalidateCache() {
	sm.cacheMu.Lock()
	sm.cache = make(map[string]*CachedSession)
	sm.cacheMu.Unlock()
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
