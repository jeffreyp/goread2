package database

import (
	"context"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/datastore"
)

const (
	// datastoreTimeout is the default timeout for all Datastore operations
	// This prevents operations from hanging indefinitely in production
	datastoreTimeout = 30 * time.Second

	// maxArticlesPerFeed limits memory usage when paginating across multiple feeds
	// This prevents OOM errors when users subscribe to many active feeds
	// With this limit, even 1000 feeds would only load ~200KB of articles
	maxArticlesPerFeed = 200
)

type DatastoreDB struct {
	client    *datastore.Client
	projectID string
}

// newDatastoreContext creates a new context with the standard Datastore timeout
func newDatastoreContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), datastoreTimeout)
}

type UserEntity struct {
	ID                   int64     `datastore:"-"`
	GoogleID             string    `datastore:"google_id"`
	Email                string    `datastore:"email"`
	Name                 string    `datastore:"name"`
	Avatar               string    `datastore:"avatar"`
	CreatedAt            time.Time `datastore:"created_at"`
	SubscriptionStatus   string    `datastore:"subscription_status"`
	SubscriptionID       string    `datastore:"subscription_id"`
	TrialEndsAt          time.Time `datastore:"trial_ends_at"`
	LastPaymentDate      time.Time `datastore:"last_payment_date"`
	NextBillingDate      time.Time `datastore:"next_billing_date"`
	IsAdmin              bool      `datastore:"is_admin"`
	FreeMonthsRemaining  int       `datastore:"free_months_remaining"`
	MaxArticlesOnFeedAdd int       `datastore:"max_articles_on_feed_add"`
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

type AdminTokenEntity struct {
	ID          int64     `datastore:"-"`
	TokenHash   string    `datastore:"token_hash"`
	Description string    `datastore:"description"`
	CreatedAt   time.Time `datastore:"created_at"`
	LastUsedAt  time.Time `datastore:"last_used_at"`
	IsActive    bool      `datastore:"is_active"`
}

type SessionEntity struct {
	ID        string    `datastore:"-"` // SessionID is the key
	UserID    int64     `datastore:"user_id"`
	CreatedAt time.Time `datastore:"created_at"`
	ExpiresAt time.Time `datastore:"expires_at"`
}

type AuditLogEntity struct {
	ID               int64     `datastore:"-"`
	Timestamp        time.Time `datastore:"timestamp"`
	AdminUserID      int64     `datastore:"admin_user_id"`
	AdminEmail       string    `datastore:"admin_email"`
	OperationType    string    `datastore:"operation_type"`
	TargetUserID     int64     `datastore:"target_user_id"`
	TargetUserEmail  string    `datastore:"target_user_email"`
	OperationDetails string    `datastore:"operation_details,noindex"`
	IPAddress        string    `datastore:"ip_address"`
	Result           string    `datastore:"result"`
	ErrorMessage     string    `datastore:"error_message,noindex"`
}

type FeedEntity struct {
	ID                    int64     `datastore:"-"`
	Title                 string    `datastore:"title"`
	URL                   string    `datastore:"url"`
	Description           string    `datastore:"description"`
	CreatedAt             time.Time `datastore:"created_at"`
	UpdatedAt             time.Time `datastore:"updated_at"`
	LastFetch             time.Time `datastore:"last_fetch"`
	LastChecked           time.Time `datastore:"last_checked"`
	LastHadNewContent     time.Time `datastore:"last_had_new_content"`
	AverageUpdateInterval int       `datastore:"average_update_interval"`
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
	ctx, cancel := newDatastoreContext()
	defer cancel()
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore client: %w", err)
	}

	return &DatastoreDB{
		client:    client,
		projectID: projectID,
	}, nil
}

// GetClient returns the underlying datastore client for direct access
func (db *DatastoreDB) GetClient() *datastore.Client {
	return db.client
}

func (db *DatastoreDB) Close() error {
	return db.client.Close()
}

func (db *DatastoreDB) AddFeed(feed *Feed) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	entity := &FeedEntity{
		Title:                 feed.Title,
		URL:                   feed.URL,
		Description:           feed.Description,
		CreatedAt:             feed.CreatedAt,
		UpdatedAt:             feed.UpdatedAt,
		LastFetch:             feed.LastFetch,
		LastChecked:           feed.LastChecked,
		LastHadNewContent:     feed.LastHadNewContent,
		AverageUpdateInterval: feed.AverageUpdateInterval,
	}

	key := datastore.IncompleteKey("Feed", nil)
	key, err := db.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to save feed: %w", err)
	}

	feed.ID = int(key.ID)
	return nil
}

func (db *DatastoreDB) UpdateFeed(feed *Feed) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Get the existing entity to preserve fields we're not updating
	key := datastore.IDKey("Feed", int64(feed.ID), nil)
	var existing FeedEntity
	err := db.client.Get(ctx, key, &existing)
	if err != nil {
		return fmt.Errorf("failed to get existing feed: %w", err)
	}

	// Update the fields we want to change
	existing.Title = feed.Title
	existing.Description = feed.Description
	existing.UpdatedAt = time.Now()

	// Save the updated entity
	_, err = db.client.Put(ctx, key, &existing)
	if err != nil {
		return fmt.Errorf("failed to update feed: %w", err)
	}

	return nil
}

