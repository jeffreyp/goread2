package services

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/jeffreyp/goread2/internal/database"
)

// setupDatastoreDB creates a test datastore database
// Note: This requires the datastore emulator to be running
// Run: gcloud beta emulators datastore start --no-store-on-disk
func setupDatastoreDB(t *testing.T) *database.DatastoreDB {
	// Skip if not in datastore test environment
	if os.Getenv("DATASTORE_EMULATOR_HOST") == "" {
		t.Skip("Datastore emulator not available - set DATASTORE_EMULATOR_HOST to run datastore tests")
	}

	projectID := "test-project-" + fmt.Sprintf("%d", time.Now().UnixNano())
	db, err := database.NewDatastoreDB(projectID)
	if err != nil {
		t.Fatalf("Failed to create datastore DB: %v", err)
	}

	return db
}

// cleanupDatastoreDB removes all test entities
func cleanupDatastoreDB(t *testing.T, db *database.DatastoreDB) {
	ctx := context.Background()
	client := db.GetClient()

	// Clean up AdminToken entities
	query := datastore.NewQuery("AdminToken").KeysOnly()
	keys, err := client.GetAll(ctx, query, nil)
	if err != nil {
		t.Logf("Failed to query AdminToken keys for cleanup: %v", err)
	} else if len(keys) > 0 {
		err = client.DeleteMulti(ctx, keys)
		if err != nil {
			t.Logf("Failed to delete AdminToken entities: %v", err)
		}
	}

	// Clean up User entities
	query = datastore.NewQuery("User").KeysOnly()
	keys, err = client.GetAll(ctx, query, nil)
	if err != nil {
		t.Logf("Failed to query User keys for cleanup: %v", err)
	} else if len(keys) > 0 {
		err = client.DeleteMulti(ctx, keys)
		if err != nil {
			t.Logf("Failed to delete User entities: %v", err)
		}
	}

	err = db.Close()
	if err != nil {
		t.Logf("Failed to close datastore DB: %v", err)
	}
}

func TestDatastoreGenerateAdminToken(t *testing.T) {
	db := setupDatastoreDB(t)
	defer cleanupDatastoreDB(t, db)

	service := NewSubscriptionService(db)

	t.Run("GenerateTokenInDatastore", func(t *testing.T) {
		description := "Datastore test token"

		token, err := service.GenerateAdminToken(description)
		if err != nil {
			t.Fatalf("Failed to generate admin token in datastore: %v", err)
		}

		// Verify token format
		if len(token) != 64 {
			t.Errorf("Expected token length 64, got %d", len(token))
		}

		// Verify token was stored in datastore
		ctx := context.Background()
		query := datastore.NewQuery("AdminToken").
			FilterField("description", "=", description)

		var entities []*database.AdminTokenEntity
		keys, err := db.GetClient().GetAll(ctx, query, &entities)
		if err != nil {
			t.Fatalf("Failed to query stored token: %v", err)
		}

		if len(entities) != 1 {
			t.Fatalf("Expected 1 token entity, got %d", len(entities))
		}

		entity := entities[0]
		if entity.Description != description {
			t.Errorf("Expected description '%s', got '%s'", description, entity.Description)
		}
		if !entity.IsActive {
			t.Error("New token should be active")
		}
		if entity.TokenHash == "" {
			t.Error("Token hash should not be empty")
		}
		if len(entity.TokenHash) != 64 {
			t.Errorf("Token hash should be 64 chars, got %d", len(entity.TokenHash))
		}
		if keys[0].ID <= 0 {
			t.Errorf("Entity key ID should be positive, got %d", keys[0].ID)
		}
	})

	t.Run("GenerateMultipleDatastoreTokens", func(t *testing.T) {
		descriptions := []string{"Token 1", "Token 2", "Token 3"}
		tokens := make([]string, len(descriptions))

		// Generate tokens
		for i, desc := range descriptions {
			token, err := service.GenerateAdminToken(desc)
			if err != nil {
				t.Fatalf("Failed to generate token '%s': %v", desc, err)
			}
			tokens[i] = token
		}

		// Verify all tokens are unique
		for i := 0; i < len(tokens); i++ {
			for j := i + 1; j < len(tokens); j++ {
				if tokens[i] == tokens[j] {
					t.Errorf("Tokens %d and %d are identical: %s", i, j, tokens[i])
				}
			}
		}

		// Verify all tokens were stored
		ctx := context.Background()
		query := datastore.NewQuery("AdminToken")

		var entities []*database.AdminTokenEntity
		_, err := db.GetClient().GetAll(ctx, query, &entities)
		if err != nil {
			t.Fatalf("Failed to query stored tokens: %v", err)
		}

		if len(entities) < len(descriptions) {
			t.Errorf("Expected at least %d entities, got %d", len(descriptions), len(entities))
		}

		// Verify descriptions are present
		foundDescriptions := make(map[string]bool)
		for _, entity := range entities {
			foundDescriptions[entity.Description] = true
		}

		for _, desc := range descriptions {
			if !foundDescriptions[desc] {
				t.Errorf("Description '%s' not found in stored entities", desc)
			}
		}
	})
}

