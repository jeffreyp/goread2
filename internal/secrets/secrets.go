package secrets

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

const (
	// secretManagerTimeout is the timeout for Secret Manager API calls
	// This prevents indefinite hangs if the service is slow or unavailable
	secretManagerTimeout = 10 * time.Second

	// secretPrefix is the marker indicating a value is a Secret Manager reference
	secretPrefix = "_secret:"
)

// Singleton Secret Manager client (reused across all calls)
// This prevents creating a new client on every GetSecret call
var (
	secretClient     *secretmanager.Client
	secretClientOnce sync.Once
	secretClientErr  error
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

// Cache for the CSRF secret (fetched once at startup)
var (
	csrfSecret string
	csrfOnce   sync.Once
	csrfErr    error
)

// Cache for the admin token (fetched once at startup)
var (
	adminToken     string
	adminTokenOnce sync.Once
	adminTokenErr  error
)

// Cache for initial admin emails (fetched once at startup)
var (
	initialAdminEmails     string
	initialAdminEmailsOnce sync.Once
	initialAdminEmailsErr  error
)

// secretFetcher is the function used to call Secret Manager; replaced in tests
var secretFetcher = fetchFromSecretManager

// fetchFromSecretManager is the real Secret Manager implementation
func fetchFromSecretManager(ctx context.Context, projectID, secretName, version string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, secretManagerTimeout)
	defer cancel()

	client, err := getOrCreateSecretClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secret manager client: %w", err)
	}

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/%s", projectID, secretName, version),
	}

	log.Printf("secrets: fetching %q version %q from Secret Manager (project=%s)", secretName, version, projectID)
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		log.Printf("secrets: failed to fetch %q: %v", secretName, err)
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}
	log.Printf("secrets: successfully fetched %q", secretName)

	return string(result.Payload.Data), nil
}

// ResetCacheForTesting resets the secret caches for testing purposes
// This should only be used in tests to allow multiple test cases
func ResetCacheForTesting() {
	// Close existing client if it exists
	if secretClient != nil {
		_ = secretClient.Close()
		secretClient = nil
	}
	secretClientOnce = sync.Once{}
	secretClientErr = nil

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

	csrfSecret = ""
	csrfOnce = sync.Once{}
	csrfErr = nil

	adminToken = ""
	adminTokenOnce = sync.Once{}
	adminTokenErr = nil

	initialAdminEmails = ""
	initialAdminEmailsOnce = sync.Once{}
	initialAdminEmailsErr = nil

	secretFetcher = fetchFromSecretManager
}

// getOrCreateSecretClient returns the singleton Secret Manager client
// Creates it on first call, reuses it on subsequent calls
func getOrCreateSecretClient(ctx context.Context) (*secretmanager.Client, error) {
	secretClientOnce.Do(func() {
		// Create the client once and reuse it
		secretClient, secretClientErr = secretmanager.NewClient(ctx)
	})
	return secretClient, secretClientErr
}

// GetSecret retrieves the latest version of a secret from Google Secret Manager
func GetSecret(ctx context.Context, secretName string) (string, error) {
	return GetSecretVersion(ctx, secretName, "latest")
}

