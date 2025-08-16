package database

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
)

type DatastoreDB struct {
	client    *datastore.Client
	projectID string
}

type UserEntity struct {
	ID                  int64     `datastore:"-"`
	GoogleID            string    `datastore:"google_id"`
	Email               string    `datastore:"email"`
	Name                string    `datastore:"name"`
	Avatar              string    `datastore:"avatar"`
	CreatedAt           time.Time `datastore:"created_at"`
	SubscriptionStatus  string    `datastore:"subscription_status"`
	SubscriptionID      string    `datastore:"subscription_id"`
	TrialEndsAt         time.Time `datastore:"trial_ends_at"`
	LastPaymentDate     time.Time `datastore:"last_payment_date"`
	IsAdmin             bool      `datastore:"is_admin"`
	FreeMonthsRemaining int       `datastore:"free_months_remaining"`
}

type UserFeedEntity struct {
	UserID int64 `datastore:"user_id"`
	FeedID int64 `datastore:"feed_id"`
}

type UserArticleEntity struct {
	UserID    int64 `datastore:"user_id"`
	ArticleID int64 `datastore:"article_id"`
	IsRead    bool  `datastore:"is_read"`
	IsStarred bool  `datastore:"is_starred"`
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

func (db *DatastoreDB) GetFeedByID(feedID int) (*Feed, error) {
	ctx := context.Background()
	
	key := datastore.IDKey("Feed", int64(feedID), nil)
	var entity FeedEntity
	err := db.client.Get(ctx, key, &entity)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, nil // Feed not found
		}
		return nil, fmt.Errorf("failed to get feed %d: %w", feedID, err)
	}
	
	entity.ID = key.ID
	feed := &Feed{
		ID:          int(entity.ID),
		Title:       entity.Title,
		URL:         entity.URL,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
		LastFetch:   entity.LastFetch,
	}
	
	return feed, nil
}

func (db *DatastoreDB) DeleteFeed(id int) error {
	ctx := context.Background()

	feedKey := datastore.IDKey("Feed", int64(id), nil)
	if err := db.client.Delete(ctx, feedKey); err != nil {
		return fmt.Errorf("failed to delete feed: %w", err)
	}

	query := datastore.NewQuery("Article").FilterField("feed_id", "=", int64(id)).KeysOnly()
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

	// Check if article already exists
	query := datastore.NewQuery("Article").FilterField("url", "=", article.URL).Limit(1)
	var existing []ArticleEntity
	keys, err := db.client.GetAll(ctx, query, &existing)
	if err != nil {
		return fmt.Errorf("failed to check for existing article: %w", err)
	}
	
	if len(existing) > 0 {
		// Article already exists, set the ID and return
		article.ID = int(keys[0].ID)
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
	key, err = db.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to save article: %w", err)
	}

	article.ID = int(key.ID)
	return nil
}

