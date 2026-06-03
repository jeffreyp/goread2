package services

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// DomainRateLimiter provides per-domain rate limiting to prevent DDoS attacks on feed servers
type DomainRateLimiter struct {
	limiters     map[string]*rate.Limiter
	lastAccessed map[string]time.Time
	mu           sync.RWMutex

	// Configuration
	requestsPerMinute int
	burstSize         int
}

// RateLimiterConfig holds configuration for the rate limiter
type RateLimiterConfig struct {
	RequestsPerMinute int // Maximum requests per minute per domain
	BurstSize         int // Burst allowance for each domain
}

// NewDomainRateLimiter creates a new domain-based rate limiter
func NewDomainRateLimiter(config RateLimiterConfig) *DomainRateLimiter {
	// Set sensible defaults if not provided
	if config.RequestsPerMinute <= 0 {
		config.RequestsPerMinute = 6 // 1 request per 10 seconds default
	}
	if config.BurstSize <= 0 {
		config.BurstSize = 1 // No burst by default
	}

	return &DomainRateLimiter{
		limiters:          make(map[string]*rate.Limiter),
		lastAccessed:      make(map[string]time.Time),
		requestsPerMinute: config.RequestsPerMinute,
		burstSize:         config.BurstSize,
	}
}

// Allow checks if a request to the given URL is allowed under rate limiting rules
func (d *DomainRateLimiter) Allow(feedURL string) bool {
	domain := d.extractDomain(feedURL)
	if domain == "" {
		return false // Invalid URL
	}

	limiter := d.getLimiterForDomain(domain)
	return limiter.Allow()
}

// Wait blocks until a request to the given URL is allowed, or returns an error if ctx is cancelled.
func (d *DomainRateLimiter) Wait(ctx context.Context, feedURL string) error {
	domain := d.extractDomain(feedURL)
	if domain == "" {
		return &RateLimitError{Domain: domain, Message: "invalid URL"}
	}

	limiter := d.getLimiterForDomain(domain)
	return limiter.Wait(ctx)
}

// getLimiterForDomain gets or creates a rate limiter for a specific domain
func (d *DomainRateLimiter) getLimiterForDomain(domain string) *rate.Limiter {
	d.mu.RLock()
	limiter, exists := d.limiters[domain]
	d.mu.RUnlock()

	if exists {
		d.mu.Lock()
		d.lastAccessed[domain] = time.Now()
		d.mu.Unlock()
		return limiter
	}

	// Create new limiter for this domain
	d.mu.Lock()
	defer d.mu.Unlock()

	// Double-check in case another goroutine created it
	if limiter, exists := d.limiters[domain]; exists {
		d.lastAccessed[domain] = time.Now()
		return limiter
	}

	// Convert requests per minute to rate.Every duration
	interval := time.Minute / time.Duration(d.requestsPerMinute)
	limiter = rate.NewLimiter(rate.Every(interval), d.burstSize)
	d.limiters[domain] = limiter
	d.lastAccessed[domain] = time.Now()

	return limiter
}

// extractDomain extracts the domain from a feed URL
func (d *DomainRateLimiter) extractDomain(feedURL string) string {
	parsedURL, err := url.Parse(feedURL)
	if err != nil {
		return ""
	}

	// Normalize domain to lowercase
	domain := strings.ToLower(parsedURL.Host)

	// Remove www. prefix for consistency
	domain = strings.TrimPrefix(domain, "www.")

	return domain
}

// GetDomainStats returns current statistics for all domains
func (d *DomainRateLimiter) GetDomainStats() map[string]DomainStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]DomainStats)
	for domain, limiter := range d.limiters {
		stats[domain] = DomainStats{
			Domain:          domain,
			TokensRemaining: int(limiter.Tokens()),
			BurstLimit:      d.burstSize,
		}
	}

	return stats
}

// cleanupIdleThreshold is the duration after which an unused limiter is evicted.
const cleanupIdleThreshold = time.Hour

// CleanupOldLimiters removes limiters that haven't been accessed within cleanupIdleThreshold.
func (d *DomainRateLimiter) CleanupOldLimiters() {
	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := time.Now().Add(-cleanupIdleThreshold)
	for domain, last := range d.lastAccessed {
		if last.Before(cutoff) {
			delete(d.limiters, domain)
			delete(d.lastAccessed, domain)
		}
	}
}

// DomainStats holds statistics for a domain's rate limiting
type DomainStats struct {
	Domain          string `json:"domain"`
	TokensRemaining int    `json:"tokens_remaining"`
	BurstLimit      int    `json:"burst_limit"`
}

// RateLimitError represents a rate limiting error
type RateLimitError struct {
	Domain  string
	Message string
}

func (e *RateLimitError) Error() string {
	return "rate limit exceeded for domain " + e.Domain + ": " + e.Message
}