func (db *DatastoreDB) UpdateFeedTracking(feedID int, lastChecked, lastHadNewContent time.Time, averageUpdateInterval int) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	key := datastore.IDKey("Feed", int64(feedID), nil)
	var entity FeedEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	entity.LastChecked = lastChecked
	// Only update LastHadNewContent if it's not zero (meaning there was new content)
	if !lastHadNewContent.IsZero() {
		entity.LastHadNewContent = lastHadNewContent
	}
	entity.AverageUpdateInterval = averageUpdateInterval

	if _, err := db.client.Put(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to update feed tracking: %w", err)
	}

	return nil
}

func (db *DatastoreDB) GetFeeds() ([]Feed, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

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
			ID:                    int(entity.ID),
			Title:                 entity.Title,
			URL:                   entity.URL,
			Description:           entity.Description,
			CreatedAt:             entity.CreatedAt,
			UpdatedAt:             entity.UpdatedAt,
			LastFetch:             entity.LastFetch,
			LastChecked:           entity.LastChecked,
			LastHadNewContent:     entity.LastHadNewContent,
			AverageUpdateInterval: entity.AverageUpdateInterval,
		}
	}

	return feeds, nil
}

func (db *DatastoreDB) GetFeedByURL(url string) (*Feed, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	query := datastore.NewQuery("Feed").FilterField("url", "=", url).Limit(1)
	var entities []FeedEntity
	keys, err := db.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed by URL: %w", err)
	}

	if len(entities) == 0 {
		return nil, nil // Feed not found
	}

	entity := entities[0]
	entity.ID = keys[0].ID
	feed := &Feed{
		ID:                    int(entity.ID),
		Title:                 entity.Title,
		URL:                   entity.URL,
		Description:           entity.Description,
		CreatedAt:             entity.CreatedAt,
		UpdatedAt:             entity.UpdatedAt,
		LastFetch:             entity.LastFetch,
		LastChecked:           entity.LastChecked,
		LastHadNewContent:     entity.LastHadNewContent,
		AverageUpdateInterval: entity.AverageUpdateInterval,
	}

	return feed, nil
}

func (db *DatastoreDB) GetFeedByID(feedID int) (*Feed, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

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
		ID:                    int(entity.ID),
		Title:                 entity.Title,
		URL:                   entity.URL,
		Description:           entity.Description,
		CreatedAt:             entity.CreatedAt,
		UpdatedAt:             entity.UpdatedAt,
		LastFetch:             entity.LastFetch,
		LastChecked:           entity.LastChecked,
		LastHadNewContent:     entity.LastHadNewContent,
		AverageUpdateInterval: entity.AverageUpdateInterval,
	}

	return feed, nil
}

func (db *DatastoreDB) DeleteFeed(id int) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

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
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Check if article already exists using keys-only query (1/3 cost of full entity read)
	query := datastore.NewQuery("Article").FilterField("url", "=", article.URL).KeysOnly().Limit(1)
	keys, err := db.client.GetAll(ctx, query, nil)
	if err != nil {
		return fmt.Errorf("failed to check for existing article: %w", err)
	}

	if len(keys) > 0 {
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
	ctx, cancel := newDatastoreContext()
	defer cancel()

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

func (db *DatastoreDB) FindArticleByURL(url string) (*Article, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	query := datastore.NewQuery("Article").FilterField("url", "=", url).Limit(1)
	var entities []ArticleEntity
	keys, err := db.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to find article by URL: %w", err)
	}

	if len(entities) == 0 {
		return nil, nil // Article not found
	}

	entity := entities[0]
	entity.ID = keys[0].ID

	article := Article{
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

	return &article, nil
}

// Legacy methods removed - use multi-user methods instead

func (db *DatastoreDB) UpdateFeedLastFetch(feedID int, lastFetch time.Time) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

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
	ctx, cancel := newDatastoreContext()
	defer cancel()

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

	entity := &UserEntity{
		GoogleID:             user.GoogleID,
		Email:                user.Email,
		Name:                 user.Name,
		Avatar:               user.Avatar,
		CreatedAt:            user.CreatedAt,
		SubscriptionStatus:   user.SubscriptionStatus,
		SubscriptionID:       user.SubscriptionID,
		TrialEndsAt:          user.TrialEndsAt,
		LastPaymentDate:      user.LastPaymentDate,
		NextBillingDate:      user.NextBillingDate,
		IsAdmin:              user.IsAdmin,
		FreeMonthsRemaining:  user.FreeMonthsRemaining,
		MaxArticlesOnFeedAdd: user.MaxArticlesOnFeedAdd,
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
	ctx, cancel := newDatastoreContext()
	defer cancel()

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

	maxArticles := entity.MaxArticlesOnFeedAdd
	if maxArticles == 0 {
		maxArticles = 100 // Default for existing users
	}

	return &User{
		ID:                   int(entity.ID),
		GoogleID:             entity.GoogleID,
		Email:                entity.Email,
		Name:                 entity.Name,
		Avatar:               entity.Avatar,
		CreatedAt:            entity.CreatedAt,
		SubscriptionStatus:   entity.SubscriptionStatus,
		SubscriptionID:       entity.SubscriptionID,
		TrialEndsAt:          entity.TrialEndsAt,
		LastPaymentDate:      entity.LastPaymentDate,
		NextBillingDate:      entity.NextBillingDate,
		IsAdmin:              entity.IsAdmin,
		FreeMonthsRemaining:  entity.FreeMonthsRemaining,
		MaxArticlesOnFeedAdd: maxArticles,
	}, nil
}

func (db *DatastoreDB) GetUserByID(userID int) (*User, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	key := datastore.IDKey("User", int64(userID), nil)
	var entity UserEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	entity.ID = int64(userID)

	maxArticles := entity.MaxArticlesOnFeedAdd
	if maxArticles == 0 {
		maxArticles = 100 // Default for existing users
	}

	return &User{
		ID:                   int(entity.ID),
		GoogleID:             entity.GoogleID,
		Email:                entity.Email,
		Name:                 entity.Name,
		Avatar:               entity.Avatar,
		CreatedAt:            entity.CreatedAt,
		SubscriptionStatus:   entity.SubscriptionStatus,
		SubscriptionID:       entity.SubscriptionID,
		TrialEndsAt:          entity.TrialEndsAt,
		LastPaymentDate:      entity.LastPaymentDate,
		NextBillingDate:      entity.NextBillingDate,
		IsAdmin:              entity.IsAdmin,
		FreeMonthsRemaining:  entity.FreeMonthsRemaining,
		MaxArticlesOnFeedAdd: maxArticles,
	}, nil
}

func (db *DatastoreDB) GetUserFeeds(userID int) ([]Feed, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Use eventually consistent query, but with retry logic for recent changes
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
						ID:                    int(entity.ID),
						Title:                 entity.Title,
						URL:                   entity.URL,
						Description:           entity.Description,
						CreatedAt:             entity.CreatedAt,
						UpdatedAt:             entity.UpdatedAt,
						LastFetch:             entity.LastFetch,
						LastChecked:           entity.LastChecked,
						LastHadNewContent:     entity.LastHadNewContent,
						AverageUpdateInterval: entity.AverageUpdateInterval,
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
			ID:                    int(entity.ID),
			Title:                 entity.Title,
			URL:                   entity.URL,
			Description:           entity.Description,
			CreatedAt:             entity.CreatedAt,
			UpdatedAt:             entity.UpdatedAt,
			LastFetch:             entity.LastFetch,
			LastChecked:           entity.LastChecked,
			LastHadNewContent:     entity.LastHadNewContent,
			AverageUpdateInterval: entity.AverageUpdateInterval,
		}
	}

	return feeds, nil
}

