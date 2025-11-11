package fixtures

import (
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

// Sample users for testing
var SampleUsers = []database.User{
	{
		GoogleID:  "google_user_1",
		Email:     "user1@example.com",
		Name:      "Test User 1",
		Avatar:    "https://example.com/avatar1.jpg",
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	{
		GoogleID:  "google_user_2",
		Email:     "user2@example.com",
		Name:      "Test User 2",
		Avatar:    "https://example.com/avatar2.jpg",
		CreatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
	},
	{
		GoogleID:  "google_admin",
		Email:     "admin@example.com",
		Name:      "Admin User",
		Avatar:    "https://example.com/admin_avatar.jpg",
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	},
}

// Sample feeds for testing
var SampleFeeds = []database.Feed{
	{
		Title:       "Tech News",
		URL:         "https://technews.com/rss.xml",
		Description: "Latest technology news and updates",
		CreatedAt:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		LastFetch:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	},
	{
		Title:       "Science Daily",
		URL:         "https://sciencedaily.com/feed.xml",
		Description: "Breaking science news and research updates",
		CreatedAt:   time.Date(2023, 1, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2023, 1, 2, 10, 0, 0, 0, time.UTC),
		LastFetch:   time.Date(2023, 1, 2, 10, 0, 0, 0, time.UTC),
	},
	{
		Title:       "Programming Blog",
		URL:         "https://programming.blog/atom.xml",
		Description: "Programming tutorials and best practices",
		CreatedAt:   time.Date(2023, 1, 3, 9, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2023, 1, 3, 9, 0, 0, 0, time.UTC),
		LastFetch:   time.Date(2023, 1, 3, 9, 0, 0, 0, time.UTC),
	},
}

// Sample articles for testing
var SampleArticles = []database.Article{
	{
		FeedID:      1, // Tech News
		Title:       "AI Breakthrough in Natural Language Processing",
		URL:         "https://technews.com/ai-breakthrough-2023",
		Content:     "Scientists have achieved a major breakthrough in natural language processing...",
		Description: "A comprehensive look at the latest AI developments",
		Author:      "Dr. Jane Smith",
		PublishedAt: time.Date(2023, 1, 1, 14, 30, 0, 0, time.UTC),
		CreatedAt:   time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC),
	},
	{
		FeedID:      1, // Tech News
		Title:       "Quantum Computing Reaches New Milestone",
		URL:         "https://technews.com/quantum-computing-milestone",
		Content:     "Researchers have demonstrated a quantum computer that can...",
		Description: "Quantum computing advances continue to accelerate",
		Author:      "Prof. John Doe",
		PublishedAt: time.Date(2023, 1, 2, 9, 15, 0, 0, time.UTC),
		CreatedAt:   time.Date(2023, 1, 2, 9, 30, 0, 0, time.UTC),
	},
	{
		FeedID:      2, // Science Daily
		Title:       "New Species of Deep-Sea Fish Discovered",
		URL:         "https://sciencedaily.com/new-fish-species-2023",
		Content:     "Marine biologists have discovered a previously unknown species...",
		Description: "Biodiversity in the deep ocean continues to surprise scientists",
		Author:      "Dr. Sarah Ocean",
		PublishedAt: time.Date(2023, 1, 2, 11, 45, 0, 0, time.UTC),
		CreatedAt:   time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
	},
	{
		FeedID:      3, // Programming Blog
		Title:       "Best Practices for Go Testing",
		URL:         "https://programming.blog/go-testing-best-practices",
		Content:     "Testing is a crucial part of software development. In this post...",
		Description: "Learn how to write effective tests in Go",
		Author:      "Alice Developer",
		PublishedAt: time.Date(2023, 1, 3, 10, 0, 0, 0, time.UTC),
		CreatedAt:   time.Date(2023, 1, 3, 10, 15, 0, 0, time.UTC),
	},
	{
		FeedID:      3, // Programming Blog
		Title:       "Microservices Architecture Patterns",
		URL:         "https://programming.blog/microservices-patterns",
		Content:     "Microservices have become a popular architectural pattern...",
		Description: "Exploring common patterns in microservices architecture",
		Author:      "Bob Architect",
		PublishedAt: time.Date(2023, 1, 3, 16, 30, 0, 0, time.UTC),
		CreatedAt:   time.Date(2023, 1, 3, 16, 45, 0, 0, time.UTC),
	},
}

// Sample RSS XML for testing feed parsing
const SampleRSSXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test RSS Feed</title>
    <description>A sample RSS feed for testing</description>
    <link>https://test.com</link>
    <item>
      <title>Sample Article 1</title>
      <link>https://test.com/article1</link>
      <description>This is a sample article for testing</description>
      <author>test@test.com (Test Author)</author>
      <pubDate>Mon, 01 Jan 2023 12:00:00 GMT</pubDate>
      <content:encoded><![CDATA[<p>Full content of the article goes here.</p>]]></content:encoded>
    </item>
    <item>
      <title>Sample Article 2</title>
      <link>https://test.com/article2</link>
      <description>Another sample article for testing</description>
      <author>test@test.com (Test Author)</author>
      <pubDate>Tue, 02 Jan 2023 10:30:00 GMT</pubDate>
      <content:encoded><![CDATA[<p>More full content here.</p>]]></content:encoded>
    </item>
  </channel>
</rss>`

// Sample Atom XML for testing feed parsing
const SampleAtomXML = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <subtitle>A sample Atom feed for testing</subtitle>
  <link href="https://test.com/atom.xml" rel="self"/>
  <updated>2023-01-01T12:00:00Z</updated>
  
  <entry>
    <title>Sample Atom Entry 1</title>
    <link href="https://test.com/atom-entry1"/>
    <id>https://test.com/atom-entry1</id>
    <published>2023-01-01T12:00:00Z</published>
    <updated>2023-01-01T12:00:00Z</updated>
    <summary>This is a sample Atom entry for testing</summary>
    <author>
      <name>Test Author</name>
    </author>
    <content type="html">
      <![CDATA[<p>Full content of the Atom entry goes here.</p>]]>
    </content>
  </entry>
  
  <entry>
    <title>Sample Atom Entry 2</title>
    <link href="https://test.com/atom-entry2"/>
    <id>https://test.com/atom-entry2</id>
    <published>2023-01-02T10:30:00Z</published>
    <updated>2023-01-02T10:30:00Z</updated>
    <summary>Another sample Atom entry for testing</summary>
    <author>
      <name>Test Author</name>
    </author>
    <content type="html">
      <![CDATA[<p>More full content here.</p>]]>
    </content>
  </entry>
</feed>`
