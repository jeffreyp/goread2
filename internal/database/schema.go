package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ArticlePaginationResult contains paginated articles and a cursor for the next page
type ArticlePaginationResult struct {
	Articles   []Article
	NextCursor string // Empty string means no more results
}

type Database interface {
	// User methods
	CreateUser(user *User) error
	GetUserByGoogleID(googleID string) (*User, error)
	GetUserByID(userID int) (*User, error)
	UpdateUserSubscription(userID int, status, subscriptionID string, lastPaymentDate, nextBillingDate time.Time) error
	IsUserSubscriptionActive(userID int) (bool, error)
	GetUserFeedCount(userID int) (int, error)
	UpdateUserMaxArticlesOnFeedAdd(userID int, maxArticles int) error

	// Admin methods
	SetUserAdmin(userID int, isAdmin bool) error
	GrantFreeMonths(userID int, months int) error
	GetUserByEmail(email string) (*User, error)

	// Feed methods
	AddFeed(feed *Feed) error
	UpdateFeed(feed *Feed) error
	UpdateFeedTracking(feedID int, lastChecked, lastHadNewContent time.Time, averageUpdateInterval int) error
	GetFeeds() ([]Feed, error)
	GetFeedByURL(url string) (*Feed, error)
	GetUserFeeds(userID int) ([]Feed, error)
	GetAllUserFeeds() ([]Feed, error)
	DeleteFeed(id int) error
	SubscribeUserToFeed(userID, feedID int) error
	UnsubscribeUserFromFeed(userID, feedID int) error

	// Article methods
	AddArticle(article *Article) error
	GetArticles(feedID int) ([]Article, error)
	FindArticleByURL(url string) (*Article, error)
	GetUserArticles(userID int) ([]Article, error)
	GetUserArticlesPaginated(userID int, limit int, cursor string, unreadOnly bool) (*ArticlePaginationResult, error)
	GetUserFeedArticles(userID, feedID int) ([]Article, error)

	// User article status methods
	GetUserArticleStatus(userID, articleID int) (*UserArticle, error)
	SetUserArticleStatus(userID, articleID int, isRead, isStarred bool) error
	BatchSetUserArticleStatus(userID int, articles []Article, isRead, isStarred bool) error
	MarkUserArticleRead(userID, articleID int, isRead bool) error
	ToggleUserArticleStar(userID, articleID int) error
	GetUserUnreadCounts(userID int) (map[int]int, error)
	GetAccountStats(userID int) (map[string]interface{}, error)
	CleanupOrphanedUserArticles(olderThanDays int) (int, error)

	// Session methods
	CreateSession(session *Session) error
	GetSession(sessionID string) (*Session, error)
	DeleteSession(sessionID string) error
	DeleteExpiredSessions() error

	// Audit log methods
	CreateAuditLog(log *AuditLog) error
	GetAuditLogs(limit, offset int, filters map[string]interface{}) ([]AuditLog, error)

	UpdateFeedLastFetch(feedID int, lastFetch time.Time) error
	Close() error
}

type DB struct {
	*sql.DB
}

type User struct {
	ID                   int       `json:"id"`
	GoogleID             string    `json:"google_id"`
	Email                string    `json:"email"`
	Name                 string    `json:"name"`
	Avatar               string    `json:"avatar"`
	CreatedAt            time.Time `json:"created_at"`
	SubscriptionStatus   string    `json:"subscription_status"`      // 'trial', 'active', 'cancelled', 'expired', 'admin'
	SubscriptionID       string    `json:"subscription_id"`          // Stripe subscription ID
	TrialEndsAt          time.Time `json:"trial_ends_at"`            // When free trial expires
	LastPaymentDate      time.Time `json:"last_payment_date"`        // Last successful payment
	NextBillingDate      time.Time `json:"next_billing_date"`        // Next billing date for active subscriptions
	IsAdmin              bool      `json:"is_admin"`                 // Admin users bypass subscription limits
	FreeMonthsRemaining  int       `json:"free_months_remaining"`    // Additional free months granted
	MaxArticlesOnFeedAdd int       `json:"max_articles_on_feed_add"` // Max articles to import when adding a new feed (0 = unlimited)
}