func (db *DatastoreDB) SubscribeUserToFeed(userID, feedID int) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Check if subscription already exists using keys-only query (1/3 cost of full entity read)
	query := datastore.NewQuery("UserFeed").
		FilterField("user_id", "=", int64(userID)).
		FilterField("feed_id", "=", int64(feedID)).
		KeysOnly().
		Limit(1)

	keys, err := db.client.GetAll(ctx, query, nil)
	if err != nil {
		return fmt.Errorf("failed to check existing subscription: %w", err)
	}

	if len(keys) > 0 {
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
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Remove the user-feed subscription
	// Note: Orphaned UserArticle entities will be cleaned up by periodic background job
	// (see CleanupOrphanedUserArticles)
	key := datastore.NameKey("UserFeed", fmt.Sprintf("%d_%d", userID, feedID), nil)
	err := db.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	return nil
}

func (db *DatastoreDB) GetUserArticles(userID int) ([]Article, error) {
	result, err := db.GetUserArticlesPaginated(userID, 50, "", false) // Default: first 50 articles
	if err != nil {
		return nil, err
	}
	return result.Articles, nil
}

func (db *DatastoreDB) GetUserArticlesPaginated(userID int, limit int, cursor string, unreadOnly bool) (*ArticlePaginationResult, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Get user's subscribed feeds
	feeds, err := db.GetUserFeeds(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user feeds: %w", err)
	}

	if len(feeds) == 0 {
		return &ArticlePaginationResult{
			Articles:   []Article{},
			NextCursor: "",
		}, nil
	}

	// Create a map of feed IDs to feed titles for efficient lookup
	feedTitleMap := make(map[int]string)
	feedIDs := make([]int64, len(feeds))
	for i, feed := range feeds {
		feedIDs[i] = int64(feed.ID)
		feedTitleMap[feed.ID] = feed.Title
	}

	// For Datastore, we need a different approach since we're querying across multiple feeds
	// We'll query each feed and merge results, then apply cursor-based pagination in memory
	// This is still more efficient than the old approach because we fetch less data per feed
	var allArticles []Article

	// Query articles from all feeds in parallel batches
	batchSize := 5 // Process 5 feeds at a time
	for i := 0; i < len(feedIDs); i += batchSize {
		end := i + batchSize
		if end > len(feedIDs) {
			end = len(feedIDs)
		}

		// Process this batch of feeds in parallel
		batch := feedIDs[i:end]
		batchArticles := make(chan []Article, len(batch))

		for _, feedID := range batch {
			go func(fid int64) {
				// Get recent articles from this feed
				// We fetch more than limit since we need to merge and sort across feeds
				// But cap it at maxArticlesPerFeed to prevent unbounded memory usage
				articlesPerFeed := limit * 2
				if articlesPerFeed > maxArticlesPerFeed {
					articlesPerFeed = maxArticlesPerFeed
				}

				query := datastore.NewQuery("Article").
					FilterField("feed_id", "=", fid).
					Order("-published_at").
					Limit(articlesPerFeed)

				var feedArticles []ArticleEntity
				keys, err := db.client.GetAll(ctx, query, &feedArticles)
				if err != nil {
					batchArticles <- []Article{} // Empty result on error
					return
				}

				// Convert to Article structs
				articles := make([]Article, len(feedArticles))
				for j, entity := range feedArticles {
					entity.ID = keys[j].ID
					feedID := int(entity.FeedID)
					articles[j] = Article{
						ID:          int(entity.ID),
						FeedID:      feedID,
						FeedTitle:   feedTitleMap[feedID],
						Title:       entity.Title,
						URL:         entity.URL,
						Content:     entity.Content,
						Description: entity.Description,
						Author:      entity.Author,
						PublishedAt: entity.PublishedAt,
						CreatedAt:   entity.CreatedAt,
						IsRead:      false, // Default, will be updated in bulk below
						IsStarred:   false, // Default, will be updated in bulk below
					}
				}
				batchArticles <- articles
			}(feedID)
		}

		// Collect results from this batch
		for j := 0; j < len(batch); j++ {
			articles := <-batchArticles
			allArticles = append(allArticles, articles...)
		}
	}

	// Sort all articles by published_at desc, then by id desc for deterministic ordering
	sort.Slice(allArticles, func(i, j int) bool {
		if allArticles[i].PublishedAt.Equal(allArticles[j].PublishedAt) {
			return allArticles[i].ID > allArticles[j].ID
		}
		return allArticles[i].PublishedAt.After(allArticles[j].PublishedAt)
	})

	// Get user status for articles using efficient batch key lookup
	// Only fetch UserArticle entities for the articles we have, not all user's articles
	if len(allArticles) > 0 {
		// Build keys for only the articles we're working with
		statusMap := make(map[int]UserArticleEntity)
		chunkSize := 1000 // Datastore GetMulti limit

		for i := 0; i < len(allArticles); i += chunkSize {
			end := i + chunkSize
			if end > len(allArticles) {
				end = len(allArticles)
			}

			// Build keys for this chunk
			chunk := allArticles[i:end]
			userArticleKeys := make([]*datastore.Key, len(chunk))
			for j, article := range chunk {
				userArticleKeys[j] = datastore.NameKey("UserArticle",
					fmt.Sprintf("%d_%d", userID, article.ID), nil)
			}

			// Batch fetch UserArticle entities for this chunk
			userArticles := make([]UserArticleEntity, len(userArticleKeys))
			err := db.client.GetMulti(ctx, userArticleKeys, userArticles)

			// Process results - handle missing entities gracefully
			if multiErr, ok := err.(datastore.MultiError); ok {
				// Some UserArticle entities may not exist (article never read/starred)
				for j, singleErr := range multiErr {
					if singleErr == nil {
						// Entity exists, add to map
						statusMap[int(userArticles[j].ArticleID)] = userArticles[j]
					}
					// If singleErr != nil, entity doesn't exist - skip (defaults to unread/unstarred)
				}
			} else if err == nil {
				// All entities exist
				for _, ua := range userArticles {
					statusMap[int(ua.ArticleID)] = ua
				}
			}
			// If err is a different error, skip this chunk (defaults to unread/unstarred)
		}

		// Update articles with user status from map
		for i := range allArticles {
			if userStatus, exists := statusMap[allArticles[i].ID]; exists {
				allArticles[i].IsRead = userStatus.IsRead
				allArticles[i].IsStarred = userStatus.IsStarred
			}
		}
	}

	// Filter for unread articles if requested
	if unreadOnly {
		filteredArticles := make([]Article, 0, len(allArticles))
		for _, article := range allArticles {
			if !article.IsRead {
				filteredArticles = append(filteredArticles, article)
			}
		}
		allArticles = filteredArticles
	}

	// Apply cursor-based pagination
	startIdx := 0
	if cursor != "" {
		// Decode cursor to find start position
		cursorData, err := decodeSQLiteCursor(cursor) // Reuse same cursor format
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}

		// Find the position of the cursor in our sorted list
		for i, article := range allArticles {
			if article.PublishedAt.Before(cursorData.PublishedAt) ||
				(article.PublishedAt.Equal(cursorData.PublishedAt) && article.ID < cursorData.ID) {
				startIdx = i
				break
			}
		}
	}

	// Calculate end index
	endIdx := startIdx + limit
	if endIdx > len(allArticles) {
		endIdx = len(allArticles)
	}

	// Check if we have a full page (more results exist beyond this page)
	var nextCursor string
	if endIdx < len(allArticles) {
		// More results exist, create cursor from the last article we're returning
		if endIdx > startIdx {
			lastArticle := allArticles[endIdx-1]
			nextCursor = encodeSQLiteCursor(lastArticle.ID, lastArticle.PublishedAt)
		}
	}

	// Get the page of articles
	var paginatedArticles []Article
	if startIdx < len(allArticles) {
		paginatedArticles = allArticles[startIdx:endIdx]
	} else {
		paginatedArticles = []Article{}
	}

	return &ArticlePaginationResult{
		Articles:   paginatedArticles,
		NextCursor: nextCursor,
	}, nil
}

