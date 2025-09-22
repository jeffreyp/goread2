//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"goread2/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run database_cleanup.go <command> [options]")
		fmt.Println("Commands:")
		fmt.Println("  audit                    - Run audit to identify integrity issues")
		fmt.Println("  cleanup --dry-run        - Show what would be cleaned (safe)")
		fmt.Println("  cleanup --execute        - Execute cleanup (DESTRUCTIVE)")
		fmt.Println("  backup [suffix]          - Create backup before cleanup")
		fmt.Println("  stats                    - Show database statistics")
		fmt.Println("")
		fmt.Println("IMPORTANT: Always run 'backup' and 'audit' before any cleanup!")
		os.Exit(1)
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT environment variable must be set")
	}

	// Initialize database connection
	db, err := database.NewDatastoreDB(projectID)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	client := db.GetClient()

	command := os.Args[1]

	switch command {
	case "audit":
		runAudit(ctx, client)
	case "cleanup":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run database_cleanup.go cleanup [--dry-run|--execute]")
			os.Exit(1)
		}
		dryRun := os.Args[2] == "--dry-run"
		runCleanup(ctx, client, dryRun)
	case "backup":
		suffix := ""
		if len(os.Args) > 2 {
			suffix = os.Args[2]
		}
		runBackup(ctx, client, suffix)
	case "stats":
		showStats(ctx, client)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func runAudit(ctx context.Context, client *datastore.Client) {
	fmt.Println("🔍 Running database integrity audit...")
	fmt.Println(strings.Repeat("=", 60))

	issues := make([]string, 0)

	// Check for orphaned articles (articles with non-existent feeds)
	orphanedArticles, err := findOrphanedArticles(ctx, client)
	if err != nil {
		log.Printf("Error checking orphaned articles: %v", err)
	} else if len(orphanedArticles) > 0 {
		issue := fmt.Sprintf("Found %d orphaned articles (articles with non-existent feeds)", len(orphanedArticles))
		issues = append(issues, issue)
		fmt.Printf("❌ %s\n", issue)
	} else {
		fmt.Printf("✅ No orphaned articles found\n")
	}

	// Check for orphaned user feeds (user feeds pointing to non-existent feeds)
	orphanedUserFeeds, err := findOrphanedUserFeeds(ctx, client)
	if err != nil {
		log.Printf("Error checking orphaned user feeds: %v", err)
	} else if len(orphanedUserFeeds) > 0 {
		issue := fmt.Sprintf("Found %d orphaned user feeds (pointing to non-existent feeds)", len(orphanedUserFeeds))
		issues = append(issues, issue)
		fmt.Printf("❌ %s\n", issue)
	} else {
		fmt.Printf("✅ No orphaned user feeds found\n")
	}

	// Check for orphaned user feeds (user feeds pointing to non-existent users)
	orphanedUserFeedsUsers, err := findOrphanedUserFeedsUsers(ctx, client)
	if err != nil {
		log.Printf("Error checking orphaned user feeds (users): %v", err)
	} else if len(orphanedUserFeedsUsers) > 0 {
		issue := fmt.Sprintf("Found %d orphaned user feeds (pointing to non-existent users)", len(orphanedUserFeedsUsers))
		issues = append(issues, issue)
		fmt.Printf("❌ %s\n", issue)
	} else {
		fmt.Printf("✅ No orphaned user feeds (users) found\n")
	}

	// Check for orphaned user articles (user articles pointing to non-existent articles)
	orphanedUserArticles, err := findOrphanedUserArticles(ctx, client)
	if err != nil {
		log.Printf("Error checking orphaned user articles: %v", err)
	} else if len(orphanedUserArticles) > 0 {
		issue := fmt.Sprintf("Found %d orphaned user articles (pointing to non-existent articles)", len(orphanedUserArticles))
		issues = append(issues, issue)
		fmt.Printf("❌ %s\n", issue)
	} else {
		fmt.Printf("✅ No orphaned user articles found\n")
	}

	// Check for orphaned user articles (user articles pointing to non-existent users)
	orphanedUserArticlesUsers, err := findOrphanedUserArticlesUsers(ctx, client)
	if err != nil {
		log.Printf("Error checking orphaned user articles (users): %v", err)
	} else if len(orphanedUserArticlesUsers) > 0 {
		issue := fmt.Sprintf("Found %d orphaned user articles (pointing to non-existent users)", len(orphanedUserArticlesUsers))
		issues = append(issues, issue)
		fmt.Printf("❌ %s\n", issue)
	} else {
		fmt.Printf("✅ No orphaned user articles (users) found\n")
	}

	// Check for feeds with no user associations
	unusedFeeds, err := findUnusedFeeds(ctx, client)
	if err != nil {
		log.Printf("Error checking unused feeds: %v", err)
	} else if len(unusedFeeds) > 0 {
		issue := fmt.Sprintf("Found %d unused feeds (feeds with no user associations)", len(unusedFeeds))
		issues = append(issues, issue)
		fmt.Printf("⚠️  %s\n", issue)
	} else {
		fmt.Printf("✅ No unused feeds found\n")
	}

	fmt.Println(strings.Repeat("=", 60))
	if len(issues) == 0 {
		fmt.Println("🎉 Database integrity audit PASSED - No issues found!")
	} else {
		fmt.Printf("⚠️  Database integrity audit found %d issues:\n", len(issues))
		for i, issue := range issues {
			fmt.Printf("  %d. %s\n", i+1, issue)
		}
		fmt.Println("\nRecommendation: Run cleanup to fix these issues")
		fmt.Println("  1. First run: go run database_cleanup.go backup pre_cleanup")
		fmt.Println("  2. Then run:  go run database_cleanup.go cleanup --dry-run")
		fmt.Println("  3. Finally:   go run database_cleanup.go cleanup --execute")
	}
}