type Feed struct {
	ID                    int       `json:"id"`
	Title                 string    `json:"title"`
	URL                   string    `json:"url"`
	Description           string    `json:"description"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	LastFetch             time.Time `json:"last_fetch"`
	LastChecked           time.Time `json:"last_checked"`            // When we last attempted to check for updates
	LastHadNewContent     time.Time `json:"last_had_new_content"`    // When we last found new articles
	AverageUpdateInterval int       `json:"average_update_interval"` // Average seconds between updates (0 = unknown)
}

type Article struct {
	ID          int       `json:"id"`
	FeedID      int       `json:"feed_id"`
	FeedTitle   string    `json:"feed_title,omitempty"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Content     string    `json:"content"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
	IsRead      bool      `json:"is_read"`
	IsStarred   bool      `json:"is_starred"`
}

type UserFeed struct {
	UserID int `json:"user_id"`
	FeedID int `json:"feed_id"`
}

type UserArticle struct {
	UserID    int  `json:"user_id"`
	ArticleID int  `json:"article_id"`
	IsRead    bool `json:"is_read"`
	IsStarred bool `json:"is_starred"`
}

type Session struct {
	ID        string    `json:"id"`
	UserID    int       `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AuditLog struct {
	ID               int       `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	AdminUserID      int       `json:"admin_user_id"`
	AdminEmail       string    `json:"admin_email"`
	OperationType    string    `json:"operation_type"`
	TargetUserID     int       `json:"target_user_id"`
	TargetUserEmail  string    `json:"target_user_email"`
	OperationDetails string    `json:"operation_details"` // JSON string
	IPAddress        string    `json:"ip_address"`
	Result           string    `json:"result"` // "success" or "failure"
	ErrorMessage     string    `json:"error_message"`
}

func InitDB() (Database, error) {
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		return NewDatastoreDB(projectID)
	}

	db, err := sql.Open("sqlite3", "./goread2.db?_loc=auto")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	dbWrapper := &DB{db}
	if err := dbWrapper.CreateTables(); err != nil {
		return nil, err
	}

	if err := dbWrapper.migrateDatabase(); err != nil {
		return nil, err
	}

	return dbWrapper, nil
}

// CreateTables creates all necessary database tables (public for testing)
func (db *DB) CreateTables() error {
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
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
	);`

	feedsTable := `
	CREATE TABLE IF NOT EXISTS feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		url TEXT UNIQUE NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_fetch DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_checked DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_had_new_content DATETIME DEFAULT CURRENT_TIMESTAMP,
		average_update_interval INTEGER DEFAULT 0
	);`

	articlesTable := `
	CREATE TABLE IF NOT EXISTS articles (
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
	);`

	userFeedsTable := `
	CREATE TABLE IF NOT EXISTS user_feeds (
		user_id INTEGER NOT NULL,
		feed_id INTEGER NOT NULL,
		PRIMARY KEY (user_id, feed_id),
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
		FOREIGN KEY (feed_id) REFERENCES feeds (id) ON DELETE CASCADE
	);`

	userArticlesTable := `
	CREATE TABLE IF NOT EXISTS user_articles (
		user_id INTEGER NOT NULL,
		article_id INTEGER NOT NULL,
		is_read BOOLEAN DEFAULT FALSE,
		is_starred BOOLEAN DEFAULT FALSE,
		PRIMARY KEY (user_id, article_id),
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
		FOREIGN KEY (article_id) REFERENCES articles (id) ON DELETE CASCADE
	);`

	adminTokensTable := `
	CREATE TABLE IF NOT EXISTS admin_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		token_hash TEXT UNIQUE NOT NULL,
		description TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_active BOOLEAN DEFAULT 1
	);`

	sessionsTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	auditLogsTable := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		admin_user_id INTEGER,
		admin_email TEXT NOT NULL,
		operation_type TEXT NOT NULL,
		target_user_id INTEGER,
		target_user_email TEXT,
		operation_details TEXT,
		ip_address TEXT,
		result TEXT NOT NULL,
		error_message TEXT
	);`

	tables := []string{usersTable, feedsTable, articlesTable, userFeedsTable, userArticlesTable, adminTokensTable, sessionsTable, auditLogsTable}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return err
		}
	}

	// Create performance indexes
	if err := db.CreateIndexes(); err != nil {
		return err
	}

	return nil
}

// CreateIndexes creates all database indexes (public for testing)
func (db *DB) CreateIndexes() error {
	indexes := []string{
		// Articles table indexes for better query performance
		`CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles (feed_id)`,
		`CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles (published_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_articles_feed_published ON articles (feed_id, published_at DESC)`,

		// User articles table indexes for read status queries
		`CREATE INDEX IF NOT EXISTS idx_user_articles_user_id ON user_articles (user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_articles_article_id ON user_articles (article_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_articles_read ON user_articles (user_id, is_read)`,
		// Critical index for unread count queries - optimizes EXISTS subquery
		`CREATE INDEX IF NOT EXISTS idx_user_articles_article_user_read ON user_articles (article_id, user_id, is_read)`,

		// User feeds table index for subscription lookups
		`CREATE INDEX IF NOT EXISTS idx_user_feeds_user_id ON user_feeds (user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_feeds_feed_id ON user_feeds (feed_id)`,

		// Users table indexes for authentication
		`CREATE INDEX IF NOT EXISTS idx_users_google_id ON users (google_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users (email)`,

		// Admin tokens table indexes for authentication
		`CREATE INDEX IF NOT EXISTS idx_admin_tokens_hash ON admin_tokens (token_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_admin_tokens_active ON admin_tokens (is_active)`,

		// Audit logs table indexes for querying
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs (timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_admin_user ON audit_logs (admin_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_target_user ON audit_logs (target_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_operation ON audit_logs (operation_type)`,
	}

	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

func (db *DB) migrateDatabase() error {
	// Add missing columns to existing users table if they don't exist
	userColumns := []string{
		"ALTER TABLE users ADD COLUMN google_id TEXT",
		"ALTER TABLE users ADD COLUMN email TEXT",
		"ALTER TABLE users ADD COLUMN name TEXT",
		"ALTER TABLE users ADD COLUMN avatar TEXT",
		"ALTER TABLE users ADD COLUMN subscription_status TEXT DEFAULT 'trial'",
		"ALTER TABLE users ADD COLUMN subscription_id TEXT",
		"ALTER TABLE users ADD COLUMN trial_ends_at DATETIME",
		"ALTER TABLE users ADD COLUMN last_payment_date DATETIME",
		"ALTER TABLE users ADD COLUMN next_billing_date DATETIME",
		"ALTER TABLE users ADD COLUMN is_admin BOOLEAN DEFAULT 0",
		"ALTER TABLE users ADD COLUMN free_months_remaining INTEGER DEFAULT 0",
		"ALTER TABLE users ADD COLUMN max_articles_on_feed_add INTEGER DEFAULT 100",
	}

	for _, alterQuery := range userColumns {
		_, err := db.Exec(alterQuery)
		if err != nil {
			// Ignore "duplicate column" errors - column already exists
			if !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("migration failed: %w", err)
			}
		}
	}

	// Add missing columns to existing feeds table for smart update tracking
	// Note: SQLite doesn't allow CURRENT_TIMESTAMP as default in ALTER TABLE
	// So we add columns with NULL default, then update existing rows
	feedColumns := []string{
		"ALTER TABLE feeds ADD COLUMN last_checked DATETIME",
		"ALTER TABLE feeds ADD COLUMN last_had_new_content DATETIME",
		"ALTER TABLE feeds ADD COLUMN average_update_interval INTEGER DEFAULT 0",
	}

	for _, alterQuery := range feedColumns {
		_, err := db.Exec(alterQuery)
		if err != nil {
			// Ignore "duplicate column" errors - column already exists
			if !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("migration failed: %w", err)
			}
		}
	}

	// Update existing feeds to have current timestamp for new tracking fields
	// This only affects feeds that existed before the migration
	_, errUpdate := db.Exec(`
		UPDATE feeds
		SET last_checked = COALESCE(last_checked, last_fetch),
		    last_had_new_content = COALESCE(last_had_new_content, last_fetch)
		WHERE last_checked IS NULL OR last_had_new_content IS NULL
	`)
	if errUpdate != nil {
		log.Printf("Warning: Failed to set initial tracking timestamps: %v", errUpdate)
	}

	// Set trial_ends_at for existing users who don't have it set
	updateTrialQuery := `
		UPDATE users
		SET trial_ends_at = datetime(created_at, '+30 days')
		WHERE trial_ends_at IS NULL AND subscription_status = 'trial'
	`
	_, err := db.Exec(updateTrialQuery)
	if err != nil {
		return fmt.Errorf("failed to set trial end dates: %w", err)
	}

	// Create sessions table if it doesn't exist
	sessionsTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	);`
	_, err = db.Exec(sessionsTable)
	if err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	// Ensure indexes are created on existing databases
	if err := db.CreateIndexes(); err != nil {
		return err
	}

	return nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) AddFeed(feed *Feed) error {
	query := `INSERT INTO feeds (title, url, description, created_at, updated_at, last_fetch, last_checked, last_had_new_content, average_update_interval)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query, feed.Title, feed.URL, feed.Description,
		feed.CreatedAt, feed.UpdatedAt, feed.LastFetch, feed.LastChecked, feed.LastHadNewContent, feed.AverageUpdateInterval)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	feed.ID = int(id)
	return nil
}

func (db *DB) UpdateFeed(feed *Feed) error {
	query := `UPDATE feeds SET title = ?, description = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, feed.Title, feed.Description, time.Now(), feed.ID)
	return err
}

func (db *DB) UpdateFeedTracking(feedID int, lastChecked, lastHadNewContent time.Time, averageUpdateInterval int) error {
	query := `UPDATE feeds SET last_checked = ?, last_had_new_content = ?, average_update_interval = ? WHERE id = ?`
	_, err := db.Exec(query, lastChecked, lastHadNewContent, averageUpdateInterval, feedID)
	return err
}

func (db *DB) GetFeeds() ([]Feed, error) {
	query := `SELECT id, title, url, description, created_at, updated_at, last_fetch,
			  last_checked, last_had_new_content, average_update_interval
			  FROM feeds ORDER BY title`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var feeds []Feed
	for rows.Next() {
		var feed Feed
		err := rows.Scan(&feed.ID, &feed.Title, &feed.URL, &feed.Description,
			&feed.CreatedAt, &feed.UpdatedAt, &feed.LastFetch, &feed.LastChecked, &feed.LastHadNewContent, &feed.AverageUpdateInterval)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, nil
}

func (db *DB) GetFeedByURL(url string) (*Feed, error) {
	query := `SELECT id, title, url, description, created_at, updated_at, last_fetch,
			  last_checked, last_had_new_content, average_update_interval
			  FROM feeds WHERE url = ?`
	var feed Feed
	err := db.QueryRow(query, url).Scan(&feed.ID, &feed.Title, &feed.URL, &feed.Description,
		&feed.CreatedAt, &feed.UpdatedAt, &feed.LastFetch, &feed.LastChecked, &feed.LastHadNewContent, &feed.AverageUpdateInterval)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &feed, nil
}

func (db *DB) DeleteFeed(id int) error {
	query := `DELETE FROM feeds WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}

func (db *DB) AddArticle(article *Article) error {
	query := `INSERT OR IGNORE INTO articles 
			  (feed_id, title, url, content, description, author, published_at, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query, article.FeedID, article.Title, article.URL, article.Content,
		article.Description, article.Author, article.PublishedAt, article.CreatedAt)
	if err != nil {
		return err
	}

	// Set the ID if this was a new insert
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if id > 0 {
		article.ID = int(id)
	} else {
		// Article already existed, fetch its ID
		query = `SELECT id FROM articles WHERE url = ?`
		err = db.QueryRow(query, article.URL).Scan(&article.ID)
	}
	return err
}

func (db *DB) GetArticles(feedID int) ([]Article, error) {
	query := `SELECT id, feed_id, title, url, content, description, author, 
			  published_at, created_at 
			  FROM articles WHERE feed_id = ? ORDER BY published_at DESC`

	rows, err := db.Query(query, feedID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var articles []Article
	for rows.Next() {
		var article Article
		err := rows.Scan(&article.ID, &article.FeedID, &article.Title, &article.URL,
			&article.Content, &article.Description, &article.Author,
			&article.PublishedAt, &article.CreatedAt)
		if err != nil {
			return nil, err
		}
		// Default read/starred status to false for this basic method
		article.IsRead = false
		article.IsStarred = false
		articles = append(articles, article)
	}

	return articles, nil
}

func (db *DB) FindArticleByURL(url string) (*Article, error) {
	query := `SELECT id, feed_id, title, url, content, description, author, published_at, created_at
			  FROM articles WHERE url = ? LIMIT 1`

	var article Article
	err := db.QueryRow(query, url).Scan(&article.ID, &article.FeedID, &article.Title, &article.URL,
		&article.Content, &article.Description, &article.Author, &article.PublishedAt, &article.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Article not found
		}
		return nil, err
	}

	// Default read/starred status to false
	article.IsRead = false
	article.IsStarred = false

	return &article, nil
}

// Legacy single-user methods - deprecated in favor of multi-user methods
// func (db *DB) MarkRead(articleID int, isRead bool) error {
// 	// This method is deprecated - use MarkUserArticleRead instead
// 	return fmt.Errorf("deprecated: use MarkUserArticleRead instead")
// }

// func (db *DB) ToggleStar(articleID int) error {
// 	// This method is deprecated - use ToggleUserArticleStar instead
// 	return fmt.Errorf("deprecated: use ToggleUserArticleStar instead")
// }

func (db *DB) UpdateFeedLastFetch(feedID int, lastFetch time.Time) error {
	query := `UPDATE feeds SET last_fetch = ? WHERE id = ?`
	_, err := db.Exec(query, lastFetch, feedID)
	return err
}

// User methods
func (db *DB) CreateUser(user *User) error {
	// Set default subscription values for new users
	if user.SubscriptionStatus == "" {
		user.SubscriptionStatus = "trial"
	}
	if user.TrialEndsAt.IsZero() {
		user.TrialEndsAt = user.CreatedAt.AddDate(0, 0, 30) // 30 days from creation
	}
	if user.MaxArticlesOnFeedAdd == 0 {
		user.MaxArticlesOnFeedAdd = 100 // Default to 100 articles
	}

	query := `INSERT INTO users (google_id, email, name, avatar, created_at, subscription_status, subscription_id, trial_ends_at, last_payment_date, next_billing_date, max_articles_on_feed_add)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query, user.GoogleID, user.Email, user.Name, user.Avatar, user.CreatedAt,
		user.SubscriptionStatus, user.SubscriptionID, user.TrialEndsAt, user.LastPaymentDate, user.NextBillingDate, user.MaxArticlesOnFeedAdd)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = int(id)
	return nil
}

func (db *DB) GetUserByGoogleID(googleID string) (*User, error) {
	query := `SELECT id, google_id, email, name, avatar, created_at,
			  COALESCE(subscription_status, 'trial') as subscription_status,
			  COALESCE(subscription_id, '') as subscription_id,
			  trial_ends_at, last_payment_date, next_billing_date,
			  COALESCE(is_admin, 0) as is_admin,
			  COALESCE(free_months_remaining, 0) as free_months_remaining,
			  COALESCE(max_articles_on_feed_add, 100) as max_articles_on_feed_add
			  FROM users WHERE google_id = ?`

	var user User
	var trialEndsAt sql.NullTime
	var lastPaymentDate sql.NullTime
	var nextBillingDate sql.NullTime

	err := db.QueryRow(query, googleID).Scan(&user.ID, &user.GoogleID, &user.Email,
		&user.Name, &user.Avatar, &user.CreatedAt, &user.SubscriptionStatus,
		&user.SubscriptionID, &trialEndsAt, &lastPaymentDate, &nextBillingDate,
		&user.IsAdmin, &user.FreeMonthsRemaining, &user.MaxArticlesOnFeedAdd)
	if err != nil {
		return nil, err
	}

	// Handle nullable datetime fields
	if trialEndsAt.Valid {
		user.TrialEndsAt = trialEndsAt.Time
	} else {
		// Set default trial end date if not set
		user.TrialEndsAt = user.CreatedAt.AddDate(0, 0, 30)
	}

	if lastPaymentDate.Valid {
		user.LastPaymentDate = lastPaymentDate.Time
	}

	if nextBillingDate.Valid {
		user.NextBillingDate = nextBillingDate.Time
	}

	return &user, nil
}

func (db *DB) GetUserByID(userID int) (*User, error) {
	query := `SELECT id, google_id, email, name, avatar, created_at,
			  COALESCE(subscription_status, 'trial') as subscription_status,
			  COALESCE(subscription_id, '') as subscription_id,
			  trial_ends_at, last_payment_date, next_billing_date,
			  COALESCE(is_admin, 0) as is_admin,
			  COALESCE(free_months_remaining, 0) as free_months_remaining,
			  COALESCE(max_articles_on_feed_add, 100) as max_articles_on_feed_add
			  FROM users WHERE id = ?`

	var user User
	var trialEndsAt sql.NullTime
	var lastPaymentDate sql.NullTime
	var nextBillingDate sql.NullTime

	err := db.QueryRow(query, userID).Scan(&user.ID, &user.GoogleID, &user.Email,
		&user.Name, &user.Avatar, &user.CreatedAt, &user.SubscriptionStatus,
		&user.SubscriptionID, &trialEndsAt, &lastPaymentDate, &nextBillingDate,
		&user.IsAdmin, &user.FreeMonthsRemaining, &user.MaxArticlesOnFeedAdd)
	if err != nil {
		return nil, err
	}

	// Handle nullable datetime fields
	if trialEndsAt.Valid {
		user.TrialEndsAt = trialEndsAt.Time
	} else {
		// Set default trial end date if not set
		user.TrialEndsAt = user.CreatedAt.AddDate(0, 0, 30)
	}

	if lastPaymentDate.Valid {
		user.LastPaymentDate = lastPaymentDate.Time
	}

	if nextBillingDate.Valid {
		user.NextBillingDate = nextBillingDate.Time
	}

	return &user, nil
}

// User feed methods
func (db *DB) GetUserFeeds(userID int) ([]Feed, error) {
	query := `SELECT f.id, f.title, f.url, f.description, f.created_at, f.updated_at, f.last_fetch,
			  f.last_checked, f.last_had_new_content, f.average_update_interval
			  FROM feeds f
			  JOIN user_feeds uf ON f.id = uf.feed_id
			  WHERE uf.user_id = ?
			  ORDER BY f.title`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var feeds []Feed
	for rows.Next() {
		var feed Feed
		err := rows.Scan(&feed.ID, &feed.Title, &feed.URL, &feed.Description,
			&feed.CreatedAt, &feed.UpdatedAt, &feed.LastFetch, &feed.LastChecked, &feed.LastHadNewContent, &feed.AverageUpdateInterval)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, nil
}

func (db *DB) GetAllUserFeeds() ([]Feed, error) {
	query := `SELECT DISTINCT f.id, f.title, f.url, f.description, f.created_at, f.updated_at, f.last_fetch,
			  f.last_checked, f.last_had_new_content, f.average_update_interval
			  FROM feeds f
			  JOIN user_feeds uf ON f.id = uf.feed_id
			  ORDER BY f.title`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var feeds []Feed
	for rows.Next() {
		var feed Feed
		err := rows.Scan(&feed.ID, &feed.Title, &feed.URL, &feed.Description,
			&feed.CreatedAt, &feed.UpdatedAt, &feed.LastFetch, &feed.LastChecked, &feed.LastHadNewContent, &feed.AverageUpdateInterval)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, nil
}

func (db *DB) SubscribeUserToFeed(userID, feedID int) error {
	query := `INSERT OR IGNORE INTO user_feeds (user_id, feed_id) VALUES (?, ?)`
	_, err := db.Exec(query, userID, feedID)
	return err
}

func (db *DB) UnsubscribeUserFromFeed(userID, feedID int) error {
	// Remove the user-feed subscription
	// Note: Orphaned user_articles will be cleaned up by periodic background job
	// (see CleanupOrphanedUserArticles)
	query := `DELETE FROM user_feeds WHERE user_id = ? AND feed_id = ?`
	_, err := db.Exec(query, userID, feedID)
	return err
}

// User article methods
func (db *DB) GetUserArticles(userID int) ([]Article, error) {
	result, err := db.GetUserArticlesPaginated(userID, 50, "", false) // Default: first 50 articles
	if err != nil {
		return nil, err
	}
	return result.Articles, nil
}

// GetUserArticlesPaginated fetches user articles with cursor-based pagination
// Uses keyset pagination for efficient querying without scanning skipped rows
func (db *DB) GetUserArticlesPaginated(userID int, limit int, cursor string, unreadOnly bool) (*ArticlePaginationResult, error) {
	baseQuery := `SELECT a.id, a.feed_id, f.title as feed_title, a.title, a.url, a.content, a.description, a.author,
			  a.published_at, a.created_at,
			  COALESCE(ua.is_read, 0) as is_read,
			  COALESCE(ua.is_starred, 0) as is_starred
			  FROM articles a
			  JOIN user_feeds uf ON a.feed_id = uf.feed_id
			  JOIN feeds f ON a.feed_id = f.id
			  LEFT JOIN user_articles ua ON a.id = ua.article_id AND ua.user_id = ?
			  WHERE uf.user_id = ?`

	args := []interface{}{userID, userID}

	// Add unread filter if requested
	if unreadOnly {
		baseQuery += ` AND COALESCE(ua.is_read, 0) = 0`
	}

	// Apply keyset pagination if cursor is provided
	if cursor != "" {
		// Decode cursor to get last article's published_at and id
		cursorData, err := decodeSQLiteCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		// Keyset pagination: use WHERE clause instead of OFFSET
		// Order by published_at DESC, id DESC for deterministic ordering
		baseQuery += ` AND (a.published_at < ? OR (a.published_at = ? AND a.id < ?))`
		args = append(args, cursorData.PublishedAt, cursorData.PublishedAt, cursorData.ID)
	}

	// Fetch limit+1 to check if there are more results
	baseQuery += ` ORDER BY a.published_at DESC, a.id DESC LIMIT ?`
	args = append(args, limit+1)

	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var articles []Article
	for rows.Next() {
		var article Article
		err := rows.Scan(&article.ID, &article.FeedID, &article.FeedTitle, &article.Title, &article.URL,
			&article.Content, &article.Description, &article.Author,
			&article.PublishedAt, &article.CreatedAt, &article.IsRead, &article.IsStarred)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}

	// Determine if there are more results and generate next cursor
	var nextCursor string
	if len(articles) > limit {
		// More results exist, create cursor from the last article we're returning
		lastArticle := articles[limit-1]
		nextCursor = encodeSQLiteCursor(lastArticle.ID, lastArticle.PublishedAt)
		articles = articles[:limit] // Trim to requested limit
	}

	return &ArticlePaginationResult{
		Articles:   articles,
		NextCursor: nextCursor,
	}, nil
}

// sqliteCursor represents the keyset values for pagination
type sqliteCursor struct {
	PublishedAt time.Time
	ID          int
}

// encodeSQLiteCursor creates a cursor from article ID and published date
func encodeSQLiteCursor(id int, publishedAt time.Time) string {
	// Format: unixnano_id to preserve full timestamp precision
	cursorStr := fmt.Sprintf("%d_%d", publishedAt.UnixNano(), id)
	return cursorStr
}

// decodeSQLiteCursor decodes a cursor string back to cursor data
func decodeSQLiteCursor(cursor string) (*sqliteCursor, error) {
	var timestampNano int64
	var id int
	_, err := fmt.Sscanf(cursor, "%d_%d", &timestampNano, &id)
	if err != nil {
		return nil, err
	}
	return &sqliteCursor{
		PublishedAt: time.Unix(0, timestampNano),
		ID:          id,
	}, nil
}

func (db *DB) GetUserFeedArticles(userID, feedID int) ([]Article, error) {
	// First verify user is subscribed to this feed
	var subscriptionExists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM user_feeds WHERE user_id = ? AND feed_id = ?)`
	err := db.QueryRow(checkQuery, userID, feedID).Scan(&subscriptionExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check user subscription: %w", err)
	}

	if !subscriptionExists {
		// User is not subscribed to this feed
		return []Article{}, nil
	}

	query := `SELECT a.id, a.feed_id, f.title as feed_title, a.title, a.url, a.content, a.description, a.author,
			  a.published_at, a.created_at,
			  COALESCE(ua.is_read, 0) as is_read,
			  COALESCE(ua.is_starred, 0) as is_starred
			  FROM articles a
			  JOIN feeds f ON a.feed_id = f.id
			  LEFT JOIN user_articles ua ON a.id = ua.article_id AND ua.user_id = ?
			  WHERE a.feed_id = ?
			  ORDER BY a.published_at DESC`

	rows, err := db.Query(query, userID, feedID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var articles []Article
	for rows.Next() {
		var article Article
		err := rows.Scan(&article.ID, &article.FeedID, &article.FeedTitle, &article.Title, &article.URL,
			&article.Content, &article.Description, &article.Author,
			&article.PublishedAt, &article.CreatedAt, &article.IsRead, &article.IsStarred)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}

	return articles, nil
}

// User article status methods
func (db *DB) GetUserArticleStatus(userID, articleID int) (*UserArticle, error) {
	query := `SELECT user_id, article_id, is_read, is_starred FROM user_articles 
			  WHERE user_id = ? AND article_id = ?`

	var userArticle UserArticle
	err := db.QueryRow(query, userID, articleID).Scan(&userArticle.UserID, &userArticle.ArticleID,
		&userArticle.IsRead, &userArticle.IsStarred)
	if err != nil {
		return nil, err
	}
	return &userArticle, nil
}

func (db *DB) SetUserArticleStatus(userID, articleID int, isRead, isStarred bool) error {
	query := `INSERT OR REPLACE INTO user_articles (user_id, article_id, is_read, is_starred) 
			  VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, userID, articleID, isRead, isStarred)
	return err
}

func (db *DB) MarkUserArticleRead(userID, articleID int, isRead bool) error {
	// First check if record exists
	var dummy int
	checkQuery := `SELECT 1 FROM user_articles WHERE user_id = ? AND article_id = ?`
	err := db.QueryRow(checkQuery, userID, articleID).Scan(&dummy)

	switch err {
	case sql.ErrNoRows:
		// Create new record
		query := `INSERT INTO user_articles (user_id, article_id, is_read, is_starred) 
				  VALUES (?, ?, ?, 0)`
		_, err = db.Exec(query, userID, articleID, isRead)
	case nil:
		// Update existing record
		query := `UPDATE user_articles SET is_read = ? WHERE user_id = ? AND article_id = ?`
		_, err = db.Exec(query, isRead, userID, articleID)
	}

	return err
}

func (db *DB) ToggleUserArticleStar(userID, articleID int) error {
	// First check if record exists
	var currentStarred bool
	checkQuery := `SELECT is_starred FROM user_articles WHERE user_id = ? AND article_id = ?`
	err := db.QueryRow(checkQuery, userID, articleID).Scan(&currentStarred)

	switch err {
	case sql.ErrNoRows:
		// Create new record with starred = true
		query := `INSERT INTO user_articles (user_id, article_id, is_read, is_starred) 
				  VALUES (?, ?, 0, 1)`
		_, err = db.Exec(query, userID, articleID)
	case nil:
		// Update existing record
		query := `UPDATE user_articles SET is_starred = ? WHERE user_id = ? AND article_id = ?`
		_, err = db.Exec(query, !currentStarred, userID, articleID)
	}

	return err
}

func (db *DB) BatchSetUserArticleStatus(userID int, articles []Article, isRead, isStarred bool) error {
	if len(articles) == 0 {
		return nil
	}

	// Use INSERT OR REPLACE for batch operation
	query := `INSERT OR REPLACE INTO user_articles (user_id, article_id, is_read, is_starred) VALUES `

	// Build values string
	values := make([]string, len(articles))
	args := make([]interface{}, len(articles)*4)

	for i, article := range articles {
		values[i] = "(?, ?, ?, ?)"
		baseIdx := i * 4
		args[baseIdx] = userID
		args[baseIdx+1] = article.ID
		args[baseIdx+2] = isRead
		args[baseIdx+3] = isStarred
	}

	query += strings.Join(values, ", ")

	_, err := db.Exec(query, args...)
	return err
}

func (db *DB) GetUserUnreadCounts(userID int) (map[int]int, error) {
	// First get user's feeds
	userFeeds, err := db.GetUserFeeds(userID)
	if err != nil {
		return nil, err
	}

	if len(userFeeds) == 0 {
		return make(map[int]int), nil
	}

	unreadCounts := make(map[int]int)

	// Process each feed individually for better performance with indexes
	for _, feed := range userFeeds {
		count, err := db.getFeedUnreadCountForUser(userID, feed.ID)
		if err != nil {
			return nil, err
		}
		unreadCounts[feed.ID] = count
	}

	return unreadCounts, nil
}

// GetAccountStats retrieves user account statistics in a single batched query
// Returns total articles, total unread, and active feeds count
func (db *DB) GetAccountStats(userID int) (map[string]interface{}, error) {
	// Single query to get all stats at once - avoids N+1 problem
	query := `
		SELECT
			COUNT(DISTINCT a.id) as total_articles,
			COUNT(DISTINCT CASE
				WHEN NOT EXISTS (
					SELECT 1 FROM user_articles ua
					WHERE ua.article_id = a.id
					AND ua.user_id = ?
					AND ua.is_read = 1
				) THEN a.id
			END) as total_unread,
			COUNT(DISTINCT CASE
				WHEN NOT EXISTS (
					SELECT 1 FROM user_articles ua
					WHERE ua.article_id = a.id
					AND ua.user_id = ?
					AND ua.is_read = 1
				) THEN a.feed_id
			END) as active_feeds
		FROM articles a
		INNER JOIN feeds f ON a.feed_id = f.id
		INNER JOIN user_feeds uf ON f.id = uf.feed_id
		WHERE uf.user_id = ?
	`

	var totalArticles, totalUnread, activeFeeds int
	err := db.QueryRow(query, userID, userID, userID).Scan(&totalArticles, &totalUnread, &activeFeeds)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_articles": totalArticles,
		"total_unread":   totalUnread,
		"active_feeds":   activeFeeds,
	}

	return stats, nil
}

