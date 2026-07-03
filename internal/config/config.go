package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jeffreyp/goread2/internal/secrets"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

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

	// Admin initialization
	InitialAdminEmails []string

	// Server
	Port string

	// Feed Rate Limiting
	RateLimitRequestsPerMinute int           // Requests per minute per domain
	RateLimitBurstSize         int           // Burst allowance per domain
	SchedulerUpdateWindow      time.Duration // Time window to spread updates across
	SchedulerMinInterval       time.Duration // Minimum time between updates for same feed
	SchedulerMaxConcurrent     int           // Maximum concurrent feed updates
	SchedulerCleanupInterval   time.Duration // How often to cleanup old rate limiters
}

var globalConfig *Config

// ResetForTesting resets the global config - used only in tests
func ResetForTesting() {
	globalConfig = nil
	secrets.ResetCacheForTesting()
}

// Load loads configuration from environment variables
func Load() *Config {
	if globalConfig != nil {
		return globalConfig
	}

	adminEmailsRaw, err := secrets.GetInitialAdminEmails(context.Background())
	if err != nil {
		log.Printf("WARNING: failed to load INITIAL_ADMIN_EMAILS from Secret Manager: %v", err)
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

		// Admin initialization
		InitialAdminEmails: parseEmailList(adminEmailsRaw),

		// Server
		Port: getEnvOrDefault("PORT", "8080"),

		// Feed Rate Limiting
		RateLimitRequestsPerMinute: parseInt(os.Getenv("RATE_LIMIT_REQUESTS_PER_MINUTE"), 120),
		RateLimitBurstSize:         parseInt(os.Getenv("RATE_LIMIT_BURST_SIZE"), 30),
		SchedulerUpdateWindow:      parseDuration(os.Getenv("SCHEDULER_UPDATE_WINDOW"), 15*time.Minute),
		SchedulerMinInterval:       parseDuration(os.Getenv("SCHEDULER_MIN_INTERVAL"), 5*time.Minute),
		SchedulerMaxConcurrent:     parseInt(os.Getenv("SCHEDULER_MAX_CONCURRENT"), 10),
		SchedulerCleanupInterval:   parseDuration(os.Getenv("SCHEDULER_CLEANUP_INTERVAL"), 1*time.Hour),
	}

	if err := validateConfig(globalConfig); err != nil {
		log.Printf("WARNING: configuration validation failed: %v", err)
	}

	return globalConfig
}

// validateConfig checks that configuration values are within acceptable ranges.
func validateConfig(cfg *Config) error {
	if cfg.RateLimitRequestsPerMinute <= 0 {
		return fmt.Errorf("RATE_LIMIT_REQUESTS_PER_MINUTE must be positive, got %d", cfg.RateLimitRequestsPerMinute)
	}
	if cfg.RateLimitBurstSize <= 0 {
		return fmt.Errorf("RATE_LIMIT_BURST_SIZE must be positive, got %d", cfg.RateLimitBurstSize)
	}
	if cfg.RateLimitRequestsPerMinute > 10000 {
		log.Printf("WARNING: very high RATE_LIMIT_REQUESTS_PER_MINUTE: %d", cfg.RateLimitRequestsPerMinute)
	}
	if cfg.SchedulerUpdateWindow <= 0 {
		return fmt.Errorf("SCHEDULER_UPDATE_WINDOW must be positive, got %v", cfg.SchedulerUpdateWindow)
	}
	if cfg.SchedulerMinInterval <= 0 {
		return fmt.Errorf("SCHEDULER_MIN_INTERVAL must be positive, got %v", cfg.SchedulerMinInterval)
	}
	if cfg.SchedulerMaxConcurrent <= 0 {
		return fmt.Errorf("SCHEDULER_MAX_CONCURRENT must be positive, got %d", cfg.SchedulerMaxConcurrent)
	}
	if cfg.SchedulerCleanupInterval <= 0 {
		return fmt.Errorf("SCHEDULER_CLEANUP_INTERVAL must be positive, got %v", cfg.SchedulerCleanupInterval)
	}
	if cfg.SchedulerMinInterval > cfg.SchedulerUpdateWindow {
		return fmt.Errorf("SCHEDULER_MIN_INTERVAL (%v) must be less than SCHEDULER_UPDATE_WINDOW (%v)",
			cfg.SchedulerMinInterval, cfg.SchedulerUpdateWindow)
	}
	if cfg.SchedulerMinInterval < time.Minute {
		log.Printf("WARNING: very short SCHEDULER_MIN_INTERVAL: %v", cfg.SchedulerMinInterval)
	}
	return nil
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

// parseEmailList parses a comma-separated list of emails
func parseEmailList(value string) []string {
	if value == "" {
		return nil
	}

	emails := strings.Split(value, ",")
	var result []string

	for _, email := range emails {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		if !emailRegex.MatchString(email) {
			log.Printf("WARNING: invalid email format in INITIAL_ADMIN_EMAILS, skipping: %s", email)
			continue
		}
		result = append(result, email)
	}

	return result
}

// parseInt parses an integer from string with a default value
func parseInt(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}

	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	log.Printf("WARNING: Failed to parse integer config value %q, using default %d", value, defaultValue)
	return defaultValue
}

// parseDuration parses a duration from string with a default value
func parseDuration(value string, defaultValue time.Duration) time.Duration {
	if value == "" {
		return defaultValue
	}

	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed
	}
	log.Printf("WARNING: Failed to parse duration config value %q, using default %s", value, defaultValue)
	return defaultValue
}
