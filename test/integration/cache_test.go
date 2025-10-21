package integration

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/gzip"
)

func setupSimpleCacheTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(gzip.Gzip(gzip.DefaultCompression))

	// Simple caching: only cache static assets aggressively, nothing else
	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Cache static assets for 24 hours (CSS, JS, images rarely change)
		if strings.HasPrefix(path, "/static/") {
			c.Header("Cache-Control", "public, max-age=86400")
			c.Header("Vary", "Accept-Encoding")
		}

		c.Next()
	})

	// Test routes
	r.GET("/static/styles.css", func(c *gin.Context) {
		c.String(http.StatusOK, "body { color: black; }")
	})

	r.GET("/api/feeds", func(c *gin.Context) {
		c.JSON(http.StatusOK, []string{"feed1", "feed2"})
	})

	return r
}

func TestSimpleCaching(t *testing.T) {
	router := setupSimpleCacheTestRouter()

	t.Run("static assets have 24 hour cache", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/styles.css", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		cacheControl := w.Header().Get("Cache-Control")
		if cacheControl != "public, max-age=86400" {
			t.Errorf("Expected Cache-Control: public, max-age=86400, got %s", cacheControl)
		}

		vary := w.Header().Get("Vary")
		if vary != "Accept-Encoding" {
			t.Errorf("Expected Vary: Accept-Encoding, got %s", vary)
		}
	})

	t.Run("API endpoints have no cache headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feeds", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		cacheControl := w.Header().Get("Cache-Control")
		if cacheControl != "" {
			t.Errorf("Expected no Cache-Control header for API, got %s", cacheControl)
		}
	})
}
