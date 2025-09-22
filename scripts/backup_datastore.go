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

	// Use appropriate entity type
	switch kind {
	case "User":
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
		// Try the current struct format first
		var userArticleEntities []database.UserArticleEntity
		keys, err := client.GetAll(ctx, query, &userArticleEntities)
		if err != nil {
			fmt.Printf("  Warning: Failed with current struct format, trying mixed format: %v\n", err)

			// Define mixed format struct - camelCase IDs, snake_case booleans
			type MixedUserArticleEntity struct {
				UserID    int64 `datastore:"UserID"`
				ArticleID int64 `datastore:"ArticleID"`
				IsRead    bool  `datastore:"is_read"`
				IsStarred bool  `datastore:"is_starred"`
			}

			var mixedEntities []MixedUserArticleEntity
			keys, err = client.GetAll(ctx, query, &mixedEntities)
			if err != nil {
				fmt.Printf("  Warning: Failed with mixed format, trying full camelCase: %v\n", err)

				// Try full camelCase format
				type FullCamelUserArticleEntity struct {
					UserID    int64 `datastore:"UserID"`
					ArticleID int64 `datastore:"ArticleID"`
					IsRead    bool  `datastore:"IsRead"`
					IsStarred bool  `datastore:"IsStarred"`
				}

				var camelEntities []FullCamelUserArticleEntity
				keys, err = client.GetAll(ctx, query, &camelEntities)
				if err != nil {
					fmt.Printf("  Warning: Full camelCase failed, trying partial camelCase: %v\n", err)

					// Try partial camelCase format - UserID camelCase, others snake_case
					type PartialCamelUserArticleEntity struct {
						UserID    int64 `datastore:"UserID"`
						ArticleID int64 `datastore:"article_id"`
						IsRead    bool  `datastore:"is_read"`
						IsStarred bool  `datastore:"is_starred"`
					}

					var partialEntities []PartialCamelUserArticleEntity
					keys, err = client.GetAll(ctx, query, &partialEntities)
					if err != nil {
						return fmt.Errorf("failed to query %s entities with all naming conventions: %w", kind, err)
					}

					// Convert and store as backup
					for i, partialEntity := range partialEntities {
						entity := database.UserArticleEntity{
							UserID:    partialEntity.UserID,
							ArticleID: partialEntity.ArticleID,
							IsRead:    partialEntity.IsRead,
							IsStarred: partialEntity.IsStarred,
						}

						originalKey := keys[i]
						backupKeyName := fmt.Sprintf("%s_backup%s", originalKey.Name, suffix)
						backupKey := datastore.NameKey(kind+"_backup", backupKeyName, nil)
						_, err = client.Put(ctx, backupKey, &entity)
						if err != nil {
							return fmt.Errorf("failed to backup %s entity %s: %w", kind, originalKey.Name, err)
						}
					}
					fmt.Printf("  Backed up %d %s entities (using partial camelCase format)\n", len(partialEntities), kind)
				} else {
					// Convert and store as backup
					for i, camelEntity := range camelEntities {
						entity := database.UserArticleEntity{
							UserID:    camelEntity.UserID,
							ArticleID: camelEntity.ArticleID,
							IsRead:    camelEntity.IsRead,
							IsStarred: camelEntity.IsStarred,
						}

						originalKey := keys[i]
						backupKeyName := fmt.Sprintf("%s_backup%s", originalKey.Name, suffix)
						backupKey := datastore.NameKey(kind+"_backup", backupKeyName, nil)
						_, err = client.Put(ctx, backupKey, &entity)
						if err != nil {
							return fmt.Errorf("failed to backup %s entity %s: %w", kind, originalKey.Name, err)
						}
					}
					fmt.Printf("  Backed up %d %s entities (using full camelCase format)\n", len(camelEntities), kind)
				}
			} else {
				// Convert and store mixed format as backup
				for i, mixedEntity := range mixedEntities {
					entity := database.UserArticleEntity{
						UserID:    mixedEntity.UserID,
						ArticleID: mixedEntity.ArticleID,
						IsRead:    mixedEntity.IsRead,
						IsStarred: mixedEntity.IsStarred,
					}

					originalKey := keys[i]
					backupKeyName := fmt.Sprintf("%s_backup%s", originalKey.Name, suffix)
					backupKey := datastore.NameKey(kind+"_backup", backupKeyName, nil)
					_, err = client.Put(ctx, backupKey, &entity)
					if err != nil {
						return fmt.Errorf("failed to backup %s entity %s: %w", kind, originalKey.Name, err)
					}
				}
				fmt.Printf("  Backed up %d %s entities (using mixed format)\n", len(mixedEntities), kind)
			}
		} else {
			// Store with current struct format
			for i, entity := range userArticleEntities {
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
	}

	return nil
}