func (db *DatastoreDB) GetUserFeedArticles(userID, feedID int) ([]Article, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// First verify user is subscribed to this feed using keys-only query (1/3 cost)
	subscriptionQuery := datastore.NewQuery("UserFeed").
		FilterField("user_id", "=", int64(userID)).
		FilterField("feed_id", "=", int64(feedID)).
		KeysOnly().
		Limit(1)

	keys, err := db.client.GetAll(ctx, subscriptionQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check user subscription: %w", err)
	}

	if len(keys) == 0 {
		// User is not subscribed to this feed
		return []Article{}, nil
	}

	// Get the feed to retrieve its title
	feed, err := db.GetFeedByID(feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	// Get articles for the feed
	query := datastore.NewQuery("Article").FilterField("feed_id", "=", int64(feedID)).Order("-published_at")
	var articleEntities []ArticleEntity
	articleKeys, err := db.client.GetAll(ctx, query, &articleEntities)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles: %w", err)
	}

	if len(articleEntities) == 0 {
		return []Article{}, nil
	}

	// Batch lookup user article statuses for better performance
	// Process in chunks to handle feeds with >1000 articles (Datastore GetMulti limit)
	statusMap := make(map[int64]UserArticleEntity)
	chunkSize := 1000

	for i := 0; i < len(articleEntities); i += chunkSize {
		end := i + chunkSize
		if end > len(articleEntities) {
			end = len(articleEntities)
		}

		// Create keys for this chunk
		chunkKeys := articleKeys[i:end]
		userArticleKeys := make([]*datastore.Key, len(chunkKeys))
		for j, key := range chunkKeys {
			articleID := key.ID
			userArticleKeys[j] = datastore.NameKey("UserArticle", fmt.Sprintf("%d_%d", userID, articleID), nil)
		}

		// Use GetMulti for efficient batch read of this chunk
		userArticles := make([]UserArticleEntity, len(userArticleKeys))
		err = db.client.GetMulti(ctx, userArticleKeys, userArticles)

		// Process results from this chunk
		if multiErr, ok := err.(datastore.MultiError); ok {
			// Handle partial results - some UserArticle entities may not exist
			for j, singleErr := range multiErr {
				if singleErr == nil {
					articleID := chunkKeys[j].ID
					statusMap[articleID] = userArticles[j]
				}
				// Ignore ErrNoSuchEntity and other errors - they mean unread/unstarred
			}
		} else if err == nil {
			// All UserArticle entities exist in this chunk
			for j, userArticle := range userArticles {
				articleID := chunkKeys[j].ID
				statusMap[articleID] = userArticle
			}
		} else {
			// Complete failure for this chunk - this is a real error, not just missing entities
			return nil, fmt.Errorf("failed to get user article statuses: %w", err)
		}
	}

	// Build articles with user status
	articles := make([]Article, len(articleEntities))
	for i, entity := range articleEntities {
		entity.ID = articleKeys[i].ID
		articleID := articleKeys[i].ID

		// Get user status from map (defaults to false if not found)
		isRead := false
		isStarred := false
		if userStatus, exists := statusMap[articleID]; exists {
			isRead = userStatus.IsRead
			isStarred = userStatus.IsStarred
		}

		articles[i] = Article{
			ID:          int(entity.ID),
			FeedID:      int(entity.FeedID),
			FeedTitle:   feed.Title,
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
	ctx, cancel := newDatastoreContext()
	defer cancel()

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
	ctx, cancel := newDatastoreContext()
	defer cancel()

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

	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Batch process articles in chunks to avoid datastore limits
	chunkSize := 100
	for i := 0; i < len(articles); i += chunkSize {
		end := i + chunkSize
		if end > len(articles) {
			end = len(articles)
		}

		chunk := articles[i:end]
		entities := make([]*UserArticleEntity, len(chunk))
		keys := make([]*datastore.Key, len(chunk))

		for j, article := range chunk {
			entities[j] = &UserArticleEntity{
				UserID:    int64(userID),
				ArticleID: int64(article.ID),
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
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Get all user feeds with strong consistency retry for recent changes
	userFeeds, err := db.getUserFeedsWithRetry(ctx, userID, 3, 500*time.Millisecond)
	if err != nil {
		return nil, err
	}

	if len(userFeeds) == 0 {
		return make(map[int]int), nil
	}

	unreadCounts := make(map[int]int)

	// Process feeds in parallel for better performance
	type feedResult struct {
		feedID int
		count  int
		err    error
	}

	results := make(chan feedResult, len(userFeeds))

	// Start goroutines for each feed
	for _, feed := range userFeeds {
		go func(feedID int) {
			count, err := db.getFeedUnreadCountForUser(ctx, userID, feedID)
			results <- feedResult{feedID: feedID, count: count, err: err}
		}(feed.ID)
	}

	// Collect results
	for i := 0; i < len(userFeeds); i++ {
		result := <-results
		if result.err != nil {
			return nil, result.err
		}
		unreadCounts[result.feedID] = result.count
	}

	return unreadCounts, nil
}

// GetUserFeedCounts returns both unread and total counts for all user's feeds
func (db *DatastoreDB) GetUserFeedCounts(userID int) (map[int]FeedCounts, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Get all user feeds with strong consistency retry for recent changes
	userFeeds, err := db.getUserFeedsWithRetry(ctx, userID, 3, 500*time.Millisecond)
	if err != nil {
		return nil, err
	}

	if len(userFeeds) == 0 {
		return make(map[int]FeedCounts), nil
	}

	feedCounts := make(map[int]FeedCounts)

	// Process feeds in parallel for better performance
	type feedCountResult struct {
		feedID      int
		unreadCount int
		totalCount  int
		err         error
	}

	results := make(chan feedCountResult, len(userFeeds))

	// Start goroutines for each feed
	for _, feed := range userFeeds {
		go func(feedID int) {
			unreadCount, err := db.getFeedUnreadCountForUser(ctx, userID, feedID)
			if err != nil {
				results <- feedCountResult{feedID: feedID, err: err}
				return
			}

			totalCount, err := db.getFeedTotalCountForUser(ctx, userID, feedID)
			if err != nil {
				results <- feedCountResult{feedID: feedID, err: err}
				return
			}

			results <- feedCountResult{
				feedID:      feedID,
				unreadCount: unreadCount,
				totalCount:  totalCount,
				err:         nil,
			}
		}(feed.ID)
	}

	// Collect results
	for i := 0; i < len(userFeeds); i++ {
		result := <-results
		if result.err != nil {
			return nil, result.err
		}
		feedCounts[result.feedID] = FeedCounts{
			Unread: result.unreadCount,
			Total:  result.totalCount,
		}
	}

	return feedCounts, nil
}

// GetAccountStats retrieves user account statistics using parallel queries
// Returns total articles, total unread, and active feeds count
func (db *DatastoreDB) GetAccountStats(userID int) (map[string]interface{}, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Get all user feeds with strong consistency retry
	userFeeds, err := db.getUserFeedsWithRetry(ctx, userID, 3, 500*time.Millisecond)
	if err != nil {
		return nil, err
	}

	if len(userFeeds) == 0 {
		return map[string]interface{}{
			"total_articles": 0,
			"total_unread":   0,
			"active_feeds":   0,
		}, nil
	}

	// Use goroutines to fetch stats in parallel for better performance
	type feedStats struct {
		totalArticles int
		unreadCount   int
		err           error
	}

	results := make(chan feedStats, len(userFeeds))

	// Process each feed in parallel
	for _, feed := range userFeeds {
		go func(feedID int) {
			// Get article count for this feed
			articleQuery := datastore.NewQuery("Article").
				FilterField("feed_id", "=", int64(feedID)).
				KeysOnly()
			articleKeys, err := db.client.GetAll(ctx, articleQuery, nil)
			if err != nil {
				results <- feedStats{err: err}
				return
			}

			totalArticles := len(articleKeys)

			// Get unread count for this feed
			unreadCount, err := db.getFeedUnreadCountForUser(ctx, userID, feedID)
			if err != nil {
				results <- feedStats{err: err}
				return
			}

			results <- feedStats{
				totalArticles: totalArticles,
				unreadCount:   unreadCount,
			}
		}(feed.ID)
	}

	// Aggregate results
	totalArticles := 0
	totalUnread := 0
	activeFeeds := 0

	for i := 0; i < len(userFeeds); i++ {
		result := <-results
		if result.err != nil {
			return nil, result.err
		}
		totalArticles += result.totalArticles
		totalUnread += result.unreadCount
		if result.unreadCount > 0 {
			activeFeeds++
		}
	}

	stats := map[string]interface{}{
		"total_articles": totalArticles,
		"total_unread":   totalUnread,
		"active_feeds":   activeFeeds,
	}

	return stats, nil
}

// Helper function to efficiently count unread articles for a specific feed
func (db *DatastoreDB) getFeedUnreadCountForUser(ctx context.Context, userID, feedID int) (int, error) {
	// Get all articles for this feed with eventual consistency retry
	var articleKeys []*datastore.Key
	var err error

	// Retry logic to handle eventual consistency issues with newly added feeds
	for attempt := 0; attempt < 3; attempt++ {
		articleQuery := datastore.NewQuery("Article").
			FilterField("feed_id", "=", int64(feedID)).
			KeysOnly()

		articleKeys, err = db.client.GetAll(ctx, articleQuery, nil)
		if err != nil {
			if attempt < 2 {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			return 0, err
		}

		// If we got articles or this isn't the first attempt, use the result
		if len(articleKeys) > 0 || attempt > 0 {
			break
		}

		// If no articles found on first attempt, wait and retry (might be consistency lag)
		time.Sleep(500 * time.Millisecond)
	}

	if len(articleKeys) == 0 {
		return 0, nil
	}

	// Batch check which articles are read by this user
	// Process in chunks to handle feeds with >1000 articles (Datastore GetMulti limit)
	unreadCount := 0
	chunkSize := 1000

	for i := 0; i < len(articleKeys); i += chunkSize {
		end := i + chunkSize
		if end > len(articleKeys) {
			end = len(articleKeys)
		}

		// Create keys for this chunk
		chunkArticleKeys := articleKeys[i:end]
		userArticleKeys := make([]*datastore.Key, len(chunkArticleKeys))
		for j, articleKey := range chunkArticleKeys {
			articleID := articleKey.ID
			userArticleKeys[j] = datastore.NameKey("UserArticle", fmt.Sprintf("%d_%d", userID, articleID), nil)
		}

		// Use GetMulti for efficient batch read of this chunk
		userArticles := make([]UserArticleEntity, len(userArticleKeys))
		err = db.client.GetMulti(ctx, userArticleKeys, userArticles)

		// Process results from this chunk
		if multiErr, ok := err.(datastore.MultiError); ok {
			// Handle partial results - some UserArticle entities may not exist
			for j, singleErr := range multiErr {
				switch singleErr {
				case datastore.ErrNoSuchEntity:
					// No UserArticle record means unread
					unreadCount++
				case nil:
					// UserArticle exists, check if it's read
					if !userArticles[j].IsRead {
						unreadCount++
					}
				}
				// Other errors are ignored (treated as read to be conservative)
			}
		} else if err == nil {
			// All UserArticle entities exist in this chunk, count unread ones
			for _, userArticle := range userArticles {
				if !userArticle.IsRead {
					unreadCount++
				}
			}
		} else {
			// Complete failure for this chunk - treat all as unread to be safe
			unreadCount += len(chunkArticleKeys)
		}
	}

	return unreadCount, nil
}

// getFeedTotalCountForUser returns the total number of articles for this user in a feed
func (db *DatastoreDB) getFeedTotalCountForUser(ctx context.Context, userID, feedID int) (int, error) {
	// Query UserArticle entities to get count of articles this user has in this feed
	var userArticleKeys []*datastore.Key
	var err error

	// Retry logic to handle eventual consistency issues with newly added feeds
	for attempt := 0; attempt < 3; attempt++ {
		userArticleQuery := datastore.NewQuery("UserArticle").
			FilterField("user_id", "=", int64(userID)).
			FilterField("feed_id", "=", int64(feedID)).
			KeysOnly()

		userArticleKeys, err = db.client.GetAll(ctx, userArticleQuery, nil)
		if err != nil {
			if attempt < 2 {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			return 0, err
		}

		// If we got articles or this isn't the first attempt, use the result
		if len(userArticleKeys) > 0 || attempt > 0 {
			break
		}

		// If no articles found on first attempt, wait and retry (might be consistency lag)
		time.Sleep(500 * time.Millisecond)
	}

	return len(userArticleKeys), nil
}

// CleanupOrphanedUserArticles removes UserArticle entities that reference articles from feeds
// the user is no longer subscribed to. Only cleans up articles older than the specified number of days.
// Returns the number of records deleted.
func (db *DatastoreDB) CleanupOrphanedUserArticles(olderThanDays int) (int, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()
	deletedCount := 0
	cutoffDate := time.Now().AddDate(0, 0, -olderThanDays)

	// Query UserArticle entities in batches
	batchSize := 500
	var cursor *datastore.Cursor

	for {
		query := datastore.NewQuery("UserArticle").Limit(batchSize)
		if cursor != nil {
			query = query.Start(*cursor)
		}

		var userArticles []UserArticleEntity
		keys, err := db.client.GetAll(ctx, query.KeysOnly(), &userArticles)
		if err != nil {
			return deletedCount, fmt.Errorf("failed to query user articles: %w", err)
		}

		if len(keys) == 0 {
			break
		}

		// Process this batch
		var keysToDelete []*datastore.Key

		for _, key := range keys {
			// Parse user_id and article_id from the key name (format: "userID_articleID")
			keyName := key.Name
			var userID, articleID int64
			_, err := fmt.Sscanf(keyName, "%d_%d", &userID, &articleID)
			if err != nil {
				continue // Skip malformed keys
			}

			// Get the article to find its feed_id and created_at
			articleKey := datastore.IDKey("Article", articleID, nil)
			var article ArticleEntity
			err = db.client.Get(ctx, articleKey, &article)
			if err != nil {
				// Article doesn't exist, definitely orphaned
				keysToDelete = append(keysToDelete, key)
				continue
			}

			// Check if article is old enough (skip check if olderThanDays is 0)
			if olderThanDays > 0 && article.CreatedAt.After(cutoffDate) {
				continue // Too recent, skip
			}

			// Check if user is still subscribed to this feed
			userFeedKey := datastore.NameKey("UserFeed", fmt.Sprintf("%d_%d", userID, article.FeedID), nil)
			var userFeed UserFeedEntity
			err = db.client.Get(ctx, userFeedKey, &userFeed)
			if err == datastore.ErrNoSuchEntity {
				// User is not subscribed, this is orphaned
				keysToDelete = append(keysToDelete, key)
			}
			// If no error, user is still subscribed, keep it
		}

		// Delete orphaned UserArticle entities in this batch
		if len(keysToDelete) > 0 {
			err := db.client.DeleteMulti(ctx, keysToDelete)
			if err != nil {
				// Log but continue with other batches
				fmt.Printf("Warning: Failed to delete some orphaned user articles: %v\n", err)
			} else {
				deletedCount += len(keysToDelete)
			}
		}

		// Check if there are more results
		if len(keys) < batchSize {
			break
		}

		// Get cursor for next page (note: we need to re-run the query to get the cursor)
		iter := db.client.Run(ctx, query)
		for i := 0; i < len(keys); i++ {
			_, err := iter.Next(nil)
			if err != nil {
				break
			}
		}
		nextCursor, err := iter.Cursor()
		if err != nil {
			break
		}
		cursor = &nextCursor
	}

	return deletedCount, nil
}

// getUserFeedsWithRetry gets user feeds with retry logic to handle eventual consistency
func (db *DatastoreDB) getUserFeedsWithRetry(ctx context.Context, userID int, maxRetries int, delay time.Duration) ([]Feed, error) {
	var lastFeeds []Feed
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		feeds, err := db.GetUserFeeds(userID)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				time.Sleep(delay)
				continue
			}
			return nil, lastErr
		}

		// If we got feeds on the first attempt or same number as previous, return
		if attempt == 0 || len(feeds) >= len(lastFeeds) {
			return feeds, nil
		}

		// If we got fewer feeds than before, retry (might be consistency issue)
		lastFeeds = feeds
		if attempt < maxRetries {
			time.Sleep(delay)
		}
	}

	// Return the last result we got
	return lastFeeds, lastErr
}

func (db *DatastoreDB) GetAllUserFeeds() ([]Feed, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

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
func (db *DatastoreDB) UpdateUserSubscription(userID int, status, subscriptionID string, lastPaymentDate, nextBillingDate time.Time) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	key := datastore.IDKey("User", int64(userID), nil)
	var entity UserEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	entity.SubscriptionStatus = status
	entity.SubscriptionID = subscriptionID
	entity.LastPaymentDate = lastPaymentDate
	entity.NextBillingDate = nextBillingDate

	_, err := db.client.Put(ctx, key, &entity)
	if err != nil {
		return fmt.Errorf("failed to update user subscription: %w", err)
	}

	return nil
}

func (db *DatastoreDB) IsUserSubscriptionActive(userID int) (bool, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	key := datastore.IDKey("User", int64(userID), nil)
	var entity UserEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	// User is active if:
	// 1. They're an admin user (unlimited access), OR
	// 2. They have an active paid subscription, OR
	// 3. They're on trial and trial hasn't expired, OR
	// 4. They have free months remaining
	if entity.IsAdmin {
		return true, nil
	}

	if entity.SubscriptionStatus == "active" {
		return true, nil
	}

	if entity.SubscriptionStatus == "trial" && time.Now().Before(entity.TrialEndsAt) {
		return true, nil
	}

	if entity.FreeMonthsRemaining > 0 {
		return true, nil
	}

	return false, nil
}

func (db *DatastoreDB) GetUserFeedCount(userID int) (int, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

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
	ctx, cancel := newDatastoreContext()
	defer cancel()

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
	ctx, cancel := newDatastoreContext()
	defer cancel()

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
	ctx, cancel := newDatastoreContext()
	defer cancel()

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

	maxArticles := user.MaxArticlesOnFeedAdd
	if maxArticles == 0 {
		maxArticles = 100 // Default for existing users
	}

	return &User{
		ID:                   int(user.ID),
		GoogleID:             user.GoogleID,
		Email:                user.Email,
		Name:                 user.Name,
		Avatar:               user.Avatar,
		CreatedAt:            user.CreatedAt,
		SubscriptionStatus:   user.SubscriptionStatus,
		SubscriptionID:       user.SubscriptionID,
		TrialEndsAt:          user.TrialEndsAt,
		LastPaymentDate:      user.LastPaymentDate,
		NextBillingDate:      user.NextBillingDate,
		IsAdmin:              user.IsAdmin,
		FreeMonthsRemaining:  user.FreeMonthsRemaining,
		MaxArticlesOnFeedAdd: maxArticles,
	}, nil
}

// Session methods for Datastore
func (db *DatastoreDB) CreateSession(session *Session) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	entity := &SessionEntity{
		ID:        session.ID,
		UserID:    int64(session.UserID),
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}

	key := datastore.NameKey("Session", session.ID, nil)
	_, err := db.client.Put(ctx, key, entity)
	return err
}

func (db *DatastoreDB) GetSession(sessionID string) (*Session, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	key := datastore.NameKey("Session", sessionID, nil)
	var entity SessionEntity

	err := db.client.Get(ctx, key, &entity)
	if err == datastore.ErrNoSuchEntity {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	entity.ID = sessionID

	return &Session{
		ID:        entity.ID,
		UserID:    int(entity.UserID),
		CreatedAt: entity.CreatedAt,
		ExpiresAt: entity.ExpiresAt,
	}, nil
}

func (db *DatastoreDB) UpdateSessionExpiry(sessionID string, newExpiry time.Time) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	key := datastore.NameKey("Session", sessionID, nil)

	// Get existing session first
	var session Session
	if err := db.client.Get(ctx, key, &session); err != nil {
		return err
	}

	// Update expiry
	session.ExpiresAt = newExpiry

	// Save back to Datastore
	_, err := db.client.Put(ctx, key, &session)
	return err
}

func (db *DatastoreDB) DeleteSession(sessionID string) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	key := datastore.NameKey("Session", sessionID, nil)
	return db.client.Delete(ctx, key)
}

func (db *DatastoreDB) DeleteExpiredSessions() error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Query for expired sessions
	query := datastore.NewQuery("Session").FilterField("expires_at", "<", time.Now()).KeysOnly()

	keys, err := db.client.GetAll(ctx, query, nil)
	if err != nil {
		return err
	}

	// Delete expired sessions in batches
	if len(keys) > 0 {
		return db.client.DeleteMulti(ctx, keys)
	}

	return nil
}

// Audit log methods for Datastore
func (db *DatastoreDB) CreateAuditLog(log *AuditLog) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	entity := &AuditLogEntity{
		Timestamp:        log.Timestamp,
		AdminUserID:      int64(log.AdminUserID),
		AdminEmail:       log.AdminEmail,
		OperationType:    log.OperationType,
		TargetUserID:     int64(log.TargetUserID),
		TargetUserEmail:  log.TargetUserEmail,
		OperationDetails: log.OperationDetails,
		IPAddress:        log.IPAddress,
		Result:           log.Result,
		ErrorMessage:     log.ErrorMessage,
	}

	key := datastore.IncompleteKey("AuditLog", nil)
	key, err := db.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to save audit log: %w", err)
	}

	log.ID = int(key.ID)
	return nil
}

