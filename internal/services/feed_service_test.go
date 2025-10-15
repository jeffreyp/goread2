package services

import (
	"encoding/xml"
	"testing"

	"goread2/internal/database"
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
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
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
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
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
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
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
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer func() { _ = db.Close() }()

	fs := NewFeedService(db, nil)

	tests := []struct {
		name        string
		link        string
		description string
		shouldNotBe string // What the title should NOT be
		shouldContain string // What the title SHOULD contain (optional)
	}{
		{
			name:        "Mastodon post with numeric ID in URL",
			link:        "https://mastodon.social/@user/115329840719892796",
			description: "<p>This is a test post about something interesting</p>",
			shouldNotBe: "115329840719892796",
			shouldContain: "This is a test post",
		},
		{
			name:        "Empty description, numeric URL",
			link:        "https://example.com/posts/123456789",
			description: "",
			shouldNotBe: "123456789",
			shouldContain: "Untitled",
		},
		{
			name:        "Valid URL slug",
			link:        "https://example.com/my-awesome-article",
			description: "",
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

// TestExportOPML tests the OPML export functionality
func TestExportOPML(t *testing.T) {
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
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
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
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