func (db *DatastoreDB) GetArticles(feedID int) ([]Article, error) {
	ctx := context.Background()

	query := datastore.NewQuery("Article").FilterField("feed_id", "=", int64(feedID)).Order("-published_at")
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

// Legacy methods removed - use multi-user methods instead

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

// User methods for Datastore
func (db *DatastoreDB) CreateUser(user *User) error {
	ctx := context.Background()

	// Set default subscription values for new users
	if user.SubscriptionStatus == "" {
		user.SubscriptionStatus = "trial"
	}
	if user.TrialEndsAt.IsZero() {
		user.TrialEndsAt = user.CreatedAt.AddDate(0, 0, 30) // 30 days from creation
	}

	entity := &UserEntity{
		GoogleID:           user.GoogleID,
		Email:              user.Email,
		Name:               user.Name,
		Avatar:             user.Avatar,
		CreatedAt:          user.CreatedAt,
		SubscriptionStatus: user.SubscriptionStatus,
		SubscriptionID:     user.SubscriptionID,
		TrialEndsAt:        user.TrialEndsAt,
		LastPaymentDate:    user.LastPaymentDate,
	}

	key := datastore.IncompleteKey("User", nil)
	key, err := db.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	user.ID = int(key.ID)
	return nil
}

func (db *DatastoreDB) GetUserByGoogleID(googleID string) (*User, error) {
	ctx := context.Background()

	query := datastore.NewQuery("User").FilterField("google_id", "=", googleID).Limit(1)
	var entities []UserEntity
	keys, err := db.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	entity := entities[0]
	entity.ID = keys[0].ID

	return &User{
		ID:                  int(entity.ID),
		GoogleID:            entity.GoogleID,
		Email:               entity.Email,
		Name:                entity.Name,
		Avatar:              entity.Avatar,
		CreatedAt:           entity.CreatedAt,
		SubscriptionStatus:  entity.SubscriptionStatus,
		SubscriptionID:      entity.SubscriptionID,
		TrialEndsAt:         entity.TrialEndsAt,
		LastPaymentDate:     entity.LastPaymentDate,
		IsAdmin:             entity.IsAdmin,
		FreeMonthsRemaining: entity.FreeMonthsRemaining,
	}, nil
}

func (db *DatastoreDB) GetUserByID(userID int) (*User, error) {
	ctx := context.Background()

	key := datastore.IDKey("User", int64(userID), nil)
	var entity UserEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	entity.ID = int64(userID)

	return &User{
		ID:                  int(entity.ID),
		GoogleID:            entity.GoogleID,
		Email:               entity.Email,
		Name:                entity.Name,
		Avatar:              entity.Avatar,
		CreatedAt:           entity.CreatedAt,
		SubscriptionStatus:  entity.SubscriptionStatus,
		SubscriptionID:      entity.SubscriptionID,
		TrialEndsAt:         entity.TrialEndsAt,
		LastPaymentDate:     entity.LastPaymentDate,
		IsAdmin:             entity.IsAdmin,
		FreeMonthsRemaining: entity.FreeMonthsRemaining,
	}, nil
}

func (db *DatastoreDB) GetUserFeeds(userID int) ([]Feed, error) {
	ctx := context.Background()

	// Query for user feed relationships
	query := datastore.NewQuery("UserFeed").FilterField("user_id", "=", int64(userID))
	var userFeedEntities []UserFeedEntity
	_, err := db.client.GetAll(ctx, query, &userFeedEntities)
	if err != nil {
		return nil, fmt.Errorf("failed to get user feeds: %w", err)
	}

	if len(userFeedEntities) == 0 {
		return []Feed{}, nil
	}

	// Get feed IDs
	feedIDs := make([]int64, len(userFeedEntities))
	for i, uf := range userFeedEntities {
		feedIDs[i] = uf.FeedID
	}

	// Query for the actual feeds
	feedKeys := make([]*datastore.Key, len(feedIDs))
	for i, feedID := range feedIDs {
		feedKeys[i] = datastore.IDKey("Feed", feedID, nil)
	}

	feedEntities := make([]FeedEntity, len(feedKeys))
	err = db.client.GetMulti(ctx, feedKeys, feedEntities)
	if err != nil {
		// Check for partial success - some feeds might exist, others might not
		if multiErr, ok := err.(datastore.MultiError); ok {
			validFeeds := []Feed{}
			for i, singleErr := range multiErr {
				if singleErr == nil {
					entity := feedEntities[i]
					entity.ID = feedIDs[i]
					validFeeds = append(validFeeds, Feed{
						ID:          int(entity.ID),
						Title:       entity.Title,
						URL:         entity.URL,
						Description: entity.Description,
						CreatedAt:   entity.CreatedAt,
						UpdatedAt:   entity.UpdatedAt,
						LastFetch:   entity.LastFetch,
					})
				}
			}
			return validFeeds, nil
		}
		return nil, fmt.Errorf("failed to get feeds: %w", err)
	}

	feeds := make([]Feed, len(feedEntities))
	for i, entity := range feedEntities {
		entity.ID = feedIDs[i]
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

func (db *DatastoreDB) SubscribeUserToFeed(userID, feedID int) error {
	ctx := context.Background()

	// Check if subscription already exists
	query := datastore.NewQuery("UserFeed").
		FilterField("user_id", "=", int64(userID)).
		FilterField("feed_id", "=", int64(feedID)).
		Limit(1)
	
	var existing []UserFeedEntity
	_, err := db.client.GetAll(ctx, query, &existing)
	if err != nil {
		return fmt.Errorf("failed to check existing subscription: %w", err)
	}

	if len(existing) > 0 {
		// Already subscribed
		return nil
	}

	// Create new subscription
	entity := &UserFeedEntity{
		UserID: int64(userID),
		FeedID: int64(feedID),
	}

	// Use a composite key to ensure uniqueness
	key := datastore.NameKey("UserFeed", fmt.Sprintf("%d_%d", userID, feedID), nil)
	_, err = db.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	return nil
}

func (db *DatastoreDB) UnsubscribeUserFromFeed(userID, feedID int) error {
	ctx := context.Background()

	// Use the same composite key format as SubscribeUserToFeed
	key := datastore.NameKey("UserFeed", fmt.Sprintf("%d_%d", userID, feedID), nil)
	err := db.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	return nil
}

func (db *DatastoreDB) GetUserArticles(userID int) ([]Article, error) {
	// First get user's subscribed feeds
	feeds, err := db.GetUserFeeds(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user feeds: %w", err)
	}

	if len(feeds) == 0 {
		return []Article{}, nil
	}

	// Get all articles from subscribed feeds
	var allArticles []Article
	for _, feed := range feeds {
		articles, err := db.GetUserFeedArticles(userID, feed.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get articles for feed %d: %w", feed.ID, err)
		}
		allArticles = append(allArticles, articles...)
	}

	return allArticles, nil
}

func (db *DatastoreDB) GetUserFeedArticles(userID, feedID int) ([]Article, error) {
	ctx := context.Background()

	// First verify user is subscribed to this feed
	subscriptionQuery := datastore.NewQuery("UserFeed").
		FilterField("user_id", "=", int64(userID)).
		FilterField("feed_id", "=", int64(feedID)).
		Limit(1)
	
	var subscriptions []UserFeedEntity
	_, err := db.client.GetAll(ctx, subscriptionQuery, &subscriptions)
	if err != nil {
		return nil, fmt.Errorf("failed to check user subscription: %w", err)
	}

	if len(subscriptions) == 0 {
		// User is not subscribed to this feed
		return []Article{}, nil
	}

	// Get articles for the feed
	query := datastore.NewQuery("Article").FilterField("feed_id", "=", int64(feedID)).Order("-published_at")
	var articleEntities []ArticleEntity
	keys, err := db.client.GetAll(ctx, query, &articleEntities)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles: %w", err)
	}

	articles := make([]Article, len(articleEntities))
	for i, entity := range articleEntities {
		entity.ID = keys[i].ID
		
		// Get user-specific read/starred status
		userArticle, err := db.GetUserArticleStatus(userID, int(entity.ID))
		isRead := false
		isStarred := false
		if err == nil && userArticle != nil {
			isRead = userArticle.IsRead
			isStarred = userArticle.IsStarred
		}

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
			IsRead:      isRead,
			IsStarred:   isStarred,
		}
	}

	return articles, nil
}

func (db *DatastoreDB) GetUserArticleStatus(userID, articleID int) (*UserArticle, error) {
	ctx := context.Background()

	key := datastore.NameKey("UserArticle", fmt.Sprintf("%d_%d", userID, articleID), nil)
	var entity UserArticleEntity
	err := db.client.Get(ctx, key, &entity)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, fmt.Errorf("user article status not found")
		}
		return nil, fmt.Errorf("failed to get user article status: %w", err)
	}

	return &UserArticle{
		UserID:    int(entity.UserID),
		ArticleID: int(entity.ArticleID),
		IsRead:    entity.IsRead,
		IsStarred: entity.IsStarred,
	}, nil
}

