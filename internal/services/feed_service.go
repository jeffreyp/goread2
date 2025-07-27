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

func NewFeedService(db database.Database) *FeedService {
	return &FeedService{db: db}
}

func (fs *FeedService) AddFeed(url string) (*database.Feed, error) {
	rss, err := fs.fetchFeed(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}

	feed := &database.Feed{
		Title:       rss.Channel.Title,
		URL:         url,
		Description: rss.Channel.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastFetch:   time.Now(),
	}

	if err := fs.db.AddFeed(feed); err != nil {
		return nil, fmt.Errorf("failed to insert feed: %w", err)
	}

	if err := fs.saveArticles(feed.ID, rss.Channel.Items); err != nil {
		return nil, fmt.Errorf("failed to save articles: %w", err)
	}

	return feed, nil
}

func (fs *FeedService) GetFeeds() ([]database.Feed, error) {
	return fs.db.GetFeeds()
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

func (fs *FeedService) MarkRead(articleID int, isRead bool) error {
	return fs.db.MarkRead(articleID, isRead)
}

func (fs *FeedService) ToggleStar(articleID int) error {
	return fs.db.ToggleStar(articleID)
}

func (fs *FeedService) fetchFeed(url string) (*RSS, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rss RSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, err
	}

	return &rss, nil
}

func (fs *FeedService) saveArticles(feedID int, items []Item) error {
	for _, item := range items {
		publishedAt, _ := time.Parse(time.RFC1123Z, item.PubDate)
		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		article := &database.Article{
			FeedID:      feedID,
			Title:       item.Title,
			URL:         item.Link,
			Content:     item.Content,
			Description: item.Description,
			Author:      item.Author,
			PublishedAt: publishedAt,
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
		rss, err := fs.fetchFeed(feed.URL)
		if err != nil {
			continue
		}

		if err := fs.saveArticles(feed.ID, rss.Channel.Items); err != nil {
			continue
		}

		fs.db.UpdateFeedLastFetch(feed.ID, time.Now())
	}

	return nil
}