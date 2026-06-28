package database

import (
	"database/sql"
	"fmt"
	"math"
	"testing"
	"testing/quick"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupBenchDB creates an isolated in-memory database for benchmarks.
// Each call opens a uniquely-named URI so benchmarks don't share state.
func setupBenchDB(b *testing.B) *DB {
	b.Helper()
	uri := fmt.Sprintf("file:bench_%d?mode=memory&cache=shared&_loc=auto", time.Now().UnixNano())
	db, err := sql.Open("sqlite3", uri)
	if err != nil {
		b.Fatalf("open bench db: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		b.Fatalf("enable fk: %v", err)
	}
	wrapped := &DB{db}
	if err := wrapped.CreateTables(); err != nil {
		b.Fatalf("create tables: %v", err)
	}
	b.Cleanup(func() { _ = wrapped.Close() })
	return wrapped
}

// seedBenchData creates a user subscribed to nFeeds feeds, each with nArticles articles.
func seedBenchData(b *testing.B, db *DB, nFeeds, nArticles int) *User {
	b.Helper()
	user := &User{
		GoogleID:             fmt.Sprintf("bench_%d", time.Now().UnixNano()),
		Email:                fmt.Sprintf("bench%d@example.com", time.Now().UnixNano()),
		Name:                 "Bench User",
		SubscriptionStatus:   "trial",
		TrialEndsAt:          time.Now().AddDate(0, 1, 0),
		MaxArticlesOnFeedAdd: 1000,
	}
	if err := db.CreateUser(user); err != nil {
		b.Fatalf("create user: %v", err)
	}

	base := time.Now()
	for f := range nFeeds {
		feed := &Feed{
			Title:     fmt.Sprintf("Bench Feed %d", f),
			URL:       fmt.Sprintf("https://bench.example.com/feed/%d/%d", time.Now().UnixNano(), f),
			CreatedAt: base,
			UpdatedAt: base,
		}
		if err := db.AddFeed(feed); err != nil {
			b.Fatalf("add feed: %v", err)
		}
		if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
			b.Fatalf("subscribe user: %v", err)
		}
		for a := range nArticles {
			article := &Article{
				FeedID:      feed.ID,
				Title:       fmt.Sprintf("Article %d-%d", f, a),
				URL:         fmt.Sprintf("https://bench.example.com/article/%d/%d/%d", time.Now().UnixNano(), f, a),
				PublishedAt: base.Add(-time.Duration(a) * time.Hour),
				CreatedAt:   base,
			}
			if err := db.AddArticle(article); err != nil {
				b.Fatalf("add article: %v", err)
			}
		}
	}
	return user
}

func BenchmarkGetUserArticlesPaginatedFirstPage(b *testing.B) {
	db := setupBenchDB(b)
	user := seedBenchData(b, db, 5, 50) // 5 feeds × 50 articles = 250 total

	b.ResetTimer()
	for range b.N {
		_, err := db.GetUserArticlesPaginated(user.ID, 20, "", false)
		if err != nil {
			b.Fatalf("paginated query: %v", err)
		}
	}
}

func BenchmarkGetUserArticlesPaginatedWithCursor(b *testing.B) {
	db := setupBenchDB(b)
	user := seedBenchData(b, db, 5, 50)

	// Fetch first page to get a real cursor.
	page1, err := db.GetUserArticlesPaginated(user.ID, 20, "", false)
	if err != nil {
		b.Fatalf("first page: %v", err)
	}
	cursor := page1.NextCursor
	if cursor == "" {
		b.Skip("not enough data to produce a cursor")
	}

	b.ResetTimer()
	for range b.N {
		_, err := db.GetUserArticlesPaginated(user.ID, 20, cursor, false)
		if err != nil {
			b.Fatalf("paginated cursor query: %v", err)
		}
	}
}

func BenchmarkGetUserArticlesPaginatedUnreadOnly(b *testing.B) {
	db := setupBenchDB(b)
	user := seedBenchData(b, db, 5, 50)

	b.ResetTimer()
	for range b.N {
		_, err := db.GetUserArticlesPaginated(user.ID, 20, "", true)
		if err != nil {
			b.Fatalf("unread paginated query: %v", err)
		}
	}
}

func BenchmarkGetUserUnreadCounts(b *testing.B) {
	db := setupBenchDB(b)
	user := seedBenchData(b, db, 10, 30) // 10 feeds × 30 articles

	b.ResetTimer()
	for range b.N {
		_, err := db.GetUserUnreadCounts(user.ID)
		if err != nil {
			b.Fatalf("unread counts: %v", err)
		}
	}
}

func BenchmarkGetAccountStats(b *testing.B) {
	db := setupBenchDB(b)
	user := seedBenchData(b, db, 5, 40)

	b.ResetTimer()
	for range b.N {
		_, err := db.GetAccountStats(user.ID)
		if err != nil {
			b.Fatalf("account stats: %v", err)
		}
	}
}

// --- Property-based tests for cursor encoding/decoding ---

// TestCursorRoundTrip verifies that any (id, timestamp) pair survives
// an encode→decode cycle with identical values.
func TestCursorRoundTrip(t *testing.T) {
	// Targeted cases first: boundaries and precision-sensitive values.
	cases := []struct {
		id int
		ts time.Time
	}{
		{1, time.Unix(0, 0)},
		{1, time.Unix(0, 1)},
		{math.MaxInt32, time.Now().Round(0)},
		{1, time.Date(2000, 1, 1, 0, 0, 0, 999999999, time.UTC)},
		{42, time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		cursor := encodeSQLiteCursor(c.id, c.ts)
		got, err := decodeSQLiteCursor(cursor)
		if err != nil {
			t.Errorf("decode(%q): %v", cursor, err)
			continue
		}
		if got.ID != c.id {
			t.Errorf("id: got %d, want %d", got.ID, c.id)
		}
		if !got.PublishedAt.Equal(c.ts) {
			t.Errorf("ts: got %v, want %v", got.PublishedAt, c.ts)
		}
	}

	// Property: holds for arbitrary positive id and nanosecond timestamp.
	prop := func(id uint32, nanos int64) bool {
		if id == 0 {
			id = 1
		}
		ts := time.Unix(0, nanos)
		cursor := encodeSQLiteCursor(int(id), ts)
		got, err := decodeSQLiteCursor(cursor)
		if err != nil {
			return false
		}
		return got.ID == int(id) && got.PublishedAt.Equal(ts)
	}
	if err := quick.Check(prop, &quick.Config{MaxCount: 1000}); err != nil {
		t.Errorf("property violation: %v", err)
	}
}

// TestDecodeCursorInvalidInputs verifies that malformed cursors return errors.
func TestDecodeCursorInvalidInputs(t *testing.T) {
	bad := []string{
		"",
		"notanumber",
		"123",       // missing _id part
		"_",         // separator only
		"abc_def",   // non-numeric parts
		"123_",      // missing id
		"_456",      // missing timestamp
	}
	for _, s := range bad {
		if _, err := decodeSQLiteCursor(s); err == nil {
			t.Errorf("decodeSQLiteCursor(%q) expected error, got nil", s)
		}
	}
}
