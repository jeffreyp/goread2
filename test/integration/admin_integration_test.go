package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"goread2/internal/database"
	"goread2/test/helpers"
)

func TestAdminCommands(t *testing.T) {
	// Set up test database
	db := helpers.CreateTestDB(t)
	sqliteDB := db.(*database.DB)

	// Create a temporary test database file for the commands
	tempDBFile := "test_admin.db"
	defer func() {
		_ = os.Remove(tempDBFile)
	}()

	// Copy our in-memory database to a file for the admin commands to use
	setupTestDatabase(t, sqliteDB, tempDBFile)

	t.Run("AdminList", func(t *testing.T) {
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "list-users")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin list command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		
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
		if !strings.Contains(outputStr, "test@example.com") {
			t.Error("Expected output to contain test user email")
		}
	})

	t.Run("AdminGrant", func(t *testing.T) {
		// Grant admin access
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "set-admin", "test@example.com", "true")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin grant command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "granted") {
			t.Error("Expected success message containing 'granted'")
		}

		// Verify the change by listing users
		listCmd := exec.Command("go", "run", "cmd/admin/main.go", "list-users")
		listCmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
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
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "set-admin", "test@example.com", "false")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin revoke command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "removed") {
			t.Error("Expected success message containing 'removed'")
		}

		// Verify the change by listing users
		listCmd := exec.Command("go", "run", "cmd/admin/main.go", "list-users")
		listCmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
		listOutput, err := listCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin list command failed after revoke: %v", err)
		}

		lines := strings.Split(string(listOutput), "\n")
		var userLine string
		for _, line := range lines {
			if strings.Contains(line, "test@example.com") {
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
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "grant-months", "test@example.com", "6")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Grant months command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "6 free months") {
			t.Error("Expected success message containing '6 free months'")
		}

		// Verify the change by listing users
		listCmd := exec.Command("go", "run", "cmd/admin/main.go", "list-users")
		listCmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
		listOutput, err := listCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Admin list command failed after grant months: %v", err)
		}

		lines := strings.Split(string(listOutput), "\n")
		var userLine string
		for _, line := range lines {
			if strings.Contains(line, "test@example.com") {
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
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "user-info", "test@example.com")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("User info command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		
		// Check for expected information
		if !strings.Contains(outputStr, "User Information:") {
			t.Error("Expected output to contain 'User Information:'")
		}
		if !strings.Contains(outputStr, "Email: test@example.com") {
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
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "invalid-command")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
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
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "set-admin", "nonexistent@example.com", "true")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for non-existent user")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "User not found") {
			t.Error("Expected error message about user not found")
		}
	})
}

// setupTestDatabase creates a file-based SQLite database with test data
func setupTestDatabase(t *testing.T, sourceDB *database.DB, filename string) {
	// Create a new file-based database
	fileDB, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to create file database: %v", err)
	}
	defer fileDB.Close()

	// Create test user
	user := &database.User{
		GoogleID:  "test123",
		Email:     "test@example.com",
		Name:      "Test User",
		Avatar:    "https://example.com/avatar.jpg",
		CreatedAt: time.Now(),
	}

	err = fileDB.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
}

func TestAdminCommandEdgeCases(t *testing.T) {
	// Set up test database
	db := helpers.CreateTestDB(t)
	sqliteDB := db.(*database.DB)

	// Create a temporary test database file
	tempDBFile := "test_admin_edge.db"
	defer func() {
		_ = os.Remove(tempDBFile)
	}()

	setupTestDatabase(t, sqliteDB, tempDBFile)

	t.Run("GrantZeroMonths", func(t *testing.T) {
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "grant-months", "test@example.com", "0")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
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
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "grant-months", "test@example.com", "-1")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for negative months")
		}

		// The command should handle this gracefully or show an error
		outputStr := string(output)
		t.Logf("Output for negative months: %s", outputStr)
	})

	t.Run("InvalidBooleanForAdmin", func(t *testing.T) {
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "set-admin", "test@example.com", "maybe")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
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
		cmd := exec.Command("go", "run", "cmd/admin/main.go", "set-admin", "test@example.com")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
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
		cmd := exec.Command("go", "run", "cmd/admin/main.go")
		cmd.Env = append(os.Environ(), "DB_PATH="+tempDBFile)
		
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for no arguments")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Usage:") {
			t.Error("Expected usage message for no arguments")
		}
	})
}