package main

import (
	"fmt"
	"log"
	"os"

	"goread2/internal/database"
	"goread2/internal/services"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/setup-stripe/main.go <command>")
		fmt.Println("Commands:")
		fmt.Println("  create-product  - Create GoRead2 Pro product and price in Stripe")
		fmt.Println("  validate        - Validate Stripe configuration")
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
	paymentService := services.NewPaymentService(db, subscriptionService)

	switch command {
	case "validate":
		if err := paymentService.ValidateStripeConfig(); err != nil {
			log.Fatal("Stripe configuration validation failed:", err)
		}
		fmt.Println("✅ Stripe configuration is valid!")
		fmt.Printf("Publishable Key: %s\n", paymentService.GetStripePublishableKey())

	case "create-product":
		fmt.Println("Creating GoRead2 Pro product and price...")

		price, err := paymentService.CreateProductAndPrice()
		if err != nil {
			log.Fatal("Failed to create product and price:", err)
		}

		fmt.Println("✅ Successfully created product and price!")
		fmt.Printf("Price ID: %s\n", price.ID)
		fmt.Printf("Amount: $%.2f/month\n", float64(price.UnitAmount)/100)
		fmt.Println("\nAdd this to your environment variables:")
		fmt.Printf("STRIPE_PRICE_ID=%s\n", price.ID)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
