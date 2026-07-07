package security

import (
	"testing"

	"github.com/jeffreyp/goread2/test/helpers"
)

// TestProtectedEndpointsRejectUnauthenticated is the single, authoritative
// list of every route registered under RequireAuth() in
// test/helpers.SetupTestServer. Adding a new protected endpoint without
// adding it here means it silently drops out of the security regression
// suite's coverage.
func TestProtectedEndpointsRejectUnauthenticated(t *testing.T) {
	t.Parallel()

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/feeds"},
		{"POST", "/api/feeds"},
		{"DELETE", "/api/feeds/1"},
		{"POST", "/api/feeds/import"},
		{"GET", "/api/feeds/export"},
		{"GET", "/api/feeds/1/articles"},
		{"POST", "/api/articles/1/read"},
		{"POST", "/api/articles/1/star"},
		{"POST", "/api/articles/mark-all-read"},
		{"POST", "/api/feeds/refresh"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+"_"+ep.path, func(t *testing.T) {
			req := helpers.CreateUnauthenticatedRequest(t, ep.method, ep.path, nil)
			rr := testServer.ExecuteRequest(req)

			if rr.Code != 401 {
				t.Errorf("expected 401 for %s %s without a session cookie, got %d: %s", ep.method, ep.path, rr.Code, rr.Body.String())
			}
		})
	}
}
