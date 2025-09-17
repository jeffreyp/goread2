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
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex

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

// Wait blocks until a request to the given URL is allowed, or returns an error if context is cancelled
func (d *DomainRateLimiter) Wait(feedURL string) error {
	domain := d.extractDomain(feedURL)
	if domain == "" {
		return &RateLimitError{Domain: domain, Message: "invalid URL"}
	}

	limiter := d.getLimiterForDomain(domain)
	return limiter.Wait(context.Background())
}

// getLimiterForDomain gets or creates a rate limiter for a specific domain
func (d *DomainRateLimiter) getLimiterForDomain(domain string) *rate.Limiter {
	d.mu.RLock()
	limiter, exists := d.limiters[domain]
	d.mu.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter for this domain
	d.mu.Lock()
	defer d.mu.Unlock()

	// Double-check in case another goroutine created it
	if limiter, exists := d.limiters[domain]; exists {
		return limiter
	}

	// Convert requests per minute to rate.Every duration
	interval := time.Minute / time.Duration(d.requestsPerMinute)
	limiter = rate.NewLimiter(rate.Every(interval), d.burstSize)
	d.limiters[domain] = limiter

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
	if strings.HasPrefix(domain, "www.") {
		domain = domain[4:]
	}

	return domain
}

// GetDomainStats returns current statistics for all domains
func (d *DomainRateLimiter) GetDomainStats() map[string]DomainStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]DomainStats)
	for domain, limiter := range d.limiters {
		stats[domain] = DomainStats{
			Domain:         domain,
			TokensRemaining: int(limiter.Tokens()),
			BurstLimit:     d.burstSize,
		}
	}

	return stats
}

// CleanupOldLimiters removes limiters that haven't been used recently to prevent memory leaks
func (d *DomainRateLimiter) CleanupOldLimiters() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Remove limiters that are at full capacity (haven't been used recently)
	for domain, limiter := range d.limiters {
		if limiter.Tokens() >= float64(d.burstSize) {
			delete(d.limiters, domain)
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