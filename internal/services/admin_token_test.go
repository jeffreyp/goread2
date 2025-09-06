package services

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"goread2/internal/database"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *database.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	dbWrapper := &database.DB{DB: db}

	// Create admin_tokens table
	_, err = dbWrapper.Exec(`CREATE TABLE admin_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		token_hash TEXT UNIQUE NOT NULL,
		description TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_active BOOLEAN DEFAULT 1
	)`)
	if err != nil {
		t.Fatalf("Failed to create admin_tokens table: %v", err)
	}

	return dbWrapper
}

func TestGenerateAdminToken(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewSubscriptionService(db)

	t.Run("GenerateValidToken", func(t *testing.T) {
		description := "Test token generation"

		token, err := service.GenerateAdminToken(description)
		if err != nil {
			t.Fatalf("Failed to generate admin token: %v", err)
		}

		// Verify token format
		if len(token) != 64 {
			t.Errorf("Expected token length 64, got %d", len(token))
		}

		// Verify token contains only hex characters
		for _, char := range token {
			if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
				t.Errorf("Token contains non-hex character: %c", char)
			}
		}
	})

	t.Run("GenerateMultipleUniqueTokens", func(t *testing.T) {
		token1, err := service.GenerateAdminToken("First token")
		if err != nil {
			t.Fatalf("Failed to generate first token: %v", err)
		}

		token2, err := service.GenerateAdminToken("Second token")
		if err != nil {
			t.Fatalf("Failed to generate second token: %v", err)
		}

		if token1 == token2 {
			t.Error("Generated tokens should be unique")
		}
	})

	t.Run("GenerateWithEmptyDescription", func(t *testing.T) {
		token, err := service.GenerateAdminToken("")
		if err != nil {
			t.Fatalf("Failed to generate token with empty description: %v", err)
		}

		if len(token) != 64 {
			t.Errorf("Expected token length 64, got %d", len(token))
		}
	})
}

func TestValidateAdminToken(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewSubscriptionService(db)

	// Generate a test token
	token, err := service.GenerateAdminToken("Test validation")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	t.Run("ValidateValidToken", func(t *testing.T) {
		valid, err := service.ValidateAdminToken(token)
		if err != nil {
			t.Fatalf("Validation error: %v", err)
		}
		if !valid {
			t.Error("Expected token to be valid")
		}
	})

	t.Run("ValidateInvalidToken", func(t *testing.T) {
		invalidToken := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

		valid, err := service.ValidateAdminToken(invalidToken)
		if err != nil {
			t.Fatalf("Validation error: %v", err)
		}
		if valid {
			t.Error("Expected invalid token to be rejected")
		}
	})

	t.Run("ValidateShortToken", func(t *testing.T) {
		shortToken := "short"

		valid, err := service.ValidateAdminToken(shortToken)
		if err != nil {
			t.Fatalf("Validation error: %v", err)
		}
		if valid {
			t.Error("Expected short token to be rejected")
		}
	})

	t.Run("ValidateLongToken", func(t *testing.T) {
		longToken := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234"

		valid, err := service.ValidateAdminToken(longToken)
		if err != nil {
			t.Fatalf("Validation error: %v", err)
		}
		if valid {
			t.Error("Expected long token to be rejected")
		}
	})

	t.Run("ValidateEmptyToken", func(t *testing.T) {
		valid, err := service.ValidateAdminToken("")
		if err != nil {
			t.Fatalf("Validation error: %v", err)
		}
		if valid {
			t.Error("Expected empty token to be rejected")
		}
	})

	t.Run("ValidateUpdatesLastUsed", func(t *testing.T) {
		// Get initial last used time
		tokens, err := service.ListAdminTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		var initialLastUsed time.Time
		for _, token := range tokens {
			if token.Description == "Test validation" {
				initialLastUsed = token.LastUsedAt
				break
			}
		}

		// Wait a brief moment to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		// Validate token again
		valid, err := service.ValidateAdminToken(token)
		if err != nil {
			t.Fatalf("Validation error: %v", err)
		}
		if !valid {
			t.Error("Expected token to be valid")
		}

		// Check that last used time was updated
		tokens, err = service.ListAdminTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		var updatedLastUsed time.Time
		for _, token := range tokens {
			if token.Description == "Test validation" {
				updatedLastUsed = token.LastUsedAt
				break
			}
		}

		if !updatedLastUsed.After(initialLastUsed) {
			t.Error("Expected last used time to be updated")
		}
	})
}

