package secrets

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

const (
	// secretManagerTimeout is the timeout for Secret Manager API calls
	// This prevents indefinite hangs if the service is slow or unavailable
	secretManagerTimeout = 10 * time.Second
)

// Cache for OAuth credentials (fetched once at startup)
var (
	oauthClientID     string
	oauthClientSecret string
	oauthOnce         sync.Once
	oauthErr          error
)

// Cache for Stripe credentials (fetched once at startup)
var (
	stripeSecretKey      string
	stripePublishableKey string
	stripeWebhookSecret  string
	stripePriceID        string
	stripeOnce           sync.Once
	stripeErr            error
)

// ResetCacheForTesting resets the secret caches for testing purposes
// This should only be used in tests to allow multiple test cases
func ResetCacheForTesting() {
	oauthClientID = ""
	oauthClientSecret = ""
	oauthOnce = sync.Once{}
	oauthErr = nil

	stripeSecretKey = ""
	stripePublishableKey = ""
	stripeWebhookSecret = ""
	stripePriceID = ""
	stripeOnce = sync.Once{}
	stripeErr = nil
}

// GetSecret retrieves a secret from Google Secret Manager
func GetSecret(ctx context.Context, secretName string) (string, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable is required")
	}

	// Create a context with timeout to prevent indefinite hangs
	ctx, cancel := context.WithTimeout(ctx, secretManagerTimeout)
	defer cancel()

	// Create the client
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secret manager client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Build the request
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName),
	}

	// Call the API
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}

	return string(result.Payload.Data), nil
}

// GetOAuthCredentials retrieves OAuth credentials from environment or secrets
// Uses sync.Once to cache credentials on first call, preventing repeated Secret Manager fetches
func GetOAuthCredentials(ctx context.Context) (clientID, clientSecret string, err error) {
	// Use sync.Once to ensure we only fetch secrets once, even with concurrent calls
	oauthOnce.Do(func() {
		// Try to get from environment variables first (for backwards compatibility)
		oauthClientID = os.Getenv("GOOGLE_CLIENT_ID")
		oauthClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")

		// If they contain secret references, or are empty, fetch from Secret Manager
		if oauthClientID == "" || (len(oauthClientID) >= 8 && oauthClientID[:8] == "_secret:") {
			secretName := os.Getenv("SECRET_CLIENT_ID_NAME")
			if secretName == "" {
				secretName = "google-client-id"
			}

			oauthClientID, oauthErr = GetSecret(ctx, secretName)
			if oauthErr != nil {
				oauthErr = fmt.Errorf("failed to get client ID secret: %w", oauthErr)
				return
			}
		}

		if oauthClientSecret == "" || (len(oauthClientSecret) >= 8 && oauthClientSecret[:8] == "_secret:") {
			secretName := os.Getenv("SECRET_CLIENT_SECRET_NAME")
			if secretName == "" {
				secretName = "google-client-secret"
			}

			oauthClientSecret, oauthErr = GetSecret(ctx, secretName)
			if oauthErr != nil {
				oauthErr = fmt.Errorf("failed to get client secret: %w", oauthErr)
				return
			}
		}
	})

	return oauthClientID, oauthClientSecret, oauthErr
}

// GetStripeCredentials retrieves Stripe credentials from environment or secrets
// Uses sync.Once to cache credentials on first call, preventing repeated Secret Manager fetches
func GetStripeCredentials(ctx context.Context) (secretKey, publishableKey, webhookSecret, priceID string, err error) {
	// Use sync.Once to ensure we only fetch secrets once, even with concurrent calls
	stripeOnce.Do(func() {
		// Get Stripe secret key
		stripeSecretKey = os.Getenv("STRIPE_SECRET_KEY")
		if stripeSecretKey == "" || stripeSecretKey == "stripe-secret-key" {
			stripeSecretKey, stripeErr = GetSecret(ctx, "stripe-secret-key")
			if stripeErr != nil {
				stripeErr = fmt.Errorf("failed to get Stripe secret key: %w", stripeErr)
				return
			}
		}

		// Get Stripe publishable key
		stripePublishableKey = os.Getenv("STRIPE_PUBLISHABLE_KEY")
		if stripePublishableKey == "" || stripePublishableKey == "stripe-publishable-key" {
			stripePublishableKey, stripeErr = GetSecret(ctx, "stripe-publishable-key")
			if stripeErr != nil {
				stripeErr = fmt.Errorf("failed to get Stripe publishable key: %w", stripeErr)
				return
			}
		}

		// Get Stripe webhook secret
		stripeWebhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")
		if stripeWebhookSecret == "" || stripeWebhookSecret == "stripe-webhook-secret" {
			stripeWebhookSecret, stripeErr = GetSecret(ctx, "stripe-webhook-secret")
			if stripeErr != nil {
				stripeErr = fmt.Errorf("failed to get Stripe webhook secret: %w", stripeErr)
				return
			}
		}

		// Get Stripe price ID
		stripePriceID = os.Getenv("STRIPE_PRICE_ID")
		if stripePriceID == "" || stripePriceID == "stripe-price-id" {
			stripePriceID, stripeErr = GetSecret(ctx, "stripe-price-id")
			if stripeErr != nil {
				stripeErr = fmt.Errorf("failed to get Stripe price ID: %w", stripeErr)
				return
			}
		}
	})

	return stripeSecretKey, stripePublishableKey, stripeWebhookSecret, stripePriceID, stripeErr
}