// Helper function to get unread count for a specific feed efficiently
func (db *DB) getFeedUnreadCountForUser(userID, feedID int) (int, error) {
	// Count articles in feed that are NOT marked as read by user
	query := `
		SELECT COUNT(*)
		FROM articles a
		WHERE a.feed_id = ?
		AND NOT EXISTS (
			SELECT 1 FROM user_articles ua 
			WHERE ua.article_id = a.id 
			AND ua.user_id = ? 
			AND ua.is_read = 1
		)
	`

	var count int
	err := db.QueryRow(query, feedID, userID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CleanupOrphanedUserArticles removes user_articles that reference articles from feeds
// the user is no longer subscribed to. Only cleans up articles older than the specified number of days.
// Returns the number of records deleted.
func (db *DB) CleanupOrphanedUserArticles(olderThanDays int) (int, error) {
	var query string
	var result sql.Result
	var err error

	if olderThanDays == 0 {
		// Clean up all orphaned user articles regardless of age
		query = `
			DELETE FROM user_articles
			WHERE rowid IN (
				SELECT ua.rowid
				FROM user_articles ua
				JOIN articles a ON ua.article_id = a.id
				LEFT JOIN user_feeds uf ON ua.user_id = uf.user_id AND a.feed_id = uf.feed_id
				WHERE uf.user_id IS NULL
			)
		`
		result, err = db.Exec(query)
	} else {
		// Clean up orphaned user articles older than the specified number of days
		query = `
			DELETE FROM user_articles
			WHERE rowid IN (
				SELECT ua.rowid
				FROM user_articles ua
				JOIN articles a ON ua.article_id = a.id
				LEFT JOIN user_feeds uf ON ua.user_id = uf.user_id AND a.feed_id = uf.feed_id
				WHERE uf.user_id IS NULL
				AND a.created_at < datetime('now', '-' || ? || ' days')
			)
		`
		result, err = db.Exec(query, olderThanDays)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to cleanup orphaned user articles: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// Subscription management methods
func (db *DB) UpdateUserSubscription(userID int, status, subscriptionID string, lastPaymentDate, nextBillingDate time.Time) error {
	// Redact subscription ID for security - only log prefix for debugging
	redactedSubID := "***"
	if len(subscriptionID) > 8 {
		redactedSubID = subscriptionID[:8] + "***"
	}
	log.Printf("UpdateUserSubscription - Updating user %d: status=%s, subscriptionID=%s, lastPaymentDate=%v, nextBillingDate=%v",
		userID, status, redactedSubID, lastPaymentDate, nextBillingDate)

	query := `UPDATE users SET subscription_status = ?, subscription_id = ?, last_payment_date = ?, next_billing_date = ? WHERE id = ?`
	result, err := db.Exec(query, status, subscriptionID, lastPaymentDate, nextBillingDate, userID)
	if err != nil {
		log.Printf("ERROR: UpdateUserSubscription - Query failed: %v", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("WARNING: UpdateUserSubscription - Could not get rows affected: %v", err)
	} else {
		log.Printf("UpdateUserSubscription - Rows affected: %d", rowsAffected)
		if rowsAffected == 0 {
			log.Printf("WARNING: UpdateUserSubscription - No rows updated for user ID %d", userID)
		}
	}

	return nil
}

func (db *DB) IsUserSubscriptionActive(userID int) (bool, error) {
	query := `SELECT subscription_status, trial_ends_at, is_admin, free_months_remaining FROM users WHERE id = ?`

	var status string
	var trialEndsAt time.Time
	var isAdmin bool
	var freeMonths int
	err := db.QueryRow(query, userID).Scan(&status, &trialEndsAt, &isAdmin, &freeMonths)
	if err != nil {
		return false, err
	}

	// User is active if:
	// 1. They're an admin user (unlimited access), OR
	// 2. They have an active paid subscription, OR
	// 3. They're on trial and trial hasn't expired, OR
	// 4. They have free months remaining
	if isAdmin {
		return true, nil
	}

	if status == "active" {
		return true, nil
	}

	if status == "trial" && time.Now().Before(trialEndsAt) {
		return true, nil
	}

	if freeMonths > 0 {
		return true, nil
	}

	return false, nil
}

func (db *DB) GetUserFeedCount(userID int) (int, error) {
	query := `SELECT COUNT(*) FROM user_feeds WHERE user_id = ?`

	var count int
	err := db.QueryRow(query, userID).Scan(&count)
	return count, err
}

func (db *DB) UpdateUserMaxArticlesOnFeedAdd(userID int, maxArticles int) error {
	query := `UPDATE users SET max_articles_on_feed_add = ? WHERE id = ?`
	_, err := db.Exec(query, maxArticles, userID)
	return err
}

// Admin management methods
func (db *DB) SetUserAdmin(userID int, isAdmin bool) error {
	query := `UPDATE users SET is_admin = ? WHERE id = ?`
	_, err := db.Exec(query, isAdmin, userID)
	return err
}

func (db *DB) GrantFreeMonths(userID int, months int) error {
	// Get current free months and add the new ones
	query := `UPDATE users SET free_months_remaining = COALESCE(free_months_remaining, 0) + ? WHERE id = ?`
	_, err := db.Exec(query, months, userID)
	return err
}

func (db *DB) GetUserByEmail(email string) (*User, error) {
	query := `SELECT id, google_id, email, name, avatar, created_at,
			  COALESCE(subscription_status, 'trial') as subscription_status,
			  COALESCE(subscription_id, '') as subscription_id,
			  trial_ends_at, last_payment_date, next_billing_date,
			  COALESCE(is_admin, 0) as is_admin,
			  COALESCE(free_months_remaining, 0) as free_months_remaining,
			  COALESCE(max_articles_on_feed_add, 100) as max_articles_on_feed_add
			  FROM users WHERE email = ?`

	var user User
	var trialEndsAt sql.NullTime
	var lastPaymentDate sql.NullTime
	var nextBillingDate sql.NullTime

	err := db.QueryRow(query, email).Scan(
		&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.Avatar,
		&user.CreatedAt, &user.SubscriptionStatus, &user.SubscriptionID,
		&trialEndsAt, &lastPaymentDate, &nextBillingDate, &user.IsAdmin, &user.FreeMonthsRemaining, &user.MaxArticlesOnFeedAdd,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable datetime fields
	if trialEndsAt.Valid {
		user.TrialEndsAt = trialEndsAt.Time
	} else {
		// Set default trial end date if not set
		user.TrialEndsAt = user.CreatedAt.AddDate(0, 0, 30)
	}

	if lastPaymentDate.Valid {
		user.LastPaymentDate = lastPaymentDate.Time
	}

	if nextBillingDate.Valid {
		user.NextBillingDate = nextBillingDate.Time
	}

	return &user, nil
}

// Session methods for SQLite
func (db *DB) CreateSession(session *Session) error {
	query := `INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, session.ID, session.UserID, session.CreatedAt, session.ExpiresAt)
	return err
}

func (db *DB) GetSession(sessionID string) (*Session, error) {
	query := `SELECT id, user_id, created_at, expires_at FROM sessions WHERE id = ?`

	var session Session
	err := db.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.CreatedAt,
		&session.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (db *DB) DeleteSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := db.Exec(query, sessionID)
	return err
}

func (db *DB) DeleteExpiredSessions() error {
	query := `DELETE FROM sessions WHERE expires_at < ?`
	_, err := db.Exec(query, time.Now())
	return err
}

// Audit log methods for SQLite
func (db *DB) CreateAuditLog(log *AuditLog) error {
	query := `INSERT INTO audit_logs
		(timestamp, admin_user_id, admin_email, operation_type, target_user_id, target_user_email,
		 operation_details, ip_address, result, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query,
		log.Timestamp,
		log.AdminUserID,
		log.AdminEmail,
		log.OperationType,
		log.TargetUserID,
		log.TargetUserEmail,
		log.OperationDetails,
		log.IPAddress,
		log.Result,
		log.ErrorMessage,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	log.ID = int(id)
	return nil
}

func (db *DB) GetAuditLogs(limit, offset int, filters map[string]interface{}) ([]AuditLog, error) {
	query := `SELECT id, timestamp, admin_user_id, admin_email, operation_type,
		target_user_id, target_user_email, operation_details, ip_address, result, error_message
		FROM audit_logs WHERE 1=1`

	args := []interface{}{}

	// Apply filters
	if userID, ok := filters["admin_user_id"]; ok {
		query += " AND admin_user_id = ?"
		args = append(args, userID)
	}
	if targetUserID, ok := filters["target_user_id"]; ok {
		query += " AND target_user_id = ?"
		args = append(args, targetUserID)
	}
	if opType, ok := filters["operation_type"]; ok {
		query += " AND operation_type = ?"
		args = append(args, opType)
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var adminUserID sql.NullInt64
		var targetUserID sql.NullInt64
		var targetUserEmail sql.NullString
		var operationDetails sql.NullString
		var ipAddress sql.NullString
		var errorMessage sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&adminUserID,
			&log.AdminEmail,
			&log.OperationType,
			&targetUserID,
			&targetUserEmail,
			&operationDetails,
			&ipAddress,
			&log.Result,
			&errorMessage,
		)
		if err != nil {
			return nil, err
		}

		if adminUserID.Valid {
			log.AdminUserID = int(adminUserID.Int64)
		}
		if targetUserID.Valid {
			log.TargetUserID = int(targetUserID.Int64)
		}
		if targetUserEmail.Valid {
			log.TargetUserEmail = targetUserEmail.String
		}
		if operationDetails.Valid {
			log.OperationDetails = operationDetails.String
		}
		if ipAddress.Valid {
			log.IPAddress = ipAddress.String
		}
		if errorMessage.Valid {
			log.ErrorMessage = errorMessage.String
		}

		logs = append(logs, log)
	}

	return logs, nil
}
