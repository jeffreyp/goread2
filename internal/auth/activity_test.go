package auth

import (
	"sync"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

// activityDB counts RecordUserActivity calls.
type activityDB struct {
	*mockDB
	mu    sync.Mutex
	calls int
}

func (db *activityDB) RecordUserActivity(int, time.Time) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.calls++
	return nil
}

func (db *activityDB) count() int {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.calls
}

func TestSessionActivityIsRecordedAtMostDaily(t *testing.T) {
	db := &activityDB{mockDB: newMockDB()}
	sm := NewSessionManager(db)
	user := &database.User{ID: 7, Email: "reader@example.com"}
	db.users[user.ID] = user

	session, err := sm.CreateSession(user)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if db.count() != 1 {
		t.Fatalf("Expected sign-in to record activity once, got %d", db.count())
	}

	if _, ok := sm.GetSession(session.ID); !ok {
		t.Fatal("Expected the new session to load")
	}
	for i := 0; i < 3; i++ {
		if err := sm.RefreshSession(session.ID); err != nil {
			t.Fatalf("RefreshSession failed: %v", err)
		}
	}
	if db.count() != 1 {
		t.Errorf("Expected requests within a day not to record activity again, got %d writes", db.count())
	}

	// A day later, the next request records activity again.
	sm.activityMu.Lock()
	sm.activityRecorded[user.ID] = time.Now().Add(-activityRecordInterval - time.Minute)
	sm.activityMu.Unlock()
	if err := sm.RefreshSession(session.ID); err != nil {
		t.Fatalf("RefreshSession failed: %v", err)
	}
	if db.count() != 2 {
		t.Errorf("Expected activity to be recorded again after a day, got %d writes", db.count())
	}
}
