package database

import (
	"sync"
	"time"
)

// defaultUserFeedsCacheTTL is how long a user's cached feed list is trusted
// before a fresh Datastore read is forced, bounding staleness from any
// invalidation this process missed (e.g. a subscribe/unsubscribe handled by
// another instance).
const defaultUserFeedsCacheTTL = 5 * time.Minute

// defaultUserFeedsCacheMaxUsers bounds cache memory. When full, the entry
// closest to expiry is evicted to make room for a new user, same policy as
// cache.UnreadCache.
const defaultUserFeedsCacheMaxUsers = 10000

// userFeedsCache caches each user's subscribed feed list in memory, keyed by
// userID. GetUserFeeds is called on every paginated articles request
// (internally by getUserArticlesPaginated as well as directly), so caching it
// here spares a UserFeed query plus a Feed GetMulti on every page load.
//
// This lives in the database package, rather than reusing internal/cache's
// FeedListCache/UnreadCache pattern, because internal/cache imports
// internal/database for the Feed type; a database-layer cache can't import
// it back without a cycle.
type userFeedsCache struct {
	mu        sync.RWMutex
	feeds     map[int][]Feed
	expiresAt map[int]time.Time
	ttl       time.Duration
	maxUsers  int
}

func newUserFeedsCache() *userFeedsCache {
	return newUserFeedsCacheWithLimits(defaultUserFeedsCacheTTL, defaultUserFeedsCacheMaxUsers)
}

func newUserFeedsCacheWithLimits(ttl time.Duration, maxUsers int) *userFeedsCache {
	return &userFeedsCache{
		feeds:     make(map[int][]Feed),
		expiresAt: make(map[int]time.Time),
		ttl:       ttl,
		maxUsers:  maxUsers,
	}
}

// get returns the cached feed list for userID if present and unexpired.
func (c *userFeedsCache) get(userID int) ([]Feed, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	feeds, ok := c.feeds[userID]
	if !ok || time.Now().After(c.expiresAt[userID]) {
		return nil, false
	}

	result := make([]Feed, len(feeds))
	copy(result, feeds)
	return result, true
}

// set stores userID's feed list with the configured TTL, evicting the entry
// closest to expiry if the cache is at capacity.
func (c *userFeedsCache) set(userID int, feeds []Feed) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.feeds[userID]; !exists && len(c.feeds) >= c.maxUsers {
		var evictID int
		var earliest time.Time
		for uid, exp := range c.expiresAt {
			if earliest.IsZero() || exp.Before(earliest) {
				evictID = uid
				earliest = exp
			}
		}
		delete(c.feeds, evictID)
		delete(c.expiresAt, evictID)
	}

	cached := make([]Feed, len(feeds))
	copy(cached, feeds)
	c.feeds[userID] = cached
	c.expiresAt[userID] = time.Now().Add(c.ttl)
}

// invalidate removes userID's cached feed list, forcing a fresh read on the
// next GetUserFeeds call. Call this after any mutation that changes the
// user's subscriptions.
func (c *userFeedsCache) invalidate(userID int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.feeds, userID)
	delete(c.expiresAt, userID)
}
