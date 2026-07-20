package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NotFoundHandler returns the handler registered via router.NoRoute for
// unmatched paths. Gin's built-in fallback writes its 404 body after the gzip
// middleware has closed its compressed writer, so for clients sending
// Accept-Encoding: gzip the body is discarded and the response reaches the
// client with an empty body and, behind the App Engine frontend, status 200.
// Vulnerability scanners probing paths like /wp-admin/phpinfo.php read that as
// a hit and keep scanning. An explicit handler writes the 404 while the gzip
// writer is still open, producing a well-formed compressed 404.
func NotFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusNotFound, "404 page not found")
	}
}
