package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/test/helpers"
)

// Global test token for integration tests
var testAdminToken string

// setupLocalAdminToken creates a valid admin token for a specific test
func setupLocalAdminToken(t *testing.T, googleID, email, name string) string {
	// Create admin user first
	setupMainTestUser(t, googleID, email, name)

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
		t.Fatalf("Failed to initialize database for admin setup: %v", err)
	}

	sqliteDB := db.(*database.DB)
	result, err := sqliteDB.Exec("UPDATE users SET is_admin = 1 WHERE email = ?", email)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	// Verify the update worked
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		t.Fatalf("No rows were updated when setting admin status")
	}

	// Double-check by querying the admin status
	var isAdmin bool
	err = sqliteDB.QueryRow("SELECT is_admin FROM users WHERE email = ?", email).Scan(&isAdmin)
	if err != nil {
		t.Fatalf("Failed to verify admin status: %v", err)
	}
	if !isAdmin {
		t.Fatalf("User was not successfully set as admin")
	}
	t.Logf("Successfully set user as admin: %t", isAdmin)

	// Close the database connection to ensure changes are flushed
	err = db.Close()
	if err != nil {
		t.Fatalf("Failed to close database connection: %v", err)
	}

	// Create admin token using bootstrap (we're already in project root)
	cmd := exec.Command("go", "run", "cmd/admin/main.go", "create-token", "Local test token")
	cmd.Env = append(os.Environ(), "ADMIN_TOKEN=bootstrap")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create admin token: %v\nOutput: %s", err, output)
	}

	// Extract token from output
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	var localToken string
	for _, line := range lines {
		if strings.HasPrefix(line, "Token: ") {
			localToken = strings.TrimPrefix(line, "Token: ")
			break
		}
	}

	if localToken == "" {
		t.Fatalf("Failed to extract token from output: %s", outputStr)
	}

	t.Logf("Created local admin token: %s", localToken)
	return localToken
}

// setupTestAdminToken creates a valid admin token for testing
func setupTestAdminToken(t *testing.T) {
	if testAdminToken != "" {
		return // Already set up
	}

	testAdminToken = setupLocalAdminToken(t, "admin123", "admin@test.com", "Admin User")
}

// createAdminCommand creates an exec.Command for admin commands with proper working directory and security
func createAdminCommand(args ...string) *exec.Cmd {
	cmdArgs := append([]string{"run", "cmd/admin/main.go"}, args...)
	cmd := exec.Command("go", cmdArgs...)

	cmd.Dir = "../.." // Set working directory to project root

	// SECURITY: Set required admin token (must be valid 64-char token)
	cmd.Env = append(os.Environ(), "ADMIN_TOKEN="+testAdminToken)

	return cmd
}

// setupMainTestUser creates a test user in the main database
func setupMainTestUser(t *testing.T, googleID, email, name string) {
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
	defer func() { _ = db.Close() }()

	user := &database.User{
		GoogleID:  googleID,
		Email:     email,
		Name:      name,
		Avatar:    "https://example.com/avatar.jpg",
		CreatedAt: time.Now(),
	}

	err = db.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Logf("Created test user: %s (%s) with ID: %d", name, email, user.ID)

	// Verify the user was created by reading it back
	retrievedUser, err := db.GetUserByEmail(email)
	if err != nil {
		t.Fatalf("Failed to retrieve created user: %v", err)
	}
	t.Logf("Verified user exists: %s (%s)", retrievedUser.Name, retrievedUser.Email)
}

// cleanupDatabase removes test users from the main database
// Deprecated: Use helpers.CleanupTestUsers instead
func cleanupDatabase(t *testing.T) {
	helpers.CleanupTestUsers(t)
}

