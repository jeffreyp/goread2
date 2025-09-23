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

	// Get all Seattle Times articles
	articles, err := db.GetArticles(seattleTimesFeed.ID)
	if err != nil {
		log.Fatalf("Failed to get articles: %v", err)
	}

	fmt.Printf("Seattle Times feed has %d articles\n", len(articles))

	client := db.GetClient()
	ctx := context.Background()

	// Count UserArticle entries specifically for Seattle Times articles
	seattleTimesUserArticleCount := 0
	chunkSize := 100

	for i := 0; i < len(articles); i += chunkSize {
		end := i + chunkSize
		if end > len(articles) {
			end = len(articles)
		}

		chunk := articles[i:end]
		for _, article := range chunk {
			// Check if UserArticle exists for this user and article
			userArticleQuery := datastore.NewQuery("UserArticle").
				FilterField("user_id", "=", int64(user.ID)).
				FilterField("article_id", "=", int64(article.ID)).
				KeysOnly()

			keys, err := client.GetAll(ctx, userArticleQuery, nil)
			if err != nil {
				log.Printf("Warning: Failed to query for article %d: %v", article.ID, err)
				continue
			}
			seattleTimesUserArticleCount += len(keys)
		}

		fmt.Printf("Checked articles %d-%d, found %d UserArticle entries so far\n", i+1, end, seattleTimesUserArticleCount)
	}

	fmt.Printf("\nFinal count: %d UserArticle entries for Seattle Times feed\n", seattleTimesUserArticleCount)
	fmt.Printf("Expected: ~100 (based on MaxArticlesOnFeedAdd setting)\n")

	if seattleTimesUserArticleCount > 100 {
		fmt.Printf("❌ PROBLEM: Still have %d entries, should be ~100\n", seattleTimesUserArticleCount)
	} else {
		fmt.Printf("✅ GOOD: UserArticle count is within expected range\n")
	}
}