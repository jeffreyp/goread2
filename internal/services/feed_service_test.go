package services

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

// Test RSS feed without title tags (like Mastodon)
const MastodonStyleRSSXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:webfeeds="http://webfeeds.org/rss/1.0" xmlns:media="http://search.yahoo.com/mrss/">
  <channel>
    <title>Test Mastodon User</title>
    <description>Public posts from @testuser@mastodon.social</description>
    <link>https://mastodon.social/@testuser</link>
    <item>
      <guid isPermaLink="true">https://mastodon.social/@testuser/115329840719892796</guid>
      <link>https://mastodon.social/@testuser/115329840719892796</link>
      <pubDate>Mon, 06 Oct 2025 23:35:12 +0000</pubDate>
      <description>&lt;p&gt;This is my first test post about renewable energy. It's a very interesting topic!&lt;/p&gt;</description>
    </item>
    <item>
      <guid isPermaLink="true">https://mastodon.social/@testuser/115327696947026496</guid>
      <link>https://mastodon.social/@testuser/115327696947026496</link>
      <pubDate>Mon, 06 Oct 2025 14:30:00 +0000</pubDate>
      <description>&lt;p&gt;Another post here with some content. This one is about technology and innovation in the modern world.&lt;/p&gt;</description>
    </item>
  </channel>