func TestAdminCommands(t *testing.T) {
	// Clean up the main database before test
	cleanupDatabase(t)

	// Set up admin token for secure authentication
	setupTestAdminToken(t)

	// Set up test user in the main database
	setupMainTestUser(t, "main123", "main@example.com", "Main Test User")

	t.Run("AdminList", func(t *testing.T) {
		cmd := createAdminCommand("list-users")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin list command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		t.Logf("Admin list output:\n%s", outputStr)

		// Check that output contains headers
		if !strings.Contains(outputStr, "ID") {
			t.Error("Expected output to contain 'ID' header")
		}
		if !strings.Contains(outputStr, "Email") {
			t.Error("Expected output to contain 'Email' header")
		}
		if !strings.Contains(outputStr, "Admin") {
			t.Error("Expected output to contain 'Admin' header")
		}

		// Check for test user
		if !strings.Contains(outputStr, "main@example.com") {
			t.Error("Expected output to contain test user email")
		}
	})

	t.Run("AdminGrant", func(t *testing.T) {
		// Grant admin access
		cmd := createAdminCommand("set-admin", "main@example.com", "true")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin grant command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "granted") {
			t.Error("Expected success message containing 'granted'")
		}

		// Verify the change by listing users
		listCmd := createAdminCommand("list-users")

		listOutput, err := listCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin list command failed after grant: %v", err)
		}

		if !strings.Contains(string(listOutput), "Yes") {
			t.Error("Expected user to be marked as admin after grant")
		}
	})

	t.Run("AdminRevoke", func(t *testing.T) {
		// Revoke admin access
		cmd := createAdminCommand("set-admin", "main@example.com", "false")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin revoke command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "removed") {
			t.Error("Expected success message containing 'removed'")
		}

		// Verify the change by listing users
		listCmd := createAdminCommand("list-users")

		listOutput, err := listCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin list command failed after revoke: %v", err)
		}

		lines := strings.Split(string(listOutput), "\n")
		var userLine string
		for _, line := range lines {
			if strings.Contains(line, "main@example.com") {
				userLine = line
				break
			}
		}

		if userLine == "" {
			t.Fatal("Could not find test user in output")
		}

		// Should show "No" for admin status
		if !strings.Contains(userLine, "No") {
			t.Error("Expected user to be marked as non-admin after revoke")
		}
	})

	t.Run("GrantFreeMonths", func(t *testing.T) {
		// Grant free months (enable subscription system for this test)
		cmd := createAdminCommand("grant-months", "main@example.com", "6")
		cmd.Env = append(cmd.Env, "SUBSCRIPTION_ENABLED=true")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Grant months command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "6 free months") {
			t.Error("Expected success message containing '6 free months'")
		}

		// Verify the change by listing users
		listCmd := createAdminCommand("list-users")

		listOutput, err := listCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin list command failed after grant months: %v", err)
		}

		lines := strings.Split(string(listOutput), "\n")
		var userLine string
		for _, line := range lines {
			if strings.Contains(line, "main@example.com") {
				userLine = line
				break
			}
		}

		if userLine == "" {
			t.Fatal("Could not find test user in output")
		}

		// Should show 6 free months
		if !strings.Contains(userLine, "6") {
			t.Errorf("Expected user to have 6 free months, got line: %s", userLine)
		}
	})

	t.Run("UserInfo", func(t *testing.T) {
		// Get user info (enable subscription system for this test)
		cmd := createAdminCommand("user-info", "main@example.com")
		cmd.Env = append(cmd.Env, "SUBSCRIPTION_ENABLED=true")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("User info command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		t.Logf("User info output:\n%s", outputStr)

		// Check for expected information
		if !strings.Contains(outputStr, "User Information") {
			t.Error("Expected output to contain 'User Information'")
		}
		if !strings.Contains(outputStr, "main@example.com") {
			t.Error("Expected output to contain user email")
		}
		if !strings.Contains(outputStr, "Subscription Details") {
			t.Error("Expected output to contain 'Subscription Details'")
		}
		if !strings.Contains(outputStr, "Free Months") {
			t.Error("Expected output to show free months field")
		}
	})

	t.Run("InvalidCommands", func(t *testing.T) {
		// Test invalid command
		cmd := createAdminCommand("invalid-command")

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for invalid command")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Unknown command") {
			t.Error("Expected error message about unknown command")
		}
	})

	t.Run("InvalidUser", func(t *testing.T) {
		// Test with non-existent user
		cmd := createAdminCommand("set-admin", "nonexistent@example.com", "true")

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for non-existent user")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "User not found") {
			t.Error("Expected error message about user not found")
		}
	})

	// Clean up after test
	cleanupDatabase(t)
}

