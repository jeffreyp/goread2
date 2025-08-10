package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database interface {
	// User methods
	CreateUser(user *User) error
	GetUserByGoogleID(googleID string) (*User, error)
	GetUserByID(userID int) (*User, error)

	// Feed methods
	AddFeed(feed *Feed) error
	GetFeeds() ([]Feed, error)
	GetUserFeeds(userID int) ([]Feed, error)
	GetAllUserFeeds() ([]Feed, error)
	DeleteFeed(id int) error
	SubscribeUserToFeed(userID, feedID int) error
	UnsubscribeUserFromFeed(userID, feedID int) error

	// Article methods
	AddArticle(article *Article) error
	GetArticles(feedID int) ([]Article, error)
	GetUserArticles(userID int) ([]Article, error)
	GetUserFeedArticles(userID, feedID int) ([]Article, error)

	// User article status methods
	GetUserArticleStatus(userID, articleID int) (*UserArticle, error)
	SetUserArticleStatus(userID, articleID int, isRead, isStarred bool) error
	BatchSetUserArticleStatus(userID int, articles []Article, isRead, isStarred bool) error
	MarkUserArticleRead(userID, articleID int, isRead bool) error
	ToggleUserArticleStar(userID, articleID int) error
	GetUserUnreadCounts(userID int) (map[int]int, error)

	// Legacy methods (for migration)
	GetAllArticles() ([]Article, error)

	UpdateFeedLastFetch(feedID int, lastFetch time.Time) error
	Close() error
}

type DB struct {
	*sql.DB
}

