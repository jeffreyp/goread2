package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ginzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// newNotFoundTestRouter mirrors main.go's setup: the gzip middleware wraps all
// responses, and NotFoundHandler is registered via NoRoute.
func newNotFoundTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginzip.Gzip(ginzip.DefaultCompression))
	r.NoRoute(NotFoundHandler())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "home")
	})
	return r
}

func TestNotFoundHandlerScannerPaths(t *testing.T) {
	router := newNotFoundTestRouter()

	paths := []string{
		"/wp-admin/phpinfo.php",
		"/core/phpinfo.php",
		"/test.php",
		"/backup.bak",
		"/config.old",
		"/nonexistent",
	}

	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("GET %s: expected status 404, got %d", path, w.Code)
		}
		if body := w.Body.String(); !strings.Contains(body, "404 page not found") {
			t.Errorf("GET %s: expected 404 body, got %q", path, body)
		}
	}
}

// TestNotFoundHandlerGzip covers the regression from gr-kspw: without an
// explicit NoRoute handler, Gin's fallback 404 body is written after the gzip
// writer closes, and clients that accept gzip receive an empty response that
// the App Engine frontend reports as status 200.
func TestNotFoundHandlerGzip(t *testing.T) {
	router := newNotFoundTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/phpinfo.php", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
	if enc := w.Header().Get("Content-Encoding"); enc != "gzip" {
		t.Fatalf("expected Content-Encoding gzip, got %q", enc)
	}

	gz, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("response body is not valid gzip: %v", err)
	}
	defer func() { _ = gz.Close() }()
	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to decompress response body: %v", err)
	}
	if !strings.Contains(string(body), "404 page not found") {
		t.Errorf("expected 404 body after decompression, got %q", body)
	}
}

func TestNotFoundHandlerLeavesKnownRoutesAlone(t *testing.T) {
	router := newNotFoundTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for known route, got %d", w.Code)
	}
}