func TestAdminTokenCommands(t *testing.T) {
	// Clean up the main database before test
	cleanupDatabase(t)

	// Set up local admin token for this test
	localAdminToken := setupLocalAdminToken(t, "tokentest123", "tokentest@test.com", "Token Test Admin")

	// Create local command function using local token
	createLocalAdminCommand := func(args ...string) *exec.Cmd {
		cmdArgs := append([]string{"run", "cmd/admin/main.go"}, args...)
		cmd := exec.Command("go", cmdArgs...)
		cmd.Dir = "../.."
		cmd.Env = append(os.Environ(), "ADMIN_TOKEN="+localAdminToken)
		return cmd
	}

	t.Run("CreateAdminToken", func(t *testing.T) {
		cmd := createLocalAdminCommand("create-token", "Integration test token 2")

		// Provide "y" input to proceed with creation (since admin tokens already exist)
		cmd.Stdin = strings.NewReader("y\n")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Create token command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		t.Logf("Create token output:\n%s", outputStr)

		if !strings.Contains(outputStr, "Admin token created successfully") {
			t.Error("Expected success message for token creation")
		}

		if !strings.Contains(outputStr, "Token: ") {
			t.Error("Expected output to contain token")
		}
	})

	t.Run("ListAdminTokens", func(t *testing.T) {
		cmd := createLocalAdminCommand("list-tokens")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("List tokens command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "ID") {
			t.Error("Expected output to contain 'ID' header")
		}
		if !strings.Contains(outputStr, "Description") {
			t.Error("Expected output to contain 'Description' header")
		}
		if !strings.Contains(outputStr, "Integration test token") {
			t.Error("Expected output to contain test token description")
		}
	})

	t.Run("RevokeAdminToken", func(t *testing.T) {
		// First list tokens to get an ID
		listCmd := createLocalAdminCommand("list-tokens")

		listOutput, err := listCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("List tokens command failed: %v", err)
		}

		// Extract first token ID from output
		lines := strings.Split(string(listOutput), "\n")
		var tokenID string
		for _, line := range lines {
			if strings.Contains(line, "Integration test token") && !strings.Contains(line, "ID") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					tokenID = fields[0]
					break
				}
			}
		}

		if tokenID == "" {
			t.Skip("No token ID found to revoke")
		}

		// Revoke the token
		revokeCmd := createLocalAdminCommand("revoke-token", tokenID)

		output, err := revokeCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Revoke token command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "revoked successfully") {
			t.Error("Expected success message for token revocation")
		}
	})

	// Clean up after test
	cleanupDatabase(t)
}

func TestAdminCommandEdgeCases(t *testing.T) {
	// Clean up the main database before test
	cleanupDatabase(t)

	// Set up local admin token for this test
	localAdminToken := setupLocalAdminToken(t, "edgetest123", "edgetest@test.com", "Edge Test Admin")

	// Create local command function using local token
	createLocalAdminCommand := func(args ...string) *exec.Cmd {
		cmdArgs := append([]string{"run", "cmd/admin/main.go"}, args...)
		cmd := exec.Command("go", cmdArgs...)
		cmd.Dir = "../.."
		cmd.Env = append(os.Environ(), "ADMIN_TOKEN="+localAdminToken)
		return cmd
	}

	// Set up test user in the main database
	setupMainTestUser(t, "edge123", "edge@example.com", "Edge Test User")

	t.Run("GrantZeroMonths", func(t *testing.T) {
		cmd := createLocalAdminCommand("grant-months", "edge@example.com", "0")
		cmd.Env = append(cmd.Env, "SUBSCRIPTION_ENABLED=true")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Grant zero months command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "0 free months") {
			t.Error("Expected success message for 0 free months")
		}
	})

	t.Run("GrantNegativeMonths", func(t *testing.T) {
		cmd := createLocalAdminCommand("grant-months", "edge@example.com", "-1")

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for negative months")
		}

		// The command should handle this gracefully or show an error
		outputStr := string(output)
		t.Logf("Output for negative months: %s", outputStr)
	})

	t.Run("InvalidBooleanForAdmin", func(t *testing.T) {
		cmd := createLocalAdminCommand("set-admin", "edge@example.com", "maybe")

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for invalid boolean value")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Invalid admin status") {
			t.Logf("Actual output: %s", outputStr)
			t.Error("Expected error message about invalid admin status")
		}
	})

	t.Run("MissingArguments", func(t *testing.T) {
		// Test set-admin with missing arguments
		cmd := createLocalAdminCommand("set-admin", "edge@example.com")

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for missing arguments")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Usage:") {
			t.Logf("Actual output: %s", outputStr)
			t.Error("Expected usage message for missing arguments")
		}
	})

	t.Run("NoArguments", func(t *testing.T) {
		// Test with no command
		cmd := createLocalAdminCommand()

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for no arguments")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Usage:") {
			t.Error("Expected usage message for no arguments")
		}
	})

	// Clean up after test
	cleanupDatabase(t)
}

