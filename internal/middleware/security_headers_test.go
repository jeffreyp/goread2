package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupSecurityRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

func doRequest(r *gin.Engine) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	return w
}

func TestSecurityHeaders_AlwaysPresent(t *testing.T) {
	if err := os.Unsetenv("CSP_ENFORCE"); err != nil {
		t.Fatal(err)
	}
	r := setupSecurityRouter()
	w := doRequest(r)

	tests := []struct {
		header   string
		contains string
	}{
		{"X-Frame-Options", "SAMEORIGIN"},
		{"X-Content-Type-Options", "nosniff"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Permissions-Policy", "geolocation=()"},
	}

	for _, tt := range tests {
		val := w.Header().Get(tt.header)
		if val == "" {
			t.Errorf("expected %s header to be set, got empty", tt.header)
		}
		if !strings.Contains(val, tt.contains) {
			t.Errorf("expected %s to contain %q, got %q", tt.header, tt.contains, val)
		}
	}
}

func TestSecurityHeaders_CSPReportOnlyByDefault(t *testing.T) {
	if err := os.Unsetenv("CSP_ENFORCE"); err != nil {
		t.Fatal(err)
	}
	r := setupSecurityRouter()
	w := doRequest(r)

	// Should use Report-Only header, not enforcing
	reportOnly := w.Header().Get("Content-Security-Policy-Report-Only")
	enforcing := w.Header().Get("Content-Security-Policy")

	if reportOnly == "" {
		t.Error("expected Content-Security-Policy-Report-Only header, got empty")
	}
	if enforcing != "" {
		t.Errorf("expected no Content-Security-Policy header in report-only mode, got %q", enforcing)
	}
}

func TestSecurityHeaders_CSPEnforced(t *testing.T) {
	t.Setenv("CSP_ENFORCE", "true")

	r := setupSecurityRouter()
	w := doRequest(r)

	enforcing := w.Header().Get("Content-Security-Policy")
	reportOnly := w.Header().Get("Content-Security-Policy-Report-Only")

	if enforcing == "" {
		t.Error("expected Content-Security-Policy header when CSP_ENFORCE=true, got empty")
	}
	if reportOnly != "" {
		t.Errorf("expected no Report-Only header when enforcing, got %q", reportOnly)
	}
}

func TestSecurityHeaders_CSPDirectives(t *testing.T) {
	if err := os.Unsetenv("CSP_ENFORCE"); err != nil {
		t.Fatal(err)
	}
	r := setupSecurityRouter()
	w := doRequest(r)

	csp := w.Header().Get("Content-Security-Policy-Report-Only")
	directives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: http: https:",
		"frame-ancestors 'self'",
		"base-uri 'self'",
		"form-action 'self'",
		"connect-src 'self'",
		"font-src 'self'",
	}

	for _, d := range directives {
		if !strings.Contains(csp, d) {
			t.Errorf("CSP missing directive %q, got: %s", d, csp)
		}
	}
}

func TestSecurityHeaders_CSPAllowsGoogleAnalytics(t *testing.T) {
	if err := os.Unsetenv("CSP_ENFORCE"); err != nil {
		t.Fatal(err)
	}
	r := setupSecurityRouter()
	w := doRequest(r)

	csp := w.Header().Get("Content-Security-Policy-Report-Only")

	// GA script loading
	if !strings.Contains(csp, "https://www.googletagmanager.com") {
		t.Error("CSP should allow googletagmanager.com in script-src")
	}
	// GA beacon sending (wildcard for regional endpoints)
	if !strings.Contains(csp, "https://*.google-analytics.com") {
		t.Error("CSP should allow *.google-analytics.com in connect-src")
	}
}

func TestSecurityHeaders_CSPAllowsDOMPurifyCDN(t *testing.T) {
	if err := os.Unsetenv("CSP_ENFORCE"); err != nil {
		t.Fatal(err)
	}
	r := setupSecurityRouter()
	w := doRequest(r)

	csp := w.Header().Get("Content-Security-Policy-Report-Only")
	if !strings.Contains(csp, "https://cdn.jsdelivr.net") {
		t.Error("CSP should allow cdn.jsdelivr.net in script-src for DOMPurify")
	}
}

func TestSecurityHeaders_HSTSOnlyInProduction(t *testing.T) {
	if err := os.Unsetenv("GAE_ENV"); err != nil {
		t.Fatal(err)
	}
	r := setupSecurityRouter()
	w := doRequest(r)

	if hsts := w.Header().Get("Strict-Transport-Security"); hsts != "" {
		t.Errorf("expected no HSTS outside production, got %q", hsts)
	}
}

func TestSecurityHeaders_HSTSInProduction(t *testing.T) {
	t.Setenv("GAE_ENV", "standard")

	r := setupSecurityRouter()
	w := doRequest(r)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts == "" {
		t.Error("expected HSTS header in production, got empty")
	}
	if !strings.Contains(hsts, "max-age=31536000") {
		t.Errorf("expected HSTS max-age=31536000, got %q", hsts)
	}
	if !strings.Contains(hsts, "includeSubDomains") {
		t.Errorf("expected HSTS includeSubDomains, got %q", hsts)
	}
}
