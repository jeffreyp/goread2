package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequestBodyLimit returns middleware that caps each request body to maxBytes.
// Requests that announce a Content-Length over the limit are rejected immediately
// with 413. The overrides map lets specific paths (e.g. file upload endpoints)
// use a higher limit than the global default.
func RequestBodyLimit(maxBytes int64, overrides map[string]int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := maxBytes
		if v, ok := overrides[c.Request.URL.Path]; ok {
			limit = v
		}
		if c.Request.ContentLength > limit {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}
