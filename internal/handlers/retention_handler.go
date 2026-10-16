package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/services"
)

// RetentionHandler runs the data retention policy from cron.
type RetentionHandler struct {
	retentionService *services.RetentionService
	taskQueue        TaskQueue
}

func NewRetentionHandler(retentionService *services.RetentionService) *RetentionHandler {
	return &RetentionHandler{retentionService: retentionService}
}

// SetTaskQueue makes the cron endpoint enqueue the work as a Cloud Task.
// Called once from main, only when Cloud Tasks is configured; left unset, the
// cron endpoint runs the policy in-process.
func (rh *RetentionHandler) SetTaskQueue(tq TaskQueue) {
	rh.taskQueue = tq
}

// CronRetention handles /cron/apply-retention.
func (rh *RetentionHandler) CronRetention(c *gin.Context) {
	if !auth.VerifyCronRequest(c) {
		return
	}
	if rh.taskQueue != nil {
		if err := rh.taskQueue.Enqueue(c.Request.Context(), "/tasks/apply-retention"); err != nil {
			log.Printf("Failed to enqueue retention task: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue retention"})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"message": "Retention enqueued"})
		return
	}
	rh.run(c)
}

// TaskRetention is the Cloud Tasks worker endpoint for /tasks/apply-retention.
func (rh *RetentionHandler) TaskRetention(c *gin.Context) {
	if !auth.VerifyTaskRequest(c) {
		return
	}
	rh.run(c)
}

func (rh *RetentionHandler) run(c *gin.Context) {
	result, err := rh.retentionService.Run(time.Now())
	log.Printf("Retention run: %d accounts deleted, %d backfilled, %d deletions failed, %d audit IPs cleared",
		result.AccountsDeleted, result.AccountsBackfilled, result.DeletionsFailed, result.AuditIPsCleared)
	if err != nil {
		log.Printf("Retention run failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Retention run failed", "result": result})
		return
	}
	c.JSON(http.StatusOK, result)
}
