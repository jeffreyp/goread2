//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"goread2/internal/database"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run main.go <user_id> <feed_id> <num_articles>")
		fmt.Println("Example: go run main.go 91 1 150")
		fmt.Println("")
		fmt.Println("This will create <num_articles> test articles for the specified feed")
		fmt.Println("and ensure they are marked as unread for the specified user.")
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

	numArticles, err := strconv.Atoi(os.Args[3])
	if err != nil {
		log.Fatalf("Invalid number of articles: %v", err)
	}

	if numArticles < 1 || numArticles > 1000 {
		log.Fatalf("Number of articles must be between 1 and 1000")
	}

	fmt.Printf("Generating %d test articles for feed %d...\n", numArticles, feedID)

	// Initialize database connection
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify user exists
	user, err := db.GetUserByID(userID)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	if user == nil {
		log.Fatalf("User %d not found", userID)
	}

	fmt.Printf("Found user: %s (%s)\n", user.Name, user.Email)

	// Verify feed exists
	feeds, err := db.GetFeeds()
	if err != nil {
		log.Fatalf("Failed to get feeds: %v", err)
	}

	feedExists := false
	var feedTitle string
	for _, feed := range feeds {
		if feed.ID == feedID {
			feedExists = true
			feedTitle = feed.Title
			break
		}
	}

	if !feedExists {
		log.Fatalf("Feed %d not found", feedID)
	}

	fmt.Printf("Found feed: %s\n", feedTitle)

	// Subscribe user to feed if not already subscribed
	err = db.SubscribeUserToFeed(userID, feedID)
	if err != nil {
		log.Fatalf("Failed to subscribe user to feed: %v", err)
	}

	// Generate and insert articles
	now := time.Now()
	var createdArticles []database.Article

	for i := 0; i < numArticles; i++ {
		article := &database.Article{
			FeedID:      feedID,
			Title:       fmt.Sprintf("Test Article %d - %s", i+1, now.Format("2006-01-02 15:04:05")),
			URL:         fmt.Sprintf("https://example.com/test-article-%d-%d", feedID, now.Unix()+int64(i)),
			Content:     fmt.Sprintf("<p>This is test article content number %d. Generated at %s.</p><p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</p>", i+1, now.Format("2006-01-02 15:04:05")),
			Description: fmt.Sprintf("Test article description %d. This is a sample description for testing purposes.", i+1),
			Author:      "Test Author",
			PublishedAt: now.Add(time.Duration(-i) * time.Minute), // Each article is 1 minute older than the previous
			CreatedAt:   now,
		}

		err = db.AddArticle(article)
		if err != nil {
			log.Printf("Warning: Failed to create article %d: %v", i+1, err)
			continue
		}

		createdArticles = append(createdArticles, *article)

		if (i+1)%50 == 0 {
			fmt.Printf("Created %d/%d articles...\n", i+1, numArticles)
		}
	}

	fmt.Printf("Successfully created %d articles\n", len(createdArticles))

	// Mark all created articles as unread for the user
	// This is done by NOT creating user_article entries, which defaults to unread
	// However, to be explicit and handle cases where entries might already exist,
	// we'll use BatchSetUserArticleStatus
	if len(createdArticles) > 0 {
		fmt.Printf("Ensuring articles are unread for user %d...\n", userID)
		err = db.BatchSetUserArticleStatus(userID, createdArticles, false, false)
		if err != nil {
			log.Fatalf("Failed to set article status: %v", err)
		}
	}

	fmt.Printf("\n✅ Success!\n")
	fmt.Printf("   Created: %d test articles\n", len(createdArticles))
	fmt.Printf("   Feed: %s (ID: %d)\n", feedTitle, feedID)
	fmt.Printf("   User: %s (%s)\n", user.Name, user.Email)
	fmt.Printf("   All articles are unread and ready for testing.\n")
}
