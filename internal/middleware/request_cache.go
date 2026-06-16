package middleware

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/database"
)

const requestCacheKey = "request_cache"

// maxRequestCacheEntries limits cache growth for requests that touch many users (e.g. admin endpoints).
const maxRequestCacheEntries = 100

// RequestCache provides request-scoped caching to eliminate duplicate database calls
// within a single HTTP request. Cache is automatically cleared when the request completes.
type RequestCache struct {
	userFeeds map[int][]database.Feed
	mu        sync.RWMutex
}

// RequestCacheMiddleware creates a new request-scoped cache for each HTTP request.
// This cache is stored in gin.Context and automatically cleaned up when the request completes.
func RequestCacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cache := &RequestCache{
			userFeeds: make(map[int][]database.Feed),
		}
		c.Set(requestCacheKey, cache)
		c.Next()
	}
}

// InvalidateUserFeeds removes the cached feeds for userID so the next call to
// GetCachedUserFeeds re-fetches from the database. Call this after any mutation
// that changes the user's feed subscriptions within the same request.
func (rc *RequestCache) InvalidateUserFeeds(userID int) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	delete(rc.userFeeds, userID)
}

// InvalidateCachedUserFeeds removes the cached feeds for userID from the
// request-scoped cache stored in c. No-op if the cache middleware is absent.
func InvalidateCachedUserFeeds(c *gin.Context, userID int) {
	cacheInterface, exists := c.Get(requestCacheKey)
	if !exists {
		return
	}
	cache, ok := cacheInterface.(*RequestCache)
	if !ok {
		return
	}
	cache.InvalidateUserFeeds(userID)
}

// GetCachedUserFeeds retrieves user feeds from request cache if available,
// otherwise fetches from database and stores in cache for subsequent calls.
// This eliminates duplicate GetUserFeeds calls within a single request.
//
// Example usage in handlers:
//
//	feeds, err := middleware.GetCachedUserFeeds(c, userID, db)
func GetCachedUserFeeds(c *gin.Context, userID int, db database.Database) ([]database.Feed, error) {
	// Try to get the cache from context
	cacheInterface, exists := c.Get(requestCacheKey)
	if !exists {
		// No cache middleware - fetch directly
		return db.GetUserFeeds(userID)
	}

	cache, ok := cacheInterface.(*RequestCache)
	if !ok {
		// Wrong type in context - fetch directly
		return db.GetUserFeeds(userID)
	}

	// Check if feeds are already cached (read lock)
	cache.mu.RLock()
	if feeds, cached := cache.userFeeds[userID]; cached {
		cache.mu.RUnlock()
		return feeds, nil // Cache hit!
	}
	cache.mu.RUnlock()

	// Cache miss - acquire write lock to fetch from database
	// Use double-check locking to prevent race condition where multiple
	// goroutines could see cache miss and all fetch from database
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// Double-check: another goroutine may have populated cache while we waited for lock
	if feeds, cached := cache.userFeeds[userID]; cached {
		return feeds, nil // Cache hit after acquiring lock!
	}

	// Still a cache miss - fetch from database
	feeds, err := db.GetUserFeeds(userID)
	if err != nil {
		return nil, err
	}

	// Store in cache only while under the entry limit.
	if len(cache.userFeeds) < maxRequestCacheEntries {
		cache.userFeeds[userID] = feeds
	}

	return feeds, nil
}
