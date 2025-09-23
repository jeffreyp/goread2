//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"goread2/internal/database"
)

func main() {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT environment variable must be set")
	}

	db, err := database.NewDatastoreDB(projectID)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Get user
	userEmail := "jeffrey@jeffreypratt.org"
	user, err := db.GetUserByEmail(userEmail)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	fmt.Printf("User ID: %d\n", user.ID)

	// Get all user feeds
	feeds, err := db.GetUserFeeds(user.ID)
	if err != nil {
		log.Fatalf("Failed to get user feeds: %v", err)
	}

	fmt.Printf("Found %d feeds\n", len(feeds))

	totalMarked := 0

	for _, feed := range feeds {
		// Skip Seattle Times - leave it unread for now
		if feed.URL == "https://www.seattletimes.com/feed/" {
			fmt.Printf("Skipping Seattle Times feed (ID: %d)\n", feed.ID)
			continue
		}

		fmt.Printf("Processing feed: %s (ID: %d)\n", feed.Title, feed.ID)

		// Get all articles for this feed
		articles, err := db.GetArticles(feed.ID)
		if err != nil {
			log.Printf("Error getting articles for feed %d: %v", feed.ID, err)
			continue
		}

		if len(articles) == 0 {
			fmt.Printf("  No articles found\n")
			continue
		}

		fmt.Printf("  Found %d articles, batch marking as READ...\n", len(articles))

		// Use BatchSetUserArticleStatus for much faster processing
		err = db.BatchSetUserArticleStatus(user.ID, articles, true, false) // read=true, starred=false
		if err != nil {
			log.Printf("  Error batch marking articles as read: %v", err)
		} else {
			totalMarked += len(articles)
			fmt.Printf("  ✅ Batch marked %d articles as READ\n", len(articles))
		}
	}

	fmt.Printf("\n✅ COMPLETED: Batch marked %d total articles as READ across all feeds\n", totalMarked)
	fmt.Printf("Your other feeds should now show 0 unread articles\n")
	fmt.Printf("Only Seattle Times should still show unread articles\n")
}