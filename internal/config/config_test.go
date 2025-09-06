package config

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	tests := []struct {
		name                        string
		envVars                     map[string]string
		expectedSubscriptionEnabled bool
		expectedPort                string
		expectedDatabasePath        string
	}{
		{
			name:                        "default values when no env vars set",
			envVars:                     map[string]string{},
			expectedSubscriptionEnabled: false,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "subscription enabled with true",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "true",
				"PORT":                 "3000",
				"DATABASE_PATH":        "/tmp/test.db",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:                "3000",
			expectedDatabasePath:        "/tmp/test.db",
		},
		{
			name: "subscription enabled with 1",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "1",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "subscription enabled with yes",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "yes",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "subscription enabled with on",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "on",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "subscription enabled with enabled",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "enabled",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "subscription disabled with false",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "false",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "subscription disabled with 0",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "0",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "subscription disabled with no",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "no",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "subscription disabled with off",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "off",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "subscription disabled with disabled",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "disabled",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "subscription disabled with invalid value",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "invalid",
			},
			expectedSubscriptionEnabled: false,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "case insensitive values",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "TRUE",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
		},
		{
			name: "whitespace handling",
			envVars: map[string]string{
				"SUBSCRIPTION_ENABLED": "  true  ",
			},
			expectedSubscriptionEnabled: true,
			expectedPort:                "8080",
			expectedDatabasePath:        "./goread2.db",
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

func TestParseEmailList(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
		desc     string
	}{
		{"", nil, "empty string"},
		{"test@example.com", []string{"test@example.com"}, "single email"},
		{"test1@example.com,test2@example.com", []string{"test1@example.com", "test2@example.com"}, "multiple emails"},
		{"test1@example.com, test2@example.com", []string{"test1@example.com", "test2@example.com"}, "multiple emails with spaces"},
		{"test1@example.com,  , test2@example.com", []string{"test1@example.com", "test2@example.com"}, "empty emails filtered out"},
		{"  test@example.com  ", []string{"test@example.com"}, "whitespace trimmed"},
		{"test1@example.com,,test2@example.com", []string{"test1@example.com", "test2@example.com"}, "consecutive commas"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			clearConfigEnvVars()
			if tt.input != "" {
				_ = os.Setenv("INITIAL_ADMIN_EMAILS", tt.input)
			}
			defer func() {
				_ = os.Unsetenv("INITIAL_ADMIN_EMAILS")
				clearConfigEnvVars()
			}()

			ResetForTesting()
			Load()
			cfg := Get()

			if len(cfg.InitialAdminEmails) != len(tt.expected) {
				t.Errorf("parseEmailList(%q) length = %v, want %v", tt.input, len(cfg.InitialAdminEmails), len(tt.expected))
				return
			}

			for i, email := range cfg.InitialAdminEmails {
				if email != tt.expected[i] {
					t.Errorf("parseEmailList(%q)[%d] = %v, want %v", tt.input, i, email, tt.expected[i])
				}
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		envKey     string
		envValue   string
		defaultVal string
		expected   string
		desc       string
	}{
		{"TEST_VAR", "custom_value", "default", "custom_value", "env var set"},
		{"UNSET_VAR", "", "default", "default", "env var not set"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.envValue != "" {
				_ = os.Setenv(tt.envKey, tt.envValue)
				defer func() {
					_ = os.Unsetenv(tt.envKey)
				}()
			}

			clearConfigEnvVars()
			ResetForTesting()

			// Test via config loading
			if tt.envKey == "PORT" {
				Load()
				cfg := Get()
				if cfg.Port != tt.expected {
					t.Errorf("Port = %v, want %v", cfg.Port, tt.expected)
				}
			}
		})
	}
}

func TestLoadSingleton(t *testing.T) {
	clearConfigEnvVars()
	ResetForTesting()

	// First call should create config
	cfg1 := Load()
	if cfg1 == nil {
		t.Error("Load() returned nil")
	}

	// Second call should return same instance
	cfg2 := Load()
	if cfg1 != cfg2 {
		t.Error("Load() should return singleton instance")
	}

	// Get() should also return same instance
	cfg3 := Get()
	if cfg1 != cfg3 {
		t.Error("Get() should return same instance as Load()")
	}
}

func TestGetWithoutLoad(t *testing.T) {
	clearConfigEnvVars()
	ResetForTesting()

	// Get() should call Load() if config not initialized
	cfg := Get()
	if cfg == nil {
		t.Fatal("Get() returned nil")
	}

	// Should have default values
	if cfg.Port != "8080" {
		t.Errorf("Default port = %v, want 8080", cfg.Port)
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
		"INITIAL_ADMIN_EMAILS",
	}

	for _, envVar := range envVars {
		_ = os.Unsetenv(envVar)
	}

	ResetForTesting()
}