func TestDatastoreValidateAdminToken(t *testing.T) {
	db := setupDatastoreDB(t)
	defer cleanupDatastoreDB(t, db)

	service := NewSubscriptionService(db)

	// Generate a test token
	token, err := service.GenerateAdminToken("Datastore validation test")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	t.Run("ValidateDatastoreToken", func(t *testing.T) {
		valid, err := service.ValidateAdminToken(token)
		if err != nil {
			t.Fatalf("Validation error: %v", err)
		}
		if !valid {
			t.Error("Expected token to be valid")
		}
	})

	t.Run("ValidateUpdatesLastUsedInDatastore", func(t *testing.T) {
		// Get initial last used time
		ctx := context.Background()
		query := datastore.NewQuery("AdminToken").
			FilterField("description", "=", "Datastore validation test")

		var entities []*database.AdminTokenEntity
		keys, err := db.GetClient().GetAll(ctx, query, &entities)
		if err != nil {
			t.Fatalf("Failed to query token: %v", err)
		}

		if len(entities) == 0 {
			t.Fatal("Token not found in datastore")
		}

		initialLastUsed := entities[0].LastUsedAt
		entityKey := keys[0]

		// Wait to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		// Validate token
		valid, err := service.ValidateAdminToken(token)
		if err != nil {
			t.Fatalf("Validation error: %v", err)
		}
		if !valid {
			t.Error("Expected token to be valid")
		}

		// Check that last used time was updated
		var updatedEntity database.AdminTokenEntity
		err = db.GetClient().Get(ctx, entityKey, &updatedEntity)
		if err != nil {
			t.Fatalf("Failed to get updated entity: %v", err)
		}

		if !updatedEntity.LastUsedAt.After(initialLastUsed) {
			t.Errorf("Expected last used time to be updated. Initial: %v, Updated: %v",
				initialLastUsed, updatedEntity.LastUsedAt)
		}
	})

	t.Run("ValidateInvalidDatastoreToken", func(t *testing.T) {
		invalidToken := "0000000000000000000000000000000000000000000000000000000000000000"

		valid, err := service.ValidateAdminToken(invalidToken)
		if err != nil {
			t.Fatalf("Validation error: %v", err)
		}
		if valid {
			t.Error("Expected invalid token to be rejected")
		}
	})
}

