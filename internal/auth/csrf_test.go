package auth

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"goread2/internal/database"
)

func TestCSRFManager(t *testing.T) {
	cm := NewCSRFManager()

	t.Run("GenerateToken", func(t *testing.T) {
		sessionID := "test-session-123"
		token, err := cm.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}
		if token == "" {
			t.Error("Generated token is empty")
		}
		if len(token) < 32 {
			t.Errorf("Token too short: %d characters", len(token))
		}
	})

	t.Run("ValidateToken_Success", func(t *testing.T) {
		sessionID := "test-session-456"
		token, err := cm.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		if !cm.ValidateToken(sessionID, token) {
			t.Error("Valid token failed validation")
		}
	})

	t.Run("ValidateToken_WrongToken", func(t *testing.T) {
		sessionID := "test-session-789"
		_, err := cm.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		if cm.ValidateToken(sessionID, "wrong-token") {
			t.Error("Invalid token passed validation")
		}
	})

	t.Run("ValidateToken_WrongSession", func(t *testing.T) {
		sessionID := "test-session-abc"
		token, err := cm.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		if cm.ValidateToken("different-session", token) {
			t.Error("Token validated for wrong session")
		}
	})

	t.Run("ValidateToken_InvalidToken", func(t *testing.T) {
		sessionID := "test-session-xyz"
		if cm.ValidateToken(sessionID, "invalid-token-format") {
			t.Error("Validation succeeded for invalid token")
		}
	})

	t.Run("DeleteToken_NoOp", func(t *testing.T) {
		// DeleteToken is a no-op in stateless implementation
		// Tokens remain valid as long as the session exists
		sessionID := "test-session-delete"
		token, err := cm.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// Verify token is valid
		if !cm.ValidateToken(sessionID, token) {
			t.Error("Token should be valid before deletion")
		}

		// Call DeleteToken (no-op)
		cm.DeleteToken(sessionID)

		// Token should still be valid (stateless - tied to session, not stored)
		if !cm.ValidateToken(sessionID, token) {
			t.Error("Token should still be valid after DeleteToken (stateless implementation)")
		}
	})

	t.Run("TokenDeterminism", func(t *testing.T) {
		// HMAC-based tokens should be deterministic - same session always produces same token
		sessionID := "test-session-determinism"

		token1, err := cm.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate first token: %v", err)
		}

		token2, err := cm.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate second token: %v", err)
		}

		if token1 != token2 {
			t.Error("Tokens should be deterministic - same session should produce same token")
		}
	})

	t.Run("TokenUniquePerSession", func(t *testing.T) {
		// Different sessions should produce different tokens
		tokens := make(map[string]bool)
		for i := 0; i < 100; i++ {
			sessionID := generateTestSessionID(i)
			token, err := cm.GenerateToken(sessionID)
			if err != nil {
				t.Fatalf("Failed to generate token for session %d: %v", i, err)
			}
			if tokens[token] {
				t.Errorf("Duplicate token generated for different session %d", i)
			}
			tokens[token] = true
		}
	})

	t.Run("StatelessAcrossInstances", func(t *testing.T) {
		// Simulate restart by creating a new manager with same secret
		sessionID := "test-session-restart"

		// Generate token with first instance
		token1, err := cm.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// Create new manager instance (simulates app restart)
		// Note: In production, secret must be configured via CSRF_SECRET env var
		// For testing, we use the same secret by accessing the struct directly
		cm2 := &CSRFManager{
			secret: cm.secret,
		}

		// Token should still be valid with new instance
		if !cm2.ValidateToken(sessionID, token1) {
			t.Error("Token should be valid across manager instances (stateless)")
		}

		// New instance should generate same token
		token2, err := cm2.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token with new instance: %v", err)
		}

		if token1 != token2 {
			t.Error("Same session should produce same token across instances")
		}
	})
}

// Helper function to generate unique session IDs for testing
func generateTestSessionID(i int) string {
	return "test-session-" + string(rune('a'+i%26)) + string(rune('a'+(i/26)%26))
}

