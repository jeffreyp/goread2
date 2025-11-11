//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/services"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run mark_feed_unread.go <user_id> <feed_id>")
		fmt.Println("Example: go run mark_feed_unread.go 1 5")
		os.Exit(1)
	}

	userID, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid user ID: %v", err)
	}

	feedID, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("Invalid feed ID: %v", err)
	}

	fmt.Printf("Marking all articles in feed %d as unread for user %d\n", feedID, userID)

	// Initialize database connection
	db, err := database.NewDatastoreDB(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create feed service
	rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
	})
	feedService := services.NewFeedService(db, rateLimiter)

	// Get all articles in the feed
	articles, err := feedService.GetArticles(feedID)
	if err != nil {
		log.Fatalf("Failed to get articles: %v", err)
	}

	fmt.Printf("Found %d articles in feed %d\n", len(articles), feedID)

	if len(articles) == 0 {
		fmt.Println("No articles found. Nothing to do.")
		return
	}

	// Mark all as unread using the existing batch function
	err = db.BatchSetUserArticleStatus(userID, articles, false, false) // unread, unstarred
	if err != nil {
		log.Fatalf("Failed to mark articles as unread: %v", err)
	}

	fmt.Printf("Successfully marked %d articles as unread for user %d\n", len(articles), userID)
}
