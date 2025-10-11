//go:build ignore

package main

import (
	"fmt"
	"os"

	"goread2/internal/config"
)

func main() {
	fmt.Println("🔍 Validating GoRead2 configuration...")
	
	// Load configuration
	cfg := config.Load()
	fmt.Printf("✓ Configuration loaded (Subscription enabled: %v)\n", cfg.SubscriptionEnabled)
	
	// Check if we should use strict validation (for production)
	strict := os.Getenv("VALIDATE_STRICT") == "true"
	if strict {
		fmt.Println("ℹ️  Using strict validation mode (for production deployment)")
	}
	
	// Validate environment configuration
	if err := config.ValidateEnvironmentConfigStrict(strict); err != nil {
		fmt.Printf("❌ Configuration validation failed:\n%v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("✓ All required credentials are available")
	
	// Check for unhandled environment variables
	fmt.Println("\n🔍 Checking for unhandled environment variables...")
	config.WarnAboutUnhandledEnvVars()
	
	fmt.Println("\n✅ Configuration validation completed successfully!")
}