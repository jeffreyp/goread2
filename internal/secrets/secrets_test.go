package secrets

import (
	"context"
	"os"
	"testing"
)

func TestGetOAuthCredentials_FromEnvironment(t *testing.T) {
	ctx := context.Background()

	t.Run("both credentials from environment", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		if err := os.Setenv("GOOGLE_CLIENT_ID", "test-client-id-123"); err != nil {
			t.Fatalf("Failed to set GOOGLE_CLIENT_ID: %v", err)
		}
		if err := os.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret-456"); err != nil {
			t.Fatalf("Failed to set GOOGLE_CLIENT_SECRET: %v", err)
		}
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_ID") }()
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_SECRET") }()

		// Execute
		clientID, clientSecret, err := GetOAuthCredentials(ctx)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if clientID != "test-client-id-123" {
			t.Errorf("Expected client ID 'test-client-id-123', got '%s'", clientID)
		}
		if clientSecret != "test-client-secret-456" {
			t.Errorf("Expected client secret 'test-client-secret-456', got '%s'", clientSecret)
		}
	})

	t.Run("missing GOOGLE_CLOUD_PROJECT for secret reference", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup - set credentials to trigger Secret Manager lookup
		if err := os.Setenv("GOOGLE_CLIENT_ID", "_secret:google-client-id"); err != nil {
			t.Fatalf("Failed to set GOOGLE_CLIENT_ID: %v", err)
		}
		if err := os.Setenv("GOOGLE_CLIENT_SECRET", "test-secret"); err != nil {
			t.Fatalf("Failed to set GOOGLE_CLIENT_SECRET: %v", err)
		}
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_ID") }()
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_SECRET") }()

		// Execute
		_, _, err := GetOAuthCredentials(ctx)

		// Assert
		if err == nil {
			t.Error("Expected error when GOOGLE_CLOUD_PROJECT is missing, got nil")
		}
		if err != nil && err.Error() != "failed to get client ID secret: GOOGLE_CLOUD_PROJECT environment variable is required" {
			t.Errorf("Expected GOOGLE_CLOUD_PROJECT error, got: %v", err)
		}
	})

	t.Run("empty client ID triggers secret lookup", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		_ = os.Unsetenv("GOOGLE_CLIENT_ID")
		if err := os.Setenv("GOOGLE_CLIENT_SECRET", "test-secret"); err != nil {
			t.Fatalf("Failed to set GOOGLE_CLIENT_SECRET: %v", err)
		}
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_SECRET") }()

		// Execute
		_, _, err := GetOAuthCredentials(ctx)

		// Assert - should fail because it tries Secret Manager without GOOGLE_CLOUD_PROJECT
		if err == nil {
			t.Error("Expected error when client ID is empty and needs Secret Manager, got nil")
		}
	})

	t.Run("secret reference prefix triggers secret lookup", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		if err := os.Setenv("GOOGLE_CLIENT_ID", "_secret:my-client-id"); err != nil {
			t.Fatalf("Failed to set GOOGLE_CLIENT_ID: %v", err)
		}
		if err := os.Setenv("GOOGLE_CLIENT_SECRET", "test-secret"); err != nil {
			t.Fatalf("Failed to set GOOGLE_CLIENT_SECRET: %v", err)
		}
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_ID") }()
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_SECRET") }()

		// Execute
		_, _, err := GetOAuthCredentials(ctx)

		// Assert
		if err == nil {
			t.Error("Expected error when using _secret: prefix without GOOGLE_CLOUD_PROJECT, got nil")
		}
	})

	t.Run("client secret with secret reference", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		if err := os.Setenv("GOOGLE_CLIENT_ID", "test-client-id"); err != nil {
			t.Fatalf("Failed to set GOOGLE_CLIENT_ID: %v", err)
		}
		if err := os.Setenv("GOOGLE_CLIENT_SECRET", "_secret:my-secret"); err != nil {
			t.Fatalf("Failed to set GOOGLE_CLIENT_SECRET: %v", err)
		}
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_ID") }()
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_SECRET") }()

		// Execute
		_, _, err := GetOAuthCredentials(ctx)

		// Assert
		if err == nil {
			t.Error("Expected error when using _secret: prefix for client secret without GOOGLE_CLOUD_PROJECT, got nil")
		}
	})
}

