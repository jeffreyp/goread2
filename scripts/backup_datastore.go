//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/datastore"
	"goread2/internal/database"
)

func main() {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT environment variable must be set")
	}

	backupSuffix := ""
	if len(os.Args) > 1 {
		backupSuffix = "_" + os.Args[1]
	}

	fmt.Printf("Creating backup of datastore for project: %s\n", projectID)
	fmt.Printf("Backup suffix: %s\n", backupSuffix)

	// Initialize datastore connection
	db, err := database.NewDatastoreDB(projectID)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	client := db.GetClient()

	// Backup each entity type
	err = backupEntities(ctx, client, "User", backupSuffix)
	if err != nil {
		log.Fatalf("Failed to backup User entities: %v", err)
	}

	err = backupEntities(ctx, client, "Feed", backupSuffix)
	if err != nil {
		log.Fatalf("Failed to backup Feed entities: %v", err)
	}

	err = backupEntities(ctx, client, "Article", backupSuffix)
	if err != nil {
		log.Fatalf("Failed to backup Article entities: %v", err)
	}

	err = backupEntities(ctx, client, "UserFeed", backupSuffix)
	if err != nil {
		log.Fatalf("Failed to backup UserFeed entities: %v", err)
	}

	err = backupEntities(ctx, client, "UserArticle", backupSuffix)
	if err != nil {
		log.Fatalf("Failed to backup UserArticle entities: %v", err)
	}

	fmt.Printf("✅ Backup completed successfully!\n")
	fmt.Printf("Backup entities created with suffix: %s\n", backupSuffix)
	fmt.Printf("To restore, use: go run restore_datastore.go %s\n", backupSuffix[1:]) // Remove the leading _
}

func backupEntities(ctx context.Context, client *datastore.Client, kind string, suffix string) error {
	fmt.Printf("Backing up %s entities...\n", kind)

	// Query all entities of this kind
	query := datastore.NewQuery(kind)
	var entities []interface{}

	// Use appropriate entity type
	switch kind {
	case "User":
		entities = make([]interface{}, 0)
		var userEntities []database.UserEntity
		keys, err := client.GetAll(ctx, query, &userEntities)
		if err != nil {
			return fmt.Errorf("failed to query %s entities: %w", kind, err)
		}

		// Convert to backup entities and save
		for i, entity := range userEntities {
			backupKey := datastore.IDKey(kind+"_backup"+suffix, keys[i].ID, nil)
			_, err = client.Put(ctx, backupKey, &entity)
			if err != nil {
				return fmt.Errorf("failed to backup %s entity %d: %w", kind, keys[i].ID, err)
			}
		}
		fmt.Printf("  Backed up %d %s entities\n", len(userEntities), kind)

	case "Feed":
		var feedEntities []database.FeedEntity
		keys, err := client.GetAll(ctx, query, &feedEntities)
		if err != nil {
			return fmt.Errorf("failed to query %s entities: %w", kind, err)
		}

		for i, entity := range feedEntities {
			backupKey := datastore.IDKey(kind+"_backup"+suffix, keys[i].ID, nil)
			_, err = client.Put(ctx, backupKey, &entity)
			if err != nil {
				return fmt.Errorf("failed to backup %s entity %d: %w", kind, keys[i].ID, err)
			}
		}
		fmt.Printf("  Backed up %d %s entities\n", len(feedEntities), kind)

	case "Article":
		var articleEntities []database.ArticleEntity
		keys, err := client.GetAll(ctx, query, &articleEntities)
		if err != nil {
			return fmt.Errorf("failed to query %s entities: %w", kind, err)
		}

		for i, entity := range articleEntities {
			backupKey := datastore.IDKey(kind+"_backup"+suffix, keys[i].ID, nil)
			_, err = client.Put(ctx, backupKey, &entity)
			if err != nil {
				return fmt.Errorf("failed to backup %s entity %d: %w", kind, keys[i].ID, err)
			}
		}
		fmt.Printf("  Backed up %d %s entities\n", len(articleEntities), kind)

	case "UserFeed":
		var userFeedEntities []database.UserFeedEntity
		keys, err := client.GetAll(ctx, query, &userFeedEntities)
		if err != nil {
			return fmt.Errorf("failed to query %s entities: %w", kind, err)
		}

		for i, entity := range userFeedEntities {
			// UserFeed uses name keys, so we need to create a backup with a modified name
			originalKey := keys[i]
			backupKeyName := fmt.Sprintf("%s_backup%s", originalKey.Name, suffix)
			backupKey := datastore.NameKey(kind+"_backup", backupKeyName, nil)
			_, err = client.Put(ctx, backupKey, &entity)
			if err != nil {
				return fmt.Errorf("failed to backup %s entity %s: %w", kind, originalKey.Name, err)
			}
		}
		fmt.Printf("  Backed up %d %s entities\n", len(userFeedEntities), kind)

	case "UserArticle":
		var userArticleEntities []database.UserArticleEntity
		keys, err := client.GetAll(ctx, query, &userArticleEntities)
		if err != nil {
			return fmt.Errorf("failed to query %s entities: %w", kind, err)
		}

		for i, entity := range userArticleEntities {
			// UserArticle uses name keys, so we need to create a backup with a modified name
			originalKey := keys[i]
			backupKeyName := fmt.Sprintf("%s_backup%s", originalKey.Name, suffix)
			backupKey := datastore.NameKey(kind+"_backup", backupKeyName, nil)
			_, err = client.Put(ctx, backupKey, &entity)
			if err != nil {
				return fmt.Errorf("failed to backup %s entity %s: %w", kind, originalKey.Name, err)
			}
		}
		fmt.Printf("  Backed up %d %s entities\n", len(userArticleEntities), kind)
	}

	return nil
}