package services

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestNewDomainRateLimiter(t *testing.T) {
	tests := []struct {
		name   string
		config RateLimiterConfig
		expect RateLimiterConfig
	}{
		{
			name: "default values when zero",
			config: RateLimiterConfig{
				RequestsPerMinute: 0,
				BurstSize:         0,
			},
			expect: RateLimiterConfig{
				RequestsPerMinute: 6,
				BurstSize:         1,
			},
		},
		{
			name: "custom values preserved",
			config: RateLimiterConfig{
				RequestsPerMinute: 12,
				BurstSize:         3,
			},
			expect: RateLimiterConfig{
				RequestsPerMinute: 12,
				BurstSize:         3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewDomainRateLimiter(tt.config)

			if limiter.requestsPerMinute != tt.expect.RequestsPerMinute {
				t.Errorf("expected requestsPerMinute %d, got %d",
					tt.expect.RequestsPerMinute, limiter.requestsPerMinute)
			}

			if limiter.burstSize != tt.expect.BurstSize {
				t.Errorf("expected burstSize %d, got %d",
					tt.expect.BurstSize, limiter.burstSize)
			}
		})
	}
}

func TestDomainRateLimiter_ExtractDomain(t *testing.T) {
	limiter := NewDomainRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 6,
		BurstSize:         1,
	})

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "basic HTTP URL",
			url:      "http://example.com/feed.xml",
			expected: "example.com",
		},
		{
			name:     "basic HTTPS URL",
			url:      "https://example.com/feed.xml",
			expected: "example.com",
		},
		{
			name:     "URL with www prefix",
			url:      "https://www.example.com/feed.xml",
			expected: "example.com",
		},
		{
			name:     "URL with port",
			url:      "https://example.com:8080/feed.xml",
			expected: "example.com:8080",
		},
		{
			name:     "URL with subdomain",
			url:      "https://feeds.example.com/rss.xml",
			expected: "feeds.example.com",
		},
		{
			name:     "invalid URL",
			url:      "not-a-url",
			expected: "",
		},
		{
			name:     "empty URL",
			url:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := limiter.extractDomain(tt.url)
			if result != tt.expected {
				t.Errorf("expected domain %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDomainRateLimiter_Allow(t *testing.T) {
	limiter := NewDomainRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 60, // 1 per second for easier testing
		BurstSize:         1,
	})

	t.Run("allows first request", func(t *testing.T) {
		if !limiter.Allow("https://example.com/feed.xml") {
			t.Error("first request should be allowed")
		}
	})

	t.Run("rate limits subsequent requests", func(t *testing.T) {
		// First request should be allowed
		if !limiter.Allow("https://test.com/feed.xml") {
			t.Error("first request should be allowed")
		}

		// Immediate second request should be rate limited
		if limiter.Allow("https://test.com/feed.xml") {
			t.Error("immediate second request should be rate limited")
		}
	})

	t.Run("different domains are independent", func(t *testing.T) {
		// These should both be allowed since they're different domains
		if !limiter.Allow("https://domain1.com/feed.xml") {
			t.Error("first domain should be allowed")
		}

		if !limiter.Allow("https://domain2.com/feed.xml") {
			t.Error("second domain should be allowed")
		}
	})

	t.Run("invalid URL returns false", func(t *testing.T) {
		if limiter.Allow("invalid-url") {
			t.Error("invalid URL should not be allowed")
		}
	})
}

func TestDomainRateLimiter_GetDomainStats(t *testing.T) {
	limiter := NewDomainRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 6,
		BurstSize:         2,
	})

	// Make some requests to populate the limiter
	limiter.Allow("https://example.com/feed.xml")
	limiter.Allow("https://test.com/feed.xml")

	stats := limiter.GetDomainStats()

	if len(stats) != 2 {
		t.Errorf("expected 2 domains in stats, got %d", len(stats))
	}

	if _, exists := stats["example.com"]; !exists {
		t.Error("expected example.com in stats")
	}

	if _, exists := stats["test.com"]; !exists {
		t.Error("expected test.com in stats")
	}

	// Check that stats contain expected fields
	for domain, stat := range stats {
		if stat.Domain != domain {
			t.Errorf("domain mismatch: expected %s, got %s", domain, stat.Domain)
		}

		if stat.BurstLimit != 2 {
			t.Errorf("expected burst limit 2, got %d", stat.BurstLimit)
		}
	}
}

