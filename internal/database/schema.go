package database

import (
	"database/sql"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database interface {
	AddFeed(feed *Feed) error
	GetFeeds() ([]Feed, error)
	DeleteFeed(id int) error
	AddArticle(article *Article) error
	GetArticles(feedID int) ([]Article, error)
	GetAllArticles() ([]Article, error)
	MarkRead(articleID int, isRead bool) error
	ToggleStar(articleID int) error
	UpdateFeedLastFetch(feedID int, lastFetch time.Time) error
	Close() error
}

type DB struct {
	*sql.DB
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
		is_read BOOLEAN DEFAULT FALSE,
		is_starred BOOLEAN DEFAULT FALSE,
		FOREIGN KEY (feed_id) REFERENCES feeds (id) ON DELETE CASCADE
	);`

	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(feedsTable); err != nil {
		return err
	}

	if _, err := db.Exec(articlesTable); err != nil {
		return err
	}

	if _, err := db.Exec(usersTable); err != nil {
		return err
	}

	return nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) AddFeed(feed *Feed) error {
	query := `INSERT INTO feeds (title, url, description, created_at, updated_at, last_fetch) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	
	result, err := db.Exec(query, feed.Title, feed.URL, feed.Description, 
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
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
	_, err := db.Exec(query, id)
	return err
}

func (db *DB) AddArticle(article *Article) error {
	query := `INSERT OR IGNORE INTO articles 
			  (feed_id, title, url, content, description, author, published_at, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := db.Exec(query, article.FeedID, article.Title, article.URL, article.Content,
		article.Description, article.Author, article.PublishedAt, article.CreatedAt)
	return err
}

func (db *DB) GetArticles(feedID int) ([]Article, error) {
	query := `SELECT id, feed_id, title, url, content, description, author, 
			  published_at, created_at, is_read, is_starred 
			  FROM articles WHERE feed_id = ? ORDER BY published_at DESC`
	
	rows, err := db.Query(query, feedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (db *DB) GetAllArticles() ([]Article, error) {
	query := `SELECT a.id, a.feed_id, a.title, a.url, a.content, a.description, a.author, 
			  a.published_at, a.created_at, a.is_read, a.is_starred 
			  FROM articles a 
			  JOIN feeds f ON a.feed_id = f.id 
			  ORDER BY a.published_at DESC`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (db *DB) MarkRead(articleID int, isRead bool) error {
	query := `UPDATE articles SET is_read = ? WHERE id = ?`
	_, err := db.Exec(query, isRead, articleID)
	return err
}

func (db *DB) ToggleStar(articleID int) error {
	query := `UPDATE articles SET is_starred = NOT is_starred WHERE id = ?`
	_, err := db.Exec(query, articleID)
	return err
}

func (db *DB) UpdateFeedLastFetch(feedID int, lastFetch time.Time) error {
	query := `UPDATE feeds SET last_fetch = ? WHERE id = ?`
	_, err := db.Exec(query, lastFetch, feedID)
	return err
}