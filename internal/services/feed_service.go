package services

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"goread2/internal/database"
)

type FeedService struct {
	db database.Database
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
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

func NewFeedService(db database.Database) *FeedService {
	return &FeedService{db: db}
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

func (fs *FeedService) AddFeedForUser(userID int, url string) (*database.Feed, error) {
	// First check if feed already exists
	feeds, err := fs.db.GetFeeds()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing feeds: %w", err)
	}

	var existingFeed *database.Feed
	for _, feed := range feeds {
		if feed.URL == url {
			existingFeed = &feed
			break
		}
	}

	if existingFeed == nil {
		// Feed doesn't exist, create it
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

		existingFeed = feed
	}

	// Subscribe user to the feed
	if err := fs.db.SubscribeUserToFeed(userID, existingFeed.ID); err != nil {
		return nil, fmt.Errorf("failed to subscribe user to feed: %w", err)
	}

	return existingFeed, nil
}

func (fs *FeedService) DeleteFeed(id int) error {
	return fs.db.DeleteFeed(id)
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

func (fs *FeedService) fetchFeed(url string) (*FeedData, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try parsing as RSS first
	var rss RSS
	if err := xml.Unmarshal(body, &rss); err == nil && rss.XMLName.Local == "rss" {
		return fs.convertRSSToFeedData(&rss), nil
	}

	// Try parsing as Atom
	var atom Atom
	if err := xml.Unmarshal(body, &atom); err == nil && atom.XMLName.Local == "feed" {
		return fs.convertAtomToFeedData(&atom), nil
	}

	return nil, fmt.Errorf("unsupported feed format or invalid XML")
}

func (fs *FeedService) convertRSSToFeedData(rss *RSS) *FeedData {
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
		Title:       rss.Channel.Title,
		Description: rss.Channel.Description,
		Articles:    articles,
	}
}

func (fs *FeedService) convertAtomToFeedData(atom *Atom) *FeedData {
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
		Title:       atom.Title,
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
	feeds, err := fs.GetFeeds()
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		feedData, err := fs.fetchFeed(feed.URL)
		if err != nil {
			continue
		}

		if err := fs.saveArticlesFromFeed(feed.ID, feedData); err != nil {
			continue
		}

		fs.db.UpdateFeedLastFetch(feed.ID, time.Now())
	}

	return nil
}
