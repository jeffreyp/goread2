package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
)

const (
	// datastoreTimeout is the default timeout for all Datastore operations
	// This prevents operations from hanging indefinitely in production
	datastoreTimeout = 30 * time.Second

	// maxArticlesPerFeed limits memory usage when paginating across multiple feeds
	// This prevents OOM errors when users subscribe to many active feeds
	// With this limit, even 1000 feeds would only load ~200KB of articles
	maxArticlesPerFeed = 200

	// unreadCountWindowDays is the lookback window for unread count queries.
	// Articles older than this are excluded from badge counts to cap read costs.
	unreadCountWindowDays = 90

	// slowQueryThreshold is the minimum duration before a datastore operation is logged as slow.
	slowQueryThreshold = 200 * time.Millisecond
)

// logSlowQuery logs a warning when a datastore operation exceeds slowQueryThreshold.
// Usage: defer logSlowQuery("OperationName", time.Now())
func logSlowQuery(op string, start time.Time) {
	if d := time.Since(start); d > slowQueryThreshold {
		log.Printf("[datastore slow] %s took %v", op, d)
	}
}

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
	ETag                  string    `datastore:"etag"`
	LastModified          string    `datastore:"last_modified"`
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
	// Use context.Background() here (not a timeout context) because gRPC dials
	// lazily and stores the dial context internally. A cancelled dial context
	// causes OAuth2 token refresh failures ("context canceled") on first use
	// after the access token expires.
	client, err := datastore.NewClient(context.Background(), projectID)
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
		ETag:                  feed.ETag,
		LastModified:          feed.LastModified,
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