func runCleanup(ctx context.Context, client *datastore.Client, dryRun bool) {
	if dryRun {
		fmt.Println("🧪 Running cleanup in DRY RUN mode (no changes will be made)")
	} else {
		fmt.Println("🔥 Running cleanup in EXECUTE mode (DESTRUCTIVE CHANGES)")
		fmt.Println("⚠️  This will permanently delete orphaned data!")
		fmt.Print("Are you sure you want to continue? (type 'yes' to confirm): ")

		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Cleanup cancelled.")
			return
		}
	}

	fmt.Println(strings.Repeat("=", 60))

	totalDeleted := 0

	// Clean orphaned articles
	orphanedArticles, err := findOrphanedArticles(ctx, client)
	if err != nil {
		log.Printf("Error finding orphaned articles: %v", err)
	} else if len(orphanedArticles) > 0 {
		fmt.Printf("🗑️  Cleaning %d orphaned articles...\n", len(orphanedArticles))
		if !dryRun {
			err = deleteOrphanedArticles(ctx, client, orphanedArticles)
			if err != nil {
				log.Printf("Error deleting orphaned articles: %v", err)
			} else {
				totalDeleted += len(orphanedArticles)
				fmt.Printf("✅ Deleted %d orphaned articles\n", len(orphanedArticles))
			}
		}
	}

	// Clean orphaned user feeds
	orphanedUserFeeds, err := findOrphanedUserFeeds(ctx, client)
	if err != nil {
		log.Printf("Error finding orphaned user feeds: %v", err)
	} else if len(orphanedUserFeeds) > 0 {
		fmt.Printf("🗑️  Cleaning %d orphaned user feeds...\n", len(orphanedUserFeeds))
		if !dryRun {
			err = deleteOrphanedUserFeeds(ctx, client, orphanedUserFeeds)
			if err != nil {
				log.Printf("Error deleting orphaned user feeds: %v", err)
			} else {
				totalDeleted += len(orphanedUserFeeds)
				fmt.Printf("✅ Deleted %d orphaned user feeds\n", len(orphanedUserFeeds))
			}
		}
	}

	// Clean orphaned user feeds (users)
	orphanedUserFeedsUsers, err := findOrphanedUserFeedsUsers(ctx, client)
	if err != nil {
		log.Printf("Error finding orphaned user feeds (users): %v", err)
	} else if len(orphanedUserFeedsUsers) > 0 {
		fmt.Printf("🗑️  Cleaning %d orphaned user feeds (users)...\n", len(orphanedUserFeedsUsers))
		if !dryRun {
			err = deleteOrphanedUserFeeds(ctx, client, orphanedUserFeedsUsers)
			if err != nil {
				log.Printf("Error deleting orphaned user feeds (users): %v", err)
			} else {
				totalDeleted += len(orphanedUserFeedsUsers)
				fmt.Printf("✅ Deleted %d orphaned user feeds (users)\n", len(orphanedUserFeedsUsers))
			}
		}
	}

	// Clean orphaned user articles
	orphanedUserArticles, err := findOrphanedUserArticles(ctx, client)
	if err != nil {
		log.Printf("Error finding orphaned user articles: %v", err)
	} else if len(orphanedUserArticles) > 0 {
		fmt.Printf("🗑️  Cleaning %d orphaned user articles...\n", len(orphanedUserArticles))
		if !dryRun {
			err = deleteOrphanedUserArticles(ctx, client, orphanedUserArticles)
			if err != nil {
				log.Printf("Error deleting orphaned user articles: %v", err)
			} else {
				totalDeleted += len(orphanedUserArticles)
				fmt.Printf("✅ Deleted %d orphaned user articles\n", len(orphanedUserArticles))
			}
		}
	}

	// Clean orphaned user articles (users)
	orphanedUserArticlesUsers, err := findOrphanedUserArticlesUsers(ctx, client)
	if err != nil {
		log.Printf("Error finding orphaned user articles (users): %v", err)
	} else if len(orphanedUserArticlesUsers) > 0 {
		fmt.Printf("🗑️  Cleaning %d orphaned user articles (users)...\n", len(orphanedUserArticlesUsers))
		if !dryRun {
			err = deleteOrphanedUserArticles(ctx, client, orphanedUserArticlesUsers)
			if err != nil {
				log.Printf("Error deleting orphaned user articles (users): %v", err)
			} else {
				totalDeleted += len(orphanedUserArticlesUsers)
				fmt.Printf("✅ Deleted %d orphaned user articles (users)\n", len(orphanedUserArticlesUsers))
			}
		}
	}

	fmt.Println(strings.Repeat("=", 60))
	if dryRun {
		fmt.Printf("🧪 DRY RUN completed - Would have deleted %d entities\n", totalDeleted)
		fmt.Println("To execute the cleanup, run: go run database_cleanup.go cleanup --execute")
	} else {
		fmt.Printf("🎉 Cleanup completed - Deleted %d orphaned entities\n", totalDeleted)
		fmt.Println("Database integrity should now be restored!")
		fmt.Println("Recommendation: Run audit again to verify cleanup was successful")
	}
}

