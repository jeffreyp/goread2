package security

import (
	"net/http"
	"testing"

	"github.com/jeffreyp/goread2/test/helpers"
)

// TestAddFeedRejectsSSRFTargets exercises SSRF protection through the real
// POST /api/feeds handler rather than calling the URL validator directly
// (see internal/services/url_validator_test.go for the unit-level cases) —
// this catches regressions where the handler or feed service stops wiring
// the validated HTTP client into the request path.
func TestAddFeedRejectsSSRFTargets(t *testing.T) {
	t.Parallel()

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)
	user := helpers.CreateTestUser(t, testServer.DB, "ssrf_google1", "ssrf_test@example.com", "SSRF Test User")

	targets := []string{
		"http://127.0.0.1/feed.xml",
		"http://localhost/feed.xml",
		"http://169.254.169.254/latest/meta-data/", // cloud metadata endpoint
		"http://10.0.0.1/feed.xml",                 // RFC1918
		"http://192.168.1.1/feed.xml",              // RFC1918
		"http://172.16.0.1/feed.xml",               // RFC1918
		"http://[::1]/feed.xml",                    // IPv6 loopback
	}

	for _, url := range targets {
		t.Run(url, func(t *testing.T) {
			req := testServer.CreateAuthenticatedRequest(t, "POST", "/api/feeds", map[string]string{"url": url}, user)
			rr := testServer.ExecuteRequest(req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected SSRF target %q to be rejected with 400, got %d: %s", url, rr.Code, rr.Body.String())
			}
		})
	}
}
