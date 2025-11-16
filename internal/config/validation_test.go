package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/jeffreyp/goread2/internal/secrets"
)

// Helper function to clear relevant environment variables
func clearValidationEnv() {
	_ = os.Unsetenv("GOOGLE_CLIENT_ID")
	_ = os.Unsetenv("GOOGLE_CLIENT_SECRET")
	_ = os.Unsetenv("GOOGLE_REDIRECT_URL")
	_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")
	_ = os.Unsetenv("SUBSCRIPTION_ENABLED")
	_ = os.Unsetenv("STRIPE_SECRET_KEY")
	_ = os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
	_ = os.Unsetenv("STRIPE_WEBHOOK_SECRET")
	_ = os.Unsetenv("STRIPE_PRICE_ID")
	_ = os.Unsetenv("INITIAL_ADMIN_EMAILS")
	_ = os.Unsetenv("SECRET_CLIENT_ID_NAME")
	_ = os.Unsetenv("SECRET_CLIENT_SECRET_NAME")
	_ = os.Unsetenv("SECRET_STRIPE_SECRET_KEY_NAME")
	_ = os.Unsetenv("SECRET_STRIPE_PUBLISHABLE_KEY_NAME")
	_ = os.Unsetenv("SECRET_STRIPE_WEBHOOK_SECRET_NAME")
	_ = os.Unsetenv("SECRET_STRIPE_PRICE_ID_NAME")
}