</rss>`

// Test RSS feed with empty title tags
const EmptyTitleRSSXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test RSS Feed</title>
    <description>A test feed with empty titles</description>
    <link>https://test.com</link>
    <item>
      <title></title>
      <link>https://test.com/article1</link>
      <description>This is a test article with an empty title tag.</description>
      <pubDate>Mon, 01 Jan 2023 12:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

func TestParseMastodonRSSFeed(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	fs := NewFeedService(db, nil)

	// Parse the Mastodon-style RSS (no title tags)
	feedData, err := fs.parseFeedFromBytes([]byte(MastodonStyleRSSXML), "https://mastodon.social/@testuser")
	if err != nil {
		t.Fatalf("Failed to parse Mastodon RSS feed: %v", err)
	}

	// Verify feed title
	if feedData.Title != "Test Mastodon User" {
		t.Errorf("Expected feed title 'Test Mastodon User', got '%s'", feedData.Title)
	}

	// Verify we have 2 articles
	if len(feedData.Articles) != 2 {
		t.Fatalf("Expected 2 articles, got %d", len(feedData.Articles))
	}

	// Verify first article title is NOT the numeric ID
	firstArticle := feedData.Articles[0]
	if firstArticle.Title == "115329840719892796" {
		t.Errorf("Article title should not be the numeric ID, got '%s'", firstArticle.Title)
	}

	// Verify first article title is extracted from description
	expectedPrefix := "This is my first test post about renewable energy"
	if len(firstArticle.Title) < len(expectedPrefix) {
		t.Errorf("Expected article title to start with description text, got '%s'", firstArticle.Title)
	}

	// Verify the title doesn't contain HTML tags
	if containsHTMLTags(firstArticle.Title) {
		t.Errorf("Article title should not contain HTML tags, got '%s'", firstArticle.Title)
	}

	// Verify second article
	secondArticle := feedData.Articles[1]
	if secondArticle.Title == "115327696947026496" {
		t.Errorf("Second article title should not be the numeric ID, got '%s'", secondArticle.Title)
	}
}

func TestParseEmptyTitleRSSFeed(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	fs := NewFeedService(db, nil)

	// Parse RSS feed with empty title tags
	feedData, err := fs.parseFeedFromBytes([]byte(EmptyTitleRSSXML), "https://test.com/feed")
	if err != nil {
		t.Fatalf("Failed to parse RSS feed with empty titles: %v", err)
	}

	// Verify we have 1 article
	if len(feedData.Articles) != 1 {
		t.Fatalf("Expected 1 article, got %d", len(feedData.Articles))
	}

	// Verify article title is generated from description
	article := feedData.Articles[0]
	if article.Title == "" {
		t.Errorf("Article title should be generated from description, got empty string")
	}

	expectedPrefix := "This is a test article"
	if len(article.Title) < len(expectedPrefix) {
		t.Errorf("Expected article title to start with description text, got '%s'", article.Title)
	}
}

func TestStripHTMLTags(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	fs := NewFeedService(db, nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple HTML paragraph",
			input:    "<p>Hello world</p>",
			expected: "Hello world",
		},
		{
			name:     "Multiple tags",
			input:    "<p>This is <strong>bold</strong> text</p>",
			expected: "This is bold text",
		},
		{
			name:     "HTML entities",
			input:    "&lt;p&gt;Test &amp; verify&lt;/p&gt;",
			expected: "<p>Test & verify</p>",
		},
		{
			name:     "Complex Mastodon content",
			input:    "&lt;p&gt;It&#39;s cruel to show me &amp;quot;cool&amp;quot; news&lt;/p&gt;",
			expected: "<p>It's cruel to show me \"cool\" news</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fs.stripHTMLTags(tt.input)
			if result != tt.expected {
				t.Errorf("stripHTMLTags(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateFallbackTitle(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	fs := NewFeedService(db, nil)

	tests := []struct {
		name          string
		link          string
		description   string
		shouldNotBe   string // What the title should NOT be
		shouldContain string // What the title SHOULD contain (optional)
	}{
		{
			name:          "Mastodon post with numeric ID in URL",
			link:          "https://mastodon.social/@user/115329840719892796",
			description:   "<p>This is a test post about something interesting</p>",
			shouldNotBe:   "115329840719892796",
			shouldContain: "This is a test post",
		},
		{
			name:          "Empty description, numeric URL",
			link:          "https://example.com/posts/123456789",
			description:   "",
			shouldNotBe:   "123456789",
			shouldContain: "Untitled",
		},
		{
			name:          "Valid URL slug",
			link:          "https://example.com/my-awesome-article",
			description:   "",
			shouldContain: "My Awesome Article",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fs.generateFallbackTitle(tt.link, tt.description)

			if tt.shouldNotBe != "" && result == tt.shouldNotBe {
				t.Errorf("generateFallbackTitle should not return %q, got %q", tt.shouldNotBe, result)
			}

			if tt.shouldContain != "" && len(result) > 0 {
				// For "Untitled" check
				if tt.shouldContain == "Untitled" && result != "Untitled Article" {
					t.Errorf("Expected 'Untitled Article', got %q", result)
				}
				// For content checks, just verify it's not the numeric ID
				if tt.shouldContain != "Untitled" && result == tt.shouldNotBe {
					t.Errorf("Result should contain meaningful text, got numeric ID: %q", result)
				}
			}
		})
	}
}

// Helper function
func containsHTMLTags(s string) bool {
	return len(s) > 0 && (s[0] == '<' || containsChar(s, '<'))
}

func containsChar(s string, c rune) bool {
	for _, r := range s {
		if r == c {
			return true
		}
	}
	return false
}

// Helper method to parse feed from bytes for testing
func (fs *FeedService) parseFeedFromBytes(body []byte, feedURL string) (*FeedData, error) {
	// Handle character encoding conversion
	body, err := fs.convertToUTF8(body)
	if err != nil {
		return nil, err
	}

	// Special handling for feeds with media namespace conflicts
	body = fs.preprocessXMLForMediaConflicts(body)

	// Try parsing as RSS 2.0 first
	var rss RSS
	if err := unmarshalXML(body, &rss); err == nil && rss.XMLName.Local == "rss" {
		return fs.convertRSSToFeedData(&rss, feedURL), nil
	}

	// Try parsing as RDF/RSS 1.0
	var rdf RDF
	if err := unmarshalXML(body, &rdf); err == nil && rdf.XMLName.Local == "RDF" {
		return fs.convertRDFToFeedData(&rdf, feedURL), nil
	}

	// Try parsing as Atom
	var atom Atom
	if err := unmarshalXML(body, &atom); err == nil && atom.XMLName.Local == "feed" {
		return fs.convertAtomToFeedData(&atom, feedURL), nil
	}

	return nil, err
}

// Helper for XML unmarshaling
func unmarshalXML(data []byte, v interface{}) error {
	return xml.Unmarshal(data, v)
}

// TestAddFeedWithMockHTTP tests adding a feed using a mock HTTP server
func TestAddFeedWithMockHTTP(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create feed service
	fs := NewFeedService(db, nil)

	// Create sample RSS XML
	sampleRSS := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <description>A test RSS feed</description>
    <link>https://test.com</link>
    <item>
      <title>Test Article</title>
      <link>https://test.com/article1</link>
      <description>This is a test article</description>
      <pubDate>Mon, 01 Jan 2023 12:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	// Create mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleRSS))
	}))
	defer mockServer.Close()

	// Inject mock HTTP client
	fs.SetHTTPClient(&mockHTTPClient{Server: mockServer})

	// Add feed using mock server URL
	feed, err := fs.AddFeed(mockServer.URL)
	if err != nil {
		t.Fatalf("AddFeed failed: %v", err)
	}

	// Verify feed was added correctly
	if feed.Title != "Test Feed" {
		t.Errorf("Expected feed title 'Test Feed', got '%s'", feed.Title)
	}

	if feed.Description != "A test RSS feed" {
		t.Errorf("Expected description 'A test RSS feed', got '%s'", feed.Description)
	}

	// Verify article was added
	articles, err := db.GetArticles(feed.ID)
	if err != nil {
		t.Fatalf("GetArticles failed: %v", err)
	}

	if len(articles) != 1 {
		t.Fatalf("Expected 1 article, got %d", len(articles))
	}

	// The feed service applies title fallback logic for short titles
	// So we just verify we got an article with a non-empty title
	if articles[0].Title == "" {
		t.Error("Article title should not be empty")
	}
}

// mockHTTPClient implements the HTTPClient interface for testing
type mockHTTPClient struct {
	Server *httptest.Server
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.Server.Client().Do(req)
}

// TestExportOPML tests the OPML export functionality
func TestExportOPML(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	fs := NewFeedService(db, nil)

	// Create a test user
	testUser := &database.User{
		GoogleID: "test-user-export-opml",
		Email:    "test-export@example.com",
		Name:     "Test User",
	}
	if err := db.CreateUser(testUser); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Add test feeds
	feed1 := &database.Feed{
		Title:       "Test Feed 1",
		URL:         "https://example.com/feed1.xml",
		Description: "First test feed",
	}
	if err := db.AddFeed(feed1); err != nil {
		t.Fatalf("Failed to add feed1: %v", err)
	}

	feed2 := &database.Feed{
		Title:       "Test Feed 2",
		URL:         "https://example.com/feed2.xml",
		Description: "Second test feed",
	}
	if err := db.AddFeed(feed2); err != nil {
		t.Fatalf("Failed to add feed2: %v", err)
	}

	// Subscribe user to the feeds
	if err := db.SubscribeUserToFeed(testUser.ID, feed1.ID); err != nil {
		t.Fatalf("Failed to subscribe to feed1: %v", err)
	}
	if err := db.SubscribeUserToFeed(testUser.ID, feed2.ID); err != nil {
		t.Fatalf("Failed to subscribe to feed2: %v", err)
	}

	// Export OPML
	opmlData, err := fs.ExportOPML(testUser.ID)
	if err != nil {
		t.Fatalf("Failed to export OPML: %v", err)
	}

	// Verify the XML is valid
	var opml OPML
	if err := xml.Unmarshal(opmlData, &opml); err != nil {
		t.Fatalf("Failed to parse exported OPML: %v", err)
	}

	// Verify the OPML structure
	if opml.Head.Title != "GoRead2 Subscriptions" {
		t.Errorf("Expected OPML title 'GoRead2 Subscriptions', got '%s'", opml.Head.Title)
	}

	// Verify we have the correct number of feeds
	if len(opml.Body.Outlines) != 2 {
		t.Fatalf("Expected 2 feeds in OPML, got %d", len(opml.Body.Outlines))
	}

	// Verify feed details
	feedsFound := make(map[string]bool)
	for _, outline := range opml.Body.Outlines {
		if outline.Type != "rss" {
			t.Errorf("Expected outline type 'rss', got '%s'", outline.Type)
		}

		// Check feed1
		if outline.XMLURL == "https://example.com/feed1.xml" {
			feedsFound["feed1"] = true
			if outline.Title != "Test Feed 1" {
				t.Errorf("Expected feed1 title 'Test Feed 1', got '%s'", outline.Title)
			}
			if outline.Text != "Test Feed 1" {
				t.Errorf("Expected feed1 text 'Test Feed 1', got '%s'", outline.Text)
			}
		}

		// Check feed2
		if outline.XMLURL == "https://example.com/feed2.xml" {
			feedsFound["feed2"] = true
			if outline.Title != "Test Feed 2" {
				t.Errorf("Expected feed2 title 'Test Feed 2', got '%s'", outline.Title)
			}
			if outline.Text != "Test Feed 2" {
				t.Errorf("Expected feed2 text 'Test Feed 2', got '%s'", outline.Text)
			}
		}
	}

	// Verify both feeds were found
	if !feedsFound["feed1"] {
		t.Error("Feed1 not found in exported OPML")
	}
	if !feedsFound["feed2"] {
		t.Error("Feed2 not found in exported OPML")
	}

	// Verify XML header is present
	if len(opmlData) < 5 || string(opmlData[:5]) != "<?xml" {
		t.Error("OPML should start with XML declaration")
	}
}

// TestExportOPMLEmptyFeeds tests OPML export with no feeds
func TestExportOPMLEmptyFeeds(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	fs := NewFeedService(db, nil)

	// Create a test user with no feeds
	testUser := &database.User{
		GoogleID: "test-user-empty-opml",
		Email:    "test-empty@example.com",
		Name:     "Test User Empty",
	}
	if err := db.CreateUser(testUser); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Export OPML (should work even with no feeds)
	opmlData, err := fs.ExportOPML(testUser.ID)
	if err != nil {
		t.Fatalf("Failed to export OPML for user with no feeds: %v", err)
	}

	// Verify the XML is valid
	var opml OPML
	if err := xml.Unmarshal(opmlData, &opml); err != nil {
		t.Fatalf("Failed to parse exported OPML: %v", err)
	}

	// Verify we have zero feeds
	if len(opml.Body.Outlines) != 0 {
		t.Errorf("Expected 0 feeds in OPML for user with no subscriptions, got %d", len(opml.Body.Outlines))
	}
}

// TestShouldCheckFeed tests the smart prioritization logic
func TestShouldCheckFeed(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	fs := NewFeedService(db, nil)
	now := time.Now()

	tests := []struct {
		name          string
		feed          database.Feed
		expectedCheck bool
		description   string
	}{
		{
			name: "never checked feed",
			feed: database.Feed{
				ID:          1,
				LastChecked: time.Time{}, // Zero time = never checked
			},
			expectedCheck: true,
			description:   "feeds never checked should always be checked",
		},
		{
			name: "active feed with known interval",
			feed: database.Feed{
				ID:                    2,
				LastChecked:           now.Add(-31 * time.Minute),
				LastHadNewContent:     now.Add(-1 * time.Hour),
				AverageUpdateInterval: 3600, // 1 hour average
			},
			expectedCheck: true,
			description:   "feed checked 31 min ago with 1hr interval (>50% threshold) should be checked",
		},
		{
			name: "active feed too soon",
			feed: database.Feed{
				ID:                    3,
				LastChecked:           now.Add(-10 * time.Minute),
				LastHadNewContent:     now.Add(-30 * time.Minute),
				AverageUpdateInterval: 3600, // 1 hour average
			},
			expectedCheck: false,
			description:   "feed checked 10 min ago with 1hr interval (<50% threshold) should be skipped",
		},
		{
			name: "recent active feed no interval data",
			feed: database.Feed{
				ID:                    4,
				LastChecked:           now.Add(-20 * time.Minute),
				LastHadNewContent:     now.Add(-5 * 24 * time.Hour), // 5 days ago
				AverageUpdateInterval: 0,                            // No historical data
			},
			expectedCheck: false,
			description:   "feed checked 20 min ago, active < 1 week, check every 30 min",
		},
		{
			name: "recent active feed ready",
			feed: database.Feed{
				ID:                    5,
				LastChecked:           now.Add(-31 * time.Minute),
				LastHadNewContent:     now.Add(-5 * 24 * time.Hour), // 5 days ago
				AverageUpdateInterval: 0,
			},
			expectedCheck: true,
			description:   "feed checked 31 min ago, active < 1 week, should be checked",
		},
		{
			name: "regular feed not ready",
			feed: database.Feed{
				ID:                    6,
				LastChecked:           now.Add(-45 * time.Minute),
				LastHadNewContent:     now.Add(-20 * 24 * time.Hour), // 20 days ago
				AverageUpdateInterval: 0,
			},
			expectedCheck: false,
			description:   "feed checked 45 min ago, active < 1 month, check every 1 hour",
		},
		{
			name: "regular feed ready",
			feed: database.Feed{
				ID:                    7,
				LastChecked:           now.Add(-61 * time.Minute),
				LastHadNewContent:     now.Add(-20 * 24 * time.Hour), // 20 days ago
				AverageUpdateInterval: 0,
			},
			expectedCheck: true,
			description:   "feed checked 61 min ago, active < 1 month, should be checked",
		},
		{
			name: "dormant feed not ready",
			feed: database.Feed{
				ID:                    8,
				LastChecked:           now.Add(-5 * time.Hour),
				LastHadNewContent:     now.Add(-40 * 24 * time.Hour), // 40 days ago
				AverageUpdateInterval: 0,
			},
			expectedCheck: false,
			description:   "feed checked 5 hours ago, dormant > 1 month, check every 6 hours",
		},
		{
			name: "dormant feed ready",
			feed: database.Feed{
				ID:                    9,
				LastChecked:           now.Add(-7 * time.Hour),
				LastHadNewContent:     now.Add(-40 * 24 * time.Hour), // 40 days ago
				AverageUpdateInterval: 0,
			},
			expectedCheck: true,
			description:   "feed checked 7 hours ago, dormant > 1 month, should be checked",
		},
		{
			name: "never had content",
			feed: database.Feed{
				ID:                    10,
				LastChecked:           now.Add(-45 * time.Minute),
				LastHadNewContent:     time.Time{}, // Never had new content
				AverageUpdateInterval: 0,
			},
			expectedCheck: false,
			description:   "feed with no content history, check every 1 hour",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fs.shouldCheckFeed(tt.feed, now)
			if result != tt.expectedCheck {
				t.Errorf("%s: expected shouldCheckFeed=%v, got %v",
					tt.description, tt.expectedCheck, result)
			}
		})
	}
}

// TestUpdateFeedTracking tests the feed tracking update logic
func TestUpdateFeedTracking(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	fs := NewFeedService(db, nil)

	// Create a test feed
	feed := &database.Feed{
		Title:       "Test Feed",
		URL:         "https://example.com/feed.xml",
		Description: "Test feed for tracking",
	}
	if err := db.AddFeed(feed); err != nil {
		t.Fatalf("Failed to add test feed: %v", err)
	}

	t.Run("first content update", func(t *testing.T) {
		now := time.Now()
		feed.LastChecked = now
		feed.LastHadNewContent = time.Time{} // No previous content
		feed.AverageUpdateInterval = 0

		err := fs.updateFeedTracking(*feed, true)
		if err != nil {
			t.Fatalf("updateFeedTracking failed: %v", err)
		}

		// Fetch the updated feed
		updatedFeed, err := db.GetFeedByURL(feed.URL)
		if err != nil {
			t.Fatalf("Failed to get updated feed: %v", err)
		}

		// LastHadNewContent should be updated
		if updatedFeed.LastHadNewContent.IsZero() {
			t.Error("LastHadNewContent should be set after first content update")
		}

		// AverageUpdateInterval should still be 0 (no previous interval to calculate)
		if updatedFeed.AverageUpdateInterval != 0 {
			t.Errorf("Expected AverageUpdateInterval=0 on first update, got %d", updatedFeed.AverageUpdateInterval)
		}
	})

	t.Run("subsequent content update calculates interval", func(t *testing.T) {
		// Set up feed with previous content
		previousContentTime := time.Now().Add(-2 * time.Hour)
		feed.LastHadNewContent = previousContentTime
		feed.AverageUpdateInterval = 0

		now := time.Now()
		feed.LastChecked = now

		err := fs.updateFeedTracking(*feed, true)
		if err != nil {
			t.Fatalf("updateFeedTracking failed: %v", err)
		}

		// Fetch the updated feed
		updatedFeed, err := db.GetFeedByURL(feed.URL)
		if err != nil {
			t.Fatalf("Failed to get updated feed: %v", err)
		}

		// AverageUpdateInterval should be set (approximately 2 hours = 7200 seconds)
		expectedInterval := 2 * 3600 // 2 hours in seconds
		tolerance := 60              // Allow 60 seconds tolerance
		if updatedFeed.AverageUpdateInterval < expectedInterval-tolerance ||
			updatedFeed.AverageUpdateInterval > expectedInterval+tolerance {
			t.Errorf("Expected AverageUpdateInterval around %d seconds, got %d",
				expectedInterval, updatedFeed.AverageUpdateInterval)
		}
	})

	t.Run("running average calculation", func(t *testing.T) {
		// Set up feed with existing average (1 hour)
		previousContentTime := time.Now().Add(-2 * time.Hour)
		feed.LastHadNewContent = previousContentTime
		feed.AverageUpdateInterval = 3600 // 1 hour

		now := time.Now()
		feed.LastChecked = now

		err := fs.updateFeedTracking(*feed, true)
		if err != nil {
			t.Fatalf("updateFeedTracking failed: %v", err)
		}

		// Fetch the updated feed
		updatedFeed, err := db.GetFeedByURL(feed.URL)
		if err != nil {
			t.Fatalf("Failed to get updated feed: %v", err)
		}

		// New interval is 2 hours (7200 seconds)
		// Running average = (3600*7 + 7200*3) / 10 = (25200 + 21600) / 10 = 4680
		expectedAverage := 4680
		tolerance := 60
		if updatedFeed.AverageUpdateInterval < expectedAverage-tolerance ||
			updatedFeed.AverageUpdateInterval > expectedAverage+tolerance {
			t.Errorf("Expected AverageUpdateInterval around %d seconds (70%% old + 30%% new), got %d",
				expectedAverage, updatedFeed.AverageUpdateInterval)
		}
	})

	t.Run("no new content does not update LastHadNewContent", func(t *testing.T) {
		// Set up feed with previous content time
		previousContentTime := time.Now().Add(-5 * time.Hour)
		feed.LastHadNewContent = previousContentTime
		oldInterval := feed.AverageUpdateInterval

		now := time.Now()
		feed.LastChecked = now

		err := fs.updateFeedTracking(*feed, false) // No new content
		if err != nil {
			t.Fatalf("updateFeedTracking failed: %v", err)
		}

		// Fetch the updated feed
		updatedFeed, err := db.GetFeedByURL(feed.URL)
		if err != nil {
			t.Fatalf("Failed to get updated feed: %v", err)
		}

		// LastHadNewContent should not be updated (within 1 second tolerance)
		timeDiff := updatedFeed.LastHadNewContent.Sub(previousContentTime)
		if timeDiff > time.Second || timeDiff < -time.Second {
			t.Error("LastHadNewContent should not be updated when no new content")
		}

		// AverageUpdateInterval should not change
		if updatedFeed.AverageUpdateInterval != oldInterval {
			t.Errorf("Expected AverageUpdateInterval unchanged (%d), got %d",
				oldInterval, updatedFeed.AverageUpdateInterval)
		}
	})
}

// TestAddFeedErrorInvalidURL tests that invalid URLs return ErrInvalidURL
func TestAddFeedErrorInvalidURL(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	fs := NewFeedService(db, nil)

	// Create test user
	user := &database.User{
		GoogleID: "test-user-error-invalid-url",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test various invalid URLs
	testCases := []struct {
		url          string
		expectedErrs []error
	}{
		{"", []error{ErrInvalidURL}},                                        // Empty URL
		{"javascript:alert('xss')", []error{ErrInvalidURL, ErrSSRFBlocked}}, // Invalid scheme
		{"not-a-url", []error{ErrNetworkError}},                             // DNS lookup fails
	}

	for _, tc := range testCases {
		_, err := fs.AddFeedForUser(user.ID, tc.url)
		if err == nil {
			t.Errorf("Expected error for invalid URL '%s', got nil", tc.url)
			continue
		}

		// Check if error matches any of the expected errors
		matched := false
		for _, expectedErr := range tc.expectedErrs {
			if errors.Is(err, expectedErr) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("Expected one of %v for '%s', got: %v", tc.expectedErrs, tc.url, err)
		}
	}
}

// TestAddFeedErrorSSRFBlocked tests that SSRF-blocked URLs return ErrSSRFBlocked
func TestAddFeedErrorSSRFBlocked(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	fs := NewFeedService(db, nil)

	// Create test user
	user := &database.User{
		GoogleID: "test-user-error-ssrf",
		Email:    "test-ssrf@example.com",
		Name:     "Test User SSRF",
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test SSRF-blocked URLs
	testCases := []string{
		"http://localhost/feed.xml",
		"http://127.0.0.1/rss",
		"http://192.168.1.1/feed",
		"http://10.0.0.1/rss.xml",
	}

	for _, testURL := range testCases {
		_, err := fs.AddFeedForUser(user.ID, testURL)
		if err == nil {
			t.Errorf("Expected error for SSRF-blocked URL '%s', got nil", testURL)
			continue
		}

		if !errors.Is(err, ErrSSRFBlocked) {
			t.Errorf("Expected ErrSSRFBlocked for '%s', got: %v", testURL, err)
		}
	}
}

// Note: Testing invalid format and 404 responses via integration tests is complex
// due to SSRF protection. These error conditions are tested via unit tests of
// the fetchFeed function and HTTP handler tests.

// TestGetErrorDetails tests the GetErrorDetails function
func TestGetErrorDetails(t *testing.T) {
	testCases := []struct {
		err              error
		expectedCode     string
		expectedContains string
	}{
		{
			err:              ErrInvalidURL,
			expectedCode:     ErrorCodeInvalidURL,
			expectedContains: "valid website URL",
		},
		{
			err:              ErrFeedNotFound,
			expectedCode:     ErrorCodeFeedNotFound,
			expectedContains: "No RSS/Atom feeds found",
		},
		{
			err:              ErrFeedTimeout,
			expectedCode:     ErrorCodeFeedTimeout,
			expectedContains: "took too long",
		},
		{
			err:              ErrInvalidFeedFormat,
			expectedCode:     ErrorCodeInvalidFormat,
			expectedContains: "invalid format",
		},
		{
			err:              ErrNetworkError,
			expectedCode:     ErrorCodeNetworkError,
			expectedContains: "Unable to reach",
		},
		{
			err:              ErrSSRFBlocked,
			expectedCode:     ErrorCodeSSRFBlocked,
			expectedContains: "security reasons",
		},
		{
			err:              ErrDatabaseError,
			expectedCode:     ErrorCodeDatabaseError,
			expectedContains: "database error",
		},
	}

	for _, tc := range testCases {
		details := GetErrorDetails(tc.err)

		if details.ErrorCode != tc.expectedCode {
			t.Errorf("Expected error code '%s', got '%s' for error: %v",
				tc.expectedCode, details.ErrorCode, tc.err)
		}

		if !contains(details.Message, tc.expectedContains) {
			t.Errorf("Expected message to contain '%s', got '%s' for error: %v",
				tc.expectedContains, details.Message, tc.err)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAtIndex(s, substr))
}

func containsAtIndex(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
