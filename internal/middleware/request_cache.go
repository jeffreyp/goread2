package middleware

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/database"
)

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
		c.Set("request_cache", cache)
		c.Next()
	}
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
	cacheInterface, exists := c.Get("request_cache")
	if !exists {
		// No cache middleware - fetch directly
		return db.GetUserFeeds(userID)
	}

	cache := cacheInterface.(*RequestCache)

	// Check if feeds are already cached (read lock)
	cache.mu.RLock()
	if feeds, cached := cache.userFeeds[userID]; cached {
		cache.mu.RUnlock()
		return feeds, nil // Cache hit!
	}
	cache.mu.RUnlock()

	// Cache miss - fetch from database
	feeds, err := db.GetUserFeeds(userID)
	if err != nil {
		return nil, err
	}

	// Store in cache for next call (write lock)
	cache.mu.Lock()
	cache.userFeeds[userID] = feeds
	cache.mu.Unlock()

	return feeds, nil
}