func (db *DatastoreDB) SetUserArticleStatus(userID, articleID int, isRead, isStarred bool) error {
	ctx := context.Background()

	entity := &UserArticleEntity{
		UserID:    int64(userID),
		ArticleID: int64(articleID),
		IsRead:    isRead,
		IsStarred: isStarred,
	}

	key := datastore.NameKey("UserArticle", fmt.Sprintf("%d_%d", userID, articleID), nil)
	_, err := db.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to set user article status: %w", err)
	}

	return nil
}

func (db *DatastoreDB) MarkUserArticleRead(userID, articleID int, isRead bool) error {
	// Get existing status or create new one
	existing, err := db.GetUserArticleStatus(userID, articleID)
	isStarred := false
	if err == nil && existing != nil {
		isStarred = existing.IsStarred
	}

	return db.SetUserArticleStatus(userID, articleID, isRead, isStarred)
}

func (db *DatastoreDB) ToggleUserArticleStar(userID, articleID int) error {
	// Get existing status
	existing, err := db.GetUserArticleStatus(userID, articleID)
	isRead := false
	isStarred := false
	
	if err == nil && existing != nil {
		isRead = existing.IsRead
		isStarred = existing.IsStarred
	}

	// Toggle starred status
	return db.SetUserArticleStatus(userID, articleID, isRead, !isStarred)
}

