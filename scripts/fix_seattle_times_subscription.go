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

	// Check current UserArticle count
	userArticles, err := db.GetUserFeedArticles(user.ID, seattleTimesFeed.ID)
	if err != nil {
		log.Printf("Warning: Failed to get current user articles: %v", err)
	} else {
		fmt.Printf("Current UserArticle entries: %d\n", len(userArticles))
	}

	// Create feed service
	feedService := services.NewFeedService(db, nil)

	// Step 1: Unsubscribe from the feed (this should clean up UserArticle entries)
	fmt.Println("\n🗑️  Unsubscribing from Seattle Times feed...")
	err = feedService.UnsubscribeUserFromFeed(user.ID, seattleTimesFeed.ID)
	if err != nil {
		log.Fatalf("Failed to unsubscribe: %v", err)
	}
	fmt.Println("✅ Unsubscribed successfully")

	// Step 2: Re-subscribe to the feed (this should apply the 100 article limit)
	fmt.Println("\n📰 Re-subscribing to Seattle Times feed...")
	_, err = feedService.AddFeedForUser(user.ID, "https://www.seattletimes.com/feed/")
	if err != nil {
		log.Fatalf("Failed to re-subscribe: %v", err)
	}
	fmt.Println("✅ Re-subscribed successfully")

	// Step 3: Check the results
	fmt.Println("\n📊 Checking results...")
	userArticles, err = db.GetUserFeedArticles(user.ID, seattleTimesFeed.ID)
	if err != nil {
		log.Printf("Warning: Failed to get final user articles: %v", err)
	} else {
		unreadCount := 0
		for _, article := range userArticles {
			if !article.IsRead {
				unreadCount++
			}
		}
		fmt.Printf("Final UserArticle entries: %d\n", len(userArticles))
		fmt.Printf("Unread articles: %d\n", unreadCount)

		if unreadCount <= 100 {
			fmt.Printf("🎉 SUCCESS! Article limit is now working correctly!\n")
		} else {
			fmt.Printf("❌ ISSUE: Still have %d unread articles (expected ~100)\n", unreadCount)
		}
	}
}