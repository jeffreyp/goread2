//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/datastore"
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

	// Find Seattle Times feed
	feeds, err := db.GetUserFeeds(user.ID)
	if err != nil {
		log.Fatalf("Failed to get user feeds: %v", err)
	}

	var seattleTimesFeed *database.Feed
	for _, feed := range feeds {
		if feed.URL == "https://www.seattletimes.com/feed/" {
			seattleTimesFeed = &feed
			break
		}
	}

	if seattleTimesFeed == nil {
		log.Fatal("Seattle Times feed not found!")
	}

	fmt.Printf("Seattle Times Feed ID: %d\n", seattleTimesFeed.ID)

	client := db.GetClient()
	ctx := context.Background()

	// Direct approach: Delete all UserArticle entries using large batch queries
	fmt.Println("Starting aggressive UserArticle cleanup...")

	deletedTotal := 0
	batchSize := 500 // Larger batches for faster processing

	for {
		// Query for UserArticle entries for this user (any feed)
		userArticleQuery := datastore.NewQuery("UserArticle").
			FilterField("user_id", "=", int64(user.ID)).
			Limit(batchSize).
			KeysOnly()

		keys, err := client.GetAll(ctx, userArticleQuery, nil)
		if err != nil {
			log.Printf("Error querying UserArticle entries: %v", err)
			break
		}

		if len(keys) == 0 {
			fmt.Println("No more UserArticle entries found")
			break
		}

		// Delete this batch
		err = client.DeleteMulti(ctx, keys)
		if err != nil {
			log.Printf("Error deleting batch: %v", err)
			// Continue with remaining batches
		} else {
			deletedTotal += len(keys)
			fmt.Printf("Deleted %d UserArticle entries (total deleted: %d)\n", len(keys), deletedTotal)
		}

		// If we got fewer than the batch size, we're done
		if len(keys) < batchSize {
			break
		}
	}

	fmt.Printf("\n✅ Cleanup completed: Deleted %d total UserArticle entries\n", deletedTotal)
	fmt.Printf("Now all your feeds should show 0 unread articles\n")
	fmt.Printf("Re-add the Seattle Times feed and it should respect the 100 article limit\n")
}