func (db *DatastoreDB) BatchSetUserArticleStatus(userID int, articles []Article, isRead, isStarred bool) error {
	if len(articles) == 0 {
		return nil
	}
	
	ctx := context.Background()
	
	// Batch process articles in chunks to avoid datastore limits
	chunkSize := 100
	for i := 0; i < len(articles); i += chunkSize {
		end := i + chunkSize
		if end > len(articles) {
			end = len(articles)
		}
		
		chunk := articles[i:end]
		entities := make([]*UserArticle, len(chunk))
		keys := make([]*datastore.Key, len(chunk))
		
		for j, article := range chunk {
			entities[j] = &UserArticle{
				UserID:    userID,
				ArticleID: article.ID,
				IsRead:    isRead,
				IsStarred: isStarred,
			}
			
			// Create composite key
			keyStr := fmt.Sprintf("%d_%d", userID, article.ID)
			keys[j] = datastore.NameKey("UserArticle", keyStr, nil)
		}
		
		_, err := db.client.PutMulti(ctx, keys, entities)
		if err != nil {
			return err
		}
	}
	
	return nil
}

func (db *DatastoreDB) GetUserUnreadCounts(userID int) (map[int]int, error) {
	ctx := context.Background()
	
	// Get all user feeds
	userFeeds, err := db.GetUserFeeds(userID)
	if err != nil {
		return nil, err
	}
	
	unreadCounts := make(map[int]int)
	
	// For each feed, count unread articles
	for _, feed := range userFeeds {
		var articles []ArticleEntity
		// Use eventually consistent query for articles (this is fine as articles don't change read status)
		q := datastore.NewQuery("Article").FilterField("feed_id", "=", int64(feed.ID))
		keys, err := db.client.GetAll(ctx, q, &articles)
		if err != nil {
			return nil, err
		}
		
		unreadCount := 0
		for i := range articles {
			articleID := int(keys[i].ID)
			// Check if user has read this article with strongly consistent read
			userArticleKey := datastore.NameKey("UserArticle", fmt.Sprintf("%d_%d", userID, articleID), nil)
			var userArticleEntity UserArticleEntity
			err := db.client.Get(ctx, userArticleKey, &userArticleEntity)
			
			// If no UserArticle record exists, or if it exists but is not read, count as unread
			if err == datastore.ErrNoSuchEntity || (err == nil && !userArticleEntity.IsRead) {
				unreadCount++
			} else if err != nil {
				// Some other error occurred, treat as unread to be safe
				unreadCount++
			}
		}
		
		unreadCounts[feed.ID] = unreadCount
	}
	
	return unreadCounts, nil
}

