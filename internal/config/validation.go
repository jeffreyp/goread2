package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"goread2/internal/secrets"
)

// ValidateEnvironmentConfig validates that all environment variables are properly configured
// Set strict=false for local development, strict=true for production validation
func ValidateEnvironmentConfig() error {
	return ValidateEnvironmentConfigStrict(false)
}

// ValidateEnvironmentConfigStrict validates with optional strict mode
func ValidateEnvironmentConfigStrict(strict bool) error {
	var errors []string

	// Check OAuth credentials
	if err := validateOAuthConfig(strict); err != nil {
		errors = append(errors, fmt.Sprintf("OAuth: %v", err))
	}

	// Check Stripe credentials if subscriptions are enabled
	if IsSubscriptionEnabled() {
		if err := validateStripeConfig(strict); err != nil {
			errors = append(errors, fmt.Sprintf("Stripe: %v", err))
		}
	}

	// Check other required variables
	if err := validateOtherConfig(strict); err != nil {
		errors = append(errors, fmt.Sprintf("Config: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n- %s", strings.Join(errors, "\n- "))
	}

	return nil
}

func validateOAuthConfig(strict bool) error {
	ctx := context.Background()
	clientID, clientSecret, err := secrets.GetOAuthCredentials(ctx)
	if err != nil && strict {
		return fmt.Errorf("failed to get OAuth credentials: %w", err)
	}

	if clientID == "" && strict {
		return fmt.Errorf("GOOGLE_CLIENT_ID is empty")
	}

	if clientSecret == "" && strict {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET is empty")
	}

	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	if redirectURL == "" {
		return fmt.Errorf("GOOGLE_REDIRECT_URL is not set")
	}

	return nil
}

func validateStripeConfig(strict bool) error {
	ctx := context.Background()
	secretKey, publishableKey, webhookSecret, priceID, err := secrets.GetStripeCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Stripe credentials: %w", err)
	}

	if secretKey == "" {
		return fmt.Errorf("STRIPE_SECRET_KEY is empty")
	}

	if publishableKey == "" {
		return fmt.Errorf("STRIPE_PUBLISHABLE_KEY is empty")
	}

	if webhookSecret == "" {
		return fmt.Errorf("STRIPE_WEBHOOK_SECRET is empty")
	}

	if priceID == "" {
		return fmt.Errorf("STRIPE_PRICE_ID is empty")
	}

	// Validate key formats
	if !strings.HasPrefix(secretKey, "sk_") {
		return fmt.Errorf("STRIPE_SECRET_KEY does not start with 'sk_'")
	}

	if !strings.HasPrefix(publishableKey, "pk_") {
		return fmt.Errorf("STRIPE_PUBLISHABLE_KEY does not start with 'pk_'")
	}

	if !strings.HasPrefix(webhookSecret, "whsec_") {
		return fmt.Errorf("STRIPE_WEBHOOK_SECRET does not start with 'whsec_'")
	}

	if !strings.HasPrefix(priceID, "price_") {
		return fmt.Errorf("STRIPE_PRICE_ID does not start with 'price_'")
	}

	return nil
}

func validateOtherConfig(strict bool) error {
	// Check GOOGLE_CLOUD_PROJECT for GAE (only in strict mode - not needed for local dev)
	if strict && os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
		return fmt.Errorf("GOOGLE_CLOUD_PROJECT is not set (required for GAE)")
	}

	// Check admin emails if configured
	adminEmails := os.Getenv("INITIAL_ADMIN_EMAILS")
	if adminEmails != "" {
		emails := parseEmailList(adminEmails)
		if len(emails) == 0 {
			return fmt.Errorf("INITIAL_ADMIN_EMAILS is set but contains no valid emails")
		}
	}

	return nil
}

// WarnAboutUnhandledEnvVars checks for potentially unhandled environment variables
func WarnAboutUnhandledEnvVars() {
	// Define all environment variables we expect to handle
	handledVars := map[string]bool{
		"GIN_MODE":                  true,
		"PORT":                      true,
		"GOOGLE_CLIENT_ID":          true,
		"GOOGLE_CLIENT_SECRET":      true,
		"GOOGLE_REDIRECT_URL":       true,
		"GOOGLE_CLOUD_PROJECT":      true,
		"SECRET_CLIENT_ID_NAME":     true,
		"SECRET_CLIENT_SECRET_NAME": true,
		"SUBSCRIPTION_ENABLED":      true,
		"INITIAL_ADMIN_EMAILS":      true,
		"ADMIN_TOKEN":               true,
		"STRIPE_SECRET_KEY":         true,
		"STRIPE_PUBLISHABLE_KEY":    true,
		"STRIPE_WEBHOOK_SECRET":     true,
		"STRIPE_PRICE_ID":           true,
		"SESSION_SECRET":            true,
	}

	// Check all environment variables
	for _, env := range os.Environ() {
		if strings.Contains(env, "=") {
			key := strings.Split(env, "=")[0]

			// Skip system variables
			if strings.HasPrefix(key, "PATH") ||
				strings.HasPrefix(key, "HOME") ||
				strings.HasPrefix(key, "USER") ||
				strings.HasPrefix(key, "SHELL") ||
				strings.HasPrefix(key, "TERM") ||
				strings.HasPrefix(key, "GOPATH") ||
				strings.HasPrefix(key, "GOROOT") ||
				strings.HasPrefix(key, "GCLOUD_") ||
				strings.HasPrefix(key, "GOOGLE_APPLICATION_CREDENTIALS") ||
				strings.HasPrefix(key, "GAE_") ||
				key == "PWD" || key == "OLDPWD" {
				continue
			}

			if !handledVars[key] {
				fmt.Printf("WARNING: Environment variable '%s' may not be handled by the application\n", key)
			}
		}
	}
}
