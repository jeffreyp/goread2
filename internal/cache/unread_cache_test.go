package cache

import (
	"testing"
	"time"
)

func TestNewUnreadCache(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)
	if cache == nil {
		t.Fatal("NewUnreadCache returned nil")
		return
	}
	if cache.ttl != 60*time.Second {
		t.Errorf("Expected TTL 60s, got %v", cache.ttl)
	}
}

func TestUnreadCache_SetAndGet(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)
	userID := 1

	counts := map[int]int{
		10: 5,  // Feed 10 has 5 unread
		20: 10, // Feed 20 has 10 unread
		30: 0,  // Feed 30 has 0 unread
	}

	// Set counts
	cache.Set(userID, counts)

	// Get counts - should be cache hit
	retrieved, hit := cache.Get(userID)
	if !hit {
		t.Fatal("Expected cache hit, got miss")
	}

	if len(retrieved) != len(counts) {
		t.Errorf("Expected %d feeds, got %d", len(counts), len(retrieved))
	}

	for feedID, expectedCount := range counts {
		if retrieved[feedID] != expectedCount {
			t.Errorf("Feed %d: expected count %d, got %d", feedID, expectedCount, retrieved[feedID])
		}
	}
}

func TestUnreadCache_GetMiss(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)

	// Get for non-existent user
	_, hit := cache.Get(999)
	if hit {
		t.Error("Expected cache miss for non-existent user")
	}
}

func TestUnreadCache_Expiration(t *testing.T) {
	cache := NewUnreadCache(100 * time.Millisecond)
	userID := 1

	counts := map[int]int{10: 5}
	cache.Set(userID, counts)

	// Immediate get should hit
	_, hit := cache.Get(userID)
	if !hit {
		t.Error("Expected cache hit immediately after set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Get should miss after expiration
	_, hit = cache.Get(userID)
	if hit {
		t.Error("Expected cache miss after expiration")
	}
}

func TestUnreadCache_UpdateCount(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)
	userID := 1
	feedID := 10

	// Set initial counts
	counts := map[int]int{feedID: 5}
	cache.Set(userID, counts)

	tests := []struct {
		name          string
		wasRead       bool
		nowRead       bool
		expectedCount int
		description   string
	}{
		{
			name:          "Mark as read decrements count",
			wasRead:       false,
			nowRead:       true,
			expectedCount: 4,
			description:   "Unread → Read should decrement",
		},
		{
			name:          "Mark as unread increments count",
			wasRead:       true,
			nowRead:       false,
			expectedCount: 6,
			description:   "Read → Unread should increment",
		},
		{
			name:          "No change when already read",
			wasRead:       true,
			nowRead:       true,
			expectedCount: 5,
			description:   "Read → Read should not change count",
		},
		{
			name:          "No change when already unread",
			wasRead:       false,
			nowRead:       false,
			expectedCount: 5,
			description:   "Unread → Unread should not change count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to known state
			cache.Set(userID, map[int]int{feedID: 5})

			// Update count
			cache.UpdateCount(userID, feedID, tt.wasRead, tt.nowRead)

			// Verify
			retrieved, hit := cache.Get(userID)
			if !hit {
				t.Fatal("Expected cache hit")
			}
			if retrieved[feedID] != tt.expectedCount {
				t.Errorf("%s: expected count %d, got %d", tt.description, tt.expectedCount, retrieved[feedID])
			}
		})
	}
}

func TestUnreadCache_UpdateCountNoCache(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)

	// Update count for user with no cache - should not panic
	cache.UpdateCount(999, 10, false, true)

	// Verify no cache was created
	_, hit := cache.Get(999)
	if hit {
		t.Error("UpdateCount should not create cache for non-existent user")
	}
}

func TestUnreadCache_UpdateCountNegativeSafety(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)
	userID := 1
	feedID := 10

	// Set count to 1
	cache.Set(userID, map[int]int{feedID: 1})

	// Mark as read twice (should not go negative)
	cache.UpdateCount(userID, feedID, false, true) // 1 → 0
	cache.UpdateCount(userID, feedID, false, true) // 0 → 0 (not -1)

	retrieved, _ := cache.Get(userID)
	if retrieved[feedID] < 0 {
		t.Errorf("Count should not go negative, got %d", retrieved[feedID])
	}
	if retrieved[feedID] != 0 {
		t.Errorf("Expected count 0, got %d", retrieved[feedID])
	}
}

func TestUnreadCache_Invalidate(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)
	userID := 1

	// Set counts
	cache.Set(userID, map[int]int{10: 5})

	// Verify cache hit
	_, hit := cache.Get(userID)
	if !hit {
		t.Fatal("Expected cache hit before invalidation")
	}

	// Invalidate
	cache.Invalidate(userID)

	// Verify cache miss
	_, hit = cache.Get(userID)
	if hit {
		t.Error("Expected cache miss after invalidation")
	}
}

