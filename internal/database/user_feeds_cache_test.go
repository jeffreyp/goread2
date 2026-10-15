package database

import (
	"testing"
	"time"
)

func TestUserFeedsCacheSetAndGet(t *testing.T) {
	c := newUserFeedsCacheWithLimits(time.Minute, 10)

	feeds := []Feed{{ID: 1, Title: "A"}, {ID: 2, Title: "B"}}
	c.set(7, feeds)

	got, hit := c.get(7)
	if !hit {
		t.Fatal("expected cache hit")
	}
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 2 {
		t.Errorf("unexpected feeds: %+v", got)
	}

	// Returned slice must be a copy; mutating it must not affect the cache.
	got[0].Title = "mutated"
	again, _ := c.get(7)
	if again[0].Title != "A" {
		t.Errorf("cache was mutated via returned slice: %+v", again)
	}
}

func TestUserFeedsCacheGetMiss(t *testing.T) {
	c := newUserFeedsCacheWithLimits(time.Minute, 10)

	if _, hit := c.get(999); hit {
		t.Error("expected cache miss for unknown user")
	}
}

func TestUserFeedsCacheExpiration(t *testing.T) {
	c := newUserFeedsCacheWithLimits(50*time.Millisecond, 10)

	c.set(1, []Feed{{ID: 1}})
	if _, hit := c.get(1); !hit {
		t.Fatal("expected cache hit immediately after set")
	}

	time.Sleep(100 * time.Millisecond)

	if _, hit := c.get(1); hit {
		t.Error("expected cache miss after expiration")
	}
}

func TestUserFeedsCacheInvalidate(t *testing.T) {
	c := newUserFeedsCacheWithLimits(time.Minute, 10)

	c.set(1, []Feed{{ID: 1}})
	c.invalidate(1)

	if _, hit := c.get(1); hit {
		t.Error("expected cache miss after invalidate")
	}

	// Invalidating a user that was never cached must be a no-op, not a panic.
	c.invalidate(42)
}

func TestUserFeedsCacheEvictsClosestToExpiryWhenFull(t *testing.T) {
	c := newUserFeedsCacheWithLimits(time.Minute, 2)

	c.set(1, []Feed{{ID: 1}})
	time.Sleep(time.Millisecond)
	c.set(2, []Feed{{ID: 2}})

	// Cache is now full with 1 (earlier expiry) and 2 (later expiry). Adding a
	// third user must evict 1, the one closest to expiry, not 2.
	c.set(3, []Feed{{ID: 3}})

	if _, hit := c.get(1); hit {
		t.Error("expected user 1 to be evicted as the entry closest to expiry")
	}
	if _, hit := c.get(2); !hit {
		t.Error("expected user 2 to remain cached")
	}
	if _, hit := c.get(3); !hit {
		t.Error("expected newly added user 3 to be cached")
	}
}
