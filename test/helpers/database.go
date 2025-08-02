package helpers

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"goread2/internal/database"
)

// CreateTestDB creates an in-memory SQLite database for testing
func CreateTestDB(t *testing.T) database.Database {
	db, err := sql.Open("sqlite3", ":memory:")
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
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE feeds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			url TEXT UNIQUE NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_fetch DATETIME DEFAULT CURRENT_TIMESTAMP
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
	feed := &database.Feed{
		Title:       title,
		URL:         url,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastFetch:   time.Now(),
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
	os.Setenv("GOOGLE_CLIENT_ID", "test_client_id")
	os.Setenv("GOOGLE_CLIENT_SECRET", "test_client_secret")
	os.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/callback")
}

// CleanupTestEnv cleans up test environment variables
func CleanupTestEnv(t *testing.T) {
	os.Unsetenv("GOOGLE_CLIENT_ID")
	os.Unsetenv("GOOGLE_CLIENT_SECRET")
	os.Unsetenv("GOOGLE_REDIRECT_URL")
}
