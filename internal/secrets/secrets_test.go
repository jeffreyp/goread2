package secrets

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func setMockSecretFetcher(values map[string]string, fetchErr error) func() {
	old := secretFetcher
	secretFetcher = func(_ context.Context, _ string, secretName string) (string, error) {
		if fetchErr != nil {
			return "", fetchErr
		}
		v, ok := values[secretName]
		if !ok {
			return "", fmt.Errorf("secret %q not found in mock", secretName)
		}
		return v, nil
	}
	return func() { secretFetcher = old }
}

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

	t.Run("secret reference prefix triggers secret lookup", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup - "_secret:" prefix triggers Secret Manager lookup
		if err := os.Setenv("STRIPE_SECRET_KEY", "_secret:stripe-secret-key"); err != nil {
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
			t.Error("Expected error when using _secret: prefix without GOOGLE_CLOUD_PROJECT, got nil")
		}
	})

	t.Run("publishable key with secret reference", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		if err := os.Setenv("STRIPE_SECRET_KEY", "sk_test_123456"); err != nil {
			t.Fatalf("Failed to set STRIPE_SECRET_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_PUBLISHABLE_KEY", "_secret:stripe-publishable-key"); err != nil {
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
			t.Error("Expected error when using _secret: prefix without GOOGLE_CLOUD_PROJECT, got nil")
		}
	})

	t.Run("webhook secret with secret reference", func(t *testing.T) {
		// Reset cache before test
		ResetCacheForTesting()

		// Setup
		if err := os.Setenv("STRIPE_SECRET_KEY", "sk_test_123456"); err != nil {
			t.Fatalf("Failed to set STRIPE_SECRET_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_789012"); err != nil {
			t.Fatalf("Failed to set STRIPE_PUBLISHABLE_KEY: %v", err)
		}
		if err := os.Setenv("STRIPE_WEBHOOK_SECRET", "_secret:stripe-webhook-secret"); err != nil {
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
			t.Error("Expected error when using _secret: prefix without GOOGLE_CLOUD_PROJECT, got nil")
		}
	})

	t.Run("price ID with secret reference", func(t *testing.T) {
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
		if err := os.Setenv("STRIPE_PRICE_ID", "_secret:stripe-price-id"); err != nil {
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
			t.Error("Expected error when using _secret: prefix without GOOGLE_CLOUD_PROJECT, got nil")
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

func TestGetOAuthCredentials_EmptyValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("empty client ID from env returns error", func(t *testing.T) {
		ResetCacheForTesting()
		t.Setenv("GOOGLE_CLIENT_ID", "")
		t.Setenv("GOOGLE_CLIENT_SECRET", "some-secret")
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")

		_, _, err := GetOAuthCredentials(ctx)
		if err == nil {
			t.Error("Expected error for empty GOOGLE_CLIENT_ID, got nil")
		}
	})

	t.Run("empty client secret from env returns error", func(t *testing.T) {
		ResetCacheForTesting()
		t.Setenv("GOOGLE_CLIENT_ID", "some-client-id")
		t.Setenv("GOOGLE_CLIENT_SECRET", "")
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")

		_, _, err := GetOAuthCredentials(ctx)
		if err == nil {
			t.Error("Expected error for empty GOOGLE_CLIENT_SECRET, got nil")
		}
	})
}

func TestGetStripeCredentials_EmptyValidation(t *testing.T) {
	ctx := context.Background()

	setAllStripe := func(t *testing.T) {
		t.Helper()
		t.Setenv("STRIPE_SECRET_KEY", "sk_test_valid")
		t.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_valid")
		t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_valid")
		t.Setenv("STRIPE_PRICE_ID", "price_valid")
	}

	t.Run("empty secret key returns error", func(t *testing.T) {
		ResetCacheForTesting()
		setAllStripe(t)
		t.Setenv("STRIPE_SECRET_KEY", "")
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")

		_, _, _, _, err := GetStripeCredentials(ctx)
		if err == nil {
			t.Error("Expected error for empty STRIPE_SECRET_KEY, got nil")
		}
	})

	t.Run("empty publishable key returns error", func(t *testing.T) {
		ResetCacheForTesting()
		setAllStripe(t)
		t.Setenv("STRIPE_PUBLISHABLE_KEY", "")
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")

		_, _, _, _, err := GetStripeCredentials(ctx)
		if err == nil {
			t.Error("Expected error for empty STRIPE_PUBLISHABLE_KEY, got nil")
		}
	})

	t.Run("empty webhook secret returns error", func(t *testing.T) {
		ResetCacheForTesting()
		setAllStripe(t)
		t.Setenv("STRIPE_WEBHOOK_SECRET", "")
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")

		_, _, _, _, err := GetStripeCredentials(ctx)
		if err == nil {
			t.Error("Expected error for empty STRIPE_WEBHOOK_SECRET, got nil")
		}
	})

	t.Run("empty price ID returns error", func(t *testing.T) {
		ResetCacheForTesting()
		setAllStripe(t)
		t.Setenv("STRIPE_PRICE_ID", "")
		_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")

		_, _, _, _, err := GetStripeCredentials(ctx)
		if err == nil {
			t.Error("Expected error for empty STRIPE_PRICE_ID, got nil")
		}
	})
}

func TestGetSecret_WithMockedSecretManager(t *testing.T) {
	ctx := context.Background()

	t.Run("returns value from Secret Manager", func(t *testing.T) {
		ResetCacheForTesting()
		t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
		defer setMockSecretFetcher(map[string]string{"my-secret": "secret-value"}, nil)()

		value, err := GetSecret(ctx, "my-secret")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if value != "secret-value" {
			t.Errorf("Expected 'secret-value', got %q", value)
		}
	})

	t.Run("propagates Secret Manager error", func(t *testing.T) {
		ResetCacheForTesting()
		t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
		defer setMockSecretFetcher(nil, fmt.Errorf("permission denied"))()

		_, err := GetSecret(ctx, "my-secret")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})
}

func TestGetOAuthCredentials_WithMockedSecretManager(t *testing.T) {
	ctx := context.Background()

	t.Run("fetches via Secret Manager when _secret: prefix used", func(t *testing.T) {
		ResetCacheForTesting()
		t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
		t.Setenv("GOOGLE_CLIENT_ID", "_secret:google-client-id")
		t.Setenv("GOOGLE_CLIENT_SECRET", "_secret:google-client-secret")
		defer setMockSecretFetcher(map[string]string{
			"google-client-id":     "real-client-id",
			"google-client-secret": "real-client-secret",
		}, nil)()

		clientID, clientSecret, err := GetOAuthCredentials(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if clientID != "real-client-id" {
			t.Errorf("Expected 'real-client-id', got %q", clientID)
		}
		if clientSecret != "real-client-secret" {
			t.Errorf("Expected 'real-client-secret', got %q", clientSecret)
		}
	})

	t.Run("fetches via Secret Manager when env vars are empty", func(t *testing.T) {
		ResetCacheForTesting()
		t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
		t.Setenv("GOOGLE_CLIENT_ID", "")
		t.Setenv("GOOGLE_CLIENT_SECRET", "")
		defer setMockSecretFetcher(map[string]string{
			"google-client-id":     "sm-client-id",
			"google-client-secret": "sm-client-secret",
		}, nil)()

		clientID, clientSecret, err := GetOAuthCredentials(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if clientID != "sm-client-id" {
			t.Errorf("Expected 'sm-client-id', got %q", clientID)
		}
		if clientSecret != "sm-client-secret" {
			t.Errorf("Expected 'sm-client-secret', got %q", clientSecret)
		}
	})

	t.Run("propagates Secret Manager error through GetOAuthCredentials", func(t *testing.T) {
		ResetCacheForTesting()
		t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
		t.Setenv("GOOGLE_CLIENT_ID", "_secret:google-client-id")
		t.Setenv("GOOGLE_CLIENT_SECRET", "direct-secret")
		defer setMockSecretFetcher(nil, fmt.Errorf("secret not found"))()

		_, _, err := GetOAuthCredentials(ctx)
		if err == nil {
			t.Fatal("Expected error from Secret Manager, got nil")
		}
	})
}

func TestGetStripeCredentials_WithMockedSecretManager(t *testing.T) {
	ctx := context.Background()

	t.Run("fetches all Stripe credentials from Secret Manager", func(t *testing.T) {
		ResetCacheForTesting()
		t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
		t.Setenv("STRIPE_SECRET_KEY", "_secret:stripe-secret-key")
		t.Setenv("STRIPE_PUBLISHABLE_KEY", "_secret:stripe-publishable-key")
		t.Setenv("STRIPE_WEBHOOK_SECRET", "_secret:stripe-webhook-secret")
		t.Setenv("STRIPE_PRICE_ID", "_secret:stripe-price-id")
		defer setMockSecretFetcher(map[string]string{
			"stripe-secret-key":      "sk_live_test",
			"stripe-publishable-key": "pk_live_test",
			"stripe-webhook-secret":  "whsec_live_test",
			"stripe-price-id":        "price_live_test",
		}, nil)()

		secretKey, publishableKey, webhookSecret, priceID, err := GetStripeCredentials(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if secretKey != "sk_live_test" {
			t.Errorf("Expected 'sk_live_test', got %q", secretKey)
		}
		if publishableKey != "pk_live_test" {
			t.Errorf("Expected 'pk_live_test', got %q", publishableKey)
		}
		if webhookSecret != "whsec_live_test" {
			t.Errorf("Expected 'whsec_live_test', got %q", webhookSecret)
		}
		if priceID != "price_live_test" {
			t.Errorf("Expected 'price_live_test', got %q", priceID)
		}
	})

	t.Run("propagates Secret Manager error through GetStripeCredentials", func(t *testing.T) {
		ResetCacheForTesting()
		t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
		t.Setenv("STRIPE_SECRET_KEY", "_secret:stripe-secret-key")
		t.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test")
		t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
		t.Setenv("STRIPE_PRICE_ID", "price_test")
		defer setMockSecretFetcher(nil, fmt.Errorf("access denied"))()

		_, _, _, _, err := GetStripeCredentials(ctx)
		if err == nil {
			t.Fatal("Expected error from Secret Manager, got nil")
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
