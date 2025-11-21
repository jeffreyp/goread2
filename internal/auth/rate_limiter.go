package auth

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipEntry stores a rate limiter and its last access time
type ipEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// RateLimiter stores rate limiters for each IP address
type RateLimiter struct {
	ips map[string]*ipEntry
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

// NewRateLimiter creates a new rate limiter
// r is the rate (requests per second)
// b is the burst size (max requests at once)
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	limiter := &RateLimiter{
		ips: make(map[string]*ipEntry),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}

	// Start cleanup goroutine to remove old entries
	go limiter.cleanupIPs()

	return limiter
}

// AddIP creates a new rate limiter for an IP address if it doesn't exist
func (rl *RateLimiter) AddIP(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter := rate.NewLimiter(rl.r, rl.b)
	rl.ips[ip] = &ipEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}

	return limiter
}

// GetLimiter returns the rate limiter for an IP address
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.ips[ip]
	if !exists {
		// Create new entry
		limiter := rate.NewLimiter(rl.r, rl.b)
		rl.ips[ip] = &ipEntry{
			limiter:    limiter,
			lastAccess: time.Now(),
		}
		return limiter
	}

	// Update last access time for existing entry
	entry.lastAccess = time.Now()
	return entry.limiter
}

// cleanupIPs removes IP entries that haven't been accessed in over 1 hour
func (rl *RateLimiter) cleanupIPs() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()

		// Remove entries inactive for more than 1 hour
		cutoff := time.Now().Add(-1 * time.Hour)
		deletedCount := 0
		totalEntries := len(rl.ips)
		for ip, entry := range rl.ips {
			if entry.lastAccess.Before(cutoff) {
				delete(rl.ips, ip)
				deletedCount++
			}
		}

		rl.mu.Unlock()

		// Log cleanup statistics for monitoring
		log.Printf("Rate limiter cleanup: deleted %d entries, %d remaining (total: %d)",
			deletedCount, totalEntries-deletedCount, totalEntries)
	}
}

// RateLimitMiddleware creates a Gin middleware for rate limiting
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP using secure method to prevent IP spoofing
		ip := GetSecureClientIP(c)

		// Get limiter for this IP
		ipLimiter := limiter.GetLimiter(ip)

		// Check if request is allowed
		if !ipLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
