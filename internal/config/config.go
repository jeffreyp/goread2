package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration
type Config struct {
	// Feature flags
	SubscriptionEnabled bool
	
	// Database
	DatabasePath string
	
	// OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	
	// Stripe (only used if subscription is enabled)
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string
	
	// Server
	Port string
}

var globalConfig *Config

// ResetForTesting resets the global config - used only in tests
func ResetForTesting() {
	globalConfig = nil
}

// Load loads configuration from environment variables
func Load() *Config {
	if globalConfig != nil {
		return globalConfig
	}
	
	globalConfig = &Config{
		// Feature flags - default to disabled for safety
		SubscriptionEnabled: parseBool(os.Getenv("SUBSCRIPTION_ENABLED"), false),
		
		// Database
		DatabasePath: getEnvOrDefault("DATABASE_PATH", "./goread2.db"),
		
		// OAuth
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		
		// Stripe
		StripeSecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
		StripePublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		StripeWebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),
		
		// Server
		Port: getEnvOrDefault("PORT", "8080"),
	}
	
	return globalConfig
}

// Get returns the current configuration
func Get() *Config {
	if globalConfig == nil {
		return Load()
	}
	return globalConfig
}

// IsSubscriptionEnabled returns true if subscription features are enabled
func IsSubscriptionEnabled() bool {
	return Get().SubscriptionEnabled
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseBool parses a boolean from string with a default value
func parseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	
	// Handle common boolean representations
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "true", "1", "yes", "on", "enabled":
		return true
	case "false", "0", "no", "off", "disabled":
		return false
	default:
		// Try standard parsing
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
		return defaultValue
	}
}