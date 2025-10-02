package auth

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"time"

	"goread2/internal/database"
)

type SessionManager struct {
	db database.Database
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
		db: db,
	}

	// Start cleanup goroutine
	go sm.cleanupExpiredSessions()

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

	return session, true
}

func (sm *SessionManager) DeleteSession(sessionID string) {
	if err := sm.db.DeleteSession(sessionID); err != nil {
		log.Printf("Error deleting session %s: %v", sessionID, err)
	}
}

func (sm *SessionManager) SetSessionCookie(w http.ResponseWriter, session *Session) {
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   false, // Set to false for local development
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
	http.SetCookie(w, cookie)
}

func (sm *SessionManager) ClearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   false, // Set to false for local development
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
	http.SetCookie(w, cookie)
}

func (sm *SessionManager) GetSessionFromRequest(r *http.Request) (*Session, bool) {
	cookie, err := r.Cookie("session_id")
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

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