type User struct {
	ID        int       `json:"id"`
	GoogleID  string    `json:"google_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
}

type Feed struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastFetch   time.Time `json:"last_fetch"`
}

type Article struct {
	ID          int       `json:"id"`
	FeedID      int       `json:"feed_id"`
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

func InitDB() (Database, error) {
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		return NewDatastoreDB(projectID)
	}

	db, err := sql.Open("sqlite3", "./goread2.db")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	dbWrapper := &DB{db}
	if err := dbWrapper.createTables(); err != nil {
		return nil, err
	}

	return dbWrapper, nil
}

func (db *DB) createTables() error {
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		google_id TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		avatar TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	feedsTable := `
	CREATE TABLE IF NOT EXISTS feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		url TEXT UNIQUE NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_fetch DATETIME DEFAULT CURRENT_TIMESTAMP
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

	tables := []string{usersTable, feedsTable, articlesTable, userFeedsTable, userArticlesTable}

	for _, table := range tables {
		if _, err := db.DB.Exec(table); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) AddFeed(feed *Feed) error {
	query := `INSERT INTO feeds (title, url, description, created_at, updated_at, last_fetch) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	result, err := db.DB.Exec(query, feed.Title, feed.URL, feed.Description,
		feed.CreatedAt, feed.UpdatedAt, feed.LastFetch)
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

func (db *DB) GetFeeds() ([]Feed, error) {
	query := `SELECT id, title, url, description, created_at, updated_at, last_fetch FROM feeds ORDER BY title`
	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var feeds []Feed
	for rows.Next() {
		var feed Feed
		err := rows.Scan(&feed.ID, &feed.Title, &feed.URL, &feed.Description,
			&feed.CreatedAt, &feed.UpdatedAt, &feed.LastFetch)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, nil
}

func (db *DB) DeleteFeed(id int) error {
	query := `DELETE FROM feeds WHERE id = ?`
	_, err := db.DB.Exec(query, id)
	return err
}

func (db *DB) AddArticle(article *Article) error {
	query := `INSERT OR IGNORE INTO articles 
			  (feed_id, title, url, content, description, author, published_at, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.DB.Exec(query, article.FeedID, article.Title, article.URL, article.Content,
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
		err = db.DB.QueryRow(query, article.URL).Scan(&article.ID)
	}
	return err
}

func (db *DB) GetArticles(feedID int) ([]Article, error) {
	query := `SELECT id, feed_id, title, url, content, description, author, 
			  published_at, created_at 
			  FROM articles WHERE feed_id = ? ORDER BY published_at DESC`

	rows, err := db.DB.Query(query, feedID)
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

func (db *DB) GetAllArticles() ([]Article, error) {
	query := `SELECT a.id, a.feed_id, a.title, a.url, a.content, a.description, a.author, 
			  a.published_at, a.created_at 
			  FROM articles a 
			  JOIN feeds f ON a.feed_id = f.id 
			  ORDER BY a.published_at DESC`

	rows, err := db.DB.Query(query)
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
	_, err := db.DB.Exec(query, lastFetch, feedID)
	return err
}

// User methods
func (db *DB) CreateUser(user *User) error {
	query := `INSERT INTO users (google_id, email, name, avatar, created_at) 
			  VALUES (?, ?, ?, ?, ?)`

	result, err := db.DB.Exec(query, user.GoogleID, user.Email, user.Name, user.Avatar, user.CreatedAt)
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
	query := `SELECT id, google_id, email, name, avatar, created_at FROM users WHERE google_id = ?`

	var user User
	err := db.DB.QueryRow(query, googleID).Scan(&user.ID, &user.GoogleID, &user.Email,
		&user.Name, &user.Avatar, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *DB) GetUserByID(userID int) (*User, error) {
	query := `SELECT id, google_id, email, name, avatar, created_at FROM users WHERE id = ?`

	var user User
	err := db.DB.QueryRow(query, userID).Scan(&user.ID, &user.GoogleID, &user.Email,
		&user.Name, &user.Avatar, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// User feed methods
func (db *DB) GetUserFeeds(userID int) ([]Feed, error) {
	query := `SELECT f.id, f.title, f.url, f.description, f.created_at, f.updated_at, f.last_fetch 
			  FROM feeds f 
			  JOIN user_feeds uf ON f.id = uf.feed_id 
			  WHERE uf.user_id = ? 
			  ORDER BY f.title`

	rows, err := db.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var feeds []Feed
	for rows.Next() {
		var feed Feed
		err := rows.Scan(&feed.ID, &feed.Title, &feed.URL, &feed.Description,
			&feed.CreatedAt, &feed.UpdatedAt, &feed.LastFetch)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, nil
}

func (db *DB) GetAllUserFeeds() ([]Feed, error) {
	query := `SELECT DISTINCT f.id, f.title, f.url, f.description, f.created_at, f.updated_at, f.last_fetch 
			  FROM feeds f 
			  JOIN user_feeds uf ON f.id = uf.feed_id 
			  ORDER BY f.title`

	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var feeds []Feed
	for rows.Next() {
		var feed Feed
		err := rows.Scan(&feed.ID, &feed.Title, &feed.URL, &feed.Description,
			&feed.CreatedAt, &feed.UpdatedAt, &feed.LastFetch)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, nil
}

func (db *DB) SubscribeUserToFeed(userID, feedID int) error {
	query := `INSERT OR IGNORE INTO user_feeds (user_id, feed_id) VALUES (?, ?)`
	_, err := db.DB.Exec(query, userID, feedID)
	return err
}

func (db *DB) UnsubscribeUserFromFeed(userID, feedID int) error {
	query := `DELETE FROM user_feeds WHERE user_id = ? AND feed_id = ?`
	_, err := db.DB.Exec(query, userID, feedID)
	return err
}

// User article methods
func (db *DB) GetUserArticles(userID int) ([]Article, error) {
	query := `SELECT a.id, a.feed_id, a.title, a.url, a.content, a.description, a.author, 
			  a.published_at, a.created_at, 
			  COALESCE(ua.is_read, 0) as is_read, 
			  COALESCE(ua.is_starred, 0) as is_starred
			  FROM articles a 
			  JOIN user_feeds uf ON a.feed_id = uf.feed_id 
			  LEFT JOIN user_articles ua ON a.id = ua.article_id AND ua.user_id = ?
			  WHERE uf.user_id = ? 
			  ORDER BY a.published_at DESC`

	rows, err := db.DB.Query(query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var articles []Article
	for rows.Next() {
		var article Article
		err := rows.Scan(&article.ID, &article.FeedID, &article.Title, &article.URL,
			&article.Content, &article.Description, &article.Author,
			&article.PublishedAt, &article.CreatedAt, &article.IsRead, &article.IsStarred)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}

	return articles, nil
}

func (db *DB) GetUserFeedArticles(userID, feedID int) ([]Article, error) {
	// First verify user is subscribed to this feed
	var subscriptionExists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM user_feeds WHERE user_id = ? AND feed_id = ?)`
	err := db.DB.QueryRow(checkQuery, userID, feedID).Scan(&subscriptionExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check user subscription: %w", err)
	}

	if !subscriptionExists {
		// User is not subscribed to this feed
		return []Article{}, nil
	}

	query := `SELECT a.id, a.feed_id, a.title, a.url, a.content, a.description, a.author, 
			  a.published_at, a.created_at, 
			  COALESCE(ua.is_read, 0) as is_read, 
			  COALESCE(ua.is_starred, 0) as is_starred
			  FROM articles a 
			  LEFT JOIN user_articles ua ON a.id = ua.article_id AND ua.user_id = ?
			  WHERE a.feed_id = ? 
			  ORDER BY a.published_at DESC`

	rows, err := db.DB.Query(query, userID, feedID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var articles []Article
	for rows.Next() {
		var article Article
		err := rows.Scan(&article.ID, &article.FeedID, &article.Title, &article.URL,
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
	err := db.DB.QueryRow(query, userID, articleID).Scan(&userArticle.UserID, &userArticle.ArticleID,
		&userArticle.IsRead, &userArticle.IsStarred)
	if err != nil {
		return nil, err
	}
	return &userArticle, nil
}

func (db *DB) SetUserArticleStatus(userID, articleID int, isRead, isStarred bool) error {
	query := `INSERT OR REPLACE INTO user_articles (user_id, article_id, is_read, is_starred) 
			  VALUES (?, ?, ?, ?)`
	_, err := db.DB.Exec(query, userID, articleID, isRead, isStarred)
	return err
}

func (db *DB) MarkUserArticleRead(userID, articleID int, isRead bool) error {
	// First check if record exists
	var dummy int
	checkQuery := `SELECT 1 FROM user_articles WHERE user_id = ? AND article_id = ?`
	err := db.DB.QueryRow(checkQuery, userID, articleID).Scan(&dummy)

	switch err {
	case sql.ErrNoRows:
		// Create new record
		query := `INSERT INTO user_articles (user_id, article_id, is_read, is_starred) 
				  VALUES (?, ?, ?, 0)`
		_, err = db.DB.Exec(query, userID, articleID, isRead)
	case nil:
		// Update existing record
		query := `UPDATE user_articles SET is_read = ? WHERE user_id = ? AND article_id = ?`
		_, err = db.DB.Exec(query, isRead, userID, articleID)
	}

	return err
}

func (db *DB) ToggleUserArticleStar(userID, articleID int) error {
	// First check if record exists
	var currentStarred bool
	checkQuery := `SELECT is_starred FROM user_articles WHERE user_id = ? AND article_id = ?`
	err := db.DB.QueryRow(checkQuery, userID, articleID).Scan(&currentStarred)

	switch err {
	case sql.ErrNoRows:
		// Create new record with starred = true
		query := `INSERT INTO user_articles (user_id, article_id, is_read, is_starred) 
				  VALUES (?, ?, 0, 1)`
		_, err = db.DB.Exec(query, userID, articleID)
	case nil:
		// Update existing record
		query := `UPDATE user_articles SET is_starred = ? WHERE user_id = ? AND article_id = ?`
		_, err = db.DB.Exec(query, !currentStarred, userID, articleID)
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
	
	_, err := db.DB.Exec(query, args...)
	return err
}

func (db *DB) GetUserUnreadCounts(userID int) (map[int]int, error) {
	query := `
		SELECT 
			a.feed_id,
			COUNT(*) as unread_count
		FROM articles a
		INNER JOIN user_feeds uf ON a.feed_id = uf.feed_id
		LEFT JOIN user_articles ua ON a.id = ua.article_id AND ua.user_id = ?
		WHERE uf.user_id = ? 
		AND (ua.is_read IS NULL OR ua.is_read = 0)
		GROUP BY a.feed_id
	`
	
	rows, err := db.DB.Query(query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	unreadCounts := make(map[int]int)
	for rows.Next() {
		var feedID, count int
		if err := rows.Scan(&feedID, &count); err != nil {
			return nil, err
		}
		unreadCounts[feedID] = count
	}
	
	return unreadCounts, rows.Err()
}