func TestDatastoreListAdminTokens(t *testing.T) {
	db := setupDatastoreDB(t)
	defer cleanupDatastoreDB(t, db)

	service := NewSubscriptionService(db)

	t.Run("ListEmptyDatastoreTokens", func(t *testing.T) {
		tokens, err := service.ListAdminTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		if len(tokens) != 0 {
			t.Errorf("Expected 0 tokens, got %d", len(tokens))
		}
	})

	t.Run("ListDatastoreTokensOrderedByCreatedAt", func(t *testing.T) {
		// Generate tokens with delays to ensure different timestamps
		_, err := service.GenerateAdminToken("Oldest datastore token")
		if err != nil {
			t.Fatalf("Failed to generate first token: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		_, err = service.GenerateAdminToken("Middle datastore token")
		if err != nil {
			t.Fatalf("Failed to generate second token: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		_, err = service.GenerateAdminToken("Newest datastore token")
		if err != nil {
			t.Fatalf("Failed to generate third token: %v", err)
		}

		tokens, err := service.ListAdminTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		if len(tokens) != 3 {
			t.Fatalf("Expected 3 tokens, got %d", len(tokens))
		}

		// Tokens should be ordered by created_at DESC (newest first)
		expectedOrder := []string{
			"Newest datastore token",
			"Middle datastore token",
			"Oldest datastore token",
		}

		for i, expectedDesc := range expectedOrder {
			if tokens[i].Description != expectedDesc {
				t.Errorf("Expected token %d to be '%s', got '%s'",
					i, expectedDesc, tokens[i].Description)
			}
		}

		// Verify all tokens have proper properties
		for i, token := range tokens {
			if token.ID <= 0 {
				t.Errorf("Token %d ID should be positive, got %d", i, token.ID)
			}
			if token.TokenHash == "" {
				t.Errorf("Token %d hash should not be empty", i)
			}
			if token.CreatedAt.IsZero() {
				t.Errorf("Token %d created at should not be zero", i)
			}
			if !token.IsActive {
				t.Errorf("Token %d should be active", i)
			}
		}
	})
}

func TestDatastoreRevokeAdminToken(t *testing.T) {
	db := setupDatastoreDB(t)
	defer cleanupDatastoreDB(t, db)

	service := NewSubscriptionService(db)

	// Generate a test token
	token, err := service.GenerateAdminToken("Datastore revocation test")
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
		if adminToken.Description == "Datastore revocation test" {
			tokenID = adminToken.ID
			break
		}
	}

	if tokenID == 0 {
		t.Fatal("Failed to find generated token")
	}

	t.Run("RevokeDatastoreToken", func(t *testing.T) {
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

		// Verify token is marked inactive in datastore
		ctx := context.Background()
		key := datastore.IDKey("AdminToken", int64(tokenID), nil)

		var entity database.AdminTokenEntity
		err = db.GetClient().Get(ctx, key, &entity)
		if err != nil {
			t.Fatalf("Failed to get token entity: %v", err)
		}

		if entity.IsActive {
			t.Error("Expected revoked token to be inactive in datastore")
		}
	})

	t.Run("RevokeNonExistentDatastoreToken", func(t *testing.T) {
		err := service.RevokeAdminToken(99999)
		if err == nil {
			t.Error("Expected error when revoking non-existent datastore token")
		}
	})
}

func TestDatastoreHasAdminTokens(t *testing.T) {
	db := setupDatastoreDB(t)
	defer cleanupDatastoreDB(t, db)

	service := NewSubscriptionService(db)

	t.Run("NoDatastoreTokensExist", func(t *testing.T) {
		hasTokens, err := service.HasAdminTokens()
		if err != nil {
			t.Fatalf("Failed to check for tokens: %v", err)
		}
		if hasTokens {
			t.Error("Expected no tokens to exist")
		}
	})

	t.Run("ActiveDatastoreTokenExists", func(t *testing.T) {
		_, err := service.GenerateAdminToken("Active datastore token")
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

	t.Run("OnlyInactiveDatastoreTokensExist", func(t *testing.T) {
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

func TestDatastoreAdminTokenCompatibility(t *testing.T) {
	db := setupDatastoreDB(t)
	defer cleanupDatastoreDB(t, db)

	service := NewSubscriptionService(db)

	t.Run("TokenCompatibilityBetweenOperations", func(t *testing.T) {
		// This test verifies that tokens created via one method work with all other methods

		// Generate token
		token, err := service.GenerateAdminToken("Compatibility test token")
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// List tokens to get ID
		tokens, err := service.ListAdminTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens: %v", err)
		}

		var tokenID int
		for _, adminToken := range tokens {
			if adminToken.Description == "Compatibility test token" {
				tokenID = adminToken.ID
				break
			}
		}

		// Validate token
		valid, err := service.ValidateAdminToken(token)
		if err != nil {
			t.Fatalf("Failed to validate token: %v", err)
		}
		if !valid {
			t.Error("Token should be valid")
		}

		// Check has tokens
		hasTokens, err := service.HasAdminTokens()
		if err != nil {
			t.Fatalf("Failed to check has tokens: %v", err)
		}
		if !hasTokens {
			t.Error("Should have tokens")
		}

		// Revoke token
		err = service.RevokeAdminToken(tokenID)
		if err != nil {
			t.Fatalf("Failed to revoke token: %v", err)
		}

		// Validate again (should fail)
		valid, err = service.ValidateAdminToken(token)
		if err != nil {
			t.Fatalf("Failed to validate revoked token: %v", err)
		}
		if valid {
			t.Error("Revoked token should be invalid")
		}

		// List tokens again (should still show token but inactive)
		tokens, err = service.ListAdminTokens()
		if err != nil {
			t.Fatalf("Failed to list tokens after revocation: %v", err)
		}

		var foundToken *AdminToken
		for _, adminToken := range tokens {
			if adminToken.ID == tokenID {
				foundToken = &adminToken
				break
			}
		}

		if foundToken == nil {
			t.Error("Token should still appear in listing after revocation")
		} else if foundToken.IsActive {
			t.Error("Revoked token should be inactive")
		}
	})
}
