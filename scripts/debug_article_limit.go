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

	// Get user by email - replace with your actual email
	userEmail := "jeffrey@jeffreypratt.org" // Update this to your email
	user, err := db.GetUserByEmail(userEmail)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	fmt.Printf("User ID: %d\n", user.ID)
	fmt.Printf("MaxArticlesOnFeedAdd setting: %d\n", user.MaxArticlesOnFeedAdd)

	// Get user's feeds
	feeds, err := db.GetUserFeeds(user.ID)
	if err != nil {
		log.Fatalf("Failed to get user feeds: %v", err)
	}

	fmt.Printf("Number of subscribed feeds: %d\n", len(feeds))

	// Show all feeds
	fmt.Printf("\nAll subscribed feeds:\n")
	for i, feed := range feeds {
		fmt.Printf("  %d. %s (%s)\n", i+1, feed.Title, feed.URL)
	}

	// Look for Seattle Times feed specifically (try multiple possible URLs)
	var seattleTimesFeed *database.Feed
	for _, feed := range feeds {
		if feed.URL == "https://www.seattletimes.com/feed/" ||
		   feed.URL == "https://seattletimes.com/feed/" ||
		   feed.URL == "http://www.seattletimes.com/feed/" ||
		   feed.URL == "http://seattletimes.com/feed/" {
			seattleTimesFeed = &feed
			break
		}
	}

	if seattleTimesFeed != nil {
		fmt.Printf("\nSeattle Times feed found (ID: %d)\n", seattleTimesFeed.ID)

		// Get all articles for this feed
		articles, err := db.GetArticles(seattleTimesFeed.ID)
		if err != nil {
			log.Fatalf("Failed to get articles: %v", err)
		}

		fmt.Printf("Total articles in Seattle Times feed: %d\n", len(articles))

		// Get unread counts
		unreadCounts, err := db.GetUserUnreadCounts(user.ID)
		if err != nil {
			log.Fatalf("Failed to get unread counts: %v", err)
		}

		if unreadCount, exists := unreadCounts[seattleTimesFeed.ID]; exists {
			fmt.Printf("Unread articles for Seattle Times: %d\n", unreadCount)
		} else {
			fmt.Printf("No unread count found for Seattle Times\n")
		}

		// Show first 5 articles (most recent)
		fmt.Printf("\nFirst 5 articles (most recent):\n")
		for i, article := range articles {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. %s (Published: %v)\n", i+1, article.Title, article.PublishedAt)
		}
	} else {
		fmt.Printf("Seattle Times feed not found in user's subscriptions\n")

		// Check if Seattle Times feed exists globally (but user isn't subscribed)
		allFeeds, err := db.GetFeeds()
		if err != nil {
			log.Printf("Failed to get all feeds: %v", err)
		} else {
			fmt.Printf("\nChecking all %d feeds in database for Seattle Times...\n", len(allFeeds))
			var globalSeattleFeed *database.Feed
			for _, feed := range allFeeds {
				if feed.URL == "https://www.seattletimes.com/feed/" ||
				   feed.URL == "https://seattletimes.com/feed/" ||
				   feed.URL == "http://www.seattletimes.com/feed/" ||
				   feed.URL == "http://seattletimes.com/feed/" {
					globalSeattleFeed = &feed
					fmt.Printf("Found Seattle Times feed globally (ID: %d, URL: %s)\n", feed.ID, feed.URL)

					// Check how many articles this feed has
					articles, err := db.GetArticles(feed.ID)
					if err != nil {
						log.Printf("Failed to get articles for global feed: %v", err)
					} else {
						fmt.Printf("Global Seattle Times feed has %d articles\n", len(articles))
						if len(articles) > 0 {
							fmt.Printf("Most recent article: %s (Published: %v)\n",
								articles[0].Title, articles[0].PublishedAt)
							fmt.Printf("Oldest article: %s (Published: %v)\n",
								articles[len(articles)-1].Title, articles[len(articles)-1].PublishedAt)
						}
					}
					break
				}
			}
			if globalSeattleFeed == nil {
				fmt.Printf("Seattle Times feed does not exist globally either\n")
			}
		}
	}
}