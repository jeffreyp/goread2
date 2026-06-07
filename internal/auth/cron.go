package auth

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// VerifyCronRequest checks that the request is authorized to trigger a cron job.
// On App Engine it validates the X-Appengine-Cron header; elsewhere it requires
// an admin session plus a matching X-Admin-Token header.
// Returns true if authorized; false after writing the error response.
func VerifyCronRequest(c *gin.Context) bool {
	if os.Getenv("GAE_ENV") == "standard" {
		if c.GetHeader("X-Appengine-Cron") != "true" {
			log.Printf("Unauthorized cron request from IP: %s", GetSecureClientIP(c))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return false
		}
		return true
	}

	// Non-App Engine: require admin session + ADMIN_TOKEN header.
	user, exists := GetUserFromContext(c)
	if !exists || !user.IsAdmin {
		log.Printf("Unauthorized cron request - requires admin authentication")
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin authentication required"})
		return false
	}
	expectedToken := os.Getenv("ADMIN_TOKEN")
	if expectedToken == "" || c.GetHeader("X-Admin-Token") != expectedToken {
		log.Printf("Unauthorized cron request - invalid or missing X-Admin-Token from IP: %s", GetSecureClientIP(c))
		c.JSON(http.StatusForbidden, gin.H{"error": "Valid X-Admin-Token header required"})
		return false
	}
	return true
}