func runBackup(ctx context.Context, client *datastore.Client, suffix string) {
	if suffix == "" {
		suffix = time.Now().Format("20060102_150405")
	}

	fmt.Printf("💾 Creating backup with suffix: %s\n", suffix)

	// This would call the backup script
	fmt.Println("Use: go run backup_datastore.go " + suffix)
}

func showStats(ctx context.Context, client *datastore.Client) {
	fmt.Println("📊 Database Statistics")
	fmt.Println(strings.Repeat("=", 40))

	// Count each entity type
	entities := []string{"User", "Feed", "Article", "UserFeed", "UserArticle"}

	for _, entity := range entities {
		query := datastore.NewQuery(entity)
		count, err := client.Count(ctx, query)
		if err != nil {
			fmt.Printf("%-12s: Error counting (%v)\n", entity, err)
		} else {
			fmt.Printf("%-12s: %d\n", entity, count)
		}
	}
}

// Helper functions for finding orphaned data

func findOrphanedArticles(ctx context.Context, client *datastore.Client) ([]*datastore.Key, error) {
	// Get all feeds first
	feedQuery := datastore.NewQuery("Feed").KeysOnly()
	feedKeys, err := client.GetAll(ctx, feedQuery, nil)
	if err != nil {
		return nil, err
	}

	feedIDMap := make(map[int64]bool)
	for _, key := range feedKeys {
		feedIDMap[key.ID] = true
	}

	// Get all articles
	articleQuery := datastore.NewQuery("Article")
	var articles []database.ArticleEntity
	articleKeys, err := client.GetAll(ctx, articleQuery, &articles)
	if err != nil {
		return nil, err
	}

	var orphanedKeys []*datastore.Key
	for i, article := range articles {
		if !feedIDMap[article.FeedID] {
			orphanedKeys = append(orphanedKeys, articleKeys[i])
		}
	}

	return orphanedKeys, nil
}

