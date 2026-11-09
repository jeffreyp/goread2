package handlers

import (
	"log"
	"net/http"
	"strings"
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
	rh.cron(c, "/tasks/apply-retention", "Retention", rh.retentionService.Run)
}

// TaskRetention is the Cloud Tasks worker endpoint for /tasks/apply-retention.
func (rh *RetentionHandler) TaskRetention(c *gin.Context) {
	if !auth.VerifyTaskRequest(c) {
		return
	}
	rh.run(c, "Retention", rh.retentionService.Run)
}

// CronRetryPendingDeletions handles /cron/retry-pending-deletions.
func (rh *RetentionHandler) CronRetryPendingDeletions(c *gin.Context) {
	rh.cron(c, "/tasks/retry-pending-deletions", "Pending deletion retry", rh.retentionService.RetryPendingDeletions)
}

// TaskRetryPendingDeletions is the Cloud Tasks worker endpoint for
// /tasks/retry-pending-deletions.
func (rh *RetentionHandler) TaskRetryPendingDeletions(c *gin.Context) {
	if !auth.VerifyTaskRequest(c) {
		return
	}
	rh.run(c, "Pending deletion retry", rh.retentionService.RetryPendingDeletions)
}

// cron enqueues taskPath as a Cloud Task when a queue is configured, and
// otherwise runs job in-process. name labels the job in logs and responses.
func (rh *RetentionHandler) cron(c *gin.Context, taskPath, name string, job func(time.Time) (services.RetentionResult, error)) {
	if !auth.VerifyCronRequest(c) {
		return
	}
	if rh.taskQueue != nil {
		if err := rh.taskQueue.Enqueue(c.Request.Context(), taskPath); err != nil {
			log.Printf("Failed to enqueue %s task: %v", strings.ToLower(name), err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue " + strings.ToLower(name)})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"message": name + " enqueued"})
		return
	}
	rh.run(c, name, job)
}

func (rh *RetentionHandler) run(c *gin.Context, name string, job func(time.Time) (services.RetentionResult, error)) {
	result, err := job(time.Now())
	log.Printf("%s run: %d accounts deleted, %d backfilled, %d deletions failed, %d audit IPs cleared, %d tombstones purged",
		name, result.AccountsDeleted, result.AccountsBackfilled, result.DeletionsFailed, result.AuditIPsCleared, result.TombstonesPurged)
	if err != nil {
		log.Printf("%s run failed: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": name + " run failed", "result": result})
		return
	}
	c.JSON(http.StatusOK, result)
}