func TestUnreadCache_InvalidateAll(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)

	// Set counts for multiple users
	cache.Set(1, map[int]int{10: 5})
	cache.Set(2, map[int]int{20: 10})
	cache.Set(3, map[int]int{30: 15})

	// Verify all cached
	for userID := 1; userID <= 3; userID++ {
		_, hit := cache.Get(userID)
		if !hit {
			t.Errorf("User %d should be cached", userID)
		}
	}

	// Invalidate all
	cache.InvalidateAll()

	// Verify all invalidated
	for userID := 1; userID <= 3; userID++ {
		_, hit := cache.Get(userID)
		if hit {
			t.Errorf("User %d should not be cached after InvalidateAll", userID)
		}
	}
}

func TestUnreadCache_GetStats(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)

	// Empty cache
	stats := cache.GetStats()
	if stats.CachedUsers != 0 || stats.TotalFeeds != 0 {
		t.Errorf("Empty cache should have 0 users and 0 feeds, got %d users and %d feeds",
			stats.CachedUsers, stats.TotalFeeds)
	}

	// Add some users
	cache.Set(1, map[int]int{10: 5, 20: 10})          // User 1: 2 feeds
	cache.Set(2, map[int]int{30: 15, 40: 20, 50: 25}) // User 2: 3 feeds

	stats = cache.GetStats()
	if stats.CachedUsers != 2 {
		t.Errorf("Expected 2 cached users, got %d", stats.CachedUsers)
	}
	if stats.TotalFeeds != 5 {
		t.Errorf("Expected 5 total feeds, got %d", stats.TotalFeeds)
	}
}

func TestUnreadCache_HitMissMetrics(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)

	// Miss on empty cache
	cache.Get(1)
	cache.Get(2)

	stats := cache.GetStats()
	if stats.Hits != 0 {
		t.Errorf("Expected 0 hits, got %d", stats.Hits)
	}
	if stats.Misses != 2 {
		t.Errorf("Expected 2 misses, got %d", stats.Misses)
	}

	// Set and hit
	cache.Set(1, map[int]int{10: 5})
	cache.Get(1) // hit
	cache.Get(1) // hit
	cache.Get(2) // miss

	stats = cache.GetStats()
	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 3 {
		t.Errorf("Expected 3 misses, got %d", stats.Misses)
	}
	// 2 hits out of 5 total = 0.4
	if stats.HitRate < 0.39 || stats.HitRate > 0.41 {
		t.Errorf("Expected HitRate ~0.4, got %f", stats.HitRate)
	}
}

func TestUnreadCache_HitMissExpiration(t *testing.T) {
	cache := NewUnreadCache(100 * time.Millisecond)

	cache.Set(1, map[int]int{10: 5})
	cache.Get(1) // hit

	time.Sleep(150 * time.Millisecond)

	cache.Get(1) // miss (expired)

	stats := cache.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
}

func TestUnreadCache_ConcurrentAccess(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)
	userID := 1
	feedID := 10

	// Set initial counts
	cache.Set(userID, map[int]int{feedID: 100})

	// Concurrent reads and updates
	done := make(chan bool)
	iterations := 100

	// Concurrent readers
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				cache.Get(userID)
			}
			done <- true
		}()
	}

	// Concurrent updaters
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				if j%2 == 0 {
					cache.UpdateCount(userID, feedID, false, true) // Decrement
				} else {
					cache.UpdateCount(userID, feedID, true, false) // Increment
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Should not panic and should have valid data
	retrieved, hit := cache.Get(userID)
	if !hit {
		t.Error("Expected cache hit after concurrent access")
	}
	if retrieved[feedID] < 0 {
		t.Error("Count should not be negative after concurrent access")
	}
}

func TestUnreadCache_IsolationBetweenUsers(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)

	// Set counts for user 1
	cache.Set(1, map[int]int{10: 5})

	// Set counts for user 2
	cache.Set(2, map[int]int{10: 15})

	// Update user 1
	cache.UpdateCount(1, 10, false, true) // Decrement to 4

	// Verify user 1
	counts1, _ := cache.Get(1)
	if counts1[10] != 4 {
		t.Errorf("User 1 feed 10: expected 4, got %d", counts1[10])
	}

	// Verify user 2 unchanged
	counts2, _ := cache.Get(2)
	if counts2[10] != 15 {
		t.Errorf("User 2 feed 10: expected 15 (unchanged), got %d", counts2[10])
	}
}

func TestUnreadCache_GetReturnsCopy(t *testing.T) {
	cache := NewUnreadCache(60 * time.Second)
	userID := 1

	original := map[int]int{10: 5}
	cache.Set(userID, original)

	// Get and modify
	retrieved, _ := cache.Get(userID)
	retrieved[10] = 999
	retrieved[99] = 123

	// Verify cache wasn't modified
	cached, _ := cache.Get(userID)
	if cached[10] != 5 {
		t.Error("Modifying retrieved map should not affect cached data")
	}
	if _, exists := cached[99]; exists {
		t.Error("Adding to retrieved map should not affect cached data")
	}
}
