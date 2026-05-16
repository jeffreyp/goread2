package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupBodyLimitRouter(maxBytes int64, overrides map[string]int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestBodyLimit(maxBytes, overrides))
	r.POST("/api/data", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"size": len(body)})
	})
	r.POST("/api/upload", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"size": len(body)})
	})
	return r
}

func TestBodyLimit_UnderLimit(t *testing.T) {
	r := setupBodyLimitRouter(1024, nil)

	body := strings.NewReader(`{"key":"value"}`)
	req, _ := http.NewRequest("POST", "/api/data", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBodyLimit_ContentLengthOverLimit(t *testing.T) {
	r := setupBodyLimitRouter(100, nil)

	body := bytes.NewReader(make([]byte, 200))
	req, _ := http.NewRequest("POST", "/api/data", body)
	req.ContentLength = 200
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBodyLimit_BodyExceedsLimitDuringRead(t *testing.T) {
	r := setupBodyLimitRouter(100, nil)

	// Send 200 bytes without declaring Content-Length
	body := bytes.NewReader(make([]byte, 200))
	req, _ := http.NewRequest("POST", "/api/data", body)
	req.ContentLength = -1 // unknown
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Handler reads the body and gets an error from MaxBytesReader
	if w.Code == http.StatusOK {
		t.Error("expected non-200 status when body exceeds limit")
	}
}

func TestBodyLimit_OverrideAllowsLargerBody(t *testing.T) {
	overrides := map[string]int64{
		"/api/upload": 500,
	}
	r := setupBodyLimitRouter(100, overrides)

	body := bytes.NewReader(make([]byte, 300))
	req, _ := http.NewRequest("POST", "/api/upload", body)
	req.ContentLength = 300
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for override path, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBodyLimit_DefaultAppliesWhenNoOverride(t *testing.T) {
	overrides := map[string]int64{
		"/api/upload": 500,
	}
	r := setupBodyLimitRouter(100, overrides)

	body := bytes.NewReader(make([]byte, 200))
	req, _ := http.NewRequest("POST", "/api/data", body)
	req.ContentLength = 200
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 for non-override path, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBodyLimit_NilOverridesUseDefault(t *testing.T) {
	r := setupBodyLimitRouter(1024, nil)

	body := bytes.NewReader(make([]byte, 2000))
	req, _ := http.NewRequest("POST", "/api/data", body)
	req.ContentLength = 2000
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d: %s", w.Code, w.Body.String())
	}
}
