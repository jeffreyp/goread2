package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/secrets"
)

func TestVerifyTaskRequest_AppEngine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GAE_ENV", "standard")

	t.Run("authorized with queue header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/refresh-feeds", nil)
		c.Request.Header.Set("X-AppEngine-QueueName", "cron-tasks")

		if !VerifyTaskRequest(c) {
			t.Error("expected request with X-AppEngine-QueueName to be authorized")
		}
	})

	t.Run("unauthorized without queue header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/refresh-feeds", nil)

		if VerifyTaskRequest(c) {
			t.Error("expected request without X-AppEngine-QueueName to be unauthorized")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})
}

func TestVerifyTaskRequest_NonAppEngine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GAE_ENV", "")
	t.Setenv("ADMIN_TOKEN", "test-admin-token-value")
	secrets.ResetCacheForTesting()
	t.Cleanup(secrets.ResetCacheForTesting)

	t.Run("falls back to admin auth and succeeds", func(t *testing.T) {
		adminUser := &database.User{ID: 1, Email: "admin@example.com", IsAdmin: true}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/refresh-feeds", nil)
		c.Request.Header.Set("X-Admin-Token", "test-admin-token-value")
		c.Set("user", adminUser)

		if !VerifyTaskRequest(c) {
			t.Error("expected admin-authenticated request to be authorized")
		}
	})

	t.Run("falls back to admin auth and rejects unauthenticated request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/tasks/refresh-feeds", nil)

		if VerifyTaskRequest(c) {
			t.Error("expected unauthenticated request to be unauthorized")
		}
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})
}