func TestListAdminTokens(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewSubscriptionService(db)

	t.Run("ListEmptyTokens", func(t *testing.T) {
		tokens, err := service.ListAdminTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		if len(tokens) != 0 {
			t.Errorf("Expected 0 tokens, got %d", len(tokens))
		}
	})

	t.Run("ListMultipleTokens", func(t *testing.T) {
		// Generate test tokens
		descriptions := []string{"First token", "Second token", "Third token"}

		for _, desc := range descriptions {
			_, err := service.GenerateAdminToken(desc)
			if err != nil {
				t.Fatalf("Failed to generate token '%s': %v", desc, err)
			}
		}

		// List tokens
		tokens, err := service.ListAdminTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		if len(tokens) != len(descriptions) {
			t.Errorf("Expected %d tokens, got %d", len(descriptions), len(tokens))
		}

		// Verify all descriptions are present
		foundDescriptions := make(map[string]bool)
		for _, token := range tokens {
			foundDescriptions[token.Description] = true

			// Verify token properties
			if token.ID <= 0 {
				t.Errorf("Token ID should be positive, got %d", token.ID)
			}
			if token.TokenHash == "" {
				t.Error("Token hash should not be empty")
			}
			if len(token.TokenHash) != 64 {
				t.Errorf("Token hash should be 64 chars, got %d", len(token.TokenHash))
			}
			if token.CreatedAt.IsZero() {
				t.Error("Created at should not be zero")
			}
			if token.LastUsedAt.IsZero() {
				t.Error("Last used at should not be zero")
			}
			if !token.IsActive {
				t.Error("New tokens should be active")
			}
		}

		for _, desc := range descriptions {
			if !foundDescriptions[desc] {
				t.Errorf("Description '%s' not found in tokens", desc)
			}
		}
	})

	t.Run("ListOrderedByCreatedAt", func(t *testing.T) {
		// Clear existing tokens for clean test
		freshDB := setupTestDB(t)
		defer func() { _ = freshDB.Close() }()
		service := NewSubscriptionService(freshDB)

		// Generate tokens with slight delay to ensure different timestamps
		_, err := service.GenerateAdminToken("Oldest token")
		if err != nil {
			t.Fatalf("Failed to generate first token: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		_, err = service.GenerateAdminToken("Newest token")
		if err != nil {
			t.Fatalf("Failed to generate second token: %v", err)
		}

		tokens, err := service.ListAdminTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		if len(tokens) != 2 {
			t.Fatalf("Expected 2 tokens, got %d", len(tokens))
		}

		// Tokens should be ordered by created_at DESC (newest first)
		if tokens[0].Description != "Newest token" {
			t.Errorf("Expected newest token first, got '%s'", tokens[0].Description)
		}
		if tokens[1].Description != "Oldest token" {
			t.Errorf("Expected oldest token second, got '%s'", tokens[1].Description)
		}
	})
}

func TestRevokeAdminToken(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewSubscriptionService(db)

	// Generate a test token
	token, err := service.GenerateAdminToken("Test revocation")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Get token ID
	tokens, err := service.ListAdminTokens()
	if err != nil {
		t.Fatalf("Failed to list tokens: %v", err)
	}

	var tokenID int
	for _, adminToken := range tokens {
		if adminToken.Description == "Test revocation" {
			tokenID = adminToken.ID
			break
		}
	}

	t.Run("RevokeValidToken", func(t *testing.T) {
		err := service.RevokeAdminToken(tokenID)
		if err != nil {
			t.Fatalf("Failed to revoke token: %v", err)
		}

		// Verify token is no longer valid
		valid, err := service.ValidateAdminToken(token)
		if err != nil {
			t.Fatalf("Validation error: %v", err)
		}
		if valid {
			t.Error("Expected revoked token to be invalid")
		}

		// Verify token shows as inactive in listing
		tokens, err := service.ListAdminTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		var found bool
		for _, adminToken := range tokens {
			if adminToken.ID == tokenID {
				if adminToken.IsActive {
					t.Error("Expected revoked token to be inactive")
				}
				found = true
				break
			}
		}

		if !found {
			t.Error("Token should still appear in listing after revocation")
		}
	})

	t.Run("RevokeNonExistentToken", func(t *testing.T) {
		err := service.RevokeAdminToken(99999)
		if err == nil {
			t.Error("Expected error when revoking non-existent token")
		}
	})

	t.Run("RevokeAlreadyRevokedToken", func(t *testing.T) {
		// Should error since token is already inactive
		err := service.RevokeAdminToken(tokenID)
		if err == nil {
			t.Error("Expected error when revoking already revoked token")
		}
	})
}

func TestHasAdminTokens(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewSubscriptionService(db)

	t.Run("NoTokensExist", func(t *testing.T) {
		hasTokens, err := service.HasAdminTokens()
		if err != nil {
			t.Fatalf("Failed to check for tokens: %v", err)
		}
		if hasTokens {
			t.Error("Expected no tokens to exist")
		}
	})

	t.Run("ActiveTokenExists", func(t *testing.T) {
		_, err := service.GenerateAdminToken("Test token")
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		hasTokens, err := service.HasAdminTokens()
		if err != nil {
			t.Fatalf("Failed to check for tokens: %v", err)
		}
		if !hasTokens {
			t.Error("Expected active token to exist")
		}
	})

	t.Run("OnlyInactiveTokensExist", func(t *testing.T) {
		// Get token ID and revoke it
		tokens, err := service.ListAdminTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		for _, token := range tokens {
			if token.IsActive {
				err := service.RevokeAdminToken(token.ID)
				if err != nil {
					t.Fatalf("Failed to revoke token: %v", err)
				}
			}
		}

		hasTokens, err := service.HasAdminTokens()
		if err != nil {
			t.Fatalf("Failed to check for tokens: %v", err)
		}
		if hasTokens {
			t.Error("Expected no active tokens to exist")
		}
	})
}

func TestAdminTokenUniqueGeneration(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewSubscriptionService(db)

	t.Run("MultipleTokenGeneration", func(t *testing.T) {
		// Test generating multiple tokens sequentially to verify uniqueness
		const numTokens = 10
		tokens := make([]string, numTokens)

		for i := 0; i < numTokens; i++ {
			token, err := service.GenerateAdminToken(fmt.Sprintf("Token %d", i))
			if err != nil {
				t.Fatalf("Failed to generate token %d: %v", i, err)
			}
			tokens[i] = token
		}

		// Verify all tokens are unique
		tokenMap := make(map[string]bool)
		for i, token := range tokens {
			if tokenMap[token] {
				t.Errorf("Duplicate token generated at index %d: %s", i, token)
			}
			tokenMap[token] = true

			// Verify token format
			if len(token) != 64 {
				t.Errorf("Token %d has wrong length: expected 64, got %d", i, len(token))
			}
		}

		if len(tokenMap) != numTokens {
			t.Errorf("Expected %d unique tokens, got %d", numTokens, len(tokenMap))
		}
	})

	t.Run("ValidationConsistency", func(t *testing.T) {
		// Generate a token and validate it multiple times
		token, err := service.GenerateAdminToken("Validation test")
		if err != nil {
			t.Fatalf("Failed to generate test token: %v", err)
		}

		// Validate multiple times
		for i := 0; i < 5; i++ {
			valid, err := service.ValidateAdminToken(token)
			if err != nil {
				t.Fatalf("Validation %d failed: %v", i, err)
			}
			if !valid {
				t.Errorf("Validation %d should have succeeded", i)
			}
		}
	})
}
