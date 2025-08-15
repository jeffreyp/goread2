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
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/admin/main.go <command> [args]")
		fmt.Println("Commands:")
		fmt.Println("  list-users                    - List all users")
		fmt.Println("  set-admin <email> <true/false> - Set admin status for user")
		if config.IsSubscriptionEnabled() {
			fmt.Println("  grant-months <email> <months>  - Grant free months to user")
		}
		fmt.Println("  user-info <email>             - Show user information")
		os.Exit(1)
	}

	command := os.Args[1]

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

		fmt.Println("ID\tEmail\t\t\t\tName\t\t\tStatus\t\tAdmin\tFree Months\tJoined")
		fmt.Println("--\t-----\t\t\t\t----\t\t\t------\t\t-----\t-----------\t------")

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

			fmt.Printf("%d\t%-25s\t%-15s\t%-10s\t%s\t%d\t\t%s\n", 
				id, email, truncate(name, 15), status, adminStr, freeMonths, createdAt[:10])
		}
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

	fmt.Printf("User Information:\n")
	fmt.Printf("  ID: %d\n", user.ID)
	fmt.Printf("  Name: %s\n", user.Name)
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Joined: %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Google ID: %s\n", user.GoogleID)
	fmt.Printf("\nSystem Configuration:\n")
	fmt.Printf("  Subscription System: %s\n", map[bool]string{true: "Enabled", false: "Disabled"}[config.IsSubscriptionEnabled()])
	fmt.Printf("\nSubscription Details:\n")
	fmt.Printf("  Status: %s\n", subscriptionInfo.Status)
	fmt.Printf("  Is Admin: %t\n", user.IsAdmin)
	if config.IsSubscriptionEnabled() {
		fmt.Printf("  Free Months Remaining: %d\n", user.FreeMonthsRemaining)
	}
	fmt.Printf("  Current Feeds: %d\n", subscriptionInfo.CurrentFeeds)
	
	if subscriptionInfo.FeedLimit == -1 {
		fmt.Printf("  Feed Limit: Unlimited\n")
	} else {
		fmt.Printf("  Feed Limit: %d\n", subscriptionInfo.FeedLimit)
	}
	
	if subscriptionInfo.Status == "trial" {
		fmt.Printf("  Trial Ends: %s\n", user.TrialEndsAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Days Remaining: %d\n", subscriptionInfo.TrialDaysRemaining)
	}
	
	if user.SubscriptionID != "" {
		fmt.Printf("  Stripe Subscription ID: %s\n", user.SubscriptionID)
	}
	
	if !user.LastPaymentDate.IsZero() {
		fmt.Printf("  Last Payment: %s\n", user.LastPaymentDate.Format("2006-01-02 15:04:05"))
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}