func findOrphanedUserFeeds(ctx context.Context, client *datastore.Client) ([]*datastore.Key, error) {
	// Get all feeds first
	feedQuery := datastore.NewQuery("Feed").KeysOnly()
	feedKeys, err := client.GetAll(ctx, feedQuery, nil)
	if err != nil {
		return nil, err
	}

	feedIDMap := make(map[int64]bool)
	for _, key := range feedKeys {
		feedIDMap[key.ID] = true
	}

	// Get all user feeds
	userFeedQuery := datastore.NewQuery("UserFeed")
	var userFeeds []database.UserFeedEntity
	userFeedKeys, err := client.GetAll(ctx, userFeedQuery, &userFeeds)
	if err != nil {
		return nil, err
	}

	var orphanedKeys []*datastore.Key
	for i, userFeed := range userFeeds {
		if !feedIDMap[userFeed.FeedID] {
			orphanedKeys = append(orphanedKeys, userFeedKeys[i])
		}
	}

	return orphanedKeys, nil
}

func findOrphanedUserFeedsUsers(ctx context.Context, client *datastore.Client) ([]*datastore.Key, error) {
	// Get all users first
	userQuery := datastore.NewQuery("User").KeysOnly()
	userKeys, err := client.GetAll(ctx, userQuery, nil)
	if err != nil {
		return nil, err
	}

	userIDMap := make(map[int64]bool)
	for _, key := range userKeys {
		userIDMap[key.ID] = true
	}

	// Get all user feeds
	userFeedQuery := datastore.NewQuery("UserFeed")
	var userFeeds []database.UserFeedEntity
	userFeedKeys, err := client.GetAll(ctx, userFeedQuery, &userFeeds)
	if err != nil {
		return nil, err
	}

	var orphanedKeys []*datastore.Key
	for i, userFeed := range userFeeds {
		if !userIDMap[userFeed.UserID] {
			orphanedKeys = append(orphanedKeys, userFeedKeys[i])
		}
	}

	return orphanedKeys, nil
}

