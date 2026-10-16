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
