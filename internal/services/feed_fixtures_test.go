package services

import (
	"os"
	"path/filepath"
	"testing"
)

// fixturesDir points at the on-disk feed fixtures shared across the test
// suite. These are real-world-shaped RSS/Atom/RDF documents (as opposed to
// the inline XML string constants elsewhere in this package) so format
// quirks are easy to spot by opening the file directly.
const fixturesDir = "../../test/fixtures/feeds"

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixturesDir, name))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return data
}

func TestParseFeedFixtures(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	fs := NewFeedService(db, nil)

	tests := []struct {
		name          string
		fixture       string
		feedURL       string
		expectError   bool
		expectedTitle string
		articleCount  int
		check         func(t *testing.T, fd *FeedData)
	}{
		{
			name:          "RSS 2.0 standard feed",
			fixture:       "rss2_standard.xml",
			feedURL:       "https://example.com/rss.xml",
			expectedTitle: "Example Tech Blog",
			articleCount:  3,
			check: func(t *testing.T, fd *FeedData) {
				if fd.Description != "Technology news and analysis" {
					t.Errorf("feed description = %q", fd.Description)
				}
				a := fd.Articles[0]
				if a.Title != "First Article Title" {
					t.Errorf("article[0] title = %q", a.Title)
				}
				if a.Link != "https://example.com/articles/first-article" {
					t.Errorf("article[0] link = %q", a.Link)
				}
				if a.Author != "Jane Doe" {
					t.Errorf("article[0] author = %q", a.Author)
				}
				if a.PublishedAt.IsZero() {
					t.Errorf("article[0] PublishedAt should be parsed from pubDate, got zero value")
				}
			},
		},
		{
			name:          "Atom 1.0 standard feed",
			fixture:       "atom_standard.xml",
			feedURL:       "https://example.org/atom.xml",
			expectedTitle: "Example Atom Feed",
			articleCount:  2,
			check: func(t *testing.T, fd *FeedData) {
				if fd.Description != "A standard Atom 1.0 test feed" {
					t.Errorf("feed description = %q", fd.Description)
				}
				a := fd.Articles[0]
				if a.Title != "Atom Entry One" {
					t.Errorf("article[0] title = %q", a.Title)
				}
				if a.Link != "https://example.org/entries/one" {
					t.Errorf("article[0] link = %q", a.Link)
				}
				if a.Author != "Alice Author" {
					t.Errorf("article[0] author = %q", a.Author)
				}
				if a.PublishedAt.IsZero() {
					t.Errorf("article[0] PublishedAt should be parsed from <published>, got zero value")
				}
			},
		},
		{
			name:          "RSS 1.0 / RDF standard feed",
			fixture:       "rdf_standard.xml",
			feedURL:       "https://example.net/rdf.xml",
			expectedTitle: "Example RDF Feed",
			articleCount:  2,
			check: func(t *testing.T, fd *FeedData) {
				if fd.Description != "A standard RSS 1.0 (RDF) test feed" {
					t.Errorf("feed description = %q", fd.Description)
				}
				a := fd.Articles[0]
				if a.Title != "RDF Item One" {
					t.Errorf("article[0] title = %q", a.Title)
				}
				if a.Link != "https://example.net/items/one" {
					t.Errorf("article[0] link = %q", a.Link)
				}
				if a.Author != "Carol Creator" {
					t.Errorf("article[0] author (dc:creator) = %q", a.Author)
				}
				if a.PublishedAt.IsZero() {
					t.Errorf("article[0] PublishedAt should be parsed from dc:date, got zero value")
				}
			},
		},
		{
			// The parser copies <link> verbatim into ArticleData.Link (feed_service.go
			// convertRSSToFeedData) — there is no url.Parse+ResolveReference against the
			// feed's own URL. This test documents that current behaviour so a future
			// change to add real resolution has to update it deliberately rather than
			// silently regress in either direction.
			name:          "RSS with relative and scheme-relative links (no resolution performed)",
			fixture:       "rss2_relative_urls.xml",
			feedURL:       "https://relative.example.com/rss.xml",
			expectedTitle: "Relative Link Blog",
			articleCount:  2,
			check: func(t *testing.T, fd *FeedData) {
				if got := fd.Articles[0].Link; got != "/articles/relative-one" {
					t.Errorf("article[0] link should pass through unresolved, got %q", got)
				}
				if got := fd.Articles[1].Link; got != "//relative.example.com/articles/relative-two" {
					t.Errorf("article[1] link should pass through unresolved, got %q", got)
				}
			},
		},
		{
			// Empty channel title/description and items missing <title>/<description>
			// must not error the whole feed — the parser substitutes fallbacks per-item
			// (sanitizeArticleTitle/generateFallbackTitle) rather than skipping items.
			name:          "RSS with missing channel and item fields",
			fixture:       "rss2_missing_fields.xml",
			feedURL:       "https://sparse.example.com/rss.xml",
			expectedTitle: "",
			articleCount:  2,
			check: func(t *testing.T, fd *FeedData) {
				if fd.Description != "" {
					t.Errorf("feed description should be empty, got %q", fd.Description)
				}
				// First item has no <title>; fallback is derived from the URL slug
				// since the description is also empty.
				first := fd.Articles[0]
				// generateFallbackTitle only keeps the final path segment
				// (feed_service.go:1359-1361), so "posts/" is dropped.
				if first.Title != "Breaking News Update" {
					t.Errorf("article[0] fallback title = %q", first.Title)
				}
				// Second item has a real title and should pass through untouched.
				second := fd.Articles[1]
				if second.Title != "Has A Title" {
					t.Errorf("article[1] title = %q", second.Title)
				}
			},
		},
		{
			name:        "Malformed XML fails gracefully",
			fixture:     "malformed.xml",
			feedURL:     "https://broken.example.com/rss.xml",
			expectError: true,
		},
		{
			// There is no per-item cap in the parse path itself (only maxFeedBodySize,
			// a 10MB cap enforced at the HTTP fetch layer in fetchFeed/TestFetchFeed_SizeLimit,
			// bounds bytes read before parsing). This guards against a future regression
			// where bulk feeds get silently truncated during unmarshaling.
			name:          "Large item count feed is fully parsed, not truncated",
			fixture:       "rss2_large_item_count.xml",
			feedURL:       "https://highvolume.example.com/rss.xml",
			expectedTitle: "High Volume Feed",
			articleCount:  3000,
			check: func(t *testing.T, fd *FeedData) {
				first := fd.Articles[0]
				if first.Title != "Bulk Article 1" {
					t.Errorf("article[0] title = %q", first.Title)
				}
				last := fd.Articles[len(fd.Articles)-1]
				if last.Title != "Bulk Article 3000" {
					t.Errorf("last article title = %q", last.Title)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := readFixture(t, tt.fixture)
			fd, err := fs.parseFeedFromBytes(body, tt.feedURL)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected an error parsing %s, got none", tt.fixture)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error parsing %s: %v", tt.fixture, err)
			}

			if fd.Title != tt.expectedTitle {
				t.Errorf("feed title = %q, want %q", fd.Title, tt.expectedTitle)
			}
			if len(fd.Articles) != tt.articleCount {
				t.Fatalf("article count = %d, want %d", len(fd.Articles), tt.articleCount)
			}
			if tt.check != nil {
				tt.check(t, fd)
			}
		})
	}
}
