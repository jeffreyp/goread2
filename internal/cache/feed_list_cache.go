package cache

import (
	"sync"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

// FeedListCache provides in-memory caching for the list of all user feeds.
// This dramatically reduces database reads during feed refresh operations by
// caching the result of GetAllUserFeeds() which is called every hour.
type FeedListCache struct {
	feeds     []database.Feed
	refreshAt time.Time
	mu        sync.RWMutex
	ttl       time.Duration
}

// NewFeedListCache creates a new feed list cache with the specified TTL.
// Typical TTL is 15-30 minutes to balance freshness with cost savings.
func NewFeedListCache(ttl time.Duration) *FeedListCache {
	fc := &FeedListCache{
		feeds: nil,
		ttl:   ttl,
	}

	// Start cleanup goroutine to prevent memory leak
	go fc.cleanupIfExpired()

	return fc
}

// Get retrieves the cached feed list if it exists and is not expired.
// Returns the feeds and true if cache hit, nil and false if cache miss.
func (fc *FeedListCache) Get() ([]database.Feed, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	if fc.feeds == nil {
		return nil, false
	}

	// Check if cache has expired
	if time.Now().After(fc.refreshAt) {
		return nil, false
	}

	// Return a copy to prevent external modification
	result := make([]database.Feed, len(fc.feeds))
	copy(result, fc.feeds)

	return result, true
}

// Set stores the feed list in the cache with the configured TTL.
func (fc *FeedListCache) Set(feeds []database.Feed) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Store a copy to prevent external modification
	cached := make([]database.Feed, len(feeds))
	copy(cached, feeds)

	fc.feeds = cached
	fc.refreshAt = time.Now().Add(fc.ttl)
}

// Invalidate clears the cached feed list, forcing a fresh fetch on next request.
// Use this when users subscribe/unsubscribe from feeds to ensure immediate updates.
func (fc *FeedListCache) Invalidate() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.feeds = nil
}

// Stats returns cache statistics for monitoring.
type FeedListCacheStats struct {
	CachedFeeds int
	IsValid     bool
}

// GetStats returns current cache statistics.
func (fc *FeedListCache) GetStats() FeedListCacheStats {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	return FeedListCacheStats{
		CachedFeeds: len(fc.feeds),
		IsValid:     fc.feeds != nil && time.Now().Before(fc.refreshAt),
	}
}

// cleanupIfExpired removes expired cache entry to prevent memory leak.
// Runs every 5 minutes to clean up the feed list if it has passed its expiry time.
func (fc *FeedListCache) cleanupIfExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		fc.mu.Lock()
		if fc.feeds != nil && time.Now().After(fc.refreshAt) {
			fc.feeds = nil
		}
		fc.mu.Unlock()
	}
}
