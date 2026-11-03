package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/services"
)

// mockDBStatsHandler serves StatsService's queries. It embeds
// database.Database so that any other call panics.
type mockDBStatsHandler struct {
	database.Database
	err error
}

func (m *mockDBStatsHandler) GetUserRetentionInfo() ([]database.UserRetentionInfo, error) {
	return []database.UserRetentionInfo{
		{UserID: 1, SubscriptionStatus: "active"},
		{UserID: 2, SubscriptionStatus: "trial", IsAdmin: true},
	}, m.err
}

func (m *mockDBStatsHandler) GetAllUserFeedSubscriptions() ([]database.UserFeed, error) {
	return []database.UserFeed{{UserID: 1, FeedID: 7}}, nil
}

func serveAdminStats(t *testing.T, db database.Database) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/stats", NewAdminStatsHandler(services.NewStatsService(db)).GetStats)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/stats", nil))
	return w
}

func TestAdminStatsGetStats(t *testing.T) {
	w := serveAdminStats(t, &mockDBStatsHandler{})
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q, want private, no-store", got)
	}
	var stats services.AdminStats
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if stats.TotalUsers != 2 || stats.Subscriptions.Active != 1 || stats.Subscriptions.Admins != 1 || stats.Feeds.ActiveFeeds != 1 {
		t.Errorf("Unexpected stats: %+v", stats)
	}
}

func TestAdminStatsGetStatsError(t *testing.T) {
	w := serveAdminStats(t, &mockDBStatsHandler{err: errors.New("datastore unavailable")})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected 500, got %d", w.Code)
	}
	if body := w.Body.String(); body != `{"error":"Usage statistics could not be loaded."}` {
		t.Errorf("Unexpected body %s", body)
	}
}
