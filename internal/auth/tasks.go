package auth

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// VerifyTaskRequest checks that the request is a Cloud Tasks dispatch to an
// App Engine task handler, not a direct call to the endpoint. On App Engine
// it validates the X-AppEngine-QueueName header, which App Engine strips
// from any inbound request that didn't originate from Cloud Tasks' own
// internal dispatch to this service (the same protection VerifyCronRequest
// relies on for X-Appengine-Cron). Elsewhere it falls back to the same
// admin-session-plus-token gate used for cron.
// Returns true if authorized; false after writing the error response.
func VerifyTaskRequest(c *gin.Context) bool {
	if os.Getenv("GAE_ENV") == "standard" {
		if c.GetHeader("X-AppEngine-QueueName") == "" {
			log.Printf("Unauthorized task request from IP: %s", GetSecureClientIP(c))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return false
		}
		return true
	}

	return VerifyCronRequest(c)
}