func TestGetStripeCredentials_FromEnvironment(t *testing.T) {
	ctx := context.Background()

	t.Run("all credentials from environment", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		if err := os.Setenv("STRIPE_SECRET_KEY", "sk_test_123456"); err != nil {
			t.Fatalf("Failed to set STRIPE_SECRET_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_789012"); err != nil {
			t.Fatalf("Failed to set STRIPE_PUBLISHABLE_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_345678"); err != nil {
			t.Fatalf("Failed to set STRIPE_WEBHOOK_SECRET: %v", err)
		}
		if err := os.Setenv("STRIPE_PRICE_ID", "price_test_901234"); err != nil {
			t.Fatalf("Failed to set STRIPE_PRICE_ID: %v", err)
		}
		defer func() { _ = os.Unsetenv("STRIPE_SECRET_KEY") }()
		defer func() { _ = os.Unsetenv("STRIPE_PUBLISHABLE_KEY") }()
		defer func() { _ = os.Unsetenv("STRIPE_WEBHOOK_SECRET") }()
		defer func() { _ = os.Unsetenv("STRIPE_PRICE_ID") }()

		// Execute
		secretKey, publishableKey, webhookSecret, priceID, err := GetStripeCredentials(ctx)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if secretKey != "sk_test_123456" {
			t.Errorf("Expected secret key 'sk_test_123456', got '%s'", secretKey)
		}
		if publishableKey != "pk_test_789012" {
			t.Errorf("Expected publishable key 'pk_test_789012', got '%s'", publishableKey)
		}
		if webhookSecret != "whsec_test_345678" {
			t.Errorf("Expected webhook secret 'whsec_test_345678', got '%s'", webhookSecret)
		}
		if priceID != "price_test_901234" {
			t.Errorf("Expected price ID 'price_test_901234', got '%s'", priceID)
		}
	})

	t.Run("missing secret key triggers secret lookup", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		_ = os.Unsetenv("STRIPE_SECRET_KEY")
		if err := os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_789012"); err != nil {
			t.Fatalf("Failed to set STRIPE_PUBLISHABLE_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_345678"); err != nil {
			t.Fatalf("Failed to set STRIPE_WEBHOOK_SECRET: %v", err)
		}
		if err := os.Setenv("STRIPE_PRICE_ID", "price_test_901234"); err != nil {
			t.Fatalf("Failed to set STRIPE_PRICE_ID: %v", err)
		}
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		defer func() { _ = os.Unsetenv("STRIPE_PUBLISHABLE_KEY") }()
		defer func() { _ = os.Unsetenv("STRIPE_WEBHOOK_SECRET") }()
		defer func() { _ = os.Unsetenv("STRIPE_PRICE_ID") }()

		// Execute
		_, _, _, _, err := GetStripeCredentials(ctx)

		// Assert
		if err == nil {
			t.Error("Expected error when secret key is missing and needs Secret Manager, got nil")
		}
	})

	t.Run("placeholder value triggers secret lookup", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup - "stripe-secret-key" is treated as a placeholder
		if err := os.Setenv("STRIPE_SECRET_KEY", "stripe-secret-key"); err != nil {
			t.Fatalf("Failed to set STRIPE_SECRET_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_789012"); err != nil {
			t.Fatalf("Failed to set STRIPE_PUBLISHABLE_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_345678"); err != nil {
			t.Fatalf("Failed to set STRIPE_WEBHOOK_SECRET: %v", err)
		}
		if err := os.Setenv("STRIPE_PRICE_ID", "price_test_901234"); err != nil {
			t.Fatalf("Failed to set STRIPE_PRICE_ID: %v", err)
		}
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		defer func() { _ = os.Unsetenv("STRIPE_SECRET_KEY") }()
		defer func() { _ = os.Unsetenv("STRIPE_PUBLISHABLE_KEY") }()
		defer func() { _ = os.Unsetenv("STRIPE_WEBHOOK_SECRET") }()
		defer func() { _ = os.Unsetenv("STRIPE_PRICE_ID") }()

		// Execute
		_, _, _, _, err := GetStripeCredentials(ctx)

		// Assert
		if err == nil {
			t.Error("Expected error when using placeholder value without GOOGLE_CLOUD_PROJECT, got nil")
		}
	})

	t.Run("publishable key placeholder triggers secret lookup", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		if err := os.Setenv("STRIPE_SECRET_KEY", "sk_test_123456"); err != nil {
			t.Fatalf("Failed to set STRIPE_SECRET_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_PUBLISHABLE_KEY", "stripe-publishable-key"); err != nil {
			t.Fatalf("Failed to set STRIPE_PUBLISHABLE_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_345678"); err != nil {
			t.Fatalf("Failed to set STRIPE_WEBHOOK_SECRET: %v", err)
		}
		if err := os.Setenv("STRIPE_PRICE_ID", "price_test_901234"); err != nil {
			t.Fatalf("Failed to set STRIPE_PRICE_ID: %v", err)
		}
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		defer func() { _ = os.Unsetenv("STRIPE_SECRET_KEY") }()
		defer func() { _ = os.Unsetenv("STRIPE_PUBLISHABLE_KEY") }()
		defer func() { _ = os.Unsetenv("STRIPE_WEBHOOK_SECRET") }()
		defer func() { _ = os.Unsetenv("STRIPE_PRICE_ID") }()

		// Execute
		_, _, _, _, err := GetStripeCredentials(ctx)

		// Assert
		if err == nil {
			t.Error("Expected error when using placeholder value without GOOGLE_CLOUD_PROJECT, got nil")
		}
	})

	t.Run("webhook secret placeholder triggers secret lookup", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		if err := os.Setenv("STRIPE_SECRET_KEY", "sk_test_123456"); err != nil {
			t.Fatalf("Failed to set STRIPE_SECRET_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_789012"); err != nil {
			t.Fatalf("Failed to set STRIPE_PUBLISHABLE_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_WEBHOOK_SECRET", "stripe-webhook-secret"); err != nil {
			t.Fatalf("Failed to set STRIPE_WEBHOOK_SECRET: %v", err)
		}
		if err := os.Setenv("STRIPE_PRICE_ID", "price_test_901234"); err != nil {
			t.Fatalf("Failed to set STRIPE_PRICE_ID: %v", err)
		}
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		defer func() { _ = os.Unsetenv("STRIPE_SECRET_KEY") }()
		defer func() { _ = os.Unsetenv("STRIPE_PUBLISHABLE_KEY") }()
		defer func() { _ = os.Unsetenv("STRIPE_WEBHOOK_SECRET") }()
		defer func() { _ = os.Unsetenv("STRIPE_PRICE_ID") }()

		// Execute
		_, _, _, _, err := GetStripeCredentials(ctx)

		// Assert
		if err == nil {
			t.Error("Expected error when using placeholder value without GOOGLE_CLOUD_PROJECT, got nil")
		}
	})

	t.Run("price ID placeholder triggers secret lookup", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		if err := os.Setenv("STRIPE_SECRET_KEY", "sk_test_123456"); err != nil {
			t.Fatalf("Failed to set STRIPE_SECRET_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_789012"); err != nil {
			t.Fatalf("Failed to set STRIPE_PUBLISHABLE_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_345678"); err != nil {
			t.Fatalf("Failed to set STRIPE_WEBHOOK_SECRET: %v", err)
		}
		if err := os.Setenv("STRIPE_PRICE_ID", "stripe-price-id"); err != nil {
			t.Fatalf("Failed to set STRIPE_PRICE_ID: %v", err)
		}
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		defer func() { _ = os.Unsetenv("STRIPE_SECRET_KEY") }()
		defer func() { _ = os.Unsetenv("STRIPE_PUBLISHABLE_KEY") }()
		defer func() { _ = os.Unsetenv("STRIPE_WEBHOOK_SECRET") }()
		defer func() { _ = os.Unsetenv("STRIPE_PRICE_ID") }()

		// Execute
		_, _, _, _, err := GetStripeCredentials(ctx)

		// Assert
		if err == nil {
			t.Error("Expected error when using placeholder value without GOOGLE_CLOUD_PROJECT, got nil")
		}
	})
}

func TestGetSecret_MissingProjectID(t *testing.T) {
	ctx := context.Background()

	t.Run("missing GOOGLE_CLOUD_PROJECT", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")

		// Execute
		_, err := GetSecret(ctx, "test-secret")

		// Assert
		if err == nil {
			t.Error("Expected error when GOOGLE_CLOUD_PROJECT is missing, got nil")
		}
		expectedError := "GOOGLE_CLOUD_PROJECT environment variable is required"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%v'", expectedError, err)
		}
	})
}

func TestGetOAuthCredentials_CustomSecretNames(t *testing.T) {
	// Note: This test verifies the logic but can't test actual Secret Manager
	// without mocking the API client
	ctx := context.Background()

	t.Run("custom secret names via environment", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup - Use environment variables to avoid Secret Manager
		if err := os.Setenv("GOOGLE_CLIENT_ID", "test-client-id"); err != nil {
			t.Fatalf("Failed to set GOOGLE_CLIENT_ID: %v", err)
		}
		if err := os.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret"); err != nil {
			t.Fatalf("Failed to set GOOGLE_CLIENT_SECRET: %v", err)
		}
		if err := os.Setenv("SECRET_CLIENT_ID_NAME", "custom-client-id"); err != nil {
			t.Fatalf("Failed to set SECRET_CLIENT_ID_NAME: %v", err)
		}
		if err := os.Setenv("SECRET_CLIENT_SECRET_NAME", "custom-client-secret"); err != nil {
			t.Fatalf("Failed to set SECRET_CLIENT_SECRET_NAME: %v", err)
		}
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_ID") }()
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_SECRET") }()
		defer func() { _ = os.Unsetenv("SECRET_CLIENT_ID_NAME") }()
		defer func() { _ = os.Unsetenv("SECRET_CLIENT_SECRET_NAME") }()

		// Execute - should use env vars and not need secret names
		clientID, clientSecret, err := GetOAuthCredentials(ctx)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if clientID != "test-client-id" {
			t.Errorf("Expected client ID 'test-client-id', got '%s'", clientID)
		}
		if clientSecret != "test-client-secret" {
			t.Errorf("Expected client secret 'test-client-secret', got '%s'", clientSecret)
		}
	})
}