func (db *DatastoreDB) UpdateFeedCacheHeaders(feedID int, etag, lastModified string) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	key := datastore.IDKey("Feed", int64(feedID), nil)
	var entity FeedEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	entity.ETag = etag
	entity.LastModified = lastModified

	if _, err := db.client.Put(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to update feed cache headers: %w", err)
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
			ETag:                  entity.ETag,
			LastModified:          entity.LastModified,
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
		ETag:                  entity.ETag,
		LastModified:          entity.LastModified,
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
		ETag:                  entity.ETag,
		LastModified:          entity.LastModified,
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

// articleURLProjection is used for projection queries that retrieve only the url field.
type articleURLProjection struct {
	URL string `datastore:"url"`
}

// articleURLDeduplicationWindow is how far back we check for duplicate article URLs.
// Feed articles never reuse URLs older than this, so there's no need to scan the
// full article history. This dramatically reduces Firestore read costs.
const articleURLDeduplicationWindow = 90 * 24 * time.Hour

// FilterExistingArticleURLs returns a map of which URLs in the given slice already exist
// for the specified feed. Uses a single projection query per feed instead of one query
// per article, reducing Firestore reads from N to 1 per feed refresh cycle.
// Only checks articles created within the last 90 days to bound read costs.
func (db *DatastoreDB) FilterExistingArticleURLs(feedID int, urls []string) (map[string]bool, error) {
	defer logSlowQuery("FilterExistingArticleURLs", time.Now())
	if len(urls) == 0 {
		return map[string]bool{}, nil
	}
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Build a set of incoming URLs for fast membership testing.
	urlSet := make(map[string]struct{}, len(urls))
	for _, u := range urls {
		urlSet[u] = struct{}{}
	}

	// One projection query returns recent stored URLs for this feed.
	// The 90-day cutoff prevents scanning the full article history on every
	// cron run — the primary driver of Firestore read costs.
	cutoff := time.Now().Add(-articleURLDeduplicationWindow)
	query := datastore.NewQuery("Article").
		FilterField("feed_id", "=", int64(feedID)).
		FilterField("created_at", ">=", cutoff).
		Project("url")
	var projections []articleURLProjection
	if _, err := db.client.GetAll(ctx, query, &projections); err != nil {
		return nil, fmt.Errorf("failed to check existing article URLs: %w", err)
	}

	existing := make(map[string]bool)
	for _, p := range projections {
		if _, ok := urlSet[p.URL]; ok {
			existing[p.URL] = true
		}
	}
	return existing, nil
}

// UpdateFeedAfterRefresh writes all post-refresh tracking fields in a single Get+Put,
// replacing the three separate UpdateFeedTracking / UpdateFeedLastFetch /
// UpdateFeedCacheHeaders calls that previously ran on every successful feed refresh.
func (db *DatastoreDB) UpdateFeedAfterRefresh(feedID int, lastChecked, lastHadNewContent time.Time, averageUpdateInterval int, lastFetch time.Time, etag, lastModified string) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	key := datastore.IDKey("Feed", int64(feedID), nil)
	var entity FeedEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	entity.LastChecked = lastChecked
	if !lastHadNewContent.IsZero() {
		entity.LastHadNewContent = lastHadNewContent
	}
	entity.AverageUpdateInterval = averageUpdateInterval
	entity.LastFetch = lastFetch
	entity.ETag = etag
	entity.LastModified = lastModified

	if _, err := db.client.Put(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to update feed after refresh: %w", err)
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
	defer logSlowQuery("GetUserFeeds", time.Now())
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
						ETag:                  entity.ETag,
						LastModified:          entity.LastModified,
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
			ETag:                  entity.ETag,
			LastModified:          entity.LastModified,
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

// articleRef is a lightweight projection result used for global sorting before fetching full entities.
type articleRef struct {
	key         *datastore.Key
	feedID      int64
	publishedAt time.Time
}

// articlePublishedAtProjection holds only the published_at field for projection queries.
// Using a projection query on indexed fields is a Datastore "small operation" (~1/6 the cost
// of a full entity read), so we use this to sort across feeds before fetching full articles.
type articlePublishedAtProjection struct {
	PublishedAt time.Time `datastore:"published_at"`
}

func (db *DatastoreDB) GetUserArticlesPaginated(userID int, limit int, cursor string, unreadOnly bool) (*ArticlePaginationResult, error) {
	return db.getUserArticlesPaginated(userID, 0, limit, cursor, unreadOnly)
}

// GetUserFeedArticlesPaginated fetches a single feed's articles with the same cursor-based
// pagination as GetUserArticlesPaginated. Returns an empty result if the user isn't
// subscribed to feedID.
func (db *DatastoreDB) GetUserFeedArticlesPaginated(userID, feedID int, limit int, cursor string, unreadOnly bool) (*ArticlePaginationResult, error) {
	return db.getUserArticlesPaginated(userID, feedID, limit, cursor, unreadOnly)
}

// getUserArticlesPaginated backs both GetUserArticlesPaginated and GetUserFeedArticlesPaginated.
// feedID of 0 means "all of the user's feeds"; a nonzero feedID restricts to that one feed.
func (db *DatastoreDB) getUserArticlesPaginated(userID, feedID int, limit int, cursor string, unreadOnly bool) (*ArticlePaginationResult, error) {
	defer logSlowQuery("GetUserArticlesPaginated", time.Now())
	ctx, cancel := newDatastoreContext()
	defer cancel()

	feeds, err := db.GetUserFeeds(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user feeds: %w", err)
	}
	if feedID != 0 {
		var only *Feed
		for i := range feeds {
			if feeds[i].ID == feedID {
				only = &feeds[i]
				break
			}
		}
		if only == nil {
			// User is not subscribed to this feed.
			return &ArticlePaginationResult{Articles: []Article{}, NextCursor: ""}, nil
		}
		feeds = []Feed{*only}
	}
	if len(feeds) == 0 {
		return &ArticlePaginationResult{Articles: []Article{}, NextCursor: ""}, nil
	}

	feedTitleMap := make(map[int]string)
	feedIDs := make([]int64, len(feeds))
	for i, feed := range feeds {
		feedIDs[i] = int64(feed.ID)
		feedTitleMap[feed.ID] = feed.Title
	}

	// How many refs to project per feed. Same ceiling as before to preserve pagination depth.
	articlesPerFeed := limit * 2
	if articlesPerFeed > maxArticlesPerFeed {
		articlesPerFeed = maxArticlesPerFeed
	}

	// Pass 1: projection queries — fetch only published_at (an indexed field) per feed.
	// These are Datastore "small operations" (~1/6 the cost of full entity reads), so we can
	// project the same number of refs as before without significantly increasing cost, while
	// deferring full entity reads until we know exactly which articles we need.
	allRefs := make([]articleRef, 0, len(feedIDs)*articlesPerFeed)

	batchSize := 5
	for i := 0; i < len(feedIDs); i += batchSize {
		end := i + batchSize
		if end > len(feedIDs) {
			end = len(feedIDs)
		}
		batch := feedIDs[i:end]
		results := make(chan []articleRef, len(batch))

		for _, fid := range batch {
			go func(fid int64) {
				query := datastore.NewQuery("Article").
					FilterField("feed_id", "=", fid).
					Order("-published_at").
					Limit(articlesPerFeed).
					Project("published_at")

				var projs []articlePublishedAtProjection
				keys, err := db.client.GetAll(ctx, query, &projs)
				if err != nil {
					results <- nil
					return
				}
				refs := make([]articleRef, len(projs))
				for j := range projs {
					refs[j] = articleRef{
						key:         keys[j],
						feedID:      fid,
						publishedAt: projs[j].PublishedAt,
					}
				}
				results <- refs
			}(fid)
		}

		for range batch {
			if refs := <-results; refs != nil {
				allRefs = append(allRefs, refs...)
			}
		}
	}

	// Sort refs globally by published_at desc, then by key ID desc for determinism.
	sort.Slice(allRefs, func(i, j int) bool {
		if allRefs[i].publishedAt.Equal(allRefs[j].publishedAt) {
			return allRefs[i].key.ID > allRefs[j].key.ID
		}
		return allRefs[i].publishedAt.After(allRefs[j].publishedAt)
	})

	// Apply cursor to find where the next page starts.
	startIdx := 0
	if cursor != "" {
		cursorData, err := decodeSQLiteCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		for i, ref := range allRefs {
			if ref.publishedAt.Before(cursorData.PublishedAt) ||
				(ref.publishedAt.Equal(cursorData.PublishedAt) && ref.key.ID < int64(cursorData.ID)) {
				startIdx = i
				break
			}
		}
	}
	remainingRefs := allRefs[startIdx:]

	// For unreadOnly we need extra candidates because some will be filtered out.
	// For all-articles we need exactly limit candidates.
	candidateCount := limit
	if unreadOnly {
		candidateCount = limit * 2
		if candidateCount > maxArticlesPerFeed {
			candidateCount = maxArticlesPerFeed
		}
	}
	if candidateCount > len(remainingRefs) {
		candidateCount = len(remainingRefs)
	}
	candidates := remainingRefs[:candidateCount]

	// Pass 2a: batch-fetch UserArticle status for candidates only (not all projected refs).
	statusMap := make(map[int64]UserArticleEntity)
	if len(candidates) > 0 {
		userArticleKeys := make([]*datastore.Key, len(candidates))
		for i, ref := range candidates {
			userArticleKeys[i] = datastore.NameKey("UserArticle",
				fmt.Sprintf("%d_%d", userID, ref.key.ID), nil)
		}
		userArticles := make([]UserArticleEntity, len(userArticleKeys))
		uaErr := db.client.GetMulti(ctx, userArticleKeys, userArticles)
		if multiErr, ok := uaErr.(datastore.MultiError); ok {
			for i, singleErr := range multiErr {
				if singleErr == nil {
					statusMap[userArticles[i].ArticleID] = userArticles[i]
				}
			}
		} else if uaErr == nil {
			for _, ua := range userArticles {
				statusMap[ua.ArticleID] = ua
			}
		}
	}

	// Determine the page refs, filtering for unread if requested.
	pageRefs := make([]articleRef, 0, limit)
	for _, ref := range candidates {
		if unreadOnly {
			ua, exists := statusMap[ref.key.ID]
			if exists && ua.IsRead {
				continue
			}
		}
		pageRefs = append(pageRefs, ref)
		if len(pageRefs) == limit {
			break
		}
	}

	// Set next cursor if there are more results beyond this page.
	var nextCursor string
	if len(pageRefs) == limit {
		// Check if any refs remain after this page (in candidates or in remainingRefs).
		if candidateCount > len(pageRefs) || len(remainingRefs) > candidateCount {
			last := pageRefs[len(pageRefs)-1]
			nextCursor = encodeSQLiteCursor(int(last.key.ID), last.publishedAt)
		}
	}

	if len(pageRefs) == 0 {
		return &ArticlePaginationResult{Articles: []Article{}, NextCursor: nextCursor}, nil
	}

	// Pass 2b: fetch full article entities for only the page we're returning.
	articleKeys := make([]*datastore.Key, len(pageRefs))
	for i, ref := range pageRefs {
		articleKeys[i] = ref.key
	}
	articleEntities := make([]ArticleEntity, len(articleKeys))
	fetchErr := db.client.GetMulti(ctx, articleKeys, articleEntities)
	multiErr, isME := fetchErr.(datastore.MultiError)

	articles := make([]Article, 0, len(pageRefs))
	for i, entity := range articleEntities {
		if isME && multiErr[i] != nil {
			continue
		}
		entity.ID = pageRefs[i].key.ID
		feedID := int(entity.FeedID)
		ua := statusMap[entity.ID]
		articles = append(articles, Article{
			ID:          int(entity.ID),
			FeedID:      feedID,
			FeedTitle:   feedTitleMap[feedID],
			Title:       entity.Title,
			URL:         entity.URL,
			Description: entity.Description,
			Author:      entity.Author,
			PublishedAt: entity.PublishedAt,
			CreatedAt:   entity.CreatedAt,
			IsRead:      ua.IsRead,
			IsStarred:   ua.IsStarred,
		})
	}
	if !isME && fetchErr != nil {
		return nil, fmt.Errorf("failed to fetch article entities: %w", fetchErr)
	}

	return &ArticlePaginationResult{
		Articles:   articles,
		NextCursor: nextCursor,
	}, nil
}

func (db *DatastoreDB) GetUserFeedArticles(userID, feedID int) ([]Article, error) {
	defer logSlowQuery("GetUserFeedArticles", time.Now())
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

func (db *DatastoreDB) GetArticleByID(userID, articleID int) (*Article, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	key := datastore.IDKey("Article", int64(articleID), nil)
	var entity ArticleEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}
	entity.ID = int64(articleID)

	// Verify user is subscribed to this feed
	subQuery := datastore.NewQuery("UserFeed").
		FilterField("user_id", "=", int64(userID)).
		FilterField("feed_id", "=", entity.FeedID).
		KeysOnly().
		Limit(1)
	subKeys, err := db.client.GetAll(ctx, subQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check user subscription: %w", err)
	}
	if len(subKeys) == 0 {
		return nil, nil
	}

	feed, err := db.GetFeedByID(int(entity.FeedID))
	if err != nil {
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	uaKey := datastore.NameKey("UserArticle", fmt.Sprintf("%d_%d", userID, articleID), nil)
	var ua UserArticleEntity
	isRead, isStarred := false, false
	if err := db.client.Get(ctx, uaKey, &ua); err == nil {
		isRead = ua.IsRead
		isStarred = ua.IsStarred
	}

	return &Article{
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
	}, nil
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
	defer logSlowQuery("BatchSetUserArticleStatus", time.Now())
	if len(articles) == 0 {
		return nil
	}

	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Wrap all chunk writes in a single transaction so a failure in any chunk
	// rolls back the entire operation instead of leaving a partial update committed.
	_, err := db.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		chunkSize := 500 // Cloud Datastore supports up to 500 entities per batch commit
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
				keyStr := fmt.Sprintf("%d_%d", userID, article.ID)
				keys[j] = datastore.NameKey("UserArticle", keyStr, nil)
			}

			if _, err := tx.PutMulti(keys, entities); err != nil {
				return fmt.Errorf("failed to write article status batch: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to batch set article statuses: %w", err)
	}
	return nil
}

func (db *DatastoreDB) MarkAllUserArticlesRead(userID int) (int, error) {
	defer logSlowQuery("MarkAllUserArticlesRead", time.Now())
	ctx, cancel := newDatastoreContext()
	defer cancel()

	feeds, err := db.GetUserFeeds(userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user feeds: %w", err)
	}
	if len(feeds) == 0 {
		return 0, nil
	}

	// Collect all article IDs via keys-only queries (avoids loading article bodies).
	var articleIDs []int64
	for _, feed := range feeds {
		q := datastore.NewQuery("Article").FilterField("feed_id", "=", int64(feed.ID)).KeysOnly()
		keys, err := db.client.GetAll(ctx, q, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to get article keys for feed %d: %w", feed.ID, err)
		}
		for _, k := range keys {
			articleIDs = append(articleIDs, k.ID)
		}
	}
	if len(articleIDs) == 0 {
		return 0, nil
	}

	_, err = db.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		chunkSize := 500
		for i := 0; i < len(articleIDs); i += chunkSize {
			end := i + chunkSize
			if end > len(articleIDs) {
				end = len(articleIDs)
			}
			chunk := articleIDs[i:end]
			entities := make([]*UserArticleEntity, len(chunk))
			keys := make([]*datastore.Key, len(chunk))
			for j, aid := range chunk {
				entities[j] = &UserArticleEntity{
					UserID:    int64(userID),
					ArticleID: aid,
					IsRead:    true,
					IsStarred: false,
				}
				keys[j] = datastore.NameKey("UserArticle", fmt.Sprintf("%d_%d", userID, aid), nil)
			}
			if _, err := tx.PutMulti(keys, entities); err != nil {
				return fmt.Errorf("failed to write read status batch: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to mark all articles read: %w", err)
	}
	return len(articleIDs), nil
}

func (db *DatastoreDB) GetUserUnreadCounts(userID int) (map[int]int, error) {
	defer logSlowQuery("GetUserUnreadCounts", time.Now())
	ctx, cancel := newDatastoreContext()
	defer cancel()

	userFeeds, err := db.getUserFeedsWithRetry(ctx, userID, 3, 500*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to get user feeds: %w", err)
	}

	if len(userFeeds) == 0 {
		return make(map[int]int), nil
	}

	cutoff := time.Now().UTC().Add(-unreadCountWindowDays * 24 * time.Hour)

	// Phase 1: fan out one keys-only Article query per feed in parallel.
	type feedArticleResult struct {
		feedID int
		keys   []*datastore.Key
		err    error
	}

	articleResults := make(chan feedArticleResult, len(userFeeds))
	for _, feed := range userFeeds {
		go func(feedID int) {
			q := datastore.NewQuery("Article").
				FilterField("feed_id", "=", int64(feedID)).
				FilterField("published_at", ">=", cutoff).
				KeysOnly()
			keys, err := db.client.GetAll(ctx, q, nil)
			articleResults <- feedArticleResult{feedID: feedID, keys: keys, err: err}
		}(feed.ID)
	}

	// Collect article keys, seeding each feed's count as "all recent = unread".
	unreadCounts := make(map[int]int)
	var allArticleKeys []*datastore.Key
	var articleFeedIDs []int

	for i := 0; i < len(userFeeds); i++ {
		r := <-articleResults
		if r.err != nil {
			return nil, r.err
		}
		unreadCounts[r.feedID] = len(r.keys)
		for _, k := range r.keys {
			allArticleKeys = append(allArticleKeys, k)
			articleFeedIDs = append(articleFeedIDs, r.feedID)
		}
	}

	if len(allArticleKeys) == 0 {
		return unreadCounts, nil
	}

	// Phase 2: one merged GetMulti for all UserArticle entities across every feed.
	// Articles with no UserArticle record (or IsRead=false) stay unread; IsRead=true subtracts one.
	userArticleKeys := make([]*datastore.Key, len(allArticleKeys))
	for i, ak := range allArticleKeys {
		userArticleKeys[i] = datastore.NameKey("UserArticle", fmt.Sprintf("%d_%d", userID, ak.ID), nil)
	}

	chunkSize := 1000
	for i := 0; i < len(userArticleKeys); i += chunkSize {
		end := i + chunkSize
		if end > len(userArticleKeys) {
			end = len(userArticleKeys)
		}
		chunk := userArticleKeys[i:end]
		userArticles := make([]UserArticleEntity, len(chunk))
		chunkErr := db.client.GetMulti(ctx, chunk, userArticles)

		if multiErr, ok := chunkErr.(datastore.MultiError); ok {
			for j, singleErr := range multiErr {
				if singleErr == nil && userArticles[j].IsRead {
					unreadCounts[articleFeedIDs[i+j]]--
				}
			}
		} else if chunkErr == nil {
			for j, ua := range userArticles {
				if ua.IsRead {
					unreadCounts[articleFeedIDs[i+j]]--
				}
			}
		}
		// On complete chunk failure leave all as unread (conservative).
	}

	return unreadCounts, nil
}

func (db *DatastoreDB) GetTotalArticleCount(userID int) (int, error) {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	userFeeds, err := db.getUserFeedsWithRetry(ctx, userID, 3, 500*time.Millisecond)
	if err != nil {
		return 0, err
	}
	if len(userFeeds) == 0 {
		return 0, nil
	}

	type countResult struct {
		count int
		err   error
	}
	results := make(chan countResult, len(userFeeds))
	for _, feed := range userFeeds {
		go func(feedID int) {
			q := datastore.NewQuery("Article").
				FilterField("feed_id", "=", int64(feedID)).
				KeysOnly()
			keys, err := db.client.GetAll(ctx, q, nil)
			results <- countResult{count: len(keys), err: err}
		}(feed.ID)
	}

	total := 0
	for range userFeeds {
		r := <-results
		if r.err != nil {
			return 0, r.err
		}
		total += r.count
	}
	return total, nil
}

// GetAccountStats retrieves user account statistics using parallel queries.
// Returns total articles, total unread, and active feeds count.
func (db *DatastoreDB) GetAccountStats(userID int) (map[string]interface{}, error) {
	defer logSlowQuery("GetAccountStats", time.Now())
	ctx, cancel := newDatastoreContext()
	defer cancel()

	userFeeds, err := db.getUserFeedsWithRetry(ctx, userID, 3, 500*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to get user feeds: %w", err)
	}

	if len(userFeeds) == 0 {
		return map[string]interface{}{
			"total_articles": 0,
			"total_unread":   0,
			"active_feeds":   0,
		}, nil
	}

	cutoff := time.Now().UTC().Add(-unreadCountWindowDays * 24 * time.Hour)

	// Phase 1: per-feed goroutines fetch total article count + recent article keys in parallel.
	type feedData struct {
		feedID     int
		totalCount int
		recentKeys []*datastore.Key
		err        error
	}

	feedResults := make(chan feedData, len(userFeeds))
	for _, feed := range userFeeds {
		go func(feedID int) {
			totalQ := datastore.NewQuery("Article").
				FilterField("feed_id", "=", int64(feedID)).
				KeysOnly()
			totalKeys, err := db.client.GetAll(ctx, totalQ, nil)
			if err != nil {
				feedResults <- feedData{err: err}
				return
			}

			recentQ := datastore.NewQuery("Article").
				FilterField("feed_id", "=", int64(feedID)).
				FilterField("published_at", ">=", cutoff).
				KeysOnly()
			recentKeys, err := db.client.GetAll(ctx, recentQ, nil)
			if err != nil {
				feedResults <- feedData{err: err}
				return
			}

			feedResults <- feedData{feedID: feedID, totalCount: len(totalKeys), recentKeys: recentKeys}
		}(feed.ID)
	}

	totalArticles := 0
	feedUnreadCounts := make(map[int]int)
	var allRecentKeys []*datastore.Key
	var recentKeyFeedIDs []int

	for i := 0; i < len(userFeeds); i++ {
		r := <-feedResults
		if r.err != nil {
			return nil, r.err
		}
		totalArticles += r.totalCount
		feedUnreadCounts[r.feedID] = len(r.recentKeys)
		for _, k := range r.recentKeys {
			allRecentKeys = append(allRecentKeys, k)
			recentKeyFeedIDs = append(recentKeyFeedIDs, r.feedID)
		}
	}

	// Phase 2: one merged GetMulti for all UserArticle entities across every feed.
	if len(allRecentKeys) > 0 {
		userArticleKeys := make([]*datastore.Key, len(allRecentKeys))
		for i, ak := range allRecentKeys {
			userArticleKeys[i] = datastore.NameKey("UserArticle", fmt.Sprintf("%d_%d", userID, ak.ID), nil)
		}

		chunkSize := 1000
		for i := 0; i < len(userArticleKeys); i += chunkSize {
			end := i + chunkSize
			if end > len(userArticleKeys) {
				end = len(userArticleKeys)
			}
			chunk := userArticleKeys[i:end]
			userArticles := make([]UserArticleEntity, len(chunk))
			chunkErr := db.client.GetMulti(ctx, chunk, userArticles)

			if multiErr, ok := chunkErr.(datastore.MultiError); ok {
				for j, singleErr := range multiErr {
					if singleErr == nil && userArticles[j].IsRead {
						feedUnreadCounts[recentKeyFeedIDs[i+j]]--
					}
				}
			} else if chunkErr == nil {
				for j, ua := range userArticles {
					if ua.IsRead {
						feedUnreadCounts[recentKeyFeedIDs[i+j]]--
					}
				}
			}
		}
	}

	totalUnread := 0
	activeFeeds := 0
	for _, count := range feedUnreadCounts {
		totalUnread += count
		if count > 0 {
			activeFeeds++
		}
	}

	return map[string]interface{}{
		"total_articles": totalArticles,
		"total_unread":   totalUnread,
		"active_feeds":   activeFeeds,
	}, nil
}


// CleanupOrphanedUserArticles removes UserArticle entities that reference articles from feeds
// the user is no longer subscribed to. Only cleans up articles older than the specified number of days.
// Returns the number of records deleted.
//
// Each page gets its own datastoreTimeout budget instead of sharing one for the whole run, and
// article/subscription lookups within a page are batched with GetMulti instead of one Get per
// candidate. A single shared 30s deadline plus per-key round trips (up to two Datastore Gets for
// every one of up to 500 keys in a page) reliably timed out once orphaned-article volume grew -
// see gr-dnhv, discovered when this started failing every real cron run in production.
func (db *DatastoreDB) CleanupOrphanedUserArticles(olderThanDays int) (int, error) {
	defer logSlowQuery("CleanupOrphanedUserArticles", time.Now())
	deletedCount := 0
	cutoffDate := time.Now().AddDate(0, 0, -olderThanDays)

	const batchSize = 500
	var cursor *datastore.Cursor

	for {
		ctx, cancel := newDatastoreContext()

		query := datastore.NewQuery("UserArticle").KeysOnly().Limit(batchSize)
		if cursor != nil {
			query = query.Start(*cursor)
		}

		var keys []*datastore.Key
		it := db.client.Run(ctx, query)
		for {
			key, err := it.Next(nil)
			if err == iterator.Done {
				break
			}
			if err != nil {
				cancel()
				return deletedCount, fmt.Errorf("failed to query user articles: %w", err)
			}
			keys = append(keys, key)
		}

		if len(keys) == 0 {
			cancel()
			break
		}

		deleted, err := db.deleteOrphanedUserArticlesBatch(ctx, keys, olderThanDays, cutoffDate)
		deletedCount += deleted
		if err != nil {
			cancel()
			// Report what was deleted before the failure so callers (and
			// Cloud Tasks' retry policy) see this as a failed run rather
			// than a successful one with an undercounted total.
			return deletedCount, err
		}

		morePages := len(keys) == batchSize
		var nextCursor datastore.Cursor
		if morePages {
			nextCursor, err = it.Cursor()
		}
		cancel()
		if !morePages || err != nil {
			break
		}
		cursor = &nextCursor
	}

	return deletedCount, nil
}

// deleteOrphanedUserArticlesBatch resolves and deletes the orphaned entries within one page of
// UserArticle keys, batching the Article/UserFeed existence checks with GetMulti instead of two
// Datastore round trips per candidate key.
func (db *DatastoreDB) deleteOrphanedUserArticlesBatch(ctx context.Context, keys []*datastore.Key, olderThanDays int, cutoffDate time.Time) (int, error) {
	type candidate struct {
		key       *datastore.Key
		userID    int64
		articleID int64
	}

	candidates := make([]candidate, 0, len(keys))
	articleKeys := make([]*datastore.Key, 0, len(keys))
	for _, key := range keys {
		// Parse user_id and article_id from the key name (format: "userID_articleID")
		var userID, articleID int64
		if _, err := fmt.Sscanf(key.Name, "%d_%d", &userID, &articleID); err != nil {
			continue // Skip malformed keys
		}
		candidates = append(candidates, candidate{key: key, userID: userID, articleID: articleID})
		articleKeys = append(articleKeys, datastore.IDKey("Article", articleID, nil))
	}

	if len(candidates) == 0 {
		return 0, nil
	}

	articles := make([]ArticleEntity, len(articleKeys))
	var articleErrs datastore.MultiError
	if err := db.client.GetMulti(ctx, articleKeys, articles); err != nil && !errors.As(err, &articleErrs) {
		return 0, fmt.Errorf("failed to batch-get articles: %w", err)
	}

	var keysToDelete []*datastore.Key
	userFeedKeys := make([]*datastore.Key, 0, len(candidates))
	pending := make([]candidate, 0, len(candidates)) // candidates awaiting a UserFeed check, aligned with userFeedKeys

	for i, c := range candidates {
		if articleErrs != nil && articleErrs[i] != nil {
			// Article doesn't exist (or failed to load), definitely orphaned.
			keysToDelete = append(keysToDelete, c.key)
			continue
		}
		article := articles[i]
		if olderThanDays > 0 && article.CreatedAt.After(cutoffDate) {
			continue // Too recent, skip
		}
		userFeedKeys = append(userFeedKeys, datastore.NameKey("UserFeed", fmt.Sprintf("%d_%d", c.userID, article.FeedID), nil))
		pending = append(pending, c)
	}

	if len(userFeedKeys) > 0 {
		userFeeds := make([]UserFeedEntity, len(userFeedKeys))
		var userFeedErrs datastore.MultiError
		if err := db.client.GetMulti(ctx, userFeedKeys, userFeeds); err != nil && !errors.As(err, &userFeedErrs) {
			return 0, fmt.Errorf("failed to batch-get user feeds: %w", err)
		}
		for i, c := range pending {
			if userFeedErrs != nil && userFeedErrs[i] == datastore.ErrNoSuchEntity {
				// User is not subscribed, this is orphaned.
				keysToDelete = append(keysToDelete, c.key)
			}
			// Any other outcome (nil, or a non-not-found error): user is still subscribed
			// or the check was inconclusive, so leave it for the next run.
		}
	}

	if len(keysToDelete) == 0 {
		return 0, nil
	}

	if err := db.client.DeleteMulti(ctx, keysToDelete); err != nil {
		return 0, fmt.Errorf("failed to delete orphaned user articles: %w", err)
	}
	return len(keysToDelete), nil
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

	// Collect unique feed IDs preserving insertion order for deterministic output
	seen := make(map[int64]bool)
	var uniqueIDs []int64
	for _, userFeed := range userFeedEntities {
		if !seen[userFeed.FeedID] {
			seen[userFeed.FeedID] = true
			uniqueIDs = append(uniqueIDs, userFeed.FeedID)
		}
	}

	if len(uniqueIDs) == 0 {
		return nil, nil
	}

	// Build keys and fetch all feeds in a single batch read instead of N round trips
	keys := make([]*datastore.Key, len(uniqueIDs))
	entities := make([]FeedEntity, len(uniqueIDs))
	for i, id := range uniqueIDs {
		keys[i] = datastore.IDKey("Feed", id, nil)
	}

	errs := db.client.GetMulti(ctx, keys, entities)
	// GetMulti returns a MultiError when some (but not all) keys are missing.
	// Treat individual ErrNoSuchEntity as non-fatal; propagate other errors.
	var multiErr datastore.MultiError
	if errs != nil && !errors.As(errs, &multiErr) {
		return nil, fmt.Errorf("failed to batch-get user feeds: %w", errs)
	}

	var feeds []Feed
	for i, entity := range entities {
		if multiErr != nil && multiErr[i] != nil {
			if multiErr[i] != datastore.ErrNoSuchEntity {
				log.Printf("Error fetching feed %d: %v", uniqueIDs[i], multiErr[i])
			}
			continue
		}
		entity.ID = keys[i].ID
		feeds = append(feeds, Feed{
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
			ETag:                  entity.ETag,
			LastModified:          entity.LastModified,
		})
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
func (db *DatastoreDB) SetUserAdminAtomic(targetID, callerID int, isAdmin bool) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	_, err := db.client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		if targetID == callerID && !isAdmin {
			return ErrSelfDemotion
		}
		userKey := datastore.IDKey("User", int64(targetID), nil)
		var user UserEntity
		if err := tx.Get(userKey, &user); err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		user.IsAdmin = isAdmin
		if _, err := tx.Put(userKey, &user); err != nil {
			return fmt.Errorf("failed to save user: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to set user admin status: %w", err)
	}
	return nil
}

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
	if _, err := db.client.Put(ctx, key, entity); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
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
		return nil, fmt.Errorf("failed to get session: %w", err)
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
	var entity SessionEntity
	if err := db.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get session for expiry update: %w", err)
	}

	// Update expiry
	entity.ExpiresAt = newExpiry

	// Save back to Datastore
	if _, err := db.client.Put(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to update session expiry: %w", err)
	}
	return nil
}

func (db *DatastoreDB) DeleteSession(sessionID string) error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	key := datastore.NameKey("Session", sessionID, nil)
	if err := db.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (db *DatastoreDB) DeleteExpiredSessions() error {
	ctx, cancel := newDatastoreContext()
	defer cancel()

	// Query for expired sessions
	query := datastore.NewQuery("Session").FilterField("expires_at", "<", time.Now()).KeysOnly()

	keys, err := db.client.GetAll(ctx, query, nil)
	if err != nil {
		return fmt.Errorf("failed to query expired sessions: %w", err)
	}

	// Delete expired sessions in batches
	if len(keys) > 0 {
		if err := db.client.DeleteMulti(ctx, keys); err != nil {
			return fmt.Errorf("failed to delete expired sessions: %w", err)
		}
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
