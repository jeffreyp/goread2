package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter stores rate limiters for each IP address
type RateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

// NewRateLimiter creates a new rate limiter
// r is the rate (requests per second)
// b is the burst size (max requests at once)
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	limiter := &RateLimiter{
		ips: make(map[string]*rate.Limiter),
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
	rl.ips[ip] = limiter

	return limiter
}

// GetLimiter returns the rate limiter for an IP address
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.ips[ip]
	rl.mu.RUnlock()

	if !exists {
		return rl.AddIP(ip)
	}

	return limiter
}

// cleanupIPs removes old IP entries every hour
func (rl *RateLimiter) cleanupIPs() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		// In a production system, you might want to track last access time
		// For simplicity, we'll clear all entries periodically
		rl.ips = make(map[string]*rate.Limiter)
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates a Gin middleware for rate limiting
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		ip := c.ClientIP()

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
