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
	fmt.Printf("MaxArticlesOnFeedAdd: %d\n", user.MaxArticlesOnFeedAdd)

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

	// Simulate the markExistingArticlesAsUnreadForUser logic
	articles, err := db.GetArticles(seattleTimesFeed.ID)
	if err != nil {
		log.Fatalf("Failed to get articles: %v", err)
	}

	fmt.Printf("Total articles retrieved: %d\n", len(articles))

	if len(articles) > 0 {
		fmt.Printf("First article published: %v\n", articles[0].PublishedAt)
		fmt.Printf("Last article published: %v\n", articles[len(articles)-1].PublishedAt)
	}

	// Check the limit condition
	fmt.Printf("\nLimit condition check:\n")
	fmt.Printf("user.MaxArticlesOnFeedAdd > 0: %d > 0 = %t\n",
		user.MaxArticlesOnFeedAdd, user.MaxArticlesOnFeedAdd > 0)
	fmt.Printf("len(articles) > user.MaxArticlesOnFeedAdd: %d > %d = %t\n",
		len(articles), user.MaxArticlesOnFeedAdd, len(articles) > user.MaxArticlesOnFeedAdd)

	shouldLimit := user.MaxArticlesOnFeedAdd > 0 && len(articles) > user.MaxArticlesOnFeedAdd
	fmt.Printf("Should apply limit: %t\n", shouldLimit)

	if shouldLimit {
		fmt.Printf("Would limit to %d articles (indices 0-%d)\n",
			user.MaxArticlesOnFeedAdd, user.MaxArticlesOnFeedAdd-1)
		fmt.Printf("Limited articles would be from %v to %v\n",
			articles[user.MaxArticlesOnFeedAdd-1].PublishedAt, articles[0].PublishedAt)
	}

	// Check actual UserArticle entries
	fmt.Printf("\nChecking UserArticle entries...\n")

	// Count user articles for this feed
	userArticles, err := db.GetUserFeedArticles(user.ID, seattleTimesFeed.ID)
	if err != nil {
		log.Printf("Failed to get user feed articles: %v", err)
	} else {
		unreadCount := 0
		for _, article := range userArticles {
			if !article.IsRead {
				unreadCount++
			}
		}
		fmt.Printf("UserArticle entries for this feed: %d\n", len(userArticles))
		fmt.Printf("Unread count: %d\n", unreadCount)
	}
}