func findOrphanedUserArticles(ctx context.Context, client *datastore.Client) ([]*datastore.Key, error) {
	// Get all articles first
	articleQuery := datastore.NewQuery("Article").KeysOnly()
	articleKeys, err := client.GetAll(ctx, articleQuery, nil)
	if err != nil {
		return nil, err
	}

	articleIDMap := make(map[int64]bool)
	for _, key := range articleKeys {
		articleIDMap[key.ID] = true
	}

	// Get all user articles - try multiple field naming formats
	userArticleQuery := datastore.NewQuery("UserArticle")
	var userArticles []database.UserArticleEntity
	userArticleKeys, err := client.GetAll(ctx, userArticleQuery, &userArticles)
	if err != nil {
		fmt.Printf("Warning: UserArticle struct query failed, trying mixed format: %v\n", err)

		// Define mixed format struct - camelCase IDs, snake_case booleans
		type MixedUserArticleEntity struct {
			UserID    int64 `datastore:"UserID"`
			ArticleID int64 `datastore:"ArticleID"`
			IsRead    bool  `datastore:"is_read"`
			IsStarred bool  `datastore:"is_starred"`
		}

		var mixedEntities []MixedUserArticleEntity
		userArticleKeys, err = client.GetAll(ctx, userArticleQuery, &mixedEntities)
		if err != nil {
			fmt.Printf("Warning: Mixed format failed, trying full camelCase: %v\n", err)

			// Try full camelCase format
			type FullCamelUserArticleEntity struct {
				UserID    int64 `datastore:"UserID"`
				ArticleID int64 `datastore:"ArticleID"`
				IsRead    bool  `datastore:"IsRead"`
				IsStarred bool  `datastore:"IsStarred"`
			}

			var camelEntities []FullCamelUserArticleEntity
			userArticleKeys, err = client.GetAll(ctx, userArticleQuery, &camelEntities)
			if err != nil {
				fmt.Printf("Warning: Full camelCase failed, trying partial camelCase: %v\n", err)

				// Try partial camelCase format - UserID camelCase, others snake_case
				type PartialCamelUserArticleEntity struct {
					UserID    int64 `datastore:"UserID"`
					ArticleID int64 `datastore:"article_id"`
					IsRead    bool  `datastore:"is_read"`
					IsStarred bool  `datastore:"is_starred"`
				}

				var partialEntities []PartialCamelUserArticleEntity
				userArticleKeys, err = client.GetAll(ctx, userArticleQuery, &partialEntities)
				if err != nil {
					return nil, fmt.Errorf("failed to query UserArticle entities with all naming conventions: %w", err)
				}

				// Process partial camelCase entities
				var orphanedKeys []*datastore.Key
				for i, partialEntity := range partialEntities {
					if !articleIDMap[partialEntity.ArticleID] {
						orphanedKeys = append(orphanedKeys, userArticleKeys[i])
					}
				}
				return orphanedKeys, nil
			}

			// Process full camelCase entities
			var orphanedKeys []*datastore.Key
			for i, camelEntity := range camelEntities {
				if !articleIDMap[camelEntity.ArticleID] {
					orphanedKeys = append(orphanedKeys, userArticleKeys[i])
				}
			}
			return orphanedKeys, nil
		}

		// Process mixed format entities
		var orphanedKeys []*datastore.Key
		for i, mixedEntity := range mixedEntities {
			if !articleIDMap[mixedEntity.ArticleID] {
				orphanedKeys = append(orphanedKeys, userArticleKeys[i])
			}
		}
		return orphanedKeys, nil
	}

	var orphanedKeys []*datastore.Key
	for i, userArticle := range userArticles {
		if !articleIDMap[userArticle.ArticleID] {
			orphanedKeys = append(orphanedKeys, userArticleKeys[i])
		}
	}

	return orphanedKeys, nil
}

