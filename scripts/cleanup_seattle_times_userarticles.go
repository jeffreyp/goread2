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
		log.Fatal("Seattle Times feed not found in user subscriptions!")
	}

	fmt.Printf("Seattle Times Feed ID: %d\n", seattleTimesFeed.ID)

	// Get all articles for this feed
	articles, err := db.GetArticles(seattleTimesFeed.ID)
	if err != nil {
		log.Fatalf("Failed to get articles: %v", err)
	}

	fmt.Printf("Total articles in feed: %d\n", len(articles))

	// Get direct access to the datastore client for cleanup
	client := db.GetClient()
	ctx := context.Background()

	// Find and delete all UserArticle entries for this user and this feed's articles
	deletedCount := 0
	chunkSize := 100

	for i := 0; i < len(articles); i += chunkSize {
		end := i + chunkSize
		if end > len(articles) {
			end = len(articles)
		}

		chunk := articles[i:end]
		var userArticleKeys []*datastore.Key

		for _, article := range chunk {
			// Query for user-article relationships with this user and article
			userArticleQuery := datastore.NewQuery("UserArticle").
				FilterField("user_id", "=", int64(user.ID)).
				FilterField("article_id", "=", int64(article.ID)).
				KeysOnly()

			keys, err := client.GetAll(ctx, userArticleQuery, nil)
			if err != nil {
				log.Printf("Warning: Failed to query UserArticle for article %d: %v", article.ID, err)
				continue
			}
			userArticleKeys = append(userArticleKeys, keys...)
		}

		// Delete the user-article relationships in this chunk
		if len(userArticleKeys) > 0 {
			if err := client.DeleteMulti(ctx, userArticleKeys); err != nil {
				log.Printf("Warning: Failed to delete some user-article relationships: %v", err)
			} else {
				deletedCount += len(userArticleKeys)
				fmt.Printf("Deleted %d UserArticle entries (chunk %d-%d)\n", len(userArticleKeys), i+1, end)
			}
		}
	}

	fmt.Printf("\n✅ Cleanup completed: Deleted %d UserArticle entries\n", deletedCount)
	fmt.Printf("Now re-subscribe to the Seattle Times feed and you should see only ~100 unread articles\n")
}