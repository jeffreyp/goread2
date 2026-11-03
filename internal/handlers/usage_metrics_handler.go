package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/services"
)

// UsageMetricsHandler records the admin usage statistics as Cloud Monitoring
// metrics from cron.
type UsageMetricsHandler struct {
	metricsService *services.UsageMetricsService
}

func NewUsageMetricsHandler(metricsService *services.UsageMetricsService) *UsageMetricsHandler {
	return &UsageMetricsHandler{metricsService: metricsService}
}

// CronRecordUsageMetrics handles /cron/record-usage-metrics. The work is a
// few queries and one API call, so it runs in the cron request rather than as
// a Cloud Task.
func (h *UsageMetricsHandler) CronRecordUsageMetrics(c *gin.Context) {
	if !auth.VerifyCronRequest(c) {
		return
	}
	points, written, err := h.metricsService.Record(c.Request.Context())
	if err != nil {
		log.Printf("Recording usage metrics failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Recording usage metrics failed"})
		return
	}
	if written {
		log.Printf("Recorded %d usage metric points", len(points))
	}
	c.JSON(http.StatusOK, gin.H{"written": written, "points": points})
}
