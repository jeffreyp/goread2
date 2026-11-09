package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/services"
)

// mockDBRetentionHandler serves one pending deletion to RetentionService. It
// embeds database.Database so that any call it does not override panics.
type mockDBRetentionHandler struct {
	database.Database
	infos      []database.UserRetentionInfo
	failDelete bool
	deletedIDs []int
}

func (m *mockDBRetentionHandler) GetUserRetentionInfo() ([]database.UserRetentionInfo, error) {
	return m.infos, nil
}

func (m *mockDBRetentionHandler) GetUserByID(int) (*database.User, error) {
	return nil, database.ErrUserNotFound
}

func (m *mockDBRetentionHandler) DeleteUserAccount(userID int) error {
	if m.failDelete {
		return errors.New("datastore unavailable")
	}
	m.deletedIDs = append(m.deletedIDs, userID)
	return nil
}

func newRetentionHandlerForTest(db *mockDBRetentionHandler) *RetentionHandler {
	return NewRetentionHandler(services.NewRetentionService(db, services.NewUserDataService(db)))
}

func TestRetentionCronEndpointsEnqueueTheirTasks(t *testing.T) {
	t.Setenv("GAE_ENV", "standard")
	tests := []struct {
		name string
		call func(*RetentionHandler, *gin.Context)
		want string
	}{
		{"apply retention", (*RetentionHandler).CronRetention, "/tasks/apply-retention"},
		{"retry pending deletions", (*RetentionHandler).CronRetryPendingDeletions, "/tasks/retry-pending-deletions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newRetentionHandlerForTest(&mockDBRetentionHandler{})
			tq := &fakeTaskQueue{}
			handler.SetTaskQueue(tq)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/cron/x", nil)
			c.Request.Header.Set("X-Appengine-Cron", "true")

			tt.call(handler, c)

			if w.Code != http.StatusAccepted {
				t.Errorf("status = %d, want 202: %s", w.Code, w.Body.String())
			}
			if len(tq.enqueued) != 1 || tq.enqueued[0] != tt.want {
				t.Errorf("enqueued = %v, want [%s]", tq.enqueued, tt.want)
			}
		})
	}
}

func TestRetentionCronReturns500WhenEnqueueFails(t *testing.T) {
	t.Setenv("GAE_ENV", "standard")
	handler := newRetentionHandlerForTest(&mockDBRetentionHandler{})
	handler.SetTaskQueue(&fakeTaskQueue{shouldFail: true})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/cron/retry-pending-deletions", nil)
	c.Request.Header.Set("X-Appengine-Cron", "true")

	handler.CronRetryPendingDeletions(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500: %s", w.Code, w.Body.String())
	}
}

func TestTaskRetryPendingDeletions(t *testing.T) {
	t.Setenv("GAE_ENV", "standard")
	stale := time.Now().Add(-2 * time.Hour)

	t.Run("deletes the pending account", func(t *testing.T) {
		db := &mockDBRetentionHandler{infos: []database.UserRetentionInfo{{UserID: 7, SubscriptionStatus: "active", DeletionRequestedAt: stale}}}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/retry-pending-deletions", nil)
		c.Request.Header.Set("X-AppEngine-QueueName", "cron-tasks")

		newRetentionHandlerForTest(db).TaskRetryPendingDeletions(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		if len(db.deletedIDs) != 1 || db.deletedIDs[0] != 7 {
			t.Errorf("deleted = %v, want [7]", db.deletedIDs)
		}
	})

	t.Run("failure returns 500 so Cloud Tasks retries", func(t *testing.T) {
		db := &mockDBRetentionHandler{
			infos:      []database.UserRetentionInfo{{UserID: 7, SubscriptionStatus: "active", DeletionRequestedAt: stale}},
			failDelete: true,
		}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/retry-pending-deletions", nil)
		c.Request.Header.Set("X-AppEngine-QueueName", "cron-tasks")

		newRetentionHandlerForTest(db).TaskRetryPendingDeletions(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects a request without the queue header", func(t *testing.T) {
		db := &mockDBRetentionHandler{}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/retry-pending-deletions", nil)

		newRetentionHandlerForTest(db).TaskRetryPendingDeletions(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})
}