// GetSecretVersion retrieves a specific version of a secret from Google Secret Manager.
// Use "latest" to get the current version, or a numeric string (e.g. "3") to pin to a specific version.
func GetSecretVersion(ctx context.Context, secretName, version string) (string, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable is required")
	}
	if version == "" {
		version = "latest"
	}
	return secretFetcher(ctx, projectID, secretName, version)
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
		if oauthClientID == "" || (strings.HasPrefix(oauthClientID, secretPrefix)) {
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

		if oauthClientSecret == "" || (strings.HasPrefix(oauthClientSecret, secretPrefix)) {
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

		if oauthClientID == "" {
			oauthErr = fmt.Errorf("GOOGLE_CLIENT_ID is empty")
			return
		}
		if oauthClientSecret == "" {
			oauthErr = fmt.Errorf("GOOGLE_CLIENT_SECRET is empty")
			return
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
		if stripeSecretKey == "" || (strings.HasPrefix(stripeSecretKey, secretPrefix)) {
			stripeSecretKey, stripeErr = GetSecret(ctx, "stripe-secret-key")
			if stripeErr != nil {
				stripeErr = fmt.Errorf("failed to get Stripe secret key: %w", stripeErr)
				return
			}
		}

		// Get Stripe publishable key
		stripePublishableKey = os.Getenv("STRIPE_PUBLISHABLE_KEY")
		if stripePublishableKey == "" || (strings.HasPrefix(stripePublishableKey, secretPrefix)) {
			stripePublishableKey, stripeErr = GetSecret(ctx, "stripe-publishable-key")
			if stripeErr != nil {
				stripeErr = fmt.Errorf("failed to get Stripe publishable key: %w", stripeErr)
				return
			}
		}

		// Get Stripe webhook secret
		stripeWebhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")
		if stripeWebhookSecret == "" || (strings.HasPrefix(stripeWebhookSecret, secretPrefix)) {
			stripeWebhookSecret, stripeErr = GetSecret(ctx, "stripe-webhook-secret")
			if stripeErr != nil {
				stripeErr = fmt.Errorf("failed to get Stripe webhook secret: %w", stripeErr)
				return
			}
		}

		// Get Stripe price ID
		stripePriceID = os.Getenv("STRIPE_PRICE_ID")
		if stripePriceID == "" || (strings.HasPrefix(stripePriceID, secretPrefix)) {
			stripePriceID, stripeErr = GetSecret(ctx, "stripe-price-id")
			if stripeErr != nil {
				stripeErr = fmt.Errorf("failed to get Stripe price ID: %w", stripeErr)
				return
			}
		}

		if stripeSecretKey == "" {
			stripeErr = fmt.Errorf("STRIPE_SECRET_KEY is empty")
			return
		}
		if stripePublishableKey == "" {
			stripeErr = fmt.Errorf("STRIPE_PUBLISHABLE_KEY is empty")
			return
		}
		if stripeWebhookSecret == "" {
			stripeErr = fmt.Errorf("STRIPE_WEBHOOK_SECRET is empty")
			return
		}
		if stripePriceID == "" {
			stripeErr = fmt.Errorf("STRIPE_PRICE_ID is empty")
			return
		}
	})

	return stripeSecretKey, stripePublishableKey, stripeWebhookSecret, stripePriceID, stripeErr
}

// GetCSRFSecret retrieves the CSRF secret from environment or Secret Manager.
// Unlike OAuth/Stripe, an empty result is not necessarily an error — callers
// (e.g. in local development) may fall back to generating an ephemeral secret.
func GetCSRFSecret(ctx context.Context) (string, error) {
	csrfOnce.Do(func() {
		csrfSecret = os.Getenv("CSRF_SECRET")
		if csrfSecret == "" || strings.HasPrefix(csrfSecret, secretPrefix) {
			csrfSecret, csrfErr = GetSecret(ctx, "csrf-secret")
		}
	})
	return csrfSecret, csrfErr
}

// GetAdminToken retrieves the admin CLI token from environment or Secret Manager.
// An empty result is not an error — it just means the ADMIN_TOKEN auth path is
// disabled; callers already fail closed on an empty expected token.
func GetAdminToken(ctx context.Context) (string, error) {
	adminTokenOnce.Do(func() {
		adminToken = os.Getenv("ADMIN_TOKEN")
		if adminToken == "" || strings.HasPrefix(adminToken, secretPrefix) {
			adminToken, adminTokenErr = GetSecret(ctx, "admin-token")
		}
	})
	return adminToken, adminTokenErr
}

// GetInitialAdminEmails retrieves the comma-separated initial admin email list
// from environment or Secret Manager. An empty result just means no initial
// admin bootstrap is configured.
func GetInitialAdminEmails(ctx context.Context) (string, error) {
	initialAdminEmailsOnce.Do(func() {
		initialAdminEmails = os.Getenv("INITIAL_ADMIN_EMAILS")
		if initialAdminEmails == "" || strings.HasPrefix(initialAdminEmails, secretPrefix) {
			initialAdminEmails, initialAdminEmailsErr = GetSecret(ctx, "initial-admin-emails")
		}
	})
	return initialAdminEmails, initialAdminEmailsErr
}
