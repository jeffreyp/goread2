package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"goread2/internal/database"
)

func TestAdminSecurityBootstrap(t *testing.T) {
	// Clean up database before test
	cleanupDatabase(t)

	t.Run("BootstrapWithoutAdminUser", func(t *testing.T) {
		// Ensure database is completely clean for this test
		cleanupDatabase(t)
		
		// Try to create token without any admin users in database
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "create-token", "Unauthorized attempt")
		cmd.Dir = "../.."
		cmd.Env = append(os.Environ(), "ADMIN_TOKEN=bootstrap")

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error when no admin users exist")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "No admin users found in database") {
			t.Error("Expected security error about no admin users")
		}
		if !strings.Contains(outputStr, "SECURITY: Initial token creation requires an existing admin user") {
			t.Error("Expected security message about admin user requirement")
		}
	})

	t.Run("BootstrapWithAdminUser", func(t *testing.T) {
		// Create an admin user first
		setupMainTestUser(t, "bootstrap123", "bootstrap@test.com", "Bootstrap Admin")

		// Change to project root directory to ensure we use the same database
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		
		err = os.Chdir("../..")
		if err != nil {
			t.Fatalf("Failed to change to project root: %v", err)
		}
		defer func() {
			if err := os.Chdir(originalDir); err != nil {
				t.Logf("Failed to restore original directory: %v", err)
			}
		}()

		// Set user as admin
		db, err := database.InitDB()
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		sqliteDB := db.(*database.DB)
		_, err = sqliteDB.Exec("UPDATE users SET is_admin = 1 WHERE email = ?", "bootstrap@test.com")
		if err != nil {
			t.Fatalf("Failed to set user as admin: %v", err)
		}

		// Close the database connection to ensure changes are flushed
		err = db.Close()
		if err != nil {
			t.Fatalf("Failed to close database connection: %v", err)
		}

		// Now bootstrap should work (already in project root)
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "create-token", "Legitimate bootstrap")
		cmd.Env = append(os.Environ(), "ADMIN_TOKEN=bootstrap")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Bootstrap failed with admin user present: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Admin token created successfully") {
			t.Error("Expected success message for legitimate bootstrap")
		}
		if !strings.Contains(outputStr, "Found admin users in database") {
			t.Error("Expected confirmation that admin users were found")
		}
	})

	t.Run("InvalidTokenFormat", func(t *testing.T) {
		// Test with invalid token format
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "list-users")
		cmd.Dir = "../.."
		cmd.Env = append(os.Environ(), "ADMIN_TOKEN=short")

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for invalid token format")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "ADMIN_TOKEN must be exactly 64 characters") {
			t.Error("Expected error message about token format")
		}
	})

	t.Run("UnauthorizedTokenValidation", func(t *testing.T) {
		// Test with properly formatted but unauthorized token
		fakeToken := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

		cmd := exec.Command("go", "run", "cmd/admin/main.go", "list-users")
		cmd.Dir = "../.."
		cmd.Env = append(os.Environ(), "ADMIN_TOKEN="+fakeToken)

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for unauthorized token")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Invalid ADMIN_TOKEN - token not found in database") {
			t.Error("Expected error message about invalid token")
		}
	})

	t.Run("MissingTokenEnvironmentVariable", func(t *testing.T) {
		// Test without ADMIN_TOKEN environment variable
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "list-users")
		cmd.Dir = "../.."
		// Don't set ADMIN_TOKEN env var

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error when ADMIN_TOKEN is missing")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "ADMIN_TOKEN environment variable must be set") {
			t.Error("Expected error message about missing environment variable")
		}
		if !strings.Contains(outputStr, "Use the 'create-token' command if you are the first admin") {
			t.Error("Expected help message about bootstrap process")
		}
	})

	// Clean up after test
	cleanupDatabase(t)
}

