package cache

import (
	"sync"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

// FeedCountsCache provides in-memory caching for both unread and total article counts.
// Similar to UnreadCache but stores FeedCounts struct with both unread and total counts.
type FeedCountsCache struct {
	counts    map[int]map[int]database.FeedCounts // userID → (feedID → FeedCounts)
	refreshAt map[int]time.Time                   // userID → cache expiry time
	mu        sync.RWMutex
	ttl       time.Duration
}

// NewFeedCountsCache creates a new feed counts cache with the specified TTL.
func NewFeedCountsCache(ttl time.Duration) *FeedCountsCache {
	fc := &FeedCountsCache{
		counts:    make(map[int]map[int]database.FeedCounts),
		refreshAt: make(map[int]time.Time),
		ttl:       ttl,
	}

	// Start cleanup goroutine to prevent memory leak
	go fc.cleanupExpiredEntries()

	return fc
}

// Get retrieves cached feed counts for a user if they exist and are not expired.
// Returns the counts and true if cache hit, nil and false if cache miss.
func (fc *FeedCountsCache) Get(userID int) (map[int]database.FeedCounts, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	counts, exists := fc.counts[userID]
	if !exists {
		return nil, false
	}

	// Check if cache has expired
	if time.Now().After(fc.refreshAt[userID]) {
		return nil, false
	}

	// Return a copy to prevent external modification
	result := make(map[int]database.FeedCounts, len(counts))
	for feedID, count := range counts {
		result[feedID] = count
	}

	return result, true
}

// Set stores feed counts for a user in the cache with the configured TTL.
func (fc *FeedCountsCache) Set(userID int, counts map[int]database.FeedCounts) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Store a copy to prevent external modification
	cached := make(map[int]database.FeedCounts, len(counts))
	for feedID, count := range counts {
		cached[feedID] = count
	}

	fc.counts[userID] = cached
	fc.refreshAt[userID] = time.Now().Add(fc.ttl)
}

// UpdateCount incrementally updates the cached unread count when an article's read status changes.
// Total count remains unchanged as it represents all articles in the feed.
func (fc *FeedCountsCache) UpdateCount(userID, feedID int, wasRead, nowRead bool) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	counts, exists := fc.counts[userID]
	if !exists {
		return // No cache to update
	}

	// Check if cache is still valid - don't update expired cache
	if time.Now().After(fc.refreshAt[userID]) {
		return
	}

	feedCount, exists := counts[feedID]
	if !exists {
		return
	}

	// Update unread count based on state transition
	if !wasRead && nowRead {
		// Article marked as read - decrement unread count
		feedCount.Unread--
		if feedCount.Unread < 0 {
			feedCount.Unread = 0 // Safety check
		}
	} else if wasRead && !nowRead {
		// Article marked as unread - increment unread count
		feedCount.Unread++
	}
	// Total count never changes from read status updates

	counts[feedID] = feedCount
}

// Invalidate removes cached counts for a user, forcing a fresh fetch on next request.
func (fc *FeedCountsCache) Invalidate(userID int) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	delete(fc.counts, userID)
	delete(fc.refreshAt, userID)
}

// InvalidateAll clears the entire cache.
func (fc *FeedCountsCache) InvalidateAll() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.counts = make(map[int]map[int]database.FeedCounts)
	fc.refreshAt = make(map[int]time.Time)
}

// cleanupExpiredEntries removes expired cache entries to prevent memory leak.
func (fc *FeedCountsCache) cleanupExpiredEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		fc.mu.Lock()
		now := time.Now()
		for userID, expiry := range fc.refreshAt {
			if now.After(expiry) {
				delete(fc.counts, userID)
				delete(fc.refreshAt, userID)
			}
		}
		fc.mu.Unlock()
	}
}
