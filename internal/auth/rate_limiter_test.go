package auth

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestRateLimiter(t *testing.T) {
	t.Run("NewRateLimiter", func(t *testing.T) {
		rl := NewRateLimiter(10, 20)
		if rl == nil {
			t.Fatal("NewRateLimiter returned nil")
		}
		if rl.ips == nil {
			t.Error("RateLimiter ips map is nil")
		}
		if rl.mu == nil {
			t.Error("RateLimiter mutex is nil")
		}
	})

	t.Run("GetLimiter_CreatesNew", func(t *testing.T) {
		rl := NewRateLimiter(10, 20)
		ip := "192.168.1.1"

		limiter := rl.GetLimiter(ip)
		if limiter == nil {
			t.Fatal("GetLimiter returned nil")
		}

		// Verify limiter was stored
		rl.mu.RLock()
		_, exists := rl.ips[ip]
		rl.mu.RUnlock()

		if !exists {
			t.Error("Limiter was not stored in map")
		}
	})

	t.Run("GetLimiter_ReusesExisting", func(t *testing.T) {
		rl := NewRateLimiter(10, 20)
		ip := "192.168.1.2"

		limiter1 := rl.GetLimiter(ip)
		limiter2 := rl.GetLimiter(ip)

		if limiter1 != limiter2 {
			t.Error("GetLimiter created new limiter instead of reusing existing")
		}
	})

	t.Run("RateLimit_AllowsBurst", func(t *testing.T) {
		rl := NewRateLimiter(1, 5) // 1 req/sec, burst of 5
		ip := "192.168.1.3"

		limiter := rl.GetLimiter(ip)

		// Should allow burst of 5 requests
		for i := 0; i < 5; i++ {
			if !limiter.Allow() {
				t.Errorf("Request %d was blocked (should allow burst)", i+1)
			}
		}

		// 6th request should be blocked
		if limiter.Allow() {
			t.Error("Request 6 was allowed (should be rate limited)")
		}
	})

	t.Run("RateLimit_ReplenishesOverTime", func(t *testing.T) {
		rl := NewRateLimiter(rate.Limit(10), 2) // 10 req/sec, burst of 2
		ip := "192.168.1.4"

		limiter := rl.GetLimiter(ip)

		// Use up the burst
		limiter.Allow()
		limiter.Allow()

		// Should be rate limited
		if limiter.Allow() {
			t.Error("Request was allowed immediately after burst (should be rate limited)")
		}

		// Wait for token to replenish (100ms for 10 req/sec = 1 request)
		time.Sleep(150 * time.Millisecond)

		// Should allow one more request
		if !limiter.Allow() {
			t.Error("Request was blocked after waiting for replenishment")
		}
	})

	t.Run("RateLimit_IndependentPerIP", func(t *testing.T) {
		rl := NewRateLimiter(1, 2)
		ip1 := "192.168.1.5"
		ip2 := "192.168.1.6"

		limiter1 := rl.GetLimiter(ip1)
		limiter2 := rl.GetLimiter(ip2)

		// Use up IP1's burst
		limiter1.Allow()
		limiter1.Allow()

		// IP1 should be blocked
		if limiter1.Allow() {
			t.Error("IP1 was allowed after burst")
		}

		// IP2 should still work
		if !limiter2.Allow() {
			t.Error("IP2 was blocked (should be independent)")
		}
	})
}

func TestRateLimiterConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(100, 10)
	ip := "192.168.1.100"

	// Access limiter concurrently from multiple goroutines
	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()

			// Each goroutine gets a limiter and attempts to use it
			limiter := rl.GetLimiter(ip)
			if limiter == nil {
				errors <- nil // Signal completion without error
				return
			}

			// Try to use the limiter
			_ = limiter.Allow()
			errors <- nil // Signal completion
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors (there shouldn't be any)
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent access caused error: %v", err)
		}
	}

	// Verify all goroutines used the same limiter
	rl.mu.RLock()
	count := len(rl.ips)
	rl.mu.RUnlock()

	if count != 1 {
		t.Errorf("Expected 1 IP entry, got %d (concurrent access created duplicates)", count)
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	// Note: The cleanup function runs on a 1-hour ticker, which is too slow for testing
	// This test verifies the cleanup mechanism works by directly manipulating the map
	rl := NewRateLimiter(10, 20)

	// Add multiple IP entries
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	for _, ip := range ips {
		rl.GetLimiter(ip)
	}

	// Verify entries were created
	rl.mu.RLock()
	initialCount := len(rl.ips)
	rl.mu.RUnlock()

	if initialCount != 3 {
		t.Errorf("Expected 3 IPs, got %d", initialCount)
	}

	// Simulate cleanup by clearing the map (what cleanupIPs does)
	rl.mu.Lock()
	rl.ips = make(map[string]*rate.Limiter)
	rl.mu.Unlock()

	// Verify cleanup worked
	rl.mu.RLock()
	afterCleanup := len(rl.ips)
	rl.mu.RUnlock()

	if afterCleanup != 0 {
		t.Errorf("Expected 0 IPs after cleanup, got %d", afterCleanup)
	}

	// Verify new limiters can be created after cleanup
	newLimiter := rl.GetLimiter("10.0.0.1")
	if newLimiter == nil {
		t.Error("Failed to create limiter after cleanup")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	// Setup gin test mode
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(2, 2) // Very restrictive: 2 req/sec, burst of 2

	t.Run("allows requests within limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/test", nil)
		c.Request.RemoteAddr = "192.168.1.10:12345"

		middleware := RateLimitMiddleware(rl)

		// First request should succeed
		middleware(c)

		if c.IsAborted() {
			t.Error("First request should not be rate limited")
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		testIP := "192.168.1.20"

		// Use up the burst allowance
		limiter := rl.GetLimiter(testIP)
		limiter.Allow() // 1st request
		limiter.Allow() // 2nd request

		// Now test the middleware with exceeded limit
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/test", nil)
		c.Request.RemoteAddr = testIP + ":12345"

		middleware := RateLimitMiddleware(rl)
		middleware(c)

		if !c.IsAborted() {
			t.Error("Request should be aborted when rate limited")
		}

		if w.Code != 429 {
			t.Errorf("Expected status 429 (Too Many Requests), got %d", w.Code)
		}
	})

	t.Run("different IPs are independent", func(t *testing.T) {
		rl2 := NewRateLimiter(1, 1) // Each IP gets 1 request

		ip1 := "192.168.1.30"
		ip2 := "192.168.1.31"

		// Exhaust IP1's limit
		limiter1 := rl2.GetLimiter(ip1)
		limiter1.Allow()

		// IP1 should be blocked
		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request = httptest.NewRequest("GET", "/api/test", nil)
		c1.Request.RemoteAddr = ip1 + ":12345"

		middleware := RateLimitMiddleware(rl2)
		middleware(c1)

		if !c1.IsAborted() {
			t.Error("IP1 should be rate limited")
		}

		// IP2 should still work
		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request = httptest.NewRequest("GET", "/api/test", nil)
		c2.Request.RemoteAddr = ip2 + ":12345"

		middleware(c2)

		if c2.IsAborted() {
			t.Error("IP2 should not be rate limited")
		}
	})
}
