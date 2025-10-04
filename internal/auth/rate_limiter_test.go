package auth

import (
	"testing"
	"time"

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
