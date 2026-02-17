package services

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/jeffreyp/goread2/internal/cache"
	"github.com/jeffreyp/goread2/internal/database"
	"golang.org/x/text/cases"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/language"
)

const (
	// maxFeedBodySize is the maximum size we'll read from a feed (10MB)
	// This prevents memory exhaustion and high bandwidth costs from malicious/broken feeds
	maxFeedBodySize = 10 * 1024 * 1024 // 10MB
)

// HTTPClient interface for making HTTP requests (allows mocking in tests)
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type FeedService struct {
	db            database.Database
	rateLimiter   *DomainRateLimiter
	urlValidator  *URLValidator
	unreadCache   *cache.UnreadCache
	feedListCache *cache.FeedListCache
	httpClient    HTTPClient // Optional: if nil, creates client using urlValidator
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type RDF struct {
	XMLName xml.Name   `xml:"RDF"`
	Channel RDFChannel `xml:"channel"`
	Items   []RDFItem  `xml:"item"`
}

type RDFChannel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
}

type RDFItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Date        string `xml:"date"`
	Creator     string `xml:"creator"`
}

type Atom struct {
	XMLName  xml.Name    `xml:"feed"`
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle"`
	Entries  []AtomEntry `xml:"entry"`
}

type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Author      string `xml:"author"`
	PubDate     string `xml:"pubDate"`
	Content     string `xml:"encoded"`
}