func (db *DatastoreDB) GetAllUserFeeds() ([]Feed, error) {
	ctx := context.Background()

	// Query for all user feed relationships
	query := datastore.NewQuery("UserFeed")
	var userFeedEntities []UserFeedEntity
	_, err := db.client.GetAll(ctx, query, &userFeedEntities)
	if err != nil {
		return nil, fmt.Errorf("failed to query user feeds: %w", err)
	}

	// Get unique feed IDs
	feedIDMap := make(map[int64]bool)
	for _, userFeed := range userFeedEntities {
		feedIDMap[userFeed.FeedID] = true
	}

	// Fetch all unique feeds
	var feeds []Feed
	for feedID := range feedIDMap {
		feed, err := db.GetFeedByID(int(feedID))
		if err != nil {
			continue
		}
		if feed != nil {
			feeds = append(feeds, *feed)
		}
	}

	return feeds, nil
}

// Subscription management methods
func (db *DatastoreDB) UpdateUserSubscription(userID int, status, subscriptionID string, lastPaymentDate time.Time) error {
	ctx := context.Background()

	key := datastore.IDKey("User", int64(userID), nil)
	var entity UserEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	entity.SubscriptionStatus = status
	entity.SubscriptionID = subscriptionID
	entity.LastPaymentDate = lastPaymentDate

	_, err := db.client.Put(ctx, key, &entity)
	if err != nil {
		return fmt.Errorf("failed to update user subscription: %w", err)
	}

	return nil
}

func (db *DatastoreDB) IsUserSubscriptionActive(userID int) (bool, error) {
	ctx := context.Background()

	key := datastore.IDKey("User", int64(userID), nil)
	var entity UserEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	// User is active if:
	// 1. They have an active paid subscription, OR
	// 2. They're on trial and trial hasn't expired
	if entity.SubscriptionStatus == "active" {
		return true, nil
	}
	
	if entity.SubscriptionStatus == "trial" && time.Now().Before(entity.TrialEndsAt) {
		return true, nil
	}
	
	return false, nil
}

func (db *DatastoreDB) GetUserFeedCount(userID int) (int, error) {
	ctx := context.Background()

	// Use Count() instead of GetAll for better performance
	query := datastore.NewQuery("UserFeed").FilterField("user_id", "=", int64(userID))
	count, err := db.client.Count(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get user feed count: %w", err)
	}
	
	return count, nil
}

// Admin management methods
func (db *DatastoreDB) SetUserAdmin(userID int, isAdmin bool) error {
	ctx := context.Background()
	
	// Get user entity first
	userKey := datastore.IDKey("User", int64(userID), nil)
	var user UserEntity
	
	err := db.client.Get(ctx, userKey, &user)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	// Update admin status
	user.IsAdmin = isAdmin
	
	// Save back to datastore
	_, err = db.client.Put(ctx, userKey, &user)
	if err != nil {
		return fmt.Errorf("failed to update user admin status: %w", err)
	}
	
	return nil
}

func (db *DatastoreDB) GrantFreeMonths(userID int, months int) error {
	ctx := context.Background()
	
	// Get user entity first
	userKey := datastore.IDKey("User", int64(userID), nil)
	var user UserEntity
	
	err := db.client.Get(ctx, userKey, &user)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	// Add free months
	user.FreeMonthsRemaining += months
	
	// Save back to datastore
	_, err = db.client.Put(ctx, userKey, &user)
	if err != nil {
		return fmt.Errorf("failed to update user free months: %w", err)
	}
	
	return nil
}

func (db *DatastoreDB) GetUserByEmail(email string) (*User, error) {
	ctx := context.Background()
	
	query := datastore.NewQuery("User").FilterField("email", "=", email).Limit(1)
	
	var users []UserEntity
	keys, err := db.client.GetAll(ctx, query, &users)
	if err != nil {
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}
	
	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	
	user := users[0]
	user.ID = keys[0].ID
	
	return &User{
		ID:                  int(user.ID),
		GoogleID:            user.GoogleID,
		Email:               user.Email,
		Name:                user.Name,
		Avatar:              user.Avatar,
		CreatedAt:           user.CreatedAt,
		SubscriptionStatus:  user.SubscriptionStatus,
		SubscriptionID:      user.SubscriptionID,
		TrialEndsAt:         user.TrialEndsAt,
		LastPaymentDate:     user.LastPaymentDate,
		IsAdmin:             user.IsAdmin,
		FreeMonthsRemaining: user.FreeMonthsRemaining,
	}, nil
}