func findOrphanedUserArticlesUsers(ctx context.Context, client *datastore.Client) ([]*datastore.Key, error) {
	// Get all users first
	userQuery := datastore.NewQuery("User").KeysOnly()
	userKeys, err := client.GetAll(ctx, userQuery, nil)
	if err != nil {
		return nil, err
	}

	userIDMap := make(map[int64]bool)
	for _, key := range userKeys {
		userIDMap[key.ID] = true
	}

	// Get all user articles - try multiple field naming formats
	userArticleQuery := datastore.NewQuery("UserArticle")
	var userArticles []database.UserArticleEntity
	userArticleKeys, err := client.GetAll(ctx, userArticleQuery, &userArticles)
	if err != nil {
		fmt.Printf("Warning: UserArticle struct query failed, trying mixed format: %v\n", err)

		// Define mixed format struct - camelCase IDs, snake_case booleans
		type MixedUserArticleEntity struct {
			UserID    int64 `datastore:"UserID"`
			ArticleID int64 `datastore:"ArticleID"`
			IsRead    bool  `datastore:"is_read"`
			IsStarred bool  `datastore:"is_starred"`
		}

		var mixedEntities []MixedUserArticleEntity
		userArticleKeys, err = client.GetAll(ctx, userArticleQuery, &mixedEntities)
		if err != nil {
			fmt.Printf("Warning: Mixed format failed, trying full camelCase: %v\n", err)

			// Try full camelCase format
			type FullCamelUserArticleEntity struct {
				UserID    int64 `datastore:"UserID"`
				ArticleID int64 `datastore:"ArticleID"`
				IsRead    bool  `datastore:"IsRead"`
				IsStarred bool  `datastore:"IsStarred"`
			}

			var camelEntities []FullCamelUserArticleEntity
			userArticleKeys, err = client.GetAll(ctx, userArticleQuery, &camelEntities)
			if err != nil {
				fmt.Printf("Warning: Full camelCase failed, trying partial camelCase: %v\n", err)

				// Try partial camelCase format - UserID camelCase, others snake_case
				type PartialCamelUserArticleEntity struct {
					UserID    int64 `datastore:"UserID"`
					ArticleID int64 `datastore:"article_id"`
					IsRead    bool  `datastore:"is_read"`
					IsStarred bool  `datastore:"is_starred"`
				}

				var partialEntities []PartialCamelUserArticleEntity
				userArticleKeys, err = client.GetAll(ctx, userArticleQuery, &partialEntities)
				if err != nil {
					return nil, fmt.Errorf("failed to query UserArticle entities with all naming conventions: %w", err)
				}

				// Process partial camelCase entities
				var orphanedKeys []*datastore.Key
				for i, partialEntity := range partialEntities {
					if !userIDMap[partialEntity.UserID] {
						orphanedKeys = append(orphanedKeys, userArticleKeys[i])
					}
				}
				return orphanedKeys, nil
			}

			// Process full camelCase entities
			var orphanedKeys []*datastore.Key
			for i, camelEntity := range camelEntities {
				if !userIDMap[camelEntity.UserID] {
					orphanedKeys = append(orphanedKeys, userArticleKeys[i])
				}
			}
			return orphanedKeys, nil
		}

		// Process mixed format entities
		var orphanedKeys []*datastore.Key
		for i, mixedEntity := range mixedEntities {
			if !userIDMap[mixedEntity.UserID] {
				orphanedKeys = append(orphanedKeys, userArticleKeys[i])
			}
		}
		return orphanedKeys, nil
	}

	var orphanedKeys []*datastore.Key
	for i, userArticle := range userArticles {
		if !userIDMap[userArticle.UserID] {
			orphanedKeys = append(orphanedKeys, userArticleKeys[i])
		}
	}

	return orphanedKeys, nil
}

func findUnusedFeeds(ctx context.Context, client *datastore.Client) ([]*datastore.Key, error) {
	// Get all feeds
	feedQuery := datastore.NewQuery("Feed").KeysOnly()
	feedKeys, err := client.GetAll(ctx, feedQuery, nil)
	if err != nil {
		return nil, err
	}

	// Get all user feeds
	userFeedQuery := datastore.NewQuery("UserFeed")
	var userFeeds []database.UserFeedEntity
	_, err = client.GetAll(ctx, userFeedQuery, &userFeeds)
	if err != nil {
		return nil, err
	}

	// Build map of used feed IDs
	usedFeedIDs := make(map[int64]bool)
	for _, userFeed := range userFeeds {
		usedFeedIDs[userFeed.FeedID] = true
	}

	// Find unused feeds
	var unusedKeys []*datastore.Key
	for _, key := range feedKeys {
		if !usedFeedIDs[key.ID] {
			unusedKeys = append(unusedKeys, key)
		}
	}

	return unusedKeys, nil
}

// Helper functions for deleting orphaned data

func deleteOrphanedArticles(ctx context.Context, client *datastore.Client, keys []*datastore.Key) error {
	if len(keys) == 0 {
		return nil
	}

	// Delete in batches to avoid datastore limits
	batchSize := 100
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		err := client.DeleteMulti(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to delete batch of articles: %w", err)
		}
	}

	return nil
}

func deleteOrphanedUserFeeds(ctx context.Context, client *datastore.Client, keys []*datastore.Key) error {
	if len(keys) == 0 {
		return nil
	}

	// Delete in batches to avoid datastore limits
	batchSize := 100
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		err := client.DeleteMulti(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to delete batch of user feeds: %w", err)
		}
	}

	return nil
}

func deleteOrphanedUserArticles(ctx context.Context, client *datastore.Client, keys []*datastore.Key) error {
	if len(keys) == 0 {
		return nil
	}

	// Delete in batches to avoid datastore limits
	batchSize := 100
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		err := client.DeleteMulti(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to delete batch of user articles: %w", err)
		}
	}

	return nil
}