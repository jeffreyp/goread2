package helpers

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

// CreateTestDB creates an in-memory SQLite database for testing
func CreateTestDB(t *testing.T) database.Database {
	db, err := sql.Open("sqlite3", ":memory:?_loc=auto")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	dbWrapper := &database.DB{DB: db}

	// Create tables
	if err := createTestTables(dbWrapper); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return dbWrapper
}

func createTestTables(db *database.DB) error {
	tables := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			google_id TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			avatar TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			subscription_status TEXT DEFAULT 'trial',
			subscription_id TEXT,
			trial_ends_at DATETIME,
			last_payment_date DATETIME,
			next_billing_date DATETIME,
			is_admin BOOLEAN DEFAULT 0,
			free_months_remaining INTEGER DEFAULT 0,
			max_articles_on_feed_add INTEGER DEFAULT 100
		)`,
		`CREATE TABLE feeds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			url TEXT UNIQUE NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_fetch DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_checked DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_had_new_content DATETIME DEFAULT CURRENT_TIMESTAMP,
			average_update_interval INTEGER DEFAULT 0,
			etag TEXT DEFAULT '',
			last_modified TEXT DEFAULT ''
		)`,
		`CREATE TABLE articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feed_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			url TEXT UNIQUE NOT NULL,
			content TEXT,
			description TEXT,
			author TEXT,
			published_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (feed_id) REFERENCES feeds (id) ON DELETE CASCADE
		)`,
		`CREATE TABLE user_feeds (
			user_id INTEGER NOT NULL,
			feed_id INTEGER NOT NULL,
			PRIMARY KEY (user_id, feed_id),
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
			FOREIGN KEY (feed_id) REFERENCES feeds (id) ON DELETE CASCADE
		)`,
		`CREATE TABLE user_articles (
			user_id INTEGER NOT NULL,
			article_id INTEGER NOT NULL,
			is_read BOOLEAN DEFAULT FALSE,
			is_starred BOOLEAN DEFAULT FALSE,
			PRIMARY KEY (user_id, article_id),
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
			FOREIGN KEY (article_id) REFERENCES articles (id) ON DELETE CASCADE
		)`,
		`CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
		)`,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// CreateTestUser creates a test user in the database
func CreateTestUser(t *testing.T, db database.Database, googleID, email, name string) *database.User {
	user := &database.User{
		GoogleID:  googleID,
		Email:     email,
		Name:      name,
		Avatar:    "https://example.com/avatar.jpg",
		CreatedAt: time.Now(),
	}

	if err := db.CreateUser(user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// CreateTestFeed creates a test feed in the database
func CreateTestFeed(t *testing.T, db database.Database, title, url, description string) *database.Feed {
	now := time.Now()
	feed := &database.Feed{
		Title:                 title,
		URL:                   url,
		Description:           description,
		CreatedAt:             now,
		UpdatedAt:             now,
		LastFetch:             now,
		LastChecked:           now,
		LastHadNewContent:     now,
		AverageUpdateInterval: 0,
	}

	if err := db.AddFeed(feed); err != nil {
		t.Fatalf("Failed to create test feed: %v", err)
	}

	return feed
}

// CreateTestArticle creates a test article in the database
func CreateTestArticle(t *testing.T, db database.Database, feedID int, title, url string) *database.Article {
	article := &database.Article{
		FeedID:      feedID,
		Title:       title,
		URL:         url,
		Content:     "Test content for " + title,
		Description: "Test description for " + title,
		Author:      "Test Author",
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	if err := db.AddArticle(article); err != nil {
		t.Fatalf("Failed to create test article: %v", err)
	}

	return article
}

// SetupTestEnv sets up environment variables for testing
func SetupTestEnv(t *testing.T) {
	_ = os.Setenv("GOOGLE_CLIENT_ID", "test_client_id")
	_ = os.Setenv("GOOGLE_CLIENT_SECRET", "test_client_secret")
	_ = os.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/callback")
}

// CleanupTestEnv cleans up test environment variables
func CleanupTestEnv(t *testing.T) {
	_ = os.Unsetenv("GOOGLE_CLIENT_ID")
	_ = os.Unsetenv("GOOGLE_CLIENT_SECRET")
	_ = os.Unsetenv("GOOGLE_REDIRECT_URL")
}

// CleanupTestUsers removes all test users from the main database
// This should be called at the start and end of integration tests to ensure clean state
func CleanupTestUsers(t *testing.T) {
	t.Helper()

	// Change to project root directory to ensure we use the same database
	originalDir, err := os.Getwd()
	if err != nil {
		t.Logf("Failed to get current directory for cleanup: %v", err)
		return
	}

	// Determine how many levels up to go based on current path
	projectRoot := "../.."
	if _, err := os.Stat(projectRoot + "/go.mod"); err != nil {
		// We might already be in test/integration or test/helpers
		// Try different path
		projectRoot = ".."
		if _, err := os.Stat(projectRoot + "/go.mod"); err != nil {
			// Try current directory
			projectRoot = "."
		}
	}

	err = os.Chdir(projectRoot)
	if err != nil {
		t.Logf("Failed to change to project root for cleanup: %v", err)
		return
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore original directory: %v", err)
		}
	}()

	db, err := database.InitDB()
	if err != nil {
		t.Logf("Failed to initialize database for cleanup: %v", err)
		return
	}
	defer func() { _ = db.Close() }()

	// Delete test users and admin tokens
	sqliteDB := db.(*database.DB)
	testEmails := []string{
		// Admin integration test users
		"main@example.com", "edge@example.com", "admin@test.com",
		"bootstrap@test.com", "lifecycle@test.com", "warning@test.com",
		"tokentest@test.com", "edgetest@test.com", "audituser@example.com",
		"auditadmin@test.com",
		// API test users (from api_test.go)
		"test@example.com", "test2@example.com", "test3@example.com",
		"user1@example.com", "user2@example.com",
		// Feature flag test users (from feature_flag_test.go)
		"api@example.com", "api2@example.com",
		"limit@example.com", "limit2@example.com",
	}

	for _, email := range testEmails {
		result, err := sqliteDB.Exec("DELETE FROM users WHERE email = ?", email)
		if err != nil {
			t.Logf("Failed to cleanup user %s: %v", email, err)
		} else {
			rowsAffected, _ := result.RowsAffected()
			if rowsAffected > 0 {
				t.Logf("Cleaned up user %s (rows affected: %d)", email, rowsAffected)
			}
		}
	}

	// Clean up admin tokens
	_, err = sqliteDB.Exec("DELETE FROM admin_tokens")
	if err != nil {
		t.Logf("Failed to cleanup admin tokens: %v", err)
	}

	// For bootstrap security tests, also remove admin privileges from all users
	// This ensures a clean state for security testing
	_, err = sqliteDB.Exec("UPDATE users SET is_admin = 0")
	if err != nil {
		t.Logf("Failed to remove admin privileges: %v", err)
	}
}

// CreateTestDatastoreDB creates a Datastore database for testing
// This requires the Datastore emulator to be running
func CreateTestDatastoreDB(t *testing.T) database.Database {
	projectID := "test-project-" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Set environment variable to use emulator
	originalHost := os.Getenv("DATASTORE_EMULATOR_HOST")
	if originalHost == "" {
		// Default emulator host
		_ = os.Setenv("DATASTORE_EMULATOR_HOST", "localhost:8081")
	}

	// Clean up environment after test
	t.Cleanup(func() {
		if originalHost == "" {
			_ = os.Unsetenv("DATASTORE_EMULATOR_HOST")
		} else {
			_ = os.Setenv("DATASTORE_EMULATOR_HOST", originalHost)
		}
	})

	db, err := database.NewDatastoreDB(projectID)
	if err != nil {
		t.Skipf("Datastore emulator not available: %v", err)
	}

	// Clean up database after test
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