func (db *DatastoreDB) GetAuditLogs(limit, offset int, filters map[string]interface{}) ([]AuditLog, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	query := datastore.NewQuery("AuditLog").Order("-timestamp")

	// Apply filters
	if userID, ok := filters["admin_user_id"]; ok {
		query = query.FilterField("admin_user_id", "=", int64(userID.(int)))
	}
	if targetUserID, ok := filters["target_user_id"]; ok {
		query = query.FilterField("target_user_id", "=", int64(targetUserID.(int)))
	}
	if opType, ok := filters["operation_type"]; ok {
		query = query.FilterField("operation_type", "=", opType.(string))
	}

	// Apply limit and offset
	query = query.Limit(limit).Offset(offset)

	var entities []AuditLogEntity
	keys, err := db.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}

	logs := make([]AuditLog, len(entities))
	for i, entity := range entities {
		entity.ID = keys[i].ID
		logs[i] = AuditLog{
			ID:               int(entity.ID),
			Timestamp:        entity.Timestamp,
			AdminUserID:      int(entity.AdminUserID),
			AdminEmail:       entity.AdminEmail,
			OperationType:    entity.OperationType,
			TargetUserID:     int(entity.TargetUserID),
			TargetUserEmail:  entity.TargetUserEmail,
			OperationDetails: entity.OperationDetails,
			IPAddress:        entity.IPAddress,
			Result:           entity.Result,
			ErrorMessage:     entity.ErrorMessage,
		}
	}

	return logs, nil
}
