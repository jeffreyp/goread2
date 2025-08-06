package secrets

import (
	"context"
	"fmt"
	"log"
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
	if clientID == "" || clientID[:8] == "_secret:" {
		secretName := os.Getenv("SECRET_CLIENT_ID_NAME")
		if secretName == "" {
			secretName = "google-client-id"
		}
		
		clientID, err = GetSecret(ctx, secretName)
		if err != nil {
			return "", "", fmt.Errorf("failed to get client ID secret: %w", err)
		}
	}

	if clientSecret == "" || clientSecret[:8] == "_secret:" {
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