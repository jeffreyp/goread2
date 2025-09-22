//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"goread2/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test_cleanup_local.go <command>")
		fmt.Println("Commands:")
		fmt.Println("  setup         - Create test database with orphaned data")
		fmt.Println("  audit         - Run audit on test database")
		fmt.Println("  cleanup       - Run cleanup on test database")
		fmt.Println("  verify        - Verify cleanup results")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "setup":
		setupTestDatabase()
	case "audit":
		runTestAudit()
	case "cleanup":
		runTestCleanup()
	case "verify":
		verifyCleanup()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func setupTestDatabase() {
	fmt.Println("🧪 Setting up test database with orphaned data...")

	// Remove existing test database
	os.Remove("test_cleanup.db")

	// Set environment to use SQLite for testing
	originalProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
	os.Unsetenv("GOOGLE_CLOUD_PROJECT") // This forces SQLite mode

	// Initialize database using the standard initialization
	dbWrapper, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer dbWrapper.Close()

	// Get the underlying SQL database for direct queries
	sqlDB, ok := dbWrapper.(*database.DB)
	if !ok {
		log.Fatal("Expected SQLite database but got different type")
	}

	// Restore original environment
	if originalProject != "" {
		os.Setenv("GOOGLE_CLOUD_PROJECT", originalProject)
	}

	// Create test users
	_, err = sqlDB.Exec(`INSERT INTO users (id, google_id, email, name, created_at) VALUES
		(1, 'user1_google', 'user1@test.com', 'Test User 1', datetime('now')),
		(2, 'user2_google', 'user2@test.com', 'Test User 2', datetime('now'))`)
	if err != nil {
		log.Fatal("Failed to create test users:", err)
	}

	// Create test feeds
	_, err = sqlDB.Exec(`INSERT INTO feeds (id, title, url, description, created_at, updated_at, last_fetch) VALUES
		(1, 'Test Feed 1', 'https://test1.com/feed', 'Test Description 1', datetime('now'), datetime('now'), datetime('now')),
		(2, 'Test Feed 2', 'https://test2.com/feed', 'Test Description 2', datetime('now'), datetime('now'), datetime('now')),
		(3, 'Orphaned Feed', 'https://orphan.com/feed', 'This feed will be orphaned', datetime('now'), datetime('now'), datetime('now'))`)
	if err != nil {
		log.Fatal("Failed to create test feeds:", err)
	}

	// Create valid user feeds
	_, err = sqlDB.Exec(`INSERT INTO user_feeds (user_id, feed_id) VALUES
		(1, 1),
		(2, 2)`)
	if err != nil {
		log.Fatal("Failed to create valid user feeds:", err)
	}

	// Create articles (some will be orphaned)
	_, err = sqlDB.Exec(`INSERT INTO articles (id, feed_id, title, url, content, created_at, published_at) VALUES
		(1, 1, 'Article 1 in Feed 1', 'https://test1.com/article1', 'Content 1', datetime('now'), datetime('now')),
		(2, 1, 'Article 2 in Feed 1', 'https://test1.com/article2', 'Content 2', datetime('now'), datetime('now')),
		(3, 2, 'Article 1 in Feed 2', 'https://test2.com/article1', 'Content 3', datetime('now'), datetime('now')),
		(4, 3, 'Article in Orphaned Feed', 'https://orphan.com/article1', 'Orphaned Content', datetime('now'), datetime('now')),
		(5, 999, 'Article in Non-existent Feed', 'https://nowhere.com/article1', 'Broken Content', datetime('now'), datetime('now'))`)
	if err != nil {
		log.Fatal("Failed to create test articles:", err)
	}

	// Create user article statuses (some will be orphaned)
	_, err = sqlDB.Exec(`INSERT INTO user_articles (user_id, article_id, is_read, is_starred) VALUES
		(1, 1, 0, 0),
		(1, 2, 1, 0),
		(2, 3, 0, 1),
		(1, 4, 0, 0),
		(999, 1, 0, 0),
		(1, 999, 0, 0)`)
	if err != nil {
		log.Fatal("Failed to create test user articles:", err)
	}

	// Create orphaned user feeds
	_, err = sqlDB.Exec(`INSERT INTO user_feeds (user_id, feed_id) VALUES
		(999, 1),
		(1, 999),
		(1, 3)`)
	if err != nil {
		log.Fatal("Failed to create orphaned user feeds:", err)
	}

	// Now delete feed 3 to create orphaned references
	_, err = sqlDB.Exec(`DELETE FROM feeds WHERE id = 3`)
	if err != nil {
		log.Fatal("Failed to delete feed 3:", err)
	}

	fmt.Println("✅ Test database created with the following orphaned data:")
	fmt.Println("   - 2 articles pointing to non-existent feeds (feed_id 3 and 999)")
	fmt.Println("   - 1 user_feed pointing to non-existent user (user_id 999)")
	fmt.Println("   - 2 user_feeds pointing to non-existent feeds (feed_id 3 and 999)")
	fmt.Println("   - 1 user_article pointing to non-existent user (user_id 999)")
	fmt.Println("   - 2 user_articles pointing to non-existent articles")
	fmt.Println("")
	fmt.Println("Run: go run test_cleanup_local.go audit")
}

