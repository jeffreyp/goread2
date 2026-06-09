package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupCORSRouter(t *testing.T, allowedOrigin string) *gin.Engine {
	t.Helper()
	if allowedOrigin != "" {
		t.Setenv("ALLOWED_ORIGIN", allowedOrigin)
	} else {
		os.Unsetenv("ALLOWED_ORIGIN")
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.OPTIONS("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func corsRequest(r *gin.Engine, method, origin string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, "/test", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	r.ServeHTTP(w, req)
	return w
}

func TestCORS_NoOriginHeader(t *testing.T) {
	r := setupCORSRouter(t, "https://example.com")
	w := corsRequest(r, "GET", "")
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS header when no Origin header")
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	r := setupCORSRouter(t, "https://example.com")
	w := corsRequest(r, "GET", "https://example.com")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("expected allowed origin, got %q", got)
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Error("expected Vary: Origin header")
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	r := setupCORSRouter(t, "https://example.com")
	w := corsRequest(r, "GET", "https://evil.com")
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS header for disallowed origin")
	}
}

func TestCORS_NoEnvVar(t *testing.T) {
	r := setupCORSRouter(t, "")
	w := corsRequest(r, "GET", "https://example.com")
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS header when ALLOWED_ORIGIN not set")
	}
}

func TestCORS_PreflightAllowed(t *testing.T) {
	r := setupCORSRouter(t, "https://example.com")
	w := corsRequest(r, "OPTIONS", "https://example.com")
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("expected allowed origin in preflight, got %q", got)
	}
}

func TestCORS_PreflightDisallowed(t *testing.T) {
	r := setupCORSRouter(t, "https://example.com")
	w := corsRequest(r, "OPTIONS", "https://evil.com")
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS header in disallowed preflight")
	}
}
