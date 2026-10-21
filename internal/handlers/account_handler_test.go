package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/secrets"
	"github.com/jeffreyp/goread2/internal/services"
)

type enqueuedTask struct {
	uri   string
	delay time.Duration
}

type fakeAccountTaskQueue struct {
	tasks []enqueuedTask
	err   error
}

func (q *fakeAccountTaskQueue) Enqueue(_ context.Context, uri string) error {
	return q.EnqueueAfter(context.Background(), uri, 0)
}

func (q *fakeAccountTaskQueue) EnqueueAfter(_ context.Context, uri string, delay time.Duration) error {
	if q.err != nil {
		return q.err
	}
	q.tasks = append(q.tasks, enqueuedTask{uri: uri, delay: delay})
	return nil
}

func runDeleteAccount(t *testing.T, queue *fakeAccountTaskQueue, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := &mockDBAuthHandler{}
	handler := NewAccountHandler(services.NewUserDataService(db), auth.NewSessionManager(db))
	handler.SetTaskQueue(queue)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/account", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(auth.UserContextKey), &database.User{ID: 42, Email: "reader@example.com"})

	handler.DeleteAccount(c)
	return w
}

func TestDeleteAccountQueuesDeletionAndSweep(t *testing.T) {
	queue := &fakeAccountTaskQueue{}
	w := runDeleteAccount(t, queue, `{"confirm_email": "Reader@Example.com"}`)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected 202, got %d: %s", w.Code, w.Body.String())
	}
	want := []enqueuedTask{
		{uri: "/tasks/delete-account?user_id=42", delay: 0},
		{uri: "/tasks/delete-account?user_id=42", delay: accountDeletionSweepDelay},
	}
	if len(queue.tasks) != len(want) || queue.tasks[0] != want[0] || queue.tasks[1] != want[1] {
		t.Errorf("Expected tasks %+v, got %+v", want, queue.tasks)
	}
	if w.Header().Get("Set-Cookie") == "" {
		t.Error("Expected the session cookie to be cleared")
	}
}

func TestDeleteAccountEnqueueFailureKeepsSession(t *testing.T) {
	queue := &fakeAccountTaskQueue{err: errors.New("queue unavailable")}
	w := runDeleteAccount(t, queue, `{"confirm_email": "reader@example.com"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Set-Cookie") != "" {
		t.Error("A failed enqueue should leave the user signed in so they can retry")
	}
}

func TestDeleteAccountRejectsMismatchedEmail(t *testing.T) {
	for _, body := range []string{`{"confirm_email": "other@example.com"}`, `{}`, `not json`} {
		queue := &fakeAccountTaskQueue{}
		w := runDeleteAccount(t, queue, body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Body %s: expected 400, got %d", body, w.Code)
		}
		if len(queue.tasks) != 0 {
			t.Errorf("Body %s: expected nothing enqueued, got %+v", body, queue.tasks)
		}
	}
}

// mockDBTaskDeletion records the user IDs DeleteAccount reaches. It embeds
// database.Database so that any call it does not override panics.
// GetUserByID reports the user as gone, which keeps the test off the Stripe
// and tombstone steps; those are covered in the services package.
type mockDBTaskDeletion struct {
	database.Database
	deleteErr  error
	deletedIDs []int
}

func (m *mockDBTaskDeletion) GetUserByID(int) (*database.User, error) {
	return nil, database.ErrUserNotFound
}

func (m *mockDBTaskDeletion) DeleteUserAccount(userID int) error {
	m.deletedIDs = append(m.deletedIDs, userID)
	return m.deleteErr
}

func runTaskDeleteAccount(t *testing.T, db *mockDBTaskDeletion, query string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("GAE_ENV", "")
	t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
	secrets.ResetCacheForTesting()
	t.Cleanup(secrets.ResetCacheForTesting)

	handler := NewAccountHandler(services.NewUserDataService(db), auth.NewSessionManager(db))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/tasks/delete-account"+query, nil)
	c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
	c.Set(string(auth.UserContextKey), &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true})

	handler.TaskDeleteAccount(c)
	return w
}

func TestTaskDeleteAccountAcknowledgesInvalidUserID(t *testing.T) {
	for _, query := range []string{"", "?user_id=", "?user_id=abc", "?user_id=0", "?user_id=-3"} {
		db := &mockDBTaskDeletion{}
		w := runTaskDeleteAccount(t, db, query)

		if w.Code != http.StatusOK {
			t.Errorf("Query %q: expected 200 so Cloud Tasks does not retry, got %d: %s", query, w.Code, w.Body.String())
		}
		if len(db.deletedIDs) != 0 {
			t.Errorf("Query %q: expected no deletion, got %v", query, db.deletedIDs)
		}
	}
}

func TestTaskDeleteAccountFailureReturns500(t *testing.T) {
	db := &mockDBTaskDeletion{deleteErr: errors.New("datastore unavailable")}
	w := runTaskDeleteAccount(t, db, "?user_id=42")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected 500 so Cloud Tasks retries, got %d: %s", w.Code, w.Body.String())
	}
	if len(db.deletedIDs) != 1 || db.deletedIDs[0] != 42 {
		t.Errorf("Expected deletion attempted for user 42, got %v", db.deletedIDs)
	}
}

func TestTaskDeleteAccountSuccess(t *testing.T) {
	db := &mockDBTaskDeletion{}
	w := runTaskDeleteAccount(t, db, "?user_id=42")

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(db.deletedIDs) != 1 || db.deletedIDs[0] != 42 {
		t.Errorf("Expected user 42 deleted, got %v", db.deletedIDs)
	}
}