func TestCSRFConcurrentGeneration(t *testing.T) {
	cm := NewCSRFManager()
	sessionID := "test-session-concurrent"

	// Generate tokens concurrently
	const numGoroutines = 100
	tokens := make([]string, numGoroutines)
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			token, err := cm.GenerateToken(sessionID)
			if err != nil {
				t.Errorf("Failed to generate token in goroutine %d: %v", index, err)
				return
			}
			tokens[index] = token
		}(i)
	}

	wg.Wait()

	// All tokens should be identical (deterministic)
	firstToken := tokens[0]
	for i, token := range tokens {
		if token != firstToken {
			t.Errorf("Token %d differs: got %s, want %s", i, token, firstToken)
		}
	}

	// Validate all tokens
	for i, token := range tokens {
		if !cm.ValidateToken(sessionID, token) {
			t.Errorf("Token %d failed validation", i)
		}
	}
}

func TestCSRFMiddleware(t *testing.T) {
	// Setup gin test mode
	gin.SetMode(gin.TestMode)

	db := newMockDB()
	sessionManager := NewSessionManager(db)
	middleware := NewMiddleware(sessionManager)
	csrfManager := NewCSRFManager()

	t.Run("safe methods bypass CSRF check", func(t *testing.T) {
		safeMethods := []string{"GET", "HEAD", "OPTIONS"}

		for _, method := range safeMethods {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(method, "/api/test", nil)

			middleware.CSRFMiddleware(csrfManager)(c)

			if c.IsAborted() {
				t.Errorf("%s request should not be aborted", method)
			}
		}
	})

	t.Run("POST without session returns unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/test", nil)

		middleware.CSRFMiddleware(csrfManager)(c)

		if w.Code != 401 {
			t.Errorf("Expected status 401, got %d", w.Code)
		}

		if !c.IsAborted() {
			t.Error("Expected request to be aborted")
		}
	})

	t.Run("POST without CSRF token returns forbidden", func(t *testing.T) {
		user := &database.User{
			ID:       1,
			Email:    "test@example.com",
			Name:     "Test User",
			GoogleID: "google-123",
		}

		// Create a valid session
		session, err := sessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/test", nil)
		c.Request.AddCookie(&http.Cookie{
			Name:  "session_id_local",
			Value: session.ID,
		})

		middleware.CSRFMiddleware(csrfManager)(c)

		if w.Code != 403 {
			t.Errorf("Expected status 403, got %d", w.Code)
		}

		if !c.IsAborted() {
			t.Error("Expected request to be aborted")
		}
	})

	t.Run("POST with invalid CSRF token returns forbidden", func(t *testing.T) {
		user := &database.User{
			ID:       1,
			Email:    "test@example.com",
			Name:     "Test User",
			GoogleID: "google-123",
		}

		// Create a valid session
		session, err := sessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/test", nil)
		c.Request.AddCookie(&http.Cookie{
			Name:  "session_id_local",
			Value: session.ID,
		})
		c.Request.Header.Set("X-CSRF-Token", "invalid-token")

		middleware.CSRFMiddleware(csrfManager)(c)

		if w.Code != 403 {
			t.Errorf("Expected status 403, got %d", w.Code)
		}

		if !c.IsAborted() {
			t.Error("Expected request to be aborted")
		}
	})

	t.Run("POST with valid CSRF token succeeds", func(t *testing.T) {
		user := &database.User{
			ID:       1,
			Email:    "test@example.com",
			Name:     "Test User",
			GoogleID: "google-123",
		}

		// Create a valid session
		session, err := sessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Generate valid CSRF token
		csrfToken, err := csrfManager.GenerateToken(session.ID)
		if err != nil {
			t.Fatalf("Failed to generate CSRF token: %v", err)
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/test", nil)
		c.Request.AddCookie(&http.Cookie{
			Name:  "session_id_local",
			Value: session.ID,
		})
		c.Request.Header.Set("X-CSRF-Token", csrfToken)

		middleware.CSRFMiddleware(csrfManager)(c)

		if c.IsAborted() {
			t.Error("Expected request not to be aborted")
		}
	})
}