func TestDomainRateLimiter_CleanupOldLimiters(t *testing.T) {
	limiter := NewDomainRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 60,
		BurstSize:         1,
	})

	// Add two limiters
	limiter.Allow("https://example.com/feed.xml")
	limiter.Allow("https://test.com/feed.xml")

	if len(limiter.limiters) != 2 {
		t.Errorf("expected 2 limiters before cleanup, got %d", len(limiter.limiters))
	}

	// Backdate one limiter to beyond the idle threshold; leave the other recent.
	limiter.mu.Lock()
	limiter.lastAccessed["example.com"] = time.Now().Add(-(cleanupIdleThreshold + time.Minute))
	limiter.mu.Unlock()

	limiter.CleanupOldLimiters()

	// Only the stale limiter should be removed.
	if len(limiter.limiters) != 1 {
		t.Errorf("expected 1 limiter after cleanup, got %d", len(limiter.limiters))
	}
	if _, ok := limiter.limiters["test.com"]; !ok {
		t.Error("expected test.com limiter to survive cleanup")
	}

	// Backdate the remaining limiter and verify full cleanup.
	limiter.mu.Lock()
	limiter.lastAccessed["test.com"] = time.Now().Add(-(cleanupIdleThreshold + time.Minute))
	limiter.mu.Unlock()

	limiter.CleanupOldLimiters()

	if len(limiter.limiters) != 0 {
		t.Errorf("expected 0 limiters after full cleanup, got %d", len(limiter.limiters))
	}
}

func TestRateLimitError(t *testing.T) {
	err := &RateLimitError{
		Domain:  "example.com",
		Message: "too many requests",
	}

	expected := "rate limit exceeded for domain example.com: too many requests"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}

func TestDomainRateLimiter_Wait(t *testing.T) {
	limiter := NewDomainRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 6,
		BurstSize:         1,
	})

	t.Run("wait succeeds for valid URL", func(t *testing.T) {
		err := limiter.Wait(context.Background(), "https://example.com/feed.xml")
		if err != nil {
			t.Errorf("wait should succeed for valid URL, got error: %v", err)
		}
	})

	t.Run("wait fails for invalid URL", func(t *testing.T) {
		err := limiter.Wait(context.Background(), "invalid-url")
		if err == nil {
			t.Error("wait should fail for invalid URL")
		}

		if _, ok := err.(*RateLimitError); !ok {
			t.Errorf("expected RateLimitError, got %T", err)
		}
	})
}

// TestDomainRateLimiter_ConcurrentAllow verifies that parallel Allow() calls on
// the same domain don't race or panic. Run with -race.
func TestDomainRateLimiter_ConcurrentAllow(t *testing.T) {
	limiter := NewDomainRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 6000,
		BurstSize:         1000,
	})

	const goroutines = 50
	const callsEach = 20

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < callsEach; j++ {
				limiter.Allow("https://example.com/feed.xml")
			}
		}()
	}

	close(start)
	wg.Wait()
}

// TestDomainRateLimiter_ConcurrentMultipleDomains stresses the double-checked
// lock in getLimiterForDomain by creating many domains simultaneously.
func TestDomainRateLimiter_ConcurrentMultipleDomains(t *testing.T) {
	limiter := NewDomainRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 600,
		BurstSize:         10,
	})

	const goroutines = 100
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			limiter.Allow(fmt.Sprintf("https://unique%d.example.com/feed.xml", id))
		}(i)
	}

	close(start)
	wg.Wait()

	limiter.mu.RLock()
	got := len(limiter.limiters)
	limiter.mu.RUnlock()

	if got != goroutines {
		t.Errorf("expected %d limiters, got %d", goroutines, got)
	}
}

// TestDomainRateLimiter_ConcurrentGetAndCleanup exercises Allow() and
// CleanupOldLimiters() racing against each other to catch map mutation bugs.
func TestDomainRateLimiter_ConcurrentGetAndCleanup(t *testing.T) {
	limiter := NewDomainRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 6000,
		BurstSize:         100,
	})

	const readers = 20
	const cleaners = 5
	const domains = 10

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			url := fmt.Sprintf("https://domain%d.example.com/feed.xml", id%domains)
			for j := 0; j < 50; j++ {
				limiter.Allow(url)
				runtime.Gosched()
			}
		}(i)
	}

	for i := 0; i < cleaners; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 20; j++ {
				limiter.CleanupOldLimiters()
				runtime.Gosched()
			}
		}()
	}

	close(start)
	wg.Wait()
}
