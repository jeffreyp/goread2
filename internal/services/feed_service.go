package services

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"goread2/internal/database"
)

type FeedService struct {
	db          database.Database
	rateLimiter *DomainRateLimiter
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

// Unified feed data structure
type FeedData struct {
	Title       string
	Description string
	Articles    []ArticleData
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
		db:          db,
		rateLimiter: rateLimiter,
	}
}

func (fs *FeedService) AddFeed(url string) (*database.Feed, error) {
	feedData, err := fs.fetchFeed(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}

	feed := &database.Feed{
		Title:       feedData.Title,
		URL:         url,
		Description: feedData.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastFetch:   time.Now(),
	}

	if err := fs.db.AddFeed(feed); err != nil {
		return nil, fmt.Errorf("failed to insert feed: %w", err)
	}

	if err := fs.saveArticlesFromFeed(feed.ID, feedData); err != nil {
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
		result, resultErr = fs.addFeedForUserInternal(userID, inputURL)
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("feed discovery timed out after 15 seconds for %s", inputURL)
	case <-done:
		return result, resultErr
	}
}

func (fs *FeedService) addFeedForUserInternal(userID int, inputURL string) (*database.Feed, error) {
	var feedURL string

	// Normalize the input URL first
	discovery := NewFeedDiscovery()
	normalizedURL, err := discovery.NormalizeURL(inputURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
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
		feedURLs, err := discovery.DiscoverFeedURL(inputURL)
		if err != nil {
			return nil, fmt.Errorf("failed to discover feed: %w", err)
		}

		if len(feedURLs) == 0 {
			return nil, fmt.Errorf("no RSS/Atom feeds found for %s. Please check if the site has feeds or try a direct feed URL", inputURL)
		}

		feedURL = feedURLs[0]
	}

	// First check if feed already exists
	feeds, err := fs.db.GetFeeds()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing feeds: %w", err)
	}

	var existingFeed *database.Feed
	for _, feed := range feeds {
		if feed.URL == feedURL {
			existingFeed = &feed
			break
		}
	}

	if existingFeed == nil {
		// Feed doesn't exist, create it
		feedData, err := fs.fetchFeed(feedURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch feed: %w", err)
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
			return nil, fmt.Errorf("failed to insert feed: %w", err)
		}

		if err := fs.saveArticlesFromFeed(feed.ID, feedData); err != nil {
			return nil, fmt.Errorf("failed to save articles: %w", err)
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
		return nil, fmt.Errorf("failed to subscribe user to feed: %w", err)
	}

	// Mark all articles in this feed as unread for the subscriber synchronously
	// This ensures unread counts are immediately accurate after feed addition
	if err := fs.markExistingArticlesAsUnreadForUser(userID, existingFeed.ID); err != nil {
		log.Printf("Failed to mark existing articles as unread for user %d, feed %d: %v", userID, existingFeed.ID, err)
		// Don't fail the entire operation, just log the error
	}

	return existingFeed, nil
}

func (fs *FeedService) DeleteFeed(id int) error {
	return fs.db.DeleteFeed(id)
}

func (fs *FeedService) UnsubscribeUserFromFeed(userID, feedID int) error {
	return fs.db.UnsubscribeUserFromFeed(userID, feedID)
}

func (fs *FeedService) GetArticles(feedID int) ([]database.Article, error) {
	return fs.db.GetArticles(feedID)
}

func (fs *FeedService) GetAllArticles() ([]database.Article, error) {
	return fs.db.GetAllArticles()
}

func (fs *FeedService) GetUserArticles(userID int) ([]database.Article, error) {
	return fs.db.GetUserArticles(userID)
}

func (fs *FeedService) GetUserArticlesPaginated(userID, limit, offset int) ([]database.Article, error) {
	return fs.db.GetUserArticlesPaginated(userID, limit, offset)
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
	return fs.db.MarkUserArticleRead(userID, articleID, isRead)
}

func (fs *FeedService) ToggleUserArticleStar(userID, articleID int) error {
	return fs.db.ToggleUserArticleStar(userID, articleID)
}

func (fs *FeedService) GetUserUnreadCounts(userID int) (map[int]int, error) {
	return fs.db.GetUserUnreadCounts(userID)
}

func (fs *FeedService) markExistingArticlesAsUnreadForUser(userID, feedID int) error {
	// Get all articles in this feed
	articles, err := fs.db.GetArticles(feedID)
	if err != nil {
		return fmt.Errorf("failed to get articles for feed %d: %w", feedID, err)
	}

	if len(articles) == 0 {
		return nil
	}

	// Use batch insert for better performance
	return fs.db.BatchSetUserArticleStatus(userID, articles, false, false) // unread, unstarred
}

func (fs *FeedService) fetchFeed(url string) (*FeedData, error) {
	// Apply rate limiting if available
	if fs.rateLimiter != nil {
		if !fs.rateLimiter.Allow(url) {
			return nil, fmt.Errorf("rate limited: too many requests to domain")
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoRead/2.0)")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Handle character encoding conversion
	body, err = fs.convertToUTF8(body)
	if err != nil {
		return nil, err
	}

	// Try parsing as RSS 2.0 first
	var rss RSS
	if err := xml.Unmarshal(body, &rss); err == nil && rss.XMLName.Local == "rss" {
		return fs.convertRSSToFeedData(&rss, url), nil
	}

	// Try parsing as RDF/RSS 1.0
	var rdf RDF
	rdfErr := xml.Unmarshal(body, &rdf)
	if rdfErr == nil {
		if rdf.XMLName.Local == "RDF" {
			return fs.convertRDFToFeedData(&rdf, url), nil
		}
	}

	// Try parsing as Atom
	var atom Atom
	if err := xml.Unmarshal(body, &atom); err == nil && atom.XMLName.Local == "feed" {
		return fs.convertAtomToFeedData(&atom, url), nil
	}

	return nil, fmt.Errorf("unsupported feed format or invalid XML")
}

func (fs *FeedService) convertRSSToFeedData(rss *RSS, feedURL string) *FeedData {
	articles := make([]ArticleData, len(rss.Channel.Items))
	for i, item := range rss.Channel.Items {
		publishedAt, _ := time.Parse(time.RFC1123Z, item.PubDate)
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		articles[i] = ArticleData{
			Title:       item.Title,
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
			Title:       item.Title,
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
			Title:       entry.Title,
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

func (fs *FeedService) saveArticlesFromFeed(feedID int, feedData *FeedData) error {
	for _, articleData := range feedData.Articles {
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
			return err
		}
	}

	return nil
}

func (fs *FeedService) RefreshFeeds() error {
	// Get all unique feeds from both global feeds and all user feeds
	globalFeeds, err := fs.GetFeeds()
	if err != nil {
		return err
	}

	// Also get all user feeds to ensure we refresh feeds that users are subscribed to
	allUserFeeds, err := fs.db.GetAllUserFeeds()
	if err != nil {
		allUserFeeds = []database.Feed{}
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

	for _, feed := range feedMap {
		feedData, err := fs.fetchFeed(feed.URL)
		if err != nil {
			continue
		}

		if err := fs.saveArticlesFromFeed(feed.ID, feedData); err != nil {
			continue
		}

		_ = fs.db.UpdateFeedLastFetch(feed.ID, time.Now())
	}

	return nil
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
