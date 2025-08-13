package integration

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"goread2/internal/database"
)

// createAdminCommand creates an exec.Command for admin commands with proper working directory
func createAdminCommand(args ...string) *exec.Cmd {
	cmdArgs := append([]string{"run", "cmd/admin/main.go"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	
	cmd.Dir = "../.." // Set working directory to project root
	return cmd
}

// setupMainTestUser creates a test user in the main database
func setupMainTestUser(t *testing.T, googleID, email, name string) {
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

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
func cleanupDatabase(t *testing.T) {
	db, err := database.InitDB()
	if err != nil {
		t.Logf("Failed to initialize database for cleanup: %v", err)
		return
	}
	defer db.Close()

	// Delete test users (this is a simple cleanup - just remove the test emails)
	sqliteDB := db.(*database.DB)
	testEmails := []string{"main@example.com", "edge@example.com"}
	
	for _, email := range testEmails {
		result, err := sqliteDB.DB.Exec("DELETE FROM users WHERE email = ?", email)
		if err != nil {
			t.Logf("Failed to cleanup user %s: %v", email, err)
		} else {
			rowsAffected, _ := result.RowsAffected()
			t.Logf("Cleaned up user %s (rows affected: %d)", email, rowsAffected)
		}
	}
}

func TestAdminCommands(t *testing.T) {
	// Clean up the main database before test
	cleanupDatabase(t)
	
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
		cmd := createAdminCommand( "set-admin", "main@example.com", "true")
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin grant command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "granted") {
			t.Error("Expected success message containing 'granted'")
		}

		// Verify the change by listing users
		listCmd := createAdminCommand( "list-users")
		
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
		cmd := createAdminCommand( "set-admin", "main@example.com", "false")
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin revoke command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "removed") {
			t.Error("Expected success message containing 'removed'")
		}

		// Verify the change by listing users
		listCmd := createAdminCommand( "list-users")
		
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
		// Grant free months
		cmd := createAdminCommand( "grant-months", "main@example.com", "6")
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Grant months command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "6 free months") {
			t.Error("Expected success message containing '6 free months'")
		}

		// Verify the change by listing users
		listCmd := createAdminCommand( "list-users")
		
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
		// Get user info
		cmd := createAdminCommand( "user-info", "main@example.com")
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("User info command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		
		// Check for expected information
		if !strings.Contains(outputStr, "User Information:") {
			t.Error("Expected output to contain 'User Information:'")
		}
		if !strings.Contains(outputStr, "Email: main@example.com") {
			t.Error("Expected output to contain user email")
		}
		if !strings.Contains(outputStr, "Subscription Details:") {
			t.Error("Expected output to contain 'Subscription Details:'")
		}
		if !strings.Contains(outputStr, "Free Months Remaining: 6") {
			t.Error("Expected output to show 6 free months remaining")
		}
	})

	t.Run("InvalidCommands", func(t *testing.T) {
		// Test invalid command
		cmd := createAdminCommand( "invalid-command")
		
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
		cmd := createAdminCommand( "set-admin", "nonexistent@example.com", "true")
		
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


func TestAdminCommandEdgeCases(t *testing.T) {
	// Clean up the main database before test
	cleanupDatabase(t)
	
	// Set up test user in the main database
	setupMainTestUser(t, "edge123", "edge@example.com", "Edge Test User")

	t.Run("GrantZeroMonths", func(t *testing.T) {
		cmd := createAdminCommand( "grant-months", "edge@example.com", "0")
		
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
		cmd := createAdminCommand( "grant-months", "edge@example.com", "-1")
		
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for negative months")
		}

		// The command should handle this gracefully or show an error
		outputStr := string(output)
		t.Logf("Output for negative months: %s", outputStr)
	})

	t.Run("InvalidBooleanForAdmin", func(t *testing.T) {
		cmd := createAdminCommand( "set-admin", "edge@example.com", "maybe")
		
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for invalid boolean value")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Invalid admin status") {
			t.Error("Expected error message about invalid admin status")
		}
	})

	t.Run("MissingArguments", func(t *testing.T) {
		// Test set-admin with missing arguments
		cmd := createAdminCommand( "set-admin", "edge@example.com")
		
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for missing arguments")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Usage:") {
			t.Error("Expected usage message for missing arguments")
		}
	})

	t.Run("NoArguments", func(t *testing.T) {
		// Test with no command
		cmd := createAdminCommand()
		
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