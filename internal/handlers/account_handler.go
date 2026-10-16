package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/services"
)

// accountDeletionSweepDelay is when the second deletion pass runs. It must
// exceed the session cache TTL (10 minutes), so that by then no instance can
// still be serving the deleted user from a cached session and recreating rows.
const accountDeletionSweepDelay = 15 * time.Minute

// AccountTaskQueue enqueues account deletion work. It is satisfied by
// *services.CloudTasksQueue; left nil, deletion runs in-process instead.
type AccountTaskQueue interface {
	Enqueue(ctx context.Context, relativeURI string) error
	EnqueueAfter(ctx context.Context, relativeURI string, delay time.Duration) error
}

// AccountHandler serves the signed-in user's data rights requests.
type AccountHandler struct {
	userDataService *services.UserDataService
	sessionManager  *auth.SessionManager
	taskQueue       AccountTaskQueue
}

func NewAccountHandler(userDataService *services.UserDataService, sessionManager *auth.SessionManager) *AccountHandler {
	return &AccountHandler{userDataService: userDataService, sessionManager: sessionManager}
}

// SetTaskQueue makes account deletion run as Cloud Tasks. Called once from
// main, only when Cloud Tasks is configured (production App Engine).
func (ah *AccountHandler) SetTaskQueue(tq AccountTaskQueue) {
	ah.taskQueue = tq
}

// ExportData handles GET /api/account/export. It sends the signed-in user's
// personal data as a JSON file download.
func (ah *AccountHandler) ExportData(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	export, err := ah.userDataService.PrepareExport(user.ID)
	if err != nil {
		log.Printf("Failed to prepare data export for user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate the data export. Please try again."})
		return
	}

	filename := fmt.Sprintf("goread2-data-%s.json", export.ExportedAt.Format("2006-01-02"))
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Status(http.StatusOK)

	if err := ah.userDataService.WriteExport(c.Writer, user.ID, export); err != nil {
		// Headers are already sent, so the client receives a truncated file.
		log.Printf("Data export for user %d failed while streaming: %v", user.ID, err)
	}
}

// DeleteAccount handles DELETE /api/account. The body must repeat the
// account's email address as {"confirm_email": "..."}. With Cloud Tasks the
// deletion is queued, followed by a second sweep after
// accountDeletionSweepDelay, and the response is 202; without it the deletion
// runs in-process and the response is 200. Either way every session is revoked
// and the session cookie cleared before responding.
func (ah *AccountHandler) DeleteAccount(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	var request struct {
		ConfirmEmail string `json:"confirm_email"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The request body could not be parsed."})
		return
	}
	if !strings.EqualFold(strings.TrimSpace(request.ConfirmEmail), user.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The email address does not match this account."})
		return
	}

	taskURI := fmt.Sprintf("/tasks/delete-account?user_id=%d", user.ID)
	if ah.taskQueue != nil {
		// Queue before revoking sessions, so a queueing failure leaves the
		// user signed in and able to retry.
		if err := ah.taskQueue.Enqueue(c.Request.Context(), taskURI); err != nil {
			log.Printf("Failed to enqueue account deletion for user %d: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start account deletion. Please try again."})
			return
		}
		if err := ah.taskQueue.EnqueueAfter(c.Request.Context(), taskURI, accountDeletionSweepDelay); err != nil {
			// The first pass still deletes everything; the sweep only catches
			// rows recreated through stale cached sessions.
			log.Printf("Failed to enqueue account deletion sweep for user %d: %v", user.ID, err)
		}
	}

	if err := ah.sessionManager.DeleteUserSessions(user.ID); err != nil {
		log.Printf("Failed to revoke sessions for user %d: %v", user.ID, err)
	}
	ah.sessionManager.ClearSessionCookie(c.Writer)

	if ah.taskQueue != nil {
		log.Printf("Account deletion queued for user %d", user.ID)
		c.JSON(http.StatusAccepted, gin.H{"message": "Account deletion started"})
		return
	}

	if err := ah.userDataService.DeleteAccount(user.ID); err != nil {
		log.Printf("Account deletion failed for user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete the account. Please sign in and try again."})
		return
	}
	log.Printf("Account deleted for user %d", user.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Account deleted"})
}

// TaskDeleteAccount is the Cloud Tasks worker endpoint for
// /tasks/delete-account. A non-2xx response makes Cloud Tasks retry, and
// DeleteAccount is idempotent, so a partial failure resumes on retry.
func (ah *AccountHandler) TaskDeleteAccount(c *gin.Context) {
	if !auth.VerifyTaskRequest(c) {
		return
	}

	userID, err := strconv.Atoi(c.Query("user_id"))
	if err != nil || userID <= 0 {
		// Retrying cannot fix a malformed task, so acknowledge it.
		log.Printf("Account deletion task has invalid user_id %q", c.Query("user_id"))
		c.JSON(http.StatusOK, gin.H{"error": "invalid user_id"})
		return
	}

	if err := ah.userDataService.DeleteAccount(userID); err != nil {
		log.Printf("Account deletion task failed for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Account deletion failed"})
		return
	}
	log.Printf("Account deletion task completed for user %d", userID)
	c.JSON(http.StatusOK, gin.H{"message": "Account deleted"})
}