func TestValidateOAuthConfig(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		strict        bool
		expectError   bool
		errorContains string
	}{
		{
			name: "valid_config_passes_strict",
			envVars: map[string]string{
				"GOOGLE_CLIENT_ID":     "test-client-id",
				"GOOGLE_CLIENT_SECRET": "test-client-secret",
				"GOOGLE_REDIRECT_URL":  "http://localhost:8080/auth/callback",
			},
			strict:      true,
			expectError: false,
		},
		{
			name: "valid_config_passes_non_strict",
			envVars: map[string]string{
				"GOOGLE_CLIENT_ID":     "test-client-id",
				"GOOGLE_CLIENT_SECRET": "test-client-secret",
				"GOOGLE_REDIRECT_URL":  "http://localhost:8080/auth/callback",
			},
			strict:      false,
			expectError: false,
		},
		{
			name: "missing_client_id_fails_strict",
			envVars: map[string]string{
				"GOOGLE_CLIENT_SECRET": "test-client-secret",
				"GOOGLE_REDIRECT_URL":  "http://localhost:8080/auth/callback",
			},
			strict:        true,
			expectError:   true,
			errorContains: "failed to get OAuth credentials",
		},
		{
			name: "missing_client_id_passes_non_strict",
			envVars: map[string]string{
				"GOOGLE_CLIENT_SECRET": "test-client-secret",
				"GOOGLE_REDIRECT_URL":  "http://localhost:8080/auth/callback",
			},
			strict:      false,
			expectError: false,
		},
		{
			name: "missing_client_secret_fails_strict",
			envVars: map[string]string{
				"GOOGLE_CLIENT_ID":    "test-client-id",
				"GOOGLE_REDIRECT_URL": "http://localhost:8080/auth/callback",
			},
			strict:        true,
			expectError:   true,
			errorContains: "failed to get OAuth credentials",
		},
		{
			name: "missing_client_secret_passes_non_strict",
			envVars: map[string]string{
				"GOOGLE_CLIENT_ID":    "test-client-id",
				"GOOGLE_REDIRECT_URL": "http://localhost:8080/auth/callback",
			},
			strict:      false,
			expectError: false,
		},
		{
			name: "missing_redirect_url_always_fails",
			envVars: map[string]string{
				"GOOGLE_CLIENT_ID":     "test-client-id",
				"GOOGLE_CLIENT_SECRET": "test-client-secret",
			},
			strict:        true,
			expectError:   true,
			errorContains: "GOOGLE_REDIRECT_URL",
		},
		{
			name: "missing_redirect_url_fails_non_strict_too",
			envVars: map[string]string{
				"GOOGLE_CLIENT_ID":     "test-client-id",
				"GOOGLE_CLIENT_SECRET": "test-client-secret",
			},
			strict:        false,
			expectError:   true,
			errorContains: "GOOGLE_REDIRECT_URL",
		},
		{
			name: "empty_values_fail_strict",
			envVars: map[string]string{
				"GOOGLE_CLIENT_ID":     "",
				"GOOGLE_CLIENT_SECRET": "",
				"GOOGLE_REDIRECT_URL":  "http://localhost:8080/auth/callback",
			},
			strict:        true,
			expectError:   true,
			errorContains: "failed to get OAuth credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		// Reset secret cache before test
		secrets.ResetCacheForTesting()

			clearValidationEnv()
			defer clearValidationEnv()

			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			err := validateOAuthConfig(tt.strict)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorContains)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateStripeConfig(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		strict        bool
		expectError   bool
		errorContains string
	}{
		{
			name: "valid_stripe_config_passes",
			envVars: map[string]string{
				"STRIPE_SECRET_KEY":      "sk_test_1234567890abcdef",
				"STRIPE_PUBLISHABLE_KEY": "pk_test_1234567890abcdef",
				"STRIPE_WEBHOOK_SECRET":  "whsec_1234567890abcdef",
				"STRIPE_PRICE_ID":        "price_1234567890abcdef",
			},
			strict:      true,
			expectError: false,
		},
		{
			name: "valid_live_keys_pass",
			envVars: map[string]string{
				"STRIPE_SECRET_KEY":      "sk_live_1234567890abcdef",
				"STRIPE_PUBLISHABLE_KEY": "pk_live_1234567890abcdef",
				"STRIPE_WEBHOOK_SECRET":  "whsec_1234567890abcdef",
				"STRIPE_PRICE_ID":        "price_1234567890abcdef",
			},
			strict:      true,
			expectError: false,
		},
		{
			name: "missing_secret_key_fails",
			envVars: map[string]string{
				"STRIPE_PUBLISHABLE_KEY": "pk_test_1234567890abcdef",
				"STRIPE_WEBHOOK_SECRET":  "whsec_1234567890abcdef",
				"STRIPE_PRICE_ID":        "price_1234567890abcdef",
			},
			strict:        true,
			expectError:   true,
			errorContains: "failed to get Stripe credentials",
		},
		{
			name: "missing_publishable_key_fails",
			envVars: map[string]string{
				"STRIPE_SECRET_KEY":     "sk_test_1234567890abcdef",
				"STRIPE_WEBHOOK_SECRET": "whsec_1234567890abcdef",
				"STRIPE_PRICE_ID":       "price_1234567890abcdef",
			},
			strict:        true,
			expectError:   true,
			errorContains: "failed to get Stripe credentials",
		},
		{
			name: "missing_webhook_secret_fails",
			envVars: map[string]string{
				"STRIPE_SECRET_KEY":      "sk_test_1234567890abcdef",
				"STRIPE_PUBLISHABLE_KEY": "pk_test_1234567890abcdef",
				"STRIPE_PRICE_ID":        "price_1234567890abcdef",
			},
			strict:        true,
			expectError:   true,
			errorContains: "failed to get Stripe credentials",
		},
		{
			name: "missing_price_id_fails",
			envVars: map[string]string{
				"STRIPE_SECRET_KEY":      "sk_test_1234567890abcdef",
				"STRIPE_PUBLISHABLE_KEY": "pk_test_1234567890abcdef",
				"STRIPE_WEBHOOK_SECRET":  "whsec_1234567890abcdef",
			},
			strict:        true,
			expectError:   true,
			errorContains: "failed to get Stripe credentials",
		},
		{
			name: "invalid_secret_key_prefix_fails",
			envVars: map[string]string{
				"STRIPE_SECRET_KEY":      "invalid_1234567890abcdef",
				"STRIPE_PUBLISHABLE_KEY": "pk_test_1234567890abcdef",
				"STRIPE_WEBHOOK_SECRET":  "whsec_1234567890abcdef",
				"STRIPE_PRICE_ID":        "price_1234567890abcdef",
			},
			strict:        true,
			expectError:   true,
			errorContains: "does not start with 'sk_'",
		},
		{
			name: "invalid_publishable_key_prefix_fails",
			envVars: map[string]string{
				"STRIPE_SECRET_KEY":      "sk_test_1234567890abcdef",
				"STRIPE_PUBLISHABLE_KEY": "invalid_1234567890abcdef",
				"STRIPE_WEBHOOK_SECRET":  "whsec_1234567890abcdef",
				"STRIPE_PRICE_ID":        "price_1234567890abcdef",
			},
			strict:        true,
			expectError:   true,
			errorContains: "does not start with 'pk_'",
		},
		{
			name: "invalid_webhook_secret_prefix_fails",
			envVars: map[string]string{
				"STRIPE_SECRET_KEY":      "sk_test_1234567890abcdef",
				"STRIPE_PUBLISHABLE_KEY": "pk_test_1234567890abcdef",
				"STRIPE_WEBHOOK_SECRET":  "invalid_1234567890abcdef",
				"STRIPE_PRICE_ID":        "price_1234567890abcdef",
			},
			strict:        true,
			expectError:   true,
			errorContains: "does not start with 'whsec_'",
		},
		{
			name: "invalid_price_id_prefix_fails",
			envVars: map[string]string{
				"STRIPE_SECRET_KEY":      "sk_test_1234567890abcdef",
				"STRIPE_PUBLISHABLE_KEY": "pk_test_1234567890abcdef",
				"STRIPE_WEBHOOK_SECRET":  "whsec_1234567890abcdef",
				"STRIPE_PRICE_ID":        "invalid_1234567890abcdef",
			},
			strict:        true,
			expectError:   true,
			errorContains: "does not start with 'price_'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		// Reset secret cache before test
		secrets.ResetCacheForTesting()

			clearValidationEnv()
			defer clearValidationEnv()

			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			err := validateStripeConfig(tt.strict)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorContains)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateOtherConfig(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		strict        bool
		expectError   bool
		errorContains string
	}{
		{
			name: "valid_config_non_strict",
			envVars: map[string]string{
				"INITIAL_ADMIN_EMAILS": "admin@example.com,user@example.com",
			},
			strict:      false,
			expectError: false,
		},
		{
			name: "valid_config_strict_with_gcp_project",
			envVars: map[string]string{
				"GOOGLE_CLOUD_PROJECT": "my-project-123",
				"INITIAL_ADMIN_EMAILS": "admin@example.com",
			},
			strict:      true,
			expectError: false,
		},
		{
			name:          "missing_gcp_project_fails_strict",
			envVars:       map[string]string{},
			strict:        true,
			expectError:   true,
			errorContains: "GOOGLE_CLOUD_PROJECT",
		},
		{
			name:        "missing_gcp_project_passes_non_strict",
			envVars:     map[string]string{},
			strict:      false,
			expectError: false,
		},
		{
			name: "invalid_admin_emails_fails",
			envVars: map[string]string{
				"INITIAL_ADMIN_EMAILS": ",,,",
			},
			strict:        false,
			expectError:   true,
			errorContains: "contains no valid emails",
		},
		{
			name: "empty_admin_emails_passes",
			envVars: map[string]string{
				"INITIAL_ADMIN_EMAILS": "",
			},
			strict:      false,
			expectError: false,
		},
		{
			name: "single_admin_email_passes",
			envVars: map[string]string{
				"INITIAL_ADMIN_EMAILS": "admin@example.com",
			},
			strict:      false,
			expectError: false,
		},
		{
			name: "multiple_admin_emails_pass",
			envVars: map[string]string{
				"INITIAL_ADMIN_EMAILS": "admin@example.com,user@example.com,test@example.com",
			},
			strict:      false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		// Reset secret cache before test
		secrets.ResetCacheForTesting()

			clearValidationEnv()
			defer clearValidationEnv()

			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			err := validateOtherConfig(tt.strict)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorContains)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateEnvironmentConfigStrict(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		strict        bool
		expectError   bool
		errorContains string
	}{
		{
			name: "complete_valid_config_strict",
			envVars: map[string]string{
				"GOOGLE_CLIENT_ID":     "test-client-id",
				"GOOGLE_CLIENT_SECRET": "test-client-secret",
				"GOOGLE_REDIRECT_URL":  "http://localhost:8080/auth/callback",
				"GOOGLE_CLOUD_PROJECT": "my-project-123",
				"SUBSCRIPTION_ENABLED": "false",
				"INITIAL_ADMIN_EMAILS": "admin@example.com",
			},
			strict:      true,
			expectError: false,
		},
		{
			name: "complete_valid_config_with_stripe",
			envVars: map[string]string{
				"GOOGLE_CLIENT_ID":       "test-client-id",
				"GOOGLE_CLIENT_SECRET":   "test-client-secret",
				"GOOGLE_REDIRECT_URL":    "http://localhost:8080/auth/callback",
				"GOOGLE_CLOUD_PROJECT":   "my-project-123",
				"SUBSCRIPTION_ENABLED":   "true",
				"STRIPE_SECRET_KEY":      "sk_test_1234567890abcdef",
				"STRIPE_PUBLISHABLE_KEY": "pk_test_1234567890abcdef",
				"STRIPE_WEBHOOK_SECRET":  "whsec_1234567890abcdef",
				"STRIPE_PRICE_ID":        "price_1234567890abcdef",
			},
			strict:      true,
			expectError: false,
		},
		{
			name: "missing_oauth_credentials_fails",
			envVars: map[string]string{
				"GOOGLE_REDIRECT_URL":  "http://localhost:8080/auth/callback",
				"GOOGLE_CLOUD_PROJECT": "my-project-123",
			},
			strict:        true,
			expectError:   true,
			errorContains: "OAuth",
		},
		{
			name: "missing_stripe_when_enabled_fails",
			envVars: map[string]string{
				"GOOGLE_CLIENT_ID":     "test-client-id",
				"GOOGLE_CLIENT_SECRET": "test-client-secret",
				"GOOGLE_REDIRECT_URL":  "http://localhost:8080/auth/callback",
				"GOOGLE_CLOUD_PROJECT": "my-project-123",
				"SUBSCRIPTION_ENABLED": "true",
			},
			strict:        true,
			expectError:   true,
			errorContains: "failed to get Stripe credentials",
		},
		{
			name: "non_strict_mode_allows_missing_credentials",
			envVars: map[string]string{
				"GOOGLE_REDIRECT_URL": "http://localhost:8080/auth/callback",
			},
			strict:      false,
			expectError: false,
		},
		{
			name: "multiple_validation_errors_combined",
			envVars: map[string]string{
				"GOOGLE_CLOUD_PROJECT": "my-project-123",
			},
			strict:        true,
			expectError:   true,
			errorContains: "configuration validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		// Reset secret cache before test
		secrets.ResetCacheForTesting()

			clearValidationEnv()
			defer clearValidationEnv()

			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			// Reset config to pick up new environment variables
			ResetForTesting()

			err := ValidateEnvironmentConfigStrict(tt.strict)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorContains)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateEnvironmentConfig(t *testing.T) {
	t.Run("calls_strict_mode_false", func(t *testing.T) {
		// Reset secret cache before test
		secrets.ResetCacheForTesting()

		clearValidationEnv()
		defer clearValidationEnv()

		// Set only redirect URL (minimal requirement for non-strict)
		_ = os.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/callback")

		err := ValidateEnvironmentConfig()
		if err != nil {
			t.Errorf("Non-strict validation should pass with minimal config, got: %v", err)
		}
	})
}

func TestWarnAboutUnhandledEnvVars(t *testing.T) {
	t.Run("unknown_variable_produces_warning", func(t *testing.T) {
		// Reset secret cache before test
		secrets.ResetCacheForTesting()

		// Set a unique unknown variable
		testVar := "GOREAD_TEST_UNKNOWN_VAR_12345"
		_ = os.Setenv(testVar, "test-value")
		defer func() { _ = os.Unsetenv(testVar) }()

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Call function
		WarnAboutUnhandledEnvVars()

		// Restore stdout and read captured output
		_ = w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		// Check that our unknown variable generates a warning
		if !strings.Contains(output, testVar) {
			t.Errorf("Expected warning for '%s', got: %s", testVar, output)
		}
		if !strings.Contains(output, "WARNING") {
			t.Errorf("Expected WARNING in output, got: %s", output)
		}
	})

	t.Run("known_variables_no_warning", func(t *testing.T) {
		// Reset secret cache before test
		secrets.ResetCacheForTesting()

		// These variables should not generate warnings
		knownVars := map[string]string{
			"PORT":                 "8080",
			"GOOGLE_CLIENT_ID":     "test-id",
			"SUBSCRIPTION_ENABLED": "true",
			"ADMIN_TOKEN":          "test-token",
		}

		for k, v := range knownVars {
			_ = os.Setenv(k, v)
		}
		defer func() {
			for k := range knownVars {
				_ = os.Unsetenv(k)
			}
		}()

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Call function
		WarnAboutUnhandledEnvVars()

		// Restore stdout and read captured output
		_ = w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		// Check that our known variables don't generate warnings
		for k := range knownVars {
			// Look for the specific warning pattern for this variable
			warningPattern := fmt.Sprintf("WARNING: Environment variable '%s'", k)
			if strings.Contains(output, warningPattern) {
				t.Errorf("Known variable '%s' should not generate warning", k)
			}
		}
	})

	t.Run("function_runs_without_error", func(t *testing.T) {
		// Reset secret cache before test
		secrets.ResetCacheForTesting()

		// Just verify the function doesn't panic
		// It will warn about many environment variables in test environment
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("WarnAboutUnhandledEnvVars panicked: %v", r)
			}
		}()

		WarnAboutUnhandledEnvVars()
	})
}
