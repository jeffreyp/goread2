package auth

import (
	"testing"
	"time"
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

	t.Run("ValidateToken_NonexistentSession", func(t *testing.T) {
		if cm.ValidateToken("nonexistent-session", "any-token") {
			t.Error("Validation succeeded for nonexistent session")
		}
	})

	t.Run("DeleteToken", func(t *testing.T) {
		sessionID := "test-session-delete"
		token, err := cm.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// Verify token exists
		if !cm.ValidateToken(sessionID, token) {
			t.Error("Token should be valid before deletion")
		}

		// Delete token
		cm.DeleteToken(sessionID)

		// Verify token is deleted
		if cm.ValidateToken(sessionID, token) {
			t.Error("Token should be invalid after deletion")
		}
	})

	t.Run("TokenExpiration", func(t *testing.T) {
		sessionID := "test-session-expiry"
		token, err := cm.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// Manually set expiration to past
		cm.mu.Lock()
		if csrfToken, exists := cm.tokens[sessionID]; exists {
			csrfToken.ExpiresAt = time.Now().Add(-1 * time.Hour)
		}
		cm.mu.Unlock()

		// Token should fail validation
		if cm.ValidateToken(sessionID, token) {
			t.Error("Expired token should not validate")
		}
	})

	t.Run("TokenUniqueness", func(t *testing.T) {
		tokens := make(map[string]bool)
		for i := 0; i < 100; i++ {
			sessionID := "test-session-unique"
			token, err := cm.GenerateToken(sessionID)
			if err != nil {
				t.Fatalf("Failed to generate token: %v", err)
			}
			if tokens[token] {
				t.Error("Duplicate token generated")
			}
			tokens[token] = true
		}
	})
}
