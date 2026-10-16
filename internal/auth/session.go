package auth

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
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
	db          database.Database
	cache       map[string]*CachedSession // sessionID -> cached session
	cacheMu     sync.RWMutex

	// activityRecorded tracks when this instance last wrote each user's
	// last-activity time, so that it is written at most once per
	// activityRecordInterval rather than on every request.
	activityRecorded map[int]time.Time
	activityMu       sync.Mutex
	cacheTTL    time.Duration // How long to cache sessions (default: 10 minutes)
	cacheHits   int64
	cacheMisses int64
	oauthStates map[string]oauthStateEntry // OAuth state -> entry (for one-time use)
	stateMu     sync.RWMutex
	authCodes   map[string]authCodeEntry // one-time code -> session handoff (native app auth)
	codeMu      sync.Mutex
}

// oauthStateEntry tracks a pending OAuth state and which kind of client
// started the flow, so the callback knows where to send the user afterwards.
type oauthStateEntry struct {
	ExpiresAt time.Time
	Native    bool
}

// authCodeEntry holds a freshly created session waiting to be claimed by a
// native app client via POST /auth/token. Codes are single-use and short-lived.
type authCodeEntry struct {
	SessionID        string
	SessionExpiresAt time.Time
	CodeExpiresAt    time.Time
}

type Session struct {
	ID        string
	UserID    int
	User      *database.User
	CreatedAt time.Time
	ExpiresAt time.Time
}

func NewSessionManager(db database.Database) *SessionManager {
	cacheTTL := 10 * time.Minute
	if v := os.Getenv("SESSION_CACHE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cacheTTL = d
		} else {
			log.Printf("WARNING: invalid SESSION_CACHE_TTL %q, using default 10m", v)
		}
	}

	sm := &SessionManager{
		db:          db,
		cache:       make(map[string]*CachedSession),

		activityRecorded: make(map[int]time.Time),
		cacheTTL:    cacheTTL,
		oauthStates: make(map[string]oauthStateEntry),
		authCodes:   make(map[string]authCodeEntry),
	}

	// Note: Cleanup is now handled by cron jobs instead of background goroutines
	// to prevent keeping instances alive 24/7 on App Engine Standard.
	// See /cron/cleanup-sessions endpoint.

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

	sm.recordActivity(user.ID)
	return session, nil
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	// Check cache first (read lock)
	sm.cacheMu.RLock()
	if cached, exists := sm.cache[sessionID]; exists {
		// Check if cache entry is still valid
		if time.Now().Before(cached.CacheExpires) {
			sm.cacheMu.RUnlock()
			atomic.AddInt64(&sm.cacheHits, 1)
			return cached.Session, true
		}
	}
	sm.cacheMu.RUnlock()

	atomic.AddInt64(&sm.cacheMisses, 1)
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

