package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// CORS returns middleware that adds Cross-Origin Resource Sharing headers.
//
// Set ALLOWED_ORIGIN to the exact origin that may call the API cross-origin
// (e.g. "https://myapp.appspot.com"). Unset means no cross-origin access is
// permitted — the browser's default same-origin policy applies.
func CORS() gin.HandlerFunc {
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" || allowedOrigin == "" {
			c.Next()
			return
		}

		if origin == allowedOrigin {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
