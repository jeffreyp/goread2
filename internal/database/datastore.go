package database

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
)

type DatastoreDB struct {
	client    *datastore.Client
	projectID string
}

type FeedEntity struct {
	ID          int64     `datastore:"-"`
	Title       string    `datastore:"title"`
	URL         string    `datastore:"url"`
	Description string    `datastore:"description"`
	CreatedAt   time.Time `datastore:"created_at"`
	UpdatedAt   time.Time `datastore:"updated_at"`
	LastFetch   time.Time `datastore:"last_fetch"`
}

type ArticleEntity struct {
	ID          int64     `datastore:"-"`
	FeedID      int64     `datastore:"feed_id"`
	Title       string    `datastore:"title"`
	URL         string    `datastore:"url"`
	Content     string    `datastore:"content,noindex"`
	Description string    `datastore:"description,noindex"`
	Author      string    `datastore:"author"`
	PublishedAt time.Time `datastore:"published_at"`
	CreatedAt   time.Time `datastore:"created_at"`
	IsRead      bool      `datastore:"is_read"`
	IsStarred   bool      `datastore:"is_starred"`
}

func NewDatastoreDB(projectID string) (*DatastoreDB, error) {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore client: %w", err)
	}

	return &DatastoreDB{
		client:    client,
		projectID: projectID,
	}, nil
}

func (db *DatastoreDB) Close() error {
	return db.client.Close()
}

func (db *DatastoreDB) AddFeed(feed *Feed) error {
	ctx := context.Background()
	
	entity := &FeedEntity{
		Title:       feed.Title,
		URL:         feed.URL,
		Description: feed.Description,
		CreatedAt:   feed.CreatedAt,
		UpdatedAt:   feed.UpdatedAt,
		LastFetch:   feed.LastFetch,
	}

	key := datastore.IncompleteKey("Feed", nil)
	key, err := db.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to save feed: %w", err)
	}

	feed.ID = int(key.ID)
	return nil
}

func (db *DatastoreDB) GetFeeds() ([]Feed, error) {
	ctx := context.Background()
	
	query := datastore.NewQuery("Feed").Order("title")
	var entities []FeedEntity
	keys, err := db.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to get feeds: %w", err)
	}

	feeds := make([]Feed, len(entities))
	for i, entity := range entities {
		entity.ID = keys[i].ID
		feeds[i] = Feed{
			ID:          int(entity.ID),
			Title:       entity.Title,
			URL:         entity.URL,
			Description: entity.Description,
			CreatedAt:   entity.CreatedAt,
			UpdatedAt:   entity.UpdatedAt,
			LastFetch:   entity.LastFetch,
		}
	}

	return feeds, nil
}

func (db *DatastoreDB) DeleteFeed(id int) error {
	ctx := context.Background()
	
	feedKey := datastore.IDKey("Feed", int64(id), nil)
	if err := db.client.Delete(ctx, feedKey); err != nil {
		return fmt.Errorf("failed to delete feed: %w", err)
	}

	query := datastore.NewQuery("Article").Filter("feed_id =", int64(id)).KeysOnly()
	keys, err := db.client.GetAll(ctx, query, nil)
	if err != nil {
		return fmt.Errorf("failed to get articles to delete: %w", err)
	}

	if len(keys) > 0 {
		if err := db.client.DeleteMulti(ctx, keys); err != nil {
			return fmt.Errorf("failed to delete articles: %w", err)
		}
	}

	return nil
}

func (db *DatastoreDB) AddArticle(article *Article) error {
	ctx := context.Background()
	
	query := datastore.NewQuery("Article").Filter("url =", article.URL).Limit(1)
	var existing []ArticleEntity
	if _, err := db.client.GetAll(ctx, query, &existing); err == nil && len(existing) > 0 {
		return nil
	}

	entity := &ArticleEntity{
		FeedID:      int64(article.FeedID),
		Title:       article.Title,
		URL:         article.URL,
		Content:     article.Content,
		Description: article.Description,
		Author:      article.Author,
		PublishedAt: article.PublishedAt,
		CreatedAt:   article.CreatedAt,
		IsRead:      article.IsRead,
		IsStarred:   article.IsStarred,
	}

	key := datastore.IncompleteKey("Article", nil)
	key, err := db.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to save article: %w", err)
	}

	article.ID = int(key.ID)
	return nil
}

func (db *DatastoreDB) GetArticles(feedID int) ([]Article, error) {
	ctx := context.Background()
	
	query := datastore.NewQuery("Article").Filter("feed_id =", int64(feedID)).Order("-published_at")
	var entities []ArticleEntity
	keys, err := db.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles: %w", err)
	}

	articles := make([]Article, len(entities))
	for i, entity := range entities {
		entity.ID = keys[i].ID
		articles[i] = Article{
			ID:          int(entity.ID),
			FeedID:      int(entity.FeedID),
			Title:       entity.Title,
			URL:         entity.URL,
			Content:     entity.Content,
			Description: entity.Description,
			Author:      entity.Author,
			PublishedAt: entity.PublishedAt,
			CreatedAt:   entity.CreatedAt,
			IsRead:      entity.IsRead,
			IsStarred:   entity.IsStarred,
		}
	}

	return articles, nil
}

func (db *DatastoreDB) GetAllArticles() ([]Article, error) {
	ctx := context.Background()
	
	query := datastore.NewQuery("Article").Order("-published_at")
	var entities []ArticleEntity
	keys, err := db.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to get all articles: %w", err)
	}

	articles := make([]Article, len(entities))
	for i, entity := range entities {
		entity.ID = keys[i].ID
		articles[i] = Article{
			ID:          int(entity.ID),
			FeedID:      int(entity.FeedID),
			Title:       entity.Title,
			URL:         entity.URL,
			Content:     entity.Content,
			Description: entity.Description,
			Author:      entity.Author,
			PublishedAt: entity.PublishedAt,
			CreatedAt:   entity.CreatedAt,
			IsRead:      entity.IsRead,
			IsStarred:   entity.IsStarred,
		}
	}

	return articles, nil
}

func (db *DatastoreDB) MarkRead(articleID int, isRead bool) error {
	ctx := context.Background()
	
	key := datastore.IDKey("Article", int64(articleID), nil)
	var entity ArticleEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get article: %w", err)
	}

	entity.IsRead = isRead
	if _, err := db.client.Put(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to update article: %w", err)
	}

	return nil
}

func (db *DatastoreDB) ToggleStar(articleID int) error {
	ctx := context.Background()
	
	key := datastore.IDKey("Article", int64(articleID), nil)
	var entity ArticleEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get article: %w", err)
	}

	entity.IsStarred = !entity.IsStarred
	if _, err := db.client.Put(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to update article: %w", err)
	}

	return nil
}

func (db *DatastoreDB) UpdateFeedLastFetch(feedID int, lastFetch time.Time) error {
	ctx := context.Background()
	
	key := datastore.IDKey("Feed", int64(feedID), nil)
	var entity FeedEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	entity.LastFetch = lastFetch
	if _, err := db.client.Put(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to update feed: %w", err)
	}

	return nil
}