func (sm *SessionManager) RefreshSession(sessionID string) error {
	// Calculate new expiry time (extend by 7 days from now)
	newExpiry := time.Now().Add(7 * 24 * time.Hour)

	// Update in database
	if err := sm.db.UpdateSessionExpiry(sessionID, newExpiry); err != nil {
		return err
	}

	// Update cache if exists (write lock)
	sm.cacheMu.Lock()
	userID := 0
	if cached, exists := sm.cache[sessionID]; exists {
		cached.Session.ExpiresAt = newExpiry
		userID = cached.Session.UserID
	}
	sm.cacheMu.Unlock()

	if userID != 0 {
		sm.recordActivity(userID)
	}
	return nil
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

// activityRecordInterval is how often each instance writes a user's
// last-activity time. The data retention policy measures inactivity in months,
// so daily precision is plenty.
const activityRecordInterval = 24 * time.Hour

// maxActivityRecorded bounds the activityRecorded map; past it, the map is
// reset, which at worst causes one extra write per active user.
const maxActivityRecorded = 10000

// recordActivity writes userID's last-activity time for the data retention
// policy, unless this instance already did so within activityRecordInterval.
// Failures are logged and retried on a later request.
func (sm *SessionManager) recordActivity(userID int) {
	now := time.Now()

	sm.activityMu.Lock()
	if last, ok := sm.activityRecorded[userID]; ok && now.Sub(last) < activityRecordInterval {
		sm.activityMu.Unlock()
		return
	}
	if len(sm.activityRecorded) >= maxActivityRecorded {
		sm.activityRecorded = make(map[int]time.Time)
	}
	sm.activityRecorded[userID] = now
	sm.activityMu.Unlock()

	if err := sm.db.RecordUserActivity(userID, now); err != nil {
		log.Printf("Failed to record activity for user %d: %v", userID, err)
		sm.activityMu.Lock()
		delete(sm.activityRecorded, userID)
		sm.activityMu.Unlock()
	}
}

// DeleteUserSessions revokes every session belonging to userID and drops them
// from this instance's cache. Other instances may keep serving a cached copy
// until its cache entry expires (cacheTTL, 10 minutes by default).
func (sm *SessionManager) DeleteUserSessions(userID int) error {
	if err := sm.db.DeleteUserSessions(userID); err != nil {
		return err
	}

	sm.cacheMu.Lock()
	for id, cached := range sm.cache {
		if cached.Session.UserID == userID {
			delete(sm.cache, id)
		}
	}
	sm.cacheMu.Unlock()
	return nil
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

// CleanupExpiredSessions removes expired sessions from the database
// This should be called by cron jobs, not background goroutines
func (sm *SessionManager) CleanupExpiredSessions() error {
	return sm.db.DeleteExpiredSessions()
}

// CleanupExpiredCache removes expired entries from the session cache
// This is called opportunistically during GetSession to keep memory usage low
func (sm *SessionManager) CleanupExpiredCache() {
	now := time.Now()
	sm.cacheMu.Lock()
	defer sm.cacheMu.Unlock()

	// Remove expired cache entries
	for sessionID, cached := range sm.cache {
		if now.After(cached.CacheExpires) {
			delete(sm.cache, sessionID)
		}
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

	hits := atomic.LoadInt64(&sm.cacheHits)
	misses := atomic.LoadInt64(&sm.cacheMisses)
	var hitRate int64
	if total := hits + misses; total > 0 {
		hitRate = hits * 100 / total
	}

	return map[string]int{
		"total":    len(sm.cache),
		"expired":  expired,
		"active":   len(sm.cache) - expired,
		"hits":     int(hits),
		"misses":   int(misses),
		"hit_rate": int(hitRate),
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

// StoreOAuthState stores an OAuth state for one-time use with a 10-minute TTL.
// native marks states started by a native app client (/auth/login?client=ios
// or client=macos), so the callback redirects to the app's custom URL scheme
// instead of /.
func (sm *SessionManager) StoreOAuthState(state string, native bool) {
	sm.stateMu.Lock()
	defer sm.stateMu.Unlock()
	sm.oauthStates[state] = oauthStateEntry{
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Native:    native,
	}
}

// ValidateAndConsumeOAuthState checks if a state exists and hasn't been used.
// Returns valid=true and the state's native flag if so, marking it as used
// (deletes it). Returns valid=false if the state doesn't exist or has expired.
func (sm *SessionManager) ValidateAndConsumeOAuthState(state string) (valid, native bool) {
	sm.stateMu.Lock()
	defer sm.stateMu.Unlock()

	entry, exists := sm.oauthStates[state]
	if !exists {
		return false, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		delete(sm.oauthStates, state)
		return false, false
	}

	// Valid state - consume it (delete so it can't be reused)
	delete(sm.oauthStates, state)
	return true, entry.Native
}

// CleanupExpiredOAuthStates removes expired OAuth states from memory
// This is called opportunistically during ValidateAndConsumeOAuthState to keep memory usage low
func (sm *SessionManager) CleanupExpiredOAuthStates() {
	sm.stateMu.Lock()
	defer sm.stateMu.Unlock()

	now := time.Now()
	for state, entry := range sm.oauthStates {
		if now.After(entry.ExpiresAt) {
			delete(sm.oauthStates, state)
		}
	}
}

// authCodeTTL bounds how long a native app client has to exchange its one-time
// code after the OAuth callback fires; the exchange happens immediately in
// practice, so this only needs to absorb network latency.
const authCodeTTL = 2 * time.Minute

// CreateAuthCode issues a single-use code a native app client exchanges for the
// session token via POST /auth/token. The indirection keeps session tokens
// out of the goread2:// callback URL, where they could leak into device logs.
func (sm *SessionManager) CreateAuthCode(session *Session) (string, error) {
	code, err := generateSessionID()
	if err != nil {
		return "", err
	}

	sm.codeMu.Lock()
	defer sm.codeMu.Unlock()
	sm.authCodes[code] = authCodeEntry{
		SessionID:        session.ID,
		SessionExpiresAt: session.ExpiresAt,
		CodeExpiresAt:    time.Now().Add(authCodeTTL),
	}
	return code, nil
}

// ExchangeAuthCode redeems a one-time code for the session token it was
// issued against. The code is consumed whether or not it is valid, so a
// code can never be redeemed twice.
func (sm *SessionManager) ExchangeAuthCode(code string) (sessionID string, sessionExpiresAt time.Time, ok bool) {
	sm.codeMu.Lock()
	defer sm.codeMu.Unlock()

	entry, exists := sm.authCodes[code]
	if !exists {
		return "", time.Time{}, false
	}

	delete(sm.authCodes, code)

	if time.Now().After(entry.CodeExpiresAt) {
		return "", time.Time{}, false
	}
	return entry.SessionID, entry.SessionExpiresAt, true
}

// CleanupExpiredAuthCodes removes expired one-time auth codes from memory.
// Called by the session-cleanup cron alongside CleanupExpiredOAuthStates.
func (sm *SessionManager) CleanupExpiredAuthCodes() {
	sm.codeMu.Lock()
	defer sm.codeMu.Unlock()

	now := time.Now()
	for code, entry := range sm.authCodes {
		if now.After(entry.CodeExpiresAt) {
			delete(sm.authCodes, code)
		}
	}
}

// SessionCookieName exposes the environment-specific session cookie name so
// API clients that set the cookie themselves (the iOS app, after POST
// /auth/token) use the right name against local and production backends.
func (sm *SessionManager) SessionCookieName() string {
	return sm.getCookieName()
}