func TestAdminTokenLifecycle(t *testing.T) {
	// Clean up database before test
	cleanupDatabase(t)

	// Set up admin user and initial token
	setupMainTestUser(t, "lifecycle123", "lifecycle@test.com", "Lifecycle Admin")

	// Change to project root directory to ensure we use the same database
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	
	err = os.Chdir("../..")
	if err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore original directory: %v", err)
		}
	}()

	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	sqliteDB := db.(*database.DB)
	_, err = sqliteDB.Exec("UPDATE users SET is_admin = 1 WHERE email = ?", "lifecycle@test.com")
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	// Close the database connection to ensure changes are flushed
	err = db.Close()
	if err != nil {
		t.Fatalf("Failed to close database connection: %v", err)
	}

	// Create initial admin token (already in project root)
	createCmd := exec.Command("go", "run", "cmd/admin/main.go", "create-token", "Lifecycle test token")
	createCmd.Env = append(os.Environ(), "ADMIN_TOKEN=bootstrap")

	createOutput, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create initial token: %v\nOutput: %s", err, createOutput)
	}

	// Extract token from output
	var adminToken string
	lines := strings.Split(string(createOutput), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Token: ") {
			adminToken = strings.TrimPrefix(line, "Token: ")
			break
		}
	}

	if adminToken == "" {
		t.Fatalf("Failed to extract token from output: %s", string(createOutput))
	}

	t.Run("TokenValidation", func(t *testing.T) {
		// Use token for a command (already in project root)
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "list-users")
		cmd.Env = append(os.Environ(), "ADMIN_TOKEN="+adminToken)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command with valid token failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "lifecycle@test.com") {
			t.Error("Expected to see admin user in output")
		}
	})

	t.Run("TokenRevocation", func(t *testing.T) {
		// List tokens to get ID (already in project root)
		listCmd := exec.Command("go", "run", "cmd/admin/main.go", "list-tokens")
		listCmd.Env = append(os.Environ(), "ADMIN_TOKEN="+adminToken)

		listOutput, err := listCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("List tokens failed: %v\nOutput: %s", err, listOutput)
		}

		// Extract token ID
		lines := strings.Split(string(listOutput), "\n")
		var tokenID string
		for _, line := range lines {
			if strings.Contains(line, "Lifecycle test token") && !strings.Contains(line, "ID") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					tokenID = fields[0]
					break
				}
			}
		}

		if tokenID == "" {
			t.Skip("Could not find token ID")
		}

		// Revoke the token (already in project root)
		revokeCmd := exec.Command("go", "run", "cmd/admin/main.go", "revoke-token", tokenID)
		revokeCmd.Env = append(os.Environ(), "ADMIN_TOKEN="+adminToken)

		revokeOutput, err := revokeCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Revoke token failed: %v\nOutput: %s", err, revokeOutput)
		}

		// Try to use revoked token (already in project root)
		testCmd := exec.Command("go", "run", "cmd/admin/main.go", "list-users")
		testCmd.Env = append(os.Environ(), "ADMIN_TOKEN="+adminToken)

		testOutput, err := testCmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error when using revoked token")
		}

		testOutputStr := string(testOutput)
		if !strings.Contains(testOutputStr, "Invalid ADMIN_TOKEN") {
			t.Error("Expected invalid token error message")
		}
	})

	// Clean up after test
	cleanupDatabase(t)
}

func TestAdminTokenSecurityWarnings(t *testing.T) {
	// Clean up database before test
	cleanupDatabase(t)

	// Set up admin user and create first token
	setupMainTestUser(t, "warning123", "warning@test.com", "Warning Admin")

	// Change to project root directory to ensure we use the same database
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	
	err = os.Chdir("../..")
	if err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore original directory: %v", err)
		}
	}()

	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	sqliteDB := db.(*database.DB)
	_, err = sqliteDB.Exec("UPDATE users SET is_admin = 1 WHERE email = ?", "warning@test.com")
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	// Close the database connection to ensure changes are flushed
	err = db.Close()
	if err != nil {
		t.Fatalf("Failed to close database connection: %v", err)
	}

	// Create first admin token (already in project root)
	createCmd := exec.Command("go", "run", "cmd/admin/main.go", "create-token", "First token")
	createCmd.Env = append(os.Environ(), "ADMIN_TOKEN=bootstrap")

	createOutput, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create first token: %v\nOutput: %s", err, createOutput)
	}

	// Extract first token
	var firstToken string
	lines := strings.Split(string(createOutput), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Token: ") {
			firstToken = strings.TrimPrefix(line, "Token: ")
			break
		}
	}

	if firstToken == "" {
		t.Fatalf("Failed to extract first token from output")
	}

	t.Run("SecurityWarningForAdditionalTokens", func(t *testing.T) {
		// Try to create another token - should show security warning (already in project root)
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "create-token", "Second token")
		cmd.Env = append(os.Environ(), "ADMIN_TOKEN="+firstToken)

		// Provide "n" input to cancel creation
		cmd.Stdin = strings.NewReader("n\n")

		output, _ := cmd.CombinedOutput()
		// Note: Cancellation exits with status 0, which is correct behavior

		outputStr := string(output)
		if !strings.Contains(outputStr, "WARNING: Admin tokens already exist") {
			t.Error("Expected security warning about existing tokens")
		}
		if !strings.Contains(outputStr, "Only create new tokens if you're an authorized administrator") {
			t.Error("Expected authorization warning")
		}
		if !strings.Contains(outputStr, "Token creation cancelled") {
			t.Error("Expected cancellation message")
		}
	})

	t.Run("ProceedWithSecurityWarning", func(t *testing.T) {
		// Try to create another token with "y" confirmation (already in project root)
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "create-token", "Confirmed second token")
		cmd.Env = append(os.Environ(), "ADMIN_TOKEN="+firstToken)

		// Provide "y" input to proceed with creation
		cmd.Stdin = strings.NewReader("y\n")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Token creation with confirmation failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "WARNING: Admin tokens already exist") {
			t.Error("Expected security warning about existing tokens")
		}
		if !strings.Contains(outputStr, "Admin token created successfully") {
			t.Error("Expected success message after confirmation")
		}
	})

	// Clean up after test
	cleanupDatabase(t)
}
