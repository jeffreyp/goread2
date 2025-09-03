package config

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	tests := []struct {
		name                    string
		envVars                 map[string]string
		expectedSubscriptionEnabled bool
		expectedPort            string
		expectedDatabasePath    string
	}{
		{
			name:                    "default values when no env vars set",
			envVars:                 map[string]string{},
			expectedSubscriptionEnabled: false,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "subscription enabled with true",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "true",
				"PORT":                 "3000",
				"DATABASE_PATH":        "/tmp/test.db",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:            "3000",
			expectedDatabasePath:    "/tmp/test.db",
		},
		{
			name: "subscription enabled with 1",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "1",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "subscription enabled with yes",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "yes",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "subscription enabled with on",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "on",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "subscription enabled with enabled",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "enabled",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "subscription disabled with false",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "false",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "subscription disabled with 0",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "0",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "subscription disabled with no",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "no",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "subscription disabled with off",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "off",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "subscription disabled with disabled",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "disabled",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "subscription disabled with invalid value",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "invalid",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "case insensitive values",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "TRUE",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
		{
			name: "whitespace handling",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "  true  ",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:            "8080",
			expectedDatabasePath:    "./goread2.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnvVars()
			
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					_ = os.Unsetenv(key)
				}
				clearConfigEnvVars()
			}()

			ResetForTesting()
			Load()
			cfg := Get()

			if cfg.SubscriptionEnabled != tt.expectedSubscriptionEnabled {
				t.Errorf("SubscriptionEnabled = %v, want %v", cfg.SubscriptionEnabled, tt.expectedSubscriptionEnabled)
			}

			if cfg.Port != tt.expectedPort {
				t.Errorf("Port = %v, want %v", cfg.Port, tt.expectedPort)
			}

			if cfg.DatabasePath != tt.expectedDatabasePath {
				t.Errorf("DatabasePath = %v, want %v", cfg.DatabasePath, tt.expectedDatabasePath)
			}

			if IsSubscriptionEnabled() != tt.expectedSubscriptionEnabled {
				t.Errorf("IsSubscriptionEnabled() = %v, want %v", IsSubscriptionEnabled(), tt.expectedSubscriptionEnabled)
			}
		})
	}
}

func TestConfigParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
		desc     string
	}{
		{"true", true, "standard true"},
		{"false", false, "standard false"},
		{"1", true, "numeric true"},
		{"0", false, "numeric false"},
		{"yes", true, "yes variant"},
		{"no", false, "no variant"},
		{"on", true, "on variant"},
		{"off", false, "off variant"},
		{"enabled", true, "enabled variant"},
		{"disabled", false, "disabled variant"},
		{"TRUE", true, "uppercase true"},
		{"FALSE", false, "uppercase false"},
		{"YES", true, "uppercase yes"},
		{"NO", false, "uppercase no"},
		{"  true  ", true, "true with whitespace"},
		{"  false  ", false, "false with whitespace"},
		{"invalid", false, "invalid value defaults to false"},
		{"", false, "empty string defaults to false"},
		{"random", false, "random string defaults to false"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			clearConfigEnvVars()
			if tt.input != "" {
				_ = os.Setenv("SUBSCRIPTION_ENABLED", tt.input)
			}
			defer func() {
				_ = os.Unsetenv("SUBSCRIPTION_ENABLED")
				clearConfigEnvVars()
			}()

			ResetForTesting()
			Load()
			result := IsSubscriptionEnabled()

			if result != tt.expected {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConfigEnvVarPrecedence(t *testing.T) {
	clearConfigEnvVars()
	
	_ = os.Setenv("SUBSCRIPTION_ENABLED", "true")
	_ = os.Setenv("PORT", "9000")
	_ = os.Setenv("DATABASE_PATH", "/custom/path.db")
	_ = os.Setenv("GOOGLE_CLIENT_ID", "test_client_id")
	_ = os.Setenv("GOOGLE_CLIENT_SECRET", "test_secret")
	_ = os.Setenv("GOOGLE_REDIRECT_URL", "http://test.com/callback")
	_ = os.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	_ = os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_123")
	_ = os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_123")

	defer func() {
		_ = os.Unsetenv("SUBSCRIPTION_ENABLED")
		_ = os.Unsetenv("PORT")
		_ = os.Unsetenv("DATABASE_PATH")
		_ = os.Unsetenv("GOOGLE_CLIENT_ID")
		_ = os.Unsetenv("GOOGLE_CLIENT_SECRET")
		_ = os.Unsetenv("GOOGLE_REDIRECT_URL")
		_ = os.Unsetenv("STRIPE_SECRET_KEY")
		_ = os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
		_ = os.Unsetenv("STRIPE_WEBHOOK_SECRET")
		clearConfigEnvVars()
	}()

	ResetForTesting()
	Load()
	cfg := Get()

	if !cfg.SubscriptionEnabled {
		t.Error("Expected SubscriptionEnabled to be true")
	}

	if cfg.Port != "9000" {
		t.Errorf("Port = %v, want 9000", cfg.Port)
	}

	if cfg.DatabasePath != "/custom/path.db" {
		t.Errorf("DatabasePath = %v, want /custom/path.db", cfg.DatabasePath)
	}

	if cfg.GoogleClientID != "test_client_id" {
		t.Errorf("GoogleClientID = %v, want test_client_id", cfg.GoogleClientID)
	}

	if cfg.StripeSecretKey != "sk_test_123" {
		t.Errorf("StripeSecretKey = %v, want sk_test_123", cfg.StripeSecretKey)
	}
}

func clearConfigEnvVars() {
	envVars := []string{
		"SUBSCRIPTION_ENABLED",
		"PORT",
		"DATABASE_PATH",
		"GOOGLE_CLIENT_ID",
		"GOOGLE_CLIENT_SECRET", 
		"GOOGLE_REDIRECT_URL",
		"STRIPE_SECRET_KEY",
		"STRIPE_PUBLISHABLE_KEY",
		"STRIPE_WEBHOOK_SECRET",
	}
	
	for _, envVar := range envVars {
		_ = os.Unsetenv(envVar)
	}
	
	ResetForTesting()
}