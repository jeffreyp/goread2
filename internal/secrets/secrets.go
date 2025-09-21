package secrets

import (
	"context"
	"fmt"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// GetSecret retrieves a secret from Google Secret Manager
func GetSecret(ctx context.Context, secretName string) (string, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable is required")
	}

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
func GetOAuthCredentials(ctx context.Context) (clientID, clientSecret string, err error) {
	// Try to get from environment variables first (for backwards compatibility)
	clientID = os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")

	// If they contain secret references, or are empty, fetch from Secret Manager
	if clientID == "" || (len(clientID) >= 8 && clientID[:8] == "_secret:") {
		secretName := os.Getenv("SECRET_CLIENT_ID_NAME")
		if secretName == "" {
			secretName = "google-client-id"
		}

		clientID, err = GetSecret(ctx, secretName)
		if err != nil {
			return "", "", fmt.Errorf("failed to get client ID secret: %w", err)
		}
	}

	if clientSecret == "" || (len(clientSecret) >= 8 && clientSecret[:8] == "_secret:") {
		secretName := os.Getenv("SECRET_CLIENT_SECRET_NAME")
		if secretName == "" {
			secretName = "google-client-secret"
		}

		clientSecret, err = GetSecret(ctx, secretName)
		if err != nil {
			return "", "", fmt.Errorf("failed to get client secret: %w", err)
		}
	}

	return clientID, clientSecret, nil
}

// GetStripeCredentials retrieves Stripe credentials from environment or secrets
func GetStripeCredentials(ctx context.Context) (secretKey, publishableKey, webhookSecret, priceID string, err error) {
	// Get Stripe secret key
	secretKey = os.Getenv("STRIPE_SECRET_KEY")
	if secretKey == "" || secretKey == "stripe-secret-key" {
		secretKey, err = GetSecret(ctx, "stripe-secret-key")
		if err != nil {
			return "", "", "", "", fmt.Errorf("failed to get Stripe secret key: %w", err)
		}
	}

	// Get Stripe publishable key
	publishableKey = os.Getenv("STRIPE_PUBLISHABLE_KEY")
	if publishableKey == "" || publishableKey == "stripe-publishable-key" {
		publishableKey, err = GetSecret(ctx, "stripe-publishable-key")
		if err != nil {
			return "", "", "", "", fmt.Errorf("failed to get Stripe publishable key: %w", err)
		}
	}

	// Get Stripe webhook secret
	webhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" || webhookSecret == "stripe-webhook-secret" {
		webhookSecret, err = GetSecret(ctx, "stripe-webhook-secret")
		if err != nil {
			return "", "", "", "", fmt.Errorf("failed to get Stripe webhook secret: %w", err)
		}
	}

	// Get Stripe price ID
	priceID = os.Getenv("STRIPE_PRICE_ID")
	if priceID == "" || priceID == "stripe-price-id" {
		priceID, err = GetSecret(ctx, "stripe-price-id")
		if err != nil {
			return "", "", "", "", fmt.Errorf("failed to get Stripe price ID: %w", err)
		}
	}

	return secretKey, publishableKey, webhookSecret, priceID, nil
}
