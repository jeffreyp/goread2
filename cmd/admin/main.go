package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"goread2/internal/config"
	"goread2/internal/database"
	"goread2/internal/services"
)

func main() {
	// SECURITY: Require admin token for sensitive operations
	adminToken := os.Getenv("ADMIN_TOKEN")
	if adminToken == "" {
		fmt.Println("ERROR: ADMIN_TOKEN environment variable must be set")
		fmt.Println("This is a security requirement to prevent unauthorized admin access.")
		fmt.Println("Set ADMIN_TOKEN to a secure random value before running admin commands.")
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/admin/main.go <command> [args]")
		fmt.Println("Commands:")
		fmt.Println("  list-users                    - List all users")
		fmt.Println("  set-admin <email> <true/false> - Set admin status for user (REQUIRES ADMIN_TOKEN)")
		if config.IsSubscriptionEnabled() {
			fmt.Println("  grant-months <email> <months>  - Grant free months to user (REQUIRES ADMIN_TOKEN)")
		}
		fmt.Println("  user-info <email>             - Show user information")
		fmt.Println("")
		fmt.Println("SECURITY NOTE: All commands require ADMIN_TOKEN environment variable to be set.")
		os.Exit(1)
	}

	command := os.Args[1]

	// Verify admin token for sensitive operations
	if command == "set-admin" || command == "grant-months" {
		providedToken := os.Getenv("ADMIN_TOKEN_VERIFY")
		if providedToken == "" {
			fmt.Println("ERROR: ADMIN_TOKEN_VERIFY environment variable must be set for sensitive operations")
			fmt.Println("This must match the ADMIN_TOKEN value as an additional security check.")
			os.Exit(1)
		}
		if providedToken != adminToken {
			fmt.Println("ERROR: ADMIN_TOKEN_VERIFY does not match ADMIN_TOKEN")
			fmt.Println("Both environment variables must have the same value for security verification.")
			os.Exit(1)
		}
	}

	// Initialize database and services
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer func() { _ = db.Close() }()

	subscriptionService := services.NewSubscriptionService(db)

	switch command {
	case "list-users":
		listUsers(db)

	case "set-admin":
		if len(os.Args) != 4 {
			fmt.Println("Usage: go run cmd/admin/main.go set-admin <email> <true/false>")
			os.Exit(1)
		}
		email := os.Args[2]
		isAdminStr := os.Args[3]
		isAdmin, err := strconv.ParseBool(isAdminStr)
		if err != nil {
			log.Fatal("Invalid admin status, use 'true' or 'false':", err)
		}
		setAdminStatus(subscriptionService, email, isAdmin)

	case "grant-months":
		if !config.IsSubscriptionEnabled() {
			fmt.Println("Error: Subscription system is disabled. Cannot grant free months.")
			fmt.Println("Set SUBSCRIPTION_ENABLED=true to enable subscription features.")
			os.Exit(1)
		}
		if len(os.Args) != 4 {
			fmt.Println("Usage: go run cmd/admin/main.go grant-months <email> <months>")
			os.Exit(1)
		}
		email := os.Args[2]
		monthsStr := os.Args[3]
		months, err := strconv.Atoi(monthsStr)
		if err != nil {
			log.Fatal("Invalid months value:", err)
		}
		grantFreeMonths(subscriptionService, email, months)

	case "user-info":
		if len(os.Args) != 3 {
			fmt.Println("Usage: go run cmd/admin/main.go user-info <email>")
			os.Exit(1)
		}
		email := os.Args[2]
		showUserInfo(subscriptionService, email)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func listUsers(db database.Database) {
	// For SQLite implementation, we can access the underlying DB
	if sqliteDB, ok := db.(*database.DB); ok {
		// Simple query to list all users
		query := `SELECT id, email, name, subscription_status, 
				  COALESCE(is_admin, 0), COALESCE(free_months_remaining, 0),
				  created_at FROM users ORDER BY id`
		
		rows, err := sqliteDB.Query(query)
		if err != nil {
			log.Fatal("Failed to query users:", err)
		}
		defer func() { _ = rows.Close() }()

		fmt.Printf("\n%-4s %-35s %-25s %-12s %-6s %-6s %-12s\n", 
			"ID", "Email", "Name", "Status", "Admin", "Free", "Joined")
		fmt.Printf("%-4s %-35s %-25s %-12s %-6s %-6s %-12s\n", 
			"----", "-----------------------------------", "-------------------------", "------------", "------", "------", "------------")

		for rows.Next() {
			var id int
			var email, name, status string
			var isAdmin bool
			var freeMonths int
			var createdAt string

			err := rows.Scan(&id, &email, &name, &status, &isAdmin, &freeMonths, &createdAt)
			if err != nil {
				log.Fatal("Failed to scan user:", err)
			}

			adminStr := "No"
			if isAdmin {
				adminStr = "Yes"
			}

			fmt.Printf("%-4d %-35s %-25s %-12s %-6s %-6d %-12s\n", 
				id, truncate(email, 34), truncate(name, 24), status, adminStr, freeMonths, createdAt[:10])
		}
		fmt.Println()
	} else {
		log.Fatal("List users command only supports SQLite database")
	}
}

func setAdminStatus(subscriptionService *services.SubscriptionService, email string, isAdmin bool) {
	// Find user by email
	user, err := subscriptionService.GetUserByEmail(email)
	if err != nil {
		log.Fatal("User not found:", err)
	}

	// Set admin status
	err = subscriptionService.SetUserAdmin(user.ID, isAdmin)
	if err != nil {
		log.Fatal("Failed to set admin status:", err)
	}

	status := "removed from"
	if isAdmin {
		status = "granted"
	}

	fmt.Printf("✅ Admin access %s for user: %s (%s)\n", status, user.Name, user.Email)
}

func grantFreeMonths(subscriptionService *services.SubscriptionService, email string, months int) {
	// Find user by email
	user, err := subscriptionService.GetUserByEmail(email)
	if err != nil {
		log.Fatal("User not found:", err)
	}

	// Grant free months
	err = subscriptionService.GrantFreeMonths(user.ID, months)
	if err != nil {
		log.Fatal("Failed to grant free months:", err)
	}

	fmt.Printf("✅ Granted %d free months to user: %s (%s)\n", months, user.Name, user.Email)
	fmt.Printf("   Total free months: %d\n", user.FreeMonthsRemaining+months)
}

func showUserInfo(subscriptionService *services.SubscriptionService, email string) {
	// Find user by email
	user, err := subscriptionService.GetUserByEmail(email)
	if err != nil {
		log.Fatal("User not found:", err)
	}

	// Get subscription info
	subscriptionInfo, err := subscriptionService.GetUserSubscriptionInfo(user.ID)
	if err != nil {
		log.Fatal("Failed to get subscription info:", err)
	}

	fmt.Printf("\n┌─────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│                         User Information                        │\n")
	fmt.Printf("├─────────────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│ %-20s │ %-42s │\n", "ID:", fmt.Sprintf("%d", user.ID))
	fmt.Printf("│ %-20s │ %-42s │\n", "Name:", user.Name)
	fmt.Printf("│ %-20s │ %-42s │\n", "Email:", user.Email)
	fmt.Printf("│ %-20s │ %-42s │\n", "Joined:", user.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("│ %-20s │ %-42s │\n", "Google ID:", user.GoogleID)
	fmt.Printf("├─────────────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│                      System Configuration                       │\n")
	fmt.Printf("├─────────────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│ %-20s │ %-42s │\n", "Subscription System:", map[bool]string{true: "Enabled", false: "Disabled"}[config.IsSubscriptionEnabled()])
	fmt.Printf("├─────────────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│                      Subscription Details                       │\n")
	fmt.Printf("├─────────────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│ %-20s │ %-42s │\n", "Status:", subscriptionInfo.Status)
	fmt.Printf("│ %-20s │ %-42s │\n", "Is Admin:", map[bool]string{true: "Yes", false: "No"}[user.IsAdmin])
	
	if config.IsSubscriptionEnabled() {
		fmt.Printf("│ %-20s │ %-42s │\n", "Free Months:", fmt.Sprintf("%d", user.FreeMonthsRemaining))
	}
	
	fmt.Printf("│ %-20s │ %-42s │\n", "Current Feeds:", fmt.Sprintf("%d", subscriptionInfo.CurrentFeeds))
	
	if subscriptionInfo.FeedLimit == -1 {
		fmt.Printf("│ %-20s │ %-42s │\n", "Feed Limit:", "Unlimited")
	} else {
		fmt.Printf("│ %-20s │ %-42s │\n", "Feed Limit:", fmt.Sprintf("%d", subscriptionInfo.FeedLimit))
	}
	
	if subscriptionInfo.Status == "trial" {
		fmt.Printf("│ %-20s │ %-42s │\n", "Trial Ends:", user.TrialEndsAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("│ %-20s │ %-42s │\n", "Days Remaining:", fmt.Sprintf("%d", subscriptionInfo.TrialDaysRemaining))
	}
	
	if user.SubscriptionID != "" {
		fmt.Printf("│ %-20s │ %-42s │\n", "Stripe Subscription:", user.SubscriptionID)
	}
	
	if !user.LastPaymentDate.IsZero() {
		fmt.Printf("│ %-20s │ %-42s │\n", "Last Payment:", user.LastPaymentDate.Format("2006-01-02 15:04:05"))
	}
	
	fmt.Printf("└─────────────────────────────────────────────────────────────────┘\n\n")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}