func TestAuditLogging(t *testing.T) {
	// Clean up the main database before test
	cleanupDatabase(t)

	// Set up local admin token for this test
	localAdminToken := setupLocalAdminToken(t, "auditadmin123", "auditadmin@test.com", "Audit Test Admin")

	// Create local command function using local token
	createLocalAdminCommand := func(args ...string) *exec.Cmd {
		cmdArgs := append([]string{"run", "cmd/admin/main.go"}, args...)
		cmd := exec.Command("go", cmdArgs...)
		cmd.Dir = "../.."
		cmd.Env = append(os.Environ(), "ADMIN_TOKEN="+localAdminToken)
		return cmd
	}

	// Set up test user in the main database
	setupMainTestUser(t, "audit123", "audituser@example.com", "Audit Test User")

	t.Run("SetAdminCreatesAuditLog", func(t *testing.T) {
		// Grant admin access
		cmd := createLocalAdminCommand("set-admin", "audituser@example.com", "true")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin grant command failed: %v\nOutput: %s", err, output)
		}

		// Query database for audit logs
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
		defer func() { _ = db.Close() }()

		// Get audit logs
		logs, err := db.GetAuditLogs(50, 0, map[string]interface{}{
			"operation_type": "grant_admin",
		})
		if err != nil {
			t.Fatalf("Failed to get audit logs: %v", err)
		}

		if len(logs) == 0 {
			t.Fatal("Expected at least one audit log for grant_admin operation")
		}

		// Verify the audit log has CLI_ADMIN
		log := logs[0]
		if log.AdminEmail != "CLI_ADMIN" {
			t.Errorf("Expected admin_email 'CLI_ADMIN', got '%s'", log.AdminEmail)
		}
		if log.AdminUserID != 0 {
			t.Errorf("Expected admin_user_id 0, got %d", log.AdminUserID)
		}
		if log.OperationType != "grant_admin" {
			t.Errorf("Expected operation_type 'grant_admin', got '%s'", log.OperationType)
		}
		if log.TargetUserEmail != "audituser@example.com" {
			t.Errorf("Expected target_user_email 'audituser@example.com', got '%s'", log.TargetUserEmail)
		}
		if log.Result != "success" {
			t.Errorf("Expected result 'success', got '%s'", log.Result)
		}
		if log.IPAddress != "CLI" {
			t.Errorf("Expected ip_address 'CLI', got '%s'", log.IPAddress)
		}
	})

	t.Run("GrantFreeMonthsCreatesAuditLog", func(t *testing.T) {
		// Grant free months
		cmd := createLocalAdminCommand("grant-months", "audituser@example.com", "6")
		cmd.Env = append(cmd.Env, "SUBSCRIPTION_ENABLED=true")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Grant months command failed: %v\nOutput: %s", err, output)
		}

		// Query database for audit logs
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
		defer func() { _ = db.Close() }()

		// Get audit logs
		logs, err := db.GetAuditLogs(50, 0, map[string]interface{}{
			"operation_type": "grant_free_months",
		})
		if err != nil {
			t.Fatalf("Failed to get audit logs: %v", err)
		}

		if len(logs) == 0 {
			t.Fatal("Expected at least one audit log for grant_free_months operation")
		}

		// Verify the audit log
		log := logs[0]
		if log.AdminEmail != "CLI_ADMIN" {
			t.Errorf("Expected admin_email 'CLI_ADMIN', got '%s'", log.AdminEmail)
		}
		if log.OperationType != "grant_free_months" {
			t.Errorf("Expected operation_type 'grant_free_months', got '%s'", log.OperationType)
		}
		if log.TargetUserEmail != "audituser@example.com" {
			t.Errorf("Expected target_user_email 'audituser@example.com', got '%s'", log.TargetUserEmail)
		}
		if log.Result != "success" {
			t.Errorf("Expected result 'success', got '%s'", log.Result)
		}
	})

	t.Run("AuditLogsCommand", func(t *testing.T) {
		// Test the audit-logs command
		cmd := createLocalAdminCommand("audit-logs")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Audit logs command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		t.Logf("Audit logs output:\n%s", outputStr)

		// Check that output contains headers
		if !strings.Contains(outputStr, "ID") {
			t.Error("Expected output to contain 'ID' header")
		}
		if !strings.Contains(outputStr, "Timestamp") {
			t.Error("Expected output to contain 'Timestamp' header")
		}
		if !strings.Contains(outputStr, "Admin") {
			t.Error("Expected output to contain 'Admin' header")
		}
		if !strings.Contains(outputStr, "Operation") {
			t.Error("Expected output to contain 'Operation' header")
		}

		// Check for CLI_ADMIN in the output
		if !strings.Contains(outputStr, "CLI_ADMIN") {
			t.Error("Expected output to contain 'CLI_ADMIN'")
		}

		// Check for our operations
		if !strings.Contains(outputStr, "grant_admin") {
			t.Error("Expected output to contain 'grant_admin' operation")
		}
		if !strings.Contains(outputStr, "grant_free_months") {
			t.Error("Expected output to contain 'grant_free_months' operation")
		}
	})

	t.Run("AuditLogsFilterByOperation", func(t *testing.T) {
		// Test audit-logs with operation filter
		cmd := createLocalAdminCommand("audit-logs", "--operation", "grant_admin")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Audit logs filter command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		t.Logf("Filtered audit logs output:\n%s", outputStr)

		// Should contain grant_admin operations
		if !strings.Contains(outputStr, "grant_admin") {
			t.Error("Expected filtered output to contain 'grant_admin' operation")
		}

		// Should NOT contain grant_free_months operations
		if strings.Contains(outputStr, "grant_free_months") {
			t.Error("Expected filtered output to NOT contain 'grant_free_months' operation")
		}
	})

	// Clean up after test
	cleanupDatabase(t)
}
