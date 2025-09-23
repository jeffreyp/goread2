//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"goread2/internal/database"
	"goread2/internal/services"
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

	feedService := services.NewFeedService(db, nil)

	for _, feed := range feeds {
		// Skip Seattle Times - leave it unread for now
		if feed.URL == "https://www.seattletimes.com/feed/" {
			fmt.Printf("Skipping Seattle Times feed (ID: %d)\n", feed.ID)
			continue
		}

		fmt.Printf("Marking all articles as READ for: %s\n", feed.Title)

		// Get all articles for this feed
		articles, err := db.GetArticles(feed.ID)
		if err != nil {
			log.Printf("Error getting articles for feed %d: %v", feed.ID, err)
			continue
		}

		fmt.Printf("  Found %d articles\n", len(articles))

		// Mark all articles as read
		for _, article := range articles {
			err := feedService.MarkUserArticleRead(user.ID, article.ID, true)
			if err != nil {
				log.Printf("  Error marking article %d as read: %v", article.ID, err)
			}
		}

		fmt.Printf("  ✅ Marked %d articles as READ\n", len(articles))
	}

	fmt.Println("\n✅ Restored read status for all feeds except Seattle Times")
	fmt.Println("Your other feeds should now show 0 unread articles")
}