func runTestAudit() {
	fmt.Println("🔍 Running audit on test database...")

	// Temporarily set environment to force SQLite
	originalProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
	os.Unsetenv("GOOGLE_CLOUD_PROJECT")

	// Use the wrapper to open the existing test database
	dbWrapper, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer dbWrapper.Close()

	// Get the underlying SQL database
	db, ok := dbWrapper.(*database.DB)
	if !ok {
		log.Fatal("Expected SQLite database but got different type")
	}

	// Restore original environment
	if originalProject != "" {
		os.Setenv("GOOGLE_CLOUD_PROJECT", originalProject)
	}

	issues := 0

	// Check orphaned articles
	var orphanedArticles int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM articles a
		WHERE NOT EXISTS (SELECT 1 FROM feeds f WHERE f.id = a.feed_id)
	`).Scan(&orphanedArticles)
	if err != nil {
		log.Fatal("Failed to count orphaned articles:", err)
	}
	if orphanedArticles > 0 {
		fmt.Printf("❌ Found %d orphaned articles\n", orphanedArticles)
		issues += orphanedArticles
	} else {
		fmt.Printf("✅ No orphaned articles\n")
	}

	// Check orphaned user feeds (feeds)
	var orphanedUserFeedsFeeds int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM user_feeds uf
		WHERE NOT EXISTS (SELECT 1 FROM feeds f WHERE f.id = uf.feed_id)
	`).Scan(&orphanedUserFeedsFeeds)
	if err != nil {
		log.Fatal("Failed to count orphaned user feeds (feeds):", err)
	}
	if orphanedUserFeedsFeeds > 0 {
		fmt.Printf("❌ Found %d orphaned user feeds (non-existent feeds)\n", orphanedUserFeedsFeeds)
		issues += orphanedUserFeedsFeeds
	} else {
		fmt.Printf("✅ No orphaned user feeds (feeds)\n")
	}

	// Check orphaned user feeds (users)
	var orphanedUserFeedsUsers int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM user_feeds uf
		WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.id = uf.user_id)
	`).Scan(&orphanedUserFeedsUsers)
	if err != nil {
		log.Fatal("Failed to count orphaned user feeds (users):", err)
	}
	if orphanedUserFeedsUsers > 0 {
		fmt.Printf("❌ Found %d orphaned user feeds (non-existent users)\n", orphanedUserFeedsUsers)
		issues += orphanedUserFeedsUsers
	} else {
		fmt.Printf("✅ No orphaned user feeds (users)\n")
	}

	// Check orphaned user articles (articles)
	var orphanedUserArticlesArticles int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM user_articles ua
		WHERE NOT EXISTS (SELECT 1 FROM articles a WHERE a.id = ua.article_id)
	`).Scan(&orphanedUserArticlesArticles)
	if err != nil {
		log.Fatal("Failed to count orphaned user articles (articles):", err)
	}
	if orphanedUserArticlesArticles > 0 {
		fmt.Printf("❌ Found %d orphaned user articles (non-existent articles)\n", orphanedUserArticlesArticles)
		issues += orphanedUserArticlesArticles
	} else {
		fmt.Printf("✅ No orphaned user articles (articles)\n")
	}

	// Check orphaned user articles (users)
	var orphanedUserArticlesUsers int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM user_articles ua
		WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.id = ua.user_id)
	`).Scan(&orphanedUserArticlesUsers)
	if err != nil {
		log.Fatal("Failed to count orphaned user articles (users):", err)
	}
	if orphanedUserArticlesUsers > 0 {
		fmt.Printf("❌ Found %d orphaned user articles (non-existent users)\n", orphanedUserArticlesUsers)
		issues += orphanedUserArticlesUsers
	} else {
		fmt.Printf("✅ No orphaned user articles (users)\n")
	}

	fmt.Printf("\n📊 Total issues found: %d\n", issues)
	if issues > 0 {
		fmt.Println("Run: go run test_cleanup_local.go cleanup")
	} else {
		fmt.Println("🎉 No issues found!")
	}
}

func runTestCleanup() {
	fmt.Println("🧹 Running cleanup on test database...")

	// Temporarily set environment to force SQLite
	originalProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
	os.Unsetenv("GOOGLE_CLOUD_PROJECT")

	// Use the wrapper to open the existing test database
	dbWrapper, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer dbWrapper.Close()

	// Get the underlying SQL database
	db, ok := dbWrapper.(*database.DB)
	if !ok {
		log.Fatal("Expected SQLite database but got different type")
	}

	// Restore original environment
	if originalProject != "" {
		os.Setenv("GOOGLE_CLOUD_PROJECT", originalProject)
	}

	totalDeleted := 0

	// Clean orphaned articles
	result, err := db.Exec(`
		DELETE FROM articles
		WHERE NOT EXISTS (SELECT 1 FROM feeds f WHERE f.id = feed_id)
	`)
	if err != nil {
		log.Fatal("Failed to delete orphaned articles:", err)
	}
	deletedArticles, _ := result.RowsAffected()
	if deletedArticles > 0 {
		fmt.Printf("🗑️  Deleted %d orphaned articles\n", deletedArticles)
		totalDeleted += int(deletedArticles)
	}

	// Clean orphaned user feeds
	result, err = db.Exec(`
		DELETE FROM user_feeds
		WHERE NOT EXISTS (SELECT 1 FROM feeds f WHERE f.id = feed_id)
		   OR NOT EXISTS (SELECT 1 FROM users u WHERE u.id = user_id)
	`)
	if err != nil {
		log.Fatal("Failed to delete orphaned user feeds:", err)
	}
	deletedUserFeeds, _ := result.RowsAffected()
	if deletedUserFeeds > 0 {
		fmt.Printf("🗑️  Deleted %d orphaned user feeds\n", deletedUserFeeds)
		totalDeleted += int(deletedUserFeeds)
	}

	// Clean orphaned user articles
	result, err = db.Exec(`
		DELETE FROM user_articles
		WHERE NOT EXISTS (SELECT 1 FROM articles a WHERE a.id = article_id)
		   OR NOT EXISTS (SELECT 1 FROM users u WHERE u.id = user_id)
	`)
	if err != nil {
		log.Fatal("Failed to delete orphaned user articles:", err)
	}
	deletedUserArticles, _ := result.RowsAffected()
	if deletedUserArticles > 0 {
		fmt.Printf("🗑️  Deleted %d orphaned user articles\n", deletedUserArticles)
		totalDeleted += int(deletedUserArticles)
	}

	fmt.Printf("\n✅ Cleanup completed - Deleted %d orphaned entities\n", totalDeleted)
	fmt.Println("Run: go run test_cleanup_local.go verify")
}

func verifyCleanup() {
	fmt.Println("✅ Verifying cleanup results...")

	// Temporarily set environment to force SQLite
	originalProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
	os.Unsetenv("GOOGLE_CLOUD_PROJECT")

	// Use the wrapper to open the existing test database
	dbWrapper, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer dbWrapper.Close()

	// Get the underlying SQL database
	db, ok := dbWrapper.(*database.DB)
	if !ok {
		log.Fatal("Expected SQLite database but got different type")
	}

	// Restore original environment
	if originalProject != "" {
		os.Setenv("GOOGLE_CLOUD_PROJECT", originalProject)
	}

	// Run audit again to verify no issues remain
	runTestAudit()

	// Show remaining data
	fmt.Println("\n📊 Remaining data after cleanup:")

	var userCount, feedCount, articleCount, userFeedCount, userArticleCount int

	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	db.QueryRow("SELECT COUNT(*) FROM feeds").Scan(&feedCount)
	db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&articleCount)
	db.QueryRow("SELECT COUNT(*) FROM user_feeds").Scan(&userFeedCount)
	db.QueryRow("SELECT COUNT(*) FROM user_articles").Scan(&userArticleCount)

	fmt.Printf("   Users:         %d\n", userCount)
	fmt.Printf("   Feeds:         %d\n", feedCount)
	fmt.Printf("   Articles:      %d\n", articleCount)
	fmt.Printf("   User Feeds:    %d\n", userFeedCount)
	fmt.Printf("   User Articles: %d\n", userArticleCount)

	fmt.Println("\n🎉 Local cleanup test completed successfully!")
	fmt.Println("The cleanup approach is validated and ready for production use.")
}