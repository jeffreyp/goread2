package cache

import (
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

func TestNewFeedListCache(t *testing.T) {
	cache := NewFeedListCache(20 * time.Minute)
	if cache == nil {
		t.Fatal("NewFeedListCache returned nil")
	}
	if cache.ttl != 20*time.Minute {
		t.Errorf("Expected TTL 20 minutes, got %v", cache.ttl)
	}
}

func TestFeedListCache_SetAndGet(t *testing.T) {
	cache := NewFeedListCache(60 * time.Second)

	feeds := []database.Feed{
		{ID: 1, Title: "Feed 1", URL: "https://example.com/feed1"},
		{ID: 2, Title: "Feed 2", URL: "https://example.com/feed2"},
		{ID: 3, Title: "Feed 3", URL: "https://example.com/feed3"},
	}

	// Set feeds
	cache.Set(feeds)

	// Get feeds - should be cache hit
	retrieved, hit := cache.Get()
	if !hit {
		t.Fatal("Expected cache hit, got miss")
	}

	if len(retrieved) != len(feeds) {
		t.Errorf("Expected %d feeds, got %d", len(feeds), len(retrieved))
	}

	for i, feed := range feeds {
		if retrieved[i].ID != feed.ID {
			t.Errorf("Feed %d: expected ID %d, got %d", i, feed.ID, retrieved[i].ID)
		}
		if retrieved[i].Title != feed.Title {
			t.Errorf("Feed %d: expected title %s, got %s", i, feed.Title, retrieved[i].Title)
		}
	}
}

func TestFeedListCache_GetMiss(t *testing.T) {
	cache := NewFeedListCache(60 * time.Second)

	// Get without setting
	_, hit := cache.Get()
	if hit {
		t.Error("Expected cache miss for uninitialized cache")
	}
}

func TestFeedListCache_Expiration(t *testing.T) {
	cache := NewFeedListCache(100 * time.Millisecond)

	feeds := []database.Feed{
		{ID: 1, Title: "Test Feed", URL: "https://example.com/feed"},
	}
	cache.Set(feeds)

	// Immediate get should hit
	_, hit := cache.Get()
	if !hit {
		t.Error("Expected cache hit immediately after set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Get should miss after expiration
	_, hit = cache.Get()
	if hit {
		t.Error("Expected cache miss after expiration")
	}
}

func TestFeedListCache_Invalidate(t *testing.T) {
	cache := NewFeedListCache(60 * time.Second)

	feeds := []database.Feed{
		{ID: 1, Title: "Test Feed", URL: "https://example.com/feed"},
	}
	cache.Set(feeds)

	// Verify cache hit
	_, hit := cache.Get()
	if !hit {
		t.Fatal("Expected cache hit before invalidation")
	}

	// Invalidate
	cache.Invalidate()

	// Verify cache miss
	_, hit = cache.Get()
	if hit {
		t.Error("Expected cache miss after invalidation")
	}
}

func TestFeedListCache_GetStats(t *testing.T) {
	cache := NewFeedListCache(60 * time.Second)

	// Empty cache
	stats := cache.GetStats()
	if stats.CachedFeeds != 0 || stats.IsValid {
		t.Errorf("Empty cache should have 0 feeds and IsValid=false, got %d feeds and IsValid=%v",
			stats.CachedFeeds, stats.IsValid)
	}

	// Add feeds
	feeds := []database.Feed{
		{ID: 1, Title: "Feed 1", URL: "https://example.com/feed1"},
		{ID: 2, Title: "Feed 2", URL: "https://example.com/feed2"},
		{ID: 3, Title: "Feed 3", URL: "https://example.com/feed3"},
	}
	cache.Set(feeds)

	stats = cache.GetStats()
	if stats.CachedFeeds != 3 {
		t.Errorf("Expected 3 cached feeds, got %d", stats.CachedFeeds)
	}
	if !stats.IsValid {
		t.Error("Expected IsValid=true for valid cache")
	}
}

func TestFeedListCache_GetReturnsCopy(t *testing.T) {
	cache := NewFeedListCache(60 * time.Second)

	original := []database.Feed{
		{ID: 1, Title: "Original", URL: "https://example.com/feed1"},
	}
	cache.Set(original)

	// Get and modify
	retrieved, _ := cache.Get()
	retrieved[0].Title = "Modified"
	retrieved = append(retrieved, database.Feed{ID: 999, Title: "Added", URL: "https://example.com/added"})
	_ = retrieved // Verify slice modification doesn't affect cache

	// Verify cache wasn't modified
	cached, _ := cache.Get()
	if cached[0].Title != "Original" {
		t.Error("Modifying retrieved slice should not affect cached data")
	}
	if len(cached) != 1 {
		t.Error("Appending to retrieved slice should not affect cached data")
	}
}

func TestFeedListCache_SetStoresCopy(t *testing.T) {
	cache := NewFeedListCache(60 * time.Second)

	original := []database.Feed{
		{ID: 1, Title: "Original", URL: "https://example.com/feed1"},
	}
	cache.Set(original)

	// Modify original after setting
	original[0].Title = "Modified"
	original = append(original, database.Feed{ID: 999, Title: "Added", URL: "https://example.com/added"})
	_ = original // Verify input slice modification doesn't affect cache

	// Verify cache wasn't affected
	cached, _ := cache.Get()
	if cached[0].Title != "Original" {
		t.Error("Modifying input slice after Set should not affect cached data")
	}
	if len(cached) != 1 {
		t.Error("Appending to input slice after Set should not affect cached data")
	}
}

func TestFeedListCache_ConcurrentAccess(t *testing.T) {
	cache := NewFeedListCache(60 * time.Second)

	feeds := []database.Feed{
		{ID: 1, Title: "Feed 1", URL: "https://example.com/feed1"},
		{ID: 2, Title: "Feed 2", URL: "https://example.com/feed2"},
	}
	cache.Set(feeds)

	// Concurrent reads
	done := make(chan bool)
	iterations := 100

	for i := 0; i < 20; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				cache.Get()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Should not panic and should have valid data
	retrieved, hit := cache.Get()
	if !hit {
		t.Error("Expected cache hit after concurrent access")
	}
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 feeds after concurrent access, got %d", len(retrieved))
	}
}

func TestFeedListCache_InvalidateDoesNotPanic(t *testing.T) {
	cache := NewFeedListCache(60 * time.Second)

	// Invalidate empty cache - should not panic
	cache.Invalidate()

	// Verify still works
	feeds := []database.Feed{
		{ID: 1, Title: "Test", URL: "https://example.com/feed"},
	}
	cache.Set(feeds)

	retrieved, hit := cache.Get()
	if !hit || len(retrieved) != 1 {
		t.Error("Cache should work normally after invalidating empty cache")
	}
}
