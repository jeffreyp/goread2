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

	client := db.GetClient()
	ctx := context.Background()

	// Query first 10 UserArticle entries for this user to see their structure
	userArticleQuery := datastore.NewQuery("UserArticle").
		FilterField("user_id", "=", int64(user.ID)).
		Limit(10)

	var entities []database.UserArticleEntity
	keys, err := client.GetAll(ctx, userArticleQuery, &entities)
	if err != nil {
		log.Fatalf("Failed to query UserArticle: %v", err)
	}

	fmt.Printf("Found %d UserArticle entries for user %d\n", len(entities), user.ID)
	fmt.Printf("Sample UserArticle entries:\n")

	for i, entity := range entities {
		key := keys[i]
		fmt.Printf("  %d. Key: %s (ID: %d, Name: %s)\n", i+1, key.String(), key.ID, key.Name)
		fmt.Printf("      UserID: %d, ArticleID: %d, IsRead: %t\n", entity.UserID, entity.ArticleID, entity.IsRead)

		// Test if we can find this entry using the expected key format
		expectedKeyStr := fmt.Sprintf("%d_%d", entity.UserID, entity.ArticleID)
		expectedKey := datastore.NameKey("UserArticle", expectedKeyStr, nil)

		var testEntity database.UserArticleEntity
		err := client.Get(ctx, expectedKey, &testEntity)
		if err == nil {
			fmt.Printf("      ✅ Found using expected key format: %s\n", expectedKeyStr)
		} else {
			fmt.Printf("      ❌ NOT found using expected key format: %s (error: %v)\n", expectedKeyStr, err)
		}
	}

	// Also check total count
	countQuery := datastore.NewQuery("UserArticle").FilterField("user_id", "=", int64(user.ID))
	total, err := client.Count(ctx, countQuery)
	if err != nil {
		log.Printf("Failed to count UserArticle entries: %v", err)
	} else {
		fmt.Printf("\nTotal UserArticle entries for user: %d\n", total)
	}
}