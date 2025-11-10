package services

import (
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
	if testing.Short() {
		t.Skip("skipping time-dependent test in short mode")
	}

	// Use higher rate for faster testing
	limiter := NewDomainRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 3600, // 1 per second for fast refill
		BurstSize:         1,
	})

	// Add some limiters
	limiter.Allow("https://example.com/feed.xml")
	limiter.Allow("https://test.com/feed.xml")

	// Should have 2 limiters
	if len(limiter.limiters) != 2 {
		t.Errorf("expected 2 limiters before cleanup, got %d", len(limiter.limiters))
	}

	// Wait for tokens to refill to full capacity
	time.Sleep(2 * time.Second)

	// Cleanup should remove limiters at full capacity
	limiter.CleanupOldLimiters()

	// Should have 0 limiters after cleanup
	if len(limiter.limiters) != 0 {
		t.Errorf("expected 0 limiters after cleanup, got %d", len(limiter.limiters))
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
		err := limiter.Wait("https://example.com/feed.xml")
		if err != nil {
			t.Errorf("wait should succeed for valid URL, got error: %v", err)
		}
	})

	t.Run("wait fails for invalid URL", func(t *testing.T) {
		err := limiter.Wait("invalid-url")
		if err == nil {
			t.Error("wait should fail for invalid URL")
		}

		if _, ok := err.(*RateLimitError); !ok {
			t.Errorf("expected RateLimitError, got %T", err)
		}
	})
}
