// Package security consolidates the regression suite for GoRead2's core
// security controls (CSRF, auth bypass, SSRF, feed-limit enforcement) into
// one location with a dedicated CI signal, instead of being scattered across
// test/integration and internal/*/  test files.
package security

import (
	"net/http"
	"testing"

	"github.com/jeffreyp/goread2/test/helpers"
)

func TestLogoutCSRF(t *testing.T) {
	t.Parallel()

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)
	user := helpers.CreateTestUser(t, testServer.DB, "csrf_google123", "csrf_test@example.com", "CSRF Test User")

	t.Run("Logout_WithValidCSRF_Succeeds", func(t *testing.T) {
		req := testServer.CreateAuthenticatedRequest(t, "POST", "/auth/logout", nil, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Logout_WithoutCSRFToken_Rejected", func(t *testing.T) {
		session, err := testServer.SessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		req, _ := http.NewRequest("POST", "/auth/logout", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session_id_local",
			Value: session.ID,
		})
		// Intentionally omit X-CSRF-Token header

		rr := testServer.ExecuteRequest(req)
		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Logout_WithInvalidCSRFToken_Rejected", func(t *testing.T) {
		session, err := testServer.SessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		req, _ := http.NewRequest("POST", "/auth/logout", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session_id_local",
			Value: session.ID,
		})
		req.Header.Set("X-CSRF-Token", "invalid-token-value")

		rr := testServer.ExecuteRequest(req)
		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Logout_Unauthenticated_Rejected", func(t *testing.T) {
		// Without any session, CSRF middleware returns 401 (can't validate token with no session)
		req, _ := http.NewRequest("POST", "/auth/logout", nil)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d: %s", rr.Code, rr.Body.String())
		}
	})

}

func TestAPICSRF(t *testing.T) {
	t.Parallel()

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)
	user := helpers.CreateTestUser(t, testServer.DB, "api_csrf_google456", "api_csrf_test@example.com", "API CSRF Test User")

	t.Run("StateChangingAPI_WithoutCSRF_Rejected", func(t *testing.T) {
		session, err := testServer.SessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		req, _ := http.NewRequest("POST", "/api/feeds", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session_id_local",
			Value: session.ID,
		})
		req.Header.Set("Content-Type", "application/json")
		// Intentionally omit X-CSRF-Token

		rr := testServer.ExecuteRequest(req)
		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("ReadOnlyAPI_WithoutCSRF_Allowed", func(t *testing.T) {
		req := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds", nil, user)
		// GET requests don't require CSRF
		req.Header.Del("X-CSRF-Token")

		rr := testServer.ExecuteRequest(req)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

}