type AtomEntry struct {
	Title     string      `xml:"title"`
	Link      AtomLink    `xml:"link"`
	Summary   string      `xml:"summary"`
	Content   AtomContent `xml:"content"`
	Author    AtomAuthor  `xml:"author"`
	Published string      `xml:"published"`
	Updated   string      `xml:"updated"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
}

type AtomContent struct {
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

type AtomAuthor struct {
	Name string `xml:"name"`
}

// FetchOptions provides HTTP conditional request headers for bandwidth optimization
type FetchOptions struct {
	ETag         string
	LastModified string
}

// Unified feed data structure
type FeedData struct {
	Title                string
	Description          string
	Articles             []ArticleData
	ResponseETag         string // ETag from the HTTP response
	ResponseLastModified string // Last-Modified from the HTTP response
}

type ArticleData struct {
	Title       string
	Link        string
	Description string
	Content     string
	Author      string
	PublishedAt time.Time
}

// OPML structures for parsing OPML files
type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Head    OPMLHead `xml:"head"`
	Body    OPMLBody `xml:"body"`
}

type OPMLHead struct {
	Title string `xml:"title"`
}

type OPMLBody struct {
	Outlines []OPMLOutline `xml:"outline"`
}

type OPMLOutline struct {
	Text    string        `xml:"text,attr"`
	Title   string        `xml:"title,attr"`
	Type    string        `xml:"type,attr"`
	XMLURL  string        `xml:"xmlUrl,attr"`
	HTMLURL string        `xml:"htmlUrl,attr"`
	Outline []OPMLOutline `xml:"outline"`
}

func NewFeedService(db database.Database, rateLimiter *DomainRateLimiter) *FeedService {
	return &FeedService{
		db:            db,
		rateLimiter:   rateLimiter,
		urlValidator:  NewURLValidator(),
		unreadCache:   cache.NewUnreadCache(5 * time.Minute),    // 5 minute TTL
		feedListCache: cache.NewFeedListCache(20 * time.Minute), // 20 minute TTL
	}
}

// SetHTTPClient sets a custom HTTP client for testing purposes
func (fs *FeedService) SetHTTPClient(client HTTPClient) {
	fs.httpClient = client
}

func (fs *FeedService) AddFeed(url string) (*database.Feed, error) {
	ctx := context.Background()
	feedData, err := fs.fetchFeed(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}

	now := time.Now()
	feed := &database.Feed{
		Title:                 feedData.Title,
		URL:                   url,
		Description:           feedData.Description,
		CreatedAt:             now,
		UpdatedAt:             now,
		LastFetch:             now,
		LastChecked:           now,
		LastHadNewContent:     now,
		AverageUpdateInterval: 0, // Will be calculated after first few updates
	}

	if err := fs.db.AddFeed(feed); err != nil {
		return nil, fmt.Errorf("failed to insert feed: %w", err)
	}

	if _, err := fs.saveArticlesFromFeed(feed.ID, feedData); err != nil {
		return nil, fmt.Errorf("failed to save articles: %w", err)
	}

	return feed, nil
}

func (fs *FeedService) GetFeeds() ([]database.Feed, error) {
	return fs.db.GetFeeds()
}

func (fs *FeedService) GetUserFeeds(userID int) ([]database.Feed, error) {
	return fs.db.GetUserFeeds(userID)
}

func (fs *FeedService) AddFeedForUser(userID int, inputURL string) (*database.Feed, error) {
	// Add overall timeout to prevent infinite hangs - reduced for better UX
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Use a channel to handle timeout
	done := make(chan struct{})
	var result *database.Feed
	var resultErr error

	go func() {
		defer close(done)
		result, resultErr = fs.addFeedForUserInternal(ctx, userID, inputURL)
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: %s", ErrFeedTimeout, inputURL)
	case <-done:
		return result, resultErr
	}
}

func (fs *FeedService) addFeedForUserInternal(ctx context.Context, userID int, inputURL string) (*database.Feed, error) {
	var feedURL string

	// Normalize the input URL first
	discovery := NewFeedDiscovery()
	normalizedURL, err := discovery.NormalizeURL(ctx, inputURL)
	if err != nil {
		// Errors from NormalizeURL are already wrapped with custom types
		return nil, err
	}

	// Check for known sites first
	if strings.Contains(normalizedURL, "slashdot.org") {
		feedURL = "https://rss.slashdot.org/Slashdot/slashdotMain"
	} else if strings.Contains(normalizedURL, "nytimes.com") {
		feedURL = "https://rss.nytimes.com/services/xml/rss/nyt/HomePage.xml"
	} else if strings.Contains(normalizedURL, "seattletimes.com") {
		feedURL = "https://www.seattletimes.com/feed/"
	} else {
		// Use feed discovery for other sites
		feedURLs, err := discovery.DiscoverFeedURL(ctx, inputURL)
		if err != nil {
			// Errors from DiscoverFeedURL are already wrapped with custom types
			return nil, err
		}

		if len(feedURLs) == 0 {
			return nil, fmt.Errorf("%w: %s", ErrFeedNotFound, inputURL)
		}

		feedURL = feedURLs[0]
	}

	// First check if feed already exists
	existingFeed, err := fs.db.GetFeedByURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to check existing feed: %v", ErrDatabaseError, err)
	}

	if existingFeed == nil {
		// Feed doesn't exist, create it
		feedData, err := fs.fetchFeed(ctx, feedURL)
		if err != nil {
			// Errors from fetchFeed are already wrapped with custom types
			return nil, err
		}

		feed := &database.Feed{
			Title:       feedData.Title,
			URL:         feedURL,
			Description: feedData.Description,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			LastFetch:   time.Now(),
		}

		if err := fs.db.AddFeed(feed); err != nil {
			return nil, fmt.Errorf("%w: failed to insert feed: %v", ErrDatabaseError, err)
		}

		// Get user's article limit preference for new feeds
		user, err := fs.db.GetUserByID(userID)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to get user %d: %v", ErrDatabaseError, userID, err)
		}

		if _, err := fs.saveArticlesFromFeedWithLimit(feed.ID, feedData, user.MaxArticlesOnFeedAdd); err != nil {
			return nil, fmt.Errorf("%w: failed to save articles: %v", ErrDatabaseError, err)
		}

		existingFeed = feed
	} else {
		// Feed exists, but check if we should enhance the title
		enhancedTitle := fs.enhanceFeedTitle(fs.cleanDuplicateTitle(existingFeed.Title), feedURL)
		if enhancedTitle != existingFeed.Title {
			// Update the feed with the enhanced title
			existingFeed.Title = enhancedTitle
			if err := fs.db.UpdateFeed(existingFeed); err != nil {
				// Log error but don't fail - title enhancement is not critical
				log.Printf("Failed to update feed title to '%s': %v", enhancedTitle, err)
			}
		}
	}

	// Subscribe user to the feed FIRST
	if err := fs.db.SubscribeUserToFeed(userID, existingFeed.ID); err != nil {
		return nil, fmt.Errorf("%w: failed to subscribe user to feed: %v", ErrDatabaseError, err)
	}

	// Mark all articles in this feed as unread for the subscriber synchronously
	// This ensures unread counts are immediately accurate after feed addition
	if err := fs.markExistingArticlesAsUnreadForUser(userID, existingFeed.ID); err != nil {
		log.Printf("Failed to mark existing articles as unread for user %d, feed %d: %v", userID, existingFeed.ID, err)
		// Don't fail the entire operation, just log the error
	}

	// Invalidate caches since subscription changed
	fs.unreadCache.Invalidate(userID)
	fs.feedListCache.Invalidate() // Feed list changed - user subscribed to new feed

	return existingFeed, nil
}

func (fs *FeedService) DeleteFeed(id int) error {
	return fs.db.DeleteFeed(id)
}

func (fs *FeedService) UnsubscribeUserFromFeed(userID, feedID int) error {
	err := fs.db.UnsubscribeUserFromFeed(userID, feedID)
	if err != nil {
		return err
	}

	// Invalidate caches since subscription changed
	fs.unreadCache.Invalidate(userID)
	fs.feedListCache.Invalidate() // Feed list may have changed - user unsubscribed

	return nil
}

func (fs *FeedService) GetArticles(feedID int) ([]database.Article, error) {
	return fs.db.GetArticles(feedID)
}

func (fs *FeedService) GetUserArticles(userID int) ([]database.Article, error) {
	return fs.db.GetUserArticles(userID)
}

func (fs *FeedService) GetUserArticlesPaginated(userID int, limit int, cursor string, unreadOnly bool) (*database.ArticlePaginationResult, error) {
	return fs.db.GetUserArticlesPaginated(userID, limit, cursor, unreadOnly)
}

func (fs *FeedService) GetUserFeedArticles(userID, feedID int) ([]database.Article, error) {
	return fs.db.GetUserFeedArticles(userID, feedID)
}

// Legacy methods removed - use multi-user methods instead
// func (fs *FeedService) MarkRead(articleID int, isRead bool) error {
// 	return fmt.Errorf("deprecated: use MarkUserArticleRead instead")
// }

// func (fs *FeedService) ToggleStar(articleID int) error {
// 	return fmt.Errorf("deprecated: use ToggleUserArticleStar instead")
// }

func (fs *FeedService) MarkUserArticleRead(userID, articleID int, isRead bool) error {
	err := fs.db.MarkUserArticleRead(userID, articleID, isRead)
	if err != nil {
		return err
	}

	// Invalidate cache to ensure accurate counts on next request
	fs.unreadCache.Invalidate(userID)

	return nil
}

func (fs *FeedService) ToggleUserArticleStar(userID, articleID int) error {
	// Starring doesn't affect unread counts, so no cache invalidation needed
	return fs.db.ToggleUserArticleStar(userID, articleID)
}

func (fs *FeedService) MarkAllArticlesRead(userID int) (int, error) {
	// Get all user's articles
	articles, err := fs.db.GetUserArticles(userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user articles: %w", err)
	}

	if len(articles) == 0 {
		return 0, nil
	}

	// Use batch operation to mark all as read
	if err := fs.db.BatchSetUserArticleStatus(userID, articles, true, false); err != nil {
		return 0, fmt.Errorf("failed to mark articles as read: %w", err)
	}

	// Invalidate cache since unread counts changed
	fs.unreadCache.Invalidate(userID)

	return len(articles), nil
}

func (fs *FeedService) GetUserUnreadCounts(userID int, userFeeds []database.Feed) (map[int]int, error) {
	// Try cache first for fast response
	if cached, hit := fs.unreadCache.Get(userID); hit {
		return cached, nil
	}

	// Cache miss - fetch from database
	unreadCounts, err := fs.db.GetUserUnreadCounts(userID)
	if err != nil {
		return nil, err
	}

	// Safety filter: only return counts for feeds that actually exist
	// This prevents orphaned data from corrupting the UI
	// userFeeds is now passed in as a parameter to avoid duplicate DB call

	// Create a map of valid feed IDs
	validFeedIDs := make(map[int]bool)
	for _, feed := range userFeeds {
		validFeedIDs[feed.ID] = true
	}

	// Filter out orphaned counts
	filteredCounts := make(map[int]int)
	for feedID, count := range unreadCounts {
		if validFeedIDs[feedID] {
			filteredCounts[feedID] = count
		} else {
			// Log orphaned data for debugging
			log.Printf("Warning: Filtered orphaned unread count for non-existent feed ID %d (%d articles)", feedID, count)
		}
	}

	// Store in cache for next time
	fs.unreadCache.Set(userID, filteredCounts)

	return filteredCounts, nil
}

func (fs *FeedService) markExistingArticlesAsUnreadForUser(userID, feedID int) error {
	// Get user's preference for max articles on feed addition
	user, err := fs.db.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user %d: %w", userID, err)
	}

	// Get all articles in this feed
	articles, err := fs.db.GetArticles(feedID)
	if err != nil {
		return fmt.Errorf("failed to get articles for feed %d: %w", feedID, err)
	}

	if len(articles) == 0 {
		return nil
	}

	// Limit articles based on user preference (0 means unlimited)
	articlesToMark := articles
	if user.MaxArticlesOnFeedAdd > 0 && len(articles) > user.MaxArticlesOnFeedAdd {
		// Sort by published date (most recent first) and take the limit
		// Articles should already be sorted by published_at DESC from GetArticles
		articlesToMark = articles[:user.MaxArticlesOnFeedAdd]
		log.Printf("User %d: Limited to %d most recent articles out of %d total for feed %d",
			userID, user.MaxArticlesOnFeedAdd, len(articles), feedID)
	}

	// Use batch insert for better performance
	return fs.db.BatchSetUserArticleStatus(userID, articlesToMark, false, false) // unread, unstarred
}

func (fs *FeedService) UpdateUserMaxArticlesOnFeedAdd(userID, maxArticles int) error {
	return fs.db.UpdateUserMaxArticlesOnFeedAdd(userID, maxArticles)
}

func (fs *FeedService) fetchFeed(ctx context.Context, url string, opts ...*FetchOptions) (*FeedData, error) {
	// Validate URL for SSRF protection (skip if using mock HTTP client for testing)
	if fs.httpClient == nil {
		if err := fs.urlValidator.ValidateURL(ctx, url); err != nil {
			// Check if it's an SSRF protection error
			if strings.Contains(err.Error(), "SSRF protection") ||
				strings.Contains(err.Error(), "blocked network") ||
				strings.Contains(err.Error(), "not allowed") {
				return nil, fmt.Errorf("%w: %v", ErrSSRFBlocked, err)
			}
			return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
		}
	}

	// Apply rate limiting if available (skip if using mock HTTP client for testing)
	if fs.rateLimiter != nil && fs.httpClient == nil {
		if !fs.rateLimiter.Allow(url) {
			// Rate limiting is a temporary network-related issue
			return nil, fmt.Errorf("%w: rate limited - too many requests to domain", ErrNetworkError)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrNetworkError, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoRead/2.0)")

	// Add conditional request headers if options provided
	if len(opts) > 0 && opts[0] != nil {
		if opts[0].ETag != "" {
			req.Header.Set("If-None-Match", opts[0].ETag)
		}
		if opts[0].LastModified != "" {
			req.Header.Set("If-Modified-Since", opts[0].LastModified)
		}
	}

	// Use injected HTTP client if available, otherwise create secure client
	var client HTTPClient
	if fs.httpClient != nil {
		client = fs.httpClient
	} else {
		client = fs.urlValidator.CreateSecureHTTPClient(30 * time.Second)
	}
	resp, err := client.Do(req)
	if err != nil {
		// Network errors: DNS failures, connection errors, timeouts
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check HTTP status code
	if resp.StatusCode == http.StatusNotModified {
		return nil, ErrFeedNotModified
	}
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("%w: feed URL returned 404 Not Found", ErrFeedNotFound)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: feed URL returned HTTP %d", ErrNetworkError, resp.StatusCode)
	}

	// Limit the amount of data we'll read to prevent memory exhaustion and bandwidth costs
	limitedBody := io.LimitReader(resp.Body, maxFeedBodySize+1)
	body, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", ErrNetworkError, err)
	}

	// Check if we hit the size limit
	if len(body) > maxFeedBodySize {
		return nil, fmt.Errorf("%w: feed exceeds maximum size of %d bytes", ErrInvalidFeedFormat, maxFeedBodySize)
	}

	// Handle character encoding conversion
	body, err = fs.convertToUTF8(body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to convert encoding: %v", ErrInvalidFeedFormat, err)
	}

	// Capture response cache headers for conditional requests
	responseETag := resp.Header.Get("ETag")
	responseLastModified := resp.Header.Get("Last-Modified")

	// Special handling for feeds with media namespace conflicts
	body = fs.preprocessXMLForMediaConflicts(body)

	// Try parsing as RSS 2.0 first
	var rss RSS
	if err := xml.Unmarshal(body, &rss); err == nil && rss.XMLName.Local == "rss" {
		feedData := fs.convertRSSToFeedData(&rss, url)
		feedData.ResponseETag = responseETag
		feedData.ResponseLastModified = responseLastModified
		return feedData, nil
	}

	// Try parsing as RDF/RSS 1.0
	var rdf RDF
	rdfErr := xml.Unmarshal(body, &rdf)
	if rdfErr == nil {
		if rdf.XMLName.Local == "RDF" {
			feedData := fs.convertRDFToFeedData(&rdf, url)
			feedData.ResponseETag = responseETag
			feedData.ResponseLastModified = responseLastModified
			return feedData, nil
		}
	}

	// Try parsing as Atom
	var atom Atom
	if err := xml.Unmarshal(body, &atom); err == nil && atom.XMLName.Local == "feed" {
		feedData := fs.convertAtomToFeedData(&atom, url)
		feedData.ResponseETag = responseETag
		feedData.ResponseLastModified = responseLastModified
		return feedData, nil
	}

	return nil, fmt.Errorf("%w: unsupported feed format or invalid XML", ErrInvalidFeedFormat)
}

func (fs *FeedService) convertRSSToFeedData(rss *RSS, feedURL string) *FeedData {
	articles := make([]ArticleData, len(rss.Channel.Items))
	for i, item := range rss.Channel.Items {
		publishedAt, _ := time.Parse(time.RFC1123Z, item.PubDate)
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		articles[i] = ArticleData{
			Title:       fs.sanitizeArticleTitle(item.Title, item.Link, item.Description),
			Link:        item.Link,
			Description: item.Description,
			Content:     item.Content,
			Author:      item.Author,
			PublishedAt: publishedAt,
		}
	}

	return &FeedData{
		Title:       fs.enhanceFeedTitle(fs.cleanDuplicateTitle(rss.Channel.Title), feedURL),
		Description: rss.Channel.Description,
		Articles:    articles,
	}
}

func (fs *FeedService) convertRDFToFeedData(rdf *RDF, feedURL string) *FeedData {
	articles := make([]ArticleData, len(rdf.Items))
	for i, item := range rdf.Items {
		publishedAt, _ := time.Parse(time.RFC3339, item.Date)
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		articles[i] = ArticleData{
			Title:       fs.sanitizeArticleTitle(item.Title, item.Link, item.Description),
			Link:        item.Link,
			Description: item.Description,
			Content:     item.Description, // RDF doesn't usually have separate content
			Author:      item.Creator,
			PublishedAt: publishedAt,
		}
	}

	return &FeedData{
		Title:       fs.enhanceFeedTitle(fs.cleanDuplicateTitle(rdf.Channel.Title), feedURL),
		Description: rdf.Channel.Description,
		Articles:    articles,
	}
}

func (fs *FeedService) convertAtomToFeedData(atom *Atom, feedURL string) *FeedData {
	articles := make([]ArticleData, len(atom.Entries))
	for i, entry := range atom.Entries {
		publishedAt, _ := time.Parse(time.RFC3339, entry.Published)
		if publishedAt.IsZero() {
			if updatedAt, err := time.Parse(time.RFC3339, entry.Updated); err == nil {
				publishedAt = updatedAt
			} else {
				publishedAt = time.Now()
			}
		}

		content := entry.Content.Content
		if content == "" {
			content = entry.Summary
		}

		articles[i] = ArticleData{
			Title:       fs.sanitizeArticleTitle(entry.Title, entry.Link.Href, entry.Summary),
			Link:        entry.Link.Href,
			Description: entry.Summary,
			Content:     content,
			Author:      entry.Author.Name,
			PublishedAt: publishedAt,
		}
	}

	description := atom.Subtitle
	if description == "" {
		description = atom.Title + " feed"
	}

	return &FeedData{
		Title:       fs.enhanceFeedTitle(fs.cleanDuplicateTitle(atom.Title), feedURL),
		Description: description,
		Articles:    articles,
	}
}

func (fs *FeedService) saveArticlesFromFeed(feedID int, feedData *FeedData) (int, error) {
	// Use unlimited (0) for backward compatibility
	return fs.saveArticlesFromFeedWithLimit(feedID, feedData, 0)
}

func (fs *FeedService) saveArticlesFromFeedWithLimit(feedID int, feedData *FeedData, maxArticles int) (int, error) {
	var savedCount int
	var errors []string

	// Sort articles by published date (most recent first) before applying limit
	articles := make([]ArticleData, len(feedData.Articles))
	copy(articles, feedData.Articles)
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].PublishedAt.After(articles[j].PublishedAt)
	})

	// Apply article limit if specified (0 means unlimited)
	articlesToSave := articles
	if maxArticles > 0 && len(articles) > maxArticles {
		// Take the most recent articles (now properly sorted)
		articlesToSave = articles[:maxArticles]
		log.Printf("Feed %d: Limited to %d most recent articles out of %d total (user preference)",
			feedID, maxArticles, len(articles))
	}

	for _, articleData := range articlesToSave {
		article := &database.Article{
			FeedID:      feedID,
			Title:       articleData.Title,
			URL:         articleData.Link,
			Content:     articleData.Content,
			Description: articleData.Description,
			Author:      articleData.Author,
			PublishedAt: articleData.PublishedAt,
			CreatedAt:   time.Now(),
		}

		if err := fs.db.AddArticle(article); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to save article '%s': %v", article.Title, err))
			continue // Continue processing other articles
		}
		savedCount++
	}

	if maxArticles > 0 && len(articles) > maxArticles {
		log.Printf("Feed %d: Saved %d/%d articles (limited by user preference to %d)",
			feedID, savedCount, len(articles), maxArticles)
	} else {
		log.Printf("Feed %d: Saved %d/%d articles", feedID, savedCount, len(articles))
	}

	if len(errors) > 0 {
		log.Printf("Feed %d: Errors saving %d articles: %v", feedID, len(errors), errors)
	}

	// Only return error if NO articles were saved
	if savedCount == 0 && len(articlesToSave) > 0 {
		return 0, fmt.Errorf("failed to save any articles from feed %d", feedID)
	}

	return savedCount, nil
}

func (fs *FeedService) RefreshFeeds() error {
	// Get all unique feeds from both global feeds and all user feeds
	globalFeeds, err := fs.GetFeeds()
	if err != nil {
		return err
	}

	// Also get all user feeds to ensure we refresh feeds that users are subscribed to
	// Use cache to reduce expensive database queries (runs every hour via cron)
	allUserFeeds, cached := fs.feedListCache.Get()
	if !cached {
		// Cache miss - fetch from database
		var err error
		allUserFeeds, err = fs.db.GetAllUserFeeds()
		if err != nil {
			allUserFeeds = []database.Feed{}
		} else {
			// Populate cache for next request
			fs.feedListCache.Set(allUserFeeds)
		}
	}

	// Combine and deduplicate feeds by URL
	feedMap := make(map[string]database.Feed)

	// Add global feeds
	for _, feed := range globalFeeds {
		feedMap[feed.URL] = feed
	}

	// Add user feeds (will overwrite if same URL)
	for _, feed := range allUserFeeds {
		feedMap[feed.URL] = feed
	}

	now := time.Now()
	checked := 0
	skipped := 0
	hasNewContent := 0
	notModified := 0

	for _, feed := range feedMap {
		// Smart feed prioritization: only check feeds that are due
		if !fs.shouldCheckFeed(feed, now) {
			skipped++
			continue
		}

		// Build conditional request options from stored cache headers
		var fetchOpts *FetchOptions
		if feed.ETag != "" || feed.LastModified != "" {
			fetchOpts = &FetchOptions{
				ETag:         feed.ETag,
				LastModified: feed.LastModified,
			}
		}

		// Fetch and save articles
		ctx := context.Background()
		feedData, err := fs.fetchFeed(ctx, feed.URL, fetchOpts)
		checked++

		// Update last_checked regardless of success/failure
		feed.LastChecked = now

		if errors.Is(err, ErrFeedNotModified) {
			notModified++
			_ = fs.updateFeedTracking(feed, false)
			continue
		}

		if err != nil {
			log.Printf("Failed to fetch feed %s: %v", feed.URL, err)
			_ = fs.updateFeedTracking(feed, false)
			continue
		}

		// Persist response cache headers for next conditional request
		if feedData.ResponseETag != "" || feedData.ResponseLastModified != "" {
			if updateErr := fs.db.UpdateFeedCacheHeaders(feed.ID, feedData.ResponseETag, feedData.ResponseLastModified); updateErr != nil {
				log.Printf("Failed to update cache headers for feed %s: %v", feed.URL, updateErr)
			}
		}

		// Save articles and get count of newly saved articles
		savedCount, err := fs.saveArticlesFromFeed(feed.ID, feedData)
		if err != nil {
			log.Printf("Failed to save articles from feed %s: %v", feed.URL, err)
			_ = fs.updateFeedTracking(feed, false)
			continue
		}

		// Check if there are new articles based on saved count
		hadNewContent := savedCount > 0

		if hadNewContent {
			hasNewContent++
		}

		// Update tracking fields
		_ = fs.updateFeedTracking(feed, hadNewContent)
		_ = fs.db.UpdateFeedLastFetch(feed.ID, now)
	}

	log.Printf("Feed refresh complete: checked=%d, skipped=%d, not_modified=%d, had_new_content=%d", checked, skipped, notModified, hasNewContent)

	return nil
}

// shouldCheckFeed determines if a feed should be checked based on smart prioritization
func (fs *FeedService) shouldCheckFeed(feed database.Feed, now time.Time) bool {
	// Always check feeds that have never been checked
	if feed.LastChecked.IsZero() {
		return true
	}

	// Calculate time since last check
	timeSinceLastCheck := now.Sub(feed.LastChecked)

	// If we have historical data about update frequency, use it
	if feed.AverageUpdateInterval > 0 {
		// Check if it's been at least 50% of the average update interval
		// This ensures we don't miss updates while avoiding excessive checks
		checkInterval := time.Duration(feed.AverageUpdateInterval) * time.Second / 2
		if timeSinceLastCheck >= checkInterval {
			return true
		}
	} else {
		// No historical data: use conservative defaults based on last activity
		if !feed.LastHadNewContent.IsZero() {
			timeSinceNewContent := now.Sub(feed.LastHadNewContent)

			// If feed had new content recently (< 1 week), check more frequently
			if timeSinceNewContent < 7*24*time.Hour {
				return timeSinceLastCheck >= 30*time.Minute
			}

			// If feed had new content in the last month, check every hour
			if timeSinceNewContent < 30*24*time.Hour {
				return timeSinceLastCheck >= 1*time.Hour
			}

			// If feed hasn't updated in months, check less frequently (every 6 hours)
			return timeSinceLastCheck >= 6*time.Hour
		}

		// Never seen new content: check every hour initially
		return timeSinceLastCheck >= 1*time.Hour
	}

	return false
}

// updateFeedTracking updates the smart tracking fields for a feed
func (fs *FeedService) updateFeedTracking(feed database.Feed, hadNewContent bool) error {
	now := time.Now()
	lastHadNewContent := feed.LastHadNewContent

	// Update last_had_new_content if there was new content
	if hadNewContent {
		// Calculate new average update interval
		if !feed.LastHadNewContent.IsZero() {
			intervalSeconds := int(now.Sub(feed.LastHadNewContent).Seconds())

			// Update running average (weighted: 70% old, 30% new)
			if feed.AverageUpdateInterval > 0 {
				feed.AverageUpdateInterval = (feed.AverageUpdateInterval*7 + intervalSeconds*3) / 10
			} else {
				feed.AverageUpdateInterval = intervalSeconds
			}
		}

		lastHadNewContent = now
	}

	// Update the feed tracking in database
	return fs.db.UpdateFeedTracking(feed.ID, now, lastHadNewContent, feed.AverageUpdateInterval)
}

func (fs *FeedService) ImportOPML(userID int, opmlData []byte) (int, error) {
	var opml OPML
	if err := xml.Unmarshal(opmlData, &opml); err != nil {
		return 0, fmt.Errorf("failed to parse OPML: %w", err)
	}

	importedCount := 0
	feeds := fs.extractFeedsFromOutlines(opml.Body.Outlines)

	for _, feedURL := range feeds {
		if feedURL == "" {
			continue
		}

		_, err := fs.AddFeedForUser(userID, feedURL)
		if err != nil {
			// Log error but continue with other feeds
			log.Printf("Failed to import feed %s: %v", feedURL, err)
			continue
		}

		importedCount++
	}

	return importedCount, nil
}

// FindArticleByURL searches for an article by its URL across all feeds
func (fs *FeedService) FindArticleByURL(url string) (*database.Article, error) {
	return fs.db.FindArticleByURL(url)
}

// ImportOPMLWithLimits imports OPML feeds while respecting subscription limits
func (fs *FeedService) ImportOPMLWithLimits(userID int, opmlData []byte, subscriptionService *SubscriptionService) (int, error) {
	var opml OPML
	if err := xml.Unmarshal(opmlData, &opml); err != nil {
		return 0, fmt.Errorf("failed to parse OPML: %w", err)
	}

	feeds := fs.extractFeedsFromOutlines(opml.Body.Outlines)
	importedCount := 0

	for _, feedURL := range feeds {
		if feedURL == "" {
			continue
		}

		// Check if user can add more feeds before each import
		if err := subscriptionService.CanUserAddFeed(userID); err != nil {
			// Return partial import count and the error
			return importedCount, err
		}

		_, err := fs.AddFeedForUser(userID, feedURL)
		if err != nil {
			// Log error but continue with other feeds
			log.Printf("Failed to import feed %s: %v", feedURL, err)
			continue
		}

		importedCount++
	}

	return importedCount, nil
}

func (fs *FeedService) extractFeedsFromOutlines(outlines []OPMLOutline) []string {
	var feeds []string

	for _, outline := range outlines {
		// If this outline has a feed URL, add it
		if outline.XMLURL != "" {
			feeds = append(feeds, outline.XMLURL)
		}

		// Recursively process nested outlines (folders)
		if len(outline.Outline) > 0 {
			nestedFeeds := fs.extractFeedsFromOutlines(outline.Outline)
			feeds = append(feeds, nestedFeeds...)
		}
	}

	return feeds
}

// ExportOPML generates an OPML XML document containing all of a user's feed subscriptions
func (fs *FeedService) ExportOPML(userID int) ([]byte, error) {
	// Get all user's feeds
	feeds, err := fs.db.GetUserFeeds(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user feeds: %w", err)
	}

	// Build OPML structure
	opml := OPML{
		Head: OPMLHead{
			Title: "GoRead2 Subscriptions",
		},
		Body: OPMLBody{
			Outlines: make([]OPMLOutline, 0, len(feeds)),
		},
	}

	// Convert each feed to an OPML outline
	for _, feed := range feeds {
		outline := OPMLOutline{
			Type:    "rss",
			Text:    feed.Title,
			Title:   feed.Title,
			XMLURL:  feed.URL,
			HTMLURL: feed.URL, // Using feed URL as HTML URL since we don't store separate HTML URLs
		}
		opml.Body.Outlines = append(opml.Body.Outlines, outline)
	}

	// Marshal to XML with proper formatting
	xmlData, err := xml.MarshalIndent(opml, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OPML: %w", err)
	}

	// Add XML header
	result := []byte(xml.Header + string(xmlData))
	return result, nil
}

func (fs *FeedService) cleanDuplicateTitle(title string) string {
	title = strings.TrimSpace(title)

	// Split title into words
	words := strings.Fields(title)
	if len(words) <= 1 {
		return title
	}

	// Check if the title is a simple duplication (first half == second half)
	halfLength := len(words) / 2
	if len(words)%2 == 0 && halfLength > 0 {
		firstHalf := strings.Join(words[:halfLength], " ")
		secondHalf := strings.Join(words[halfLength:], " ")

		if firstHalf == secondHalf {
			return firstHalf
		}
	}

	return title
}

func (fs *FeedService) enhanceFeedTitle(title, feedURL string) string {
	title = strings.TrimSpace(title)

	// Skip enhancement if title is already descriptive (more than 1 word and doesn't look generic)
	words := strings.Fields(title)
	if len(words) > 1 {
		return title
	}

	// Check for generic single-word titles that could be enhanced
	genericTitles := map[string]bool{
		"blog":    true,
		"feed":    true,
		"rss":     true,
		"news":    true,
		"posts":   true,
		"updates": true,
	}

	if !genericTitles[strings.ToLower(title)] {
		return title
	}

	// Extract domain from feed URL for enhancement
	u, err := url.Parse(feedURL)
	if err != nil {
		return title
	}

	domain := u.Hostname()
	if domain != "" {
		// Remove www. prefix for cleaner display
		domain = strings.TrimPrefix(domain, "www.")
		return domain
	}

	return title
}

func (fs *FeedService) sanitizeArticleTitle(title, link, description string) string {
	// Remove excessive whitespace and trim
	title = strings.TrimSpace(title)

	// If title is empty, try to generate one from other sources
	if title == "" {
		return fs.generateFallbackTitle(link, description)
	}

	// Check if title looks like a filename, URL fragment, or other non-title content
	if fs.isInvalidTitle(title) {
		fallback := fs.generateFallbackTitle(link, description)
		if fallback != "" {
			log.Printf("Replacing invalid title '%s' with fallback: '%s'", title, fallback)
			return fallback
		}
	}

	// Clean up the title
	title = fs.cleanTitle(title)

	return title
}

func (fs *FeedService) isInvalidTitle(title string) bool {
	title = strings.ToLower(strings.TrimSpace(title))

	// Check for patterns that suggest this isn't a real title
	invalidPatterns := []string{
		// File extensions
		".jpg", ".jpeg", ".png", ".gif", ".pdf", ".doc", ".docx",
		// Common filename patterns
		"-v2-", "_v2_", "(2)", "(3)", "(4)",
		// URL-like patterns
		"http://", "https://", "www.",
		// Generic short codes
		"temp", "tmp", "test", "draft",
	}

	for _, pattern := range invalidPatterns {
		if strings.Contains(title, pattern) {
			return true
		}
	}

	// Check if title is suspiciously short and contains mostly non-letter characters
	if len(title) < 15 {
		letterCount := 0
		for _, r := range title {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				letterCount++
			}
		}
		// If less than 50% letters, it's likely not a real title
		if float64(letterCount)/float64(len(title)) < 0.5 {
			return true
		}
	}

	// Check for very short titles that look like codes or fragments
	if len(title) < 12 {
		// Count meaningful words (more than 2 characters)
		words := strings.Fields(title)
		meaningfulWords := 0
		for _, word := range words {
			if len(word) > 2 {
				meaningfulWords++
			}
		}
		// If no meaningful words, it's probably not a real title
		if meaningfulWords == 0 {
			return true
		}
	}

	return false
}

func (fs *FeedService) generateFallbackTitle(link, description string) string {
	// Try to use first sentence of description first (better for social media posts)
	if description != "" {
		description = strings.TrimSpace(description)
		// Strip HTML tags from description for better title extraction
		description = fs.stripHTMLTags(description)
		description = strings.TrimSpace(description)

		if len(description) > 10 {
			// Find first sentence or take first 80 characters
			if dotIndex := strings.Index(description, ". "); dotIndex > 10 && dotIndex < 80 {
				return description[:dotIndex]
			}
			if len(description) > 80 {
				return description[:77] + "..."
			}
			return description
		}
	}

	// Try to extract a meaningful title from the URL
	if link != "" {
		if u, err := url.Parse(link); err == nil {
			path := strings.TrimPrefix(u.Path, "/")
			// Remove file extensions and URL encoding
			path = strings.TrimSuffix(path, "/")
			if lastSlash := strings.LastIndex(path, "/"); lastSlash != -1 {
				path = path[lastSlash+1:]
			}

			// Skip if path looks like a numeric ID (common in Mastodon, social media)
			isNumeric := true
			for _, r := range path {
				if r < '0' || r > '9' {
					isNumeric = false
					break
				}
			}
			if isNumeric {
				return "Untitled Article"
			}

			// Clean up the path to make it more readable
			path = strings.ReplaceAll(path, "-", " ")
			path = strings.ReplaceAll(path, "_", " ")
			caser := cases.Title(language.English)
			path = caser.String(strings.ToLower(path))
			if len(path) > 5 && len(path) < 100 {
				return path
			}
		}
	}

	return "Untitled Article"
}

func (fs *FeedService) stripHTMLTags(s string) string {
	// Simple HTML tag stripper - removes content between < and >
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	// Decode common HTML entities
	text := result.String()
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&apos;", "'")
	return text
}

func (fs *FeedService) cleanTitle(title string) string {
	// Remove HTML tags if any
	title = strings.ReplaceAll(title, "<", "&lt;")
	title = strings.ReplaceAll(title, ">", "&gt;")

	// Normalize whitespace
	title = strings.Join(strings.Fields(title), " ")

	// Limit length to reasonable bounds
	if len(title) > 200 {
		title = title[:197] + "..."
	}

	return title
}

func (fs *FeedService) preprocessXMLForMediaConflicts(body []byte) []byte {
	content := string(body)

	// Replace media:title and media:description tags to prevent namespace conflicts
	// This prevents Go's XML parser from accidentally binding to media tags instead of regular tags
	content = strings.ReplaceAll(content, "<media:title>", "<media-title>")
	content = strings.ReplaceAll(content, "</media:title>", "</media-title>")
	content = strings.ReplaceAll(content, "<media:description", "<media-description")
	content = strings.ReplaceAll(content, "</media:description>", "</media-description>")

	return []byte(content)
}

// GetCacheStats returns statistics from both the unread and feed list caches.
func (fs *FeedService) GetCacheStats() (unread cache.CacheStats, feedList cache.FeedListCacheStats) {
	return fs.unreadCache.GetStats(), fs.feedListCache.GetStats()
}

func (fs *FeedService) convertToUTF8(body []byte) ([]byte, error) {
	// Check if content declares a specific encoding
	content := string(body)
	if strings.Contains(content, "encoding=\"ISO-8859-1\"") || strings.Contains(content, "encoding='ISO-8859-1'") {
		// Convert from ISO-8859-1 to UTF-8
		decoder := charmap.ISO8859_1.NewDecoder()
		utf8Content, err := decoder.Bytes(body)
		if err != nil {
			return nil, fmt.Errorf("failed to convert ISO-8859-1 to UTF-8: %w", err)
		}

		// Replace the encoding declaration in the XML
		utf8String := string(utf8Content)
		utf8String = strings.Replace(utf8String, "encoding=\"ISO-8859-1\"", "encoding=\"UTF-8\"", 1)
		utf8String = strings.Replace(utf8String, "encoding='ISO-8859-1'", "encoding='UTF-8'", 1)

		return []byte(utf8String), nil
	}

	// If no special encoding, return as-is
	return body, nil
}
