package cache

import (
	"sync"
	"time"
)

// UnreadCache provides in-memory caching for unread article counts with incremental updates.
// This dramatically reduces database reads by serving cached counts and updating them
// incrementally when articles are marked as read/unread.
type UnreadCache struct {
	counts    map[int]map[int]int // userID → (feedID → unread count)
	refreshAt map[int]time.Time   // userID → cache expiry time
	mu        sync.RWMutex
	ttl       time.Duration
}

// NewUnreadCache creates a new unread count cache with the specified TTL.
// Typical TTL is 90-120 seconds for background refresh of counts.
func NewUnreadCache(ttl time.Duration) *UnreadCache {
	uc := &UnreadCache{
		counts:    make(map[int]map[int]int),
		refreshAt: make(map[int]time.Time),
		ttl:       ttl,
	}

	// Start cleanup goroutine to prevent memory leak
	go uc.cleanupExpiredEntries()

	return uc
}

// Get retrieves cached unread counts for a user if they exist and are not expired.
// Returns the counts and true if cache hit, nil and false if cache miss.
func (uc *UnreadCache) Get(userID int) (map[int]int, bool) {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	counts, exists := uc.counts[userID]
	if !exists {
		return nil, false
	}

	// Check if cache has expired
	if time.Now().After(uc.refreshAt[userID]) {
		return nil, false
	}

	// Return a copy to prevent external modification
	result := make(map[int]int, len(counts))
	for feedID, count := range counts {
		result[feedID] = count
	}

	return result, true
}

// Set stores unread counts for a user in the cache with the configured TTL.
func (uc *UnreadCache) Set(userID int, counts map[int]int) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	// Store a copy to prevent external modification
	cached := make(map[int]int, len(counts))
	for feedID, count := range counts {
		cached[feedID] = count
	}

	uc.counts[userID] = cached
	uc.refreshAt[userID] = time.Now().Add(uc.ttl)
}

// UpdateCount incrementally updates the cached count when an article's read status changes.
// This provides immediate feedback to users while maintaining cache accuracy.
//
// Parameters:
//   - userID: The user whose cache to update
//   - feedID: The feed containing the article
//   - wasRead: Previous read status of the article
//   - nowRead: New read status of the article
func (uc *UnreadCache) UpdateCount(userID, feedID int, wasRead, nowRead bool) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	counts, exists := uc.counts[userID]
	if !exists {
		return // No cache to update
	}

	// Check if cache is still valid - don't update expired cache
	if time.Now().After(uc.refreshAt[userID]) {
		return
	}

	// Update count based on state transition
	if !wasRead && nowRead {
		// Article marked as read - decrement unread count
		counts[feedID]--
		if counts[feedID] < 0 {
			counts[feedID] = 0 // Safety check
		}
	} else if wasRead && !nowRead {
		// Article marked as unread - increment unread count
		counts[feedID]++
	}
	// If wasRead == nowRead, no change needed
}

// Invalidate removes cached counts for a user, forcing a fresh fetch on next request.
// Use this for complex operations where incremental updates are difficult (e.g., batch operations).
func (uc *UnreadCache) Invalidate(userID int) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	delete(uc.counts, userID)
	delete(uc.refreshAt, userID)
}

// InvalidateAll clears the entire cache. Useful for testing or maintenance.
func (uc *UnreadCache) InvalidateAll() {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	uc.counts = make(map[int]map[int]int)
	uc.refreshAt = make(map[int]time.Time)
}

// Stats returns cache statistics for monitoring.
type CacheStats struct {
	CachedUsers int
	TotalFeeds  int
}

// GetStats returns current cache statistics.
func (uc *UnreadCache) GetStats() CacheStats {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	now := time.Now()
	activeUsers := 0
	totalFeeds := 0

	for userID, counts := range uc.counts {
		if now.Before(uc.refreshAt[userID]) { // Only count valid entries
			activeUsers++
			totalFeeds += len(counts)
		}
	}

	return CacheStats{
		CachedUsers: activeUsers,
		TotalFeeds:  totalFeeds,
	}
}

// cleanupExpiredEntries removes expired cache entries to prevent memory leak.
// Runs every 5 minutes to clean up entries that have passed their expiry time.
func (uc *UnreadCache) cleanupExpiredEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		uc.mu.Lock()
		now := time.Now()
		for userID, expiry := range uc.refreshAt {
			if now.After(expiry) {
				delete(uc.counts, userID)
				delete(uc.refreshAt, userID)
			}
		}
		uc.mu.Unlock()
	}
}
