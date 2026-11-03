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

func serveUsageMetrics(t *testing.T, db database.Database, cronHeader bool) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("GAE_ENV", "standard")
	gin.SetMode(gin.TestMode)
	handler := NewUsageMetricsHandler(services.NewUsageMetricsService(services.NewStatsService(db), nil))
	router := gin.New()
	router.GET("/cron/record-usage-metrics", handler.CronRecordUsageMetrics)
	req := httptest.NewRequest(http.MethodGet, "/cron/record-usage-metrics", nil)
	if cronHeader {
		req.Header.Set("X-Appengine-Cron", "true")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestCronRecordUsageMetrics(t *testing.T) {
	w := serveUsageMetrics(t, &mockDBStatsHandler{}, true)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Written bool                   `json:"written"`
		Points  []services.MetricPoint `json:"points"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if body.Written || len(body.Points) == 0 {
		t.Errorf("Expected unwritten points without a writer, got %+v", body)
	}
}

func TestCronRecordUsageMetricsRequiresCron(t *testing.T) {
	if w := serveUsageMetrics(t, &mockDBStatsHandler{}, false); w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 without the cron header, got %d", w.Code)
	}
}

func TestCronRecordUsageMetricsError(t *testing.T) {
	if w := serveUsageMetrics(t, &mockDBStatsHandler{err: errors.New("down")}, true); w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}
