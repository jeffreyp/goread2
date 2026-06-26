package services

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

func TestNewFeedScheduler(t *testing.T) {
	rateLimiter := NewDomainRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 6,
		BurstSize:         1,
	})

	tests := []struct {
		name   string
		config SchedulerConfig
		expect SchedulerConfig
	}{
		{
			name: "default values when zero",
			config: SchedulerConfig{
				UpdateWindow:    0,
				MinInterval:     0,
				MaxConcurrent:   0,
				CleanupInterval: 0,
			},
			expect: SchedulerConfig{
				UpdateWindow:    6 * time.Hour,
				MinInterval:     30 * time.Minute,
				MaxConcurrent:   10,
				CleanupInterval: 1 * time.Hour,
			},
		},
		{
			name: "custom values preserved",
			config: SchedulerConfig{
				UpdateWindow:    2 * time.Hour,
				MinInterval:     15 * time.Minute,
				MaxConcurrent:   5,
				CleanupInterval: 30 * time.Minute,
			},
			expect: SchedulerConfig{
				UpdateWindow:    2 * time.Hour,
				MinInterval:     15 * time.Minute,
				MaxConcurrent:   5,
				CleanupInterval: 30 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use nil for feedService since we're only testing config
			scheduler := NewFeedScheduler(nil, rateLimiter, tt.config)

			if scheduler.updateWindow != tt.expect.UpdateWindow {
				t.Errorf("expected updateWindow %v, got %v",
					tt.expect.UpdateWindow, scheduler.updateWindow)
			}

			if scheduler.minInterval != tt.expect.MinInterval {
				t.Errorf("expected minInterval %v, got %v",
					tt.expect.MinInterval, scheduler.minInterval)
			}

			if scheduler.maxConcurrent != tt.expect.MaxConcurrent {
				t.Errorf("expected maxConcurrent %d, got %d",
					tt.expect.MaxConcurrent, scheduler.maxConcurrent)
			}

			if scheduler.cleanupInterval != tt.expect.CleanupInterval {
				t.Errorf("expected cleanupInterval %v, got %v",
					tt.expect.CleanupInterval, scheduler.cleanupInterval)
			}
		})
	}
}

func TestFeedScheduler_CalculateStaggeredDelay(t *testing.T) {
	scheduler := &FeedScheduler{
		updateWindow: 1 * time.Hour,
		minInterval:  30 * time.Minute,
	}

	t.Run("different feed IDs produce different delays", func(t *testing.T) {
		lastFetch := time.Now().Add(-2 * time.Hour) // Old enough to not trigger min interval

		delay1 := scheduler.calculateStaggeredDelay(1, lastFetch)
		delay2 := scheduler.calculateStaggeredDelay(2, lastFetch)

		if delay1 == delay2 {
			t.Error("different feed IDs should produce different delays")
		}

		// Both delays should be within the update window
		if delay1 < 0 || delay1 > 1*time.Hour {
			t.Errorf("delay1 %v should be within update window", delay1)
		}

		if delay2 < 0 || delay2 > 1*time.Hour {
			t.Errorf("delay2 %v should be within update window", delay2)
		}
	})

	t.Run("respects minimum interval", func(t *testing.T) {
		recentFetch := time.Now().Add(-10 * time.Minute) // Recent fetch

		delay := scheduler.calculateStaggeredDelay(1, recentFetch)

		// Should include additional delay for min interval
		expectedMinDelay := 30*time.Minute - 10*time.Minute // 20 minutes additional
		if delay < expectedMinDelay {
			t.Errorf("delay %v should be at least %v to respect min interval", delay, expectedMinDelay)
		}
	})

	t.Run("same feed ID produces same delay", func(t *testing.T) {
		lastFetch := time.Now().Add(-2 * time.Hour)

		delay1 := scheduler.calculateStaggeredDelay(42, lastFetch)
		delay2 := scheduler.calculateStaggeredDelay(42, lastFetch)

		if delay1 != delay2 {
			t.Error("same feed ID should produce same delay")
		}
	})
}

func TestFeedScheduler_CalculateFeedPriority(t *testing.T) {
	scheduler := &FeedScheduler{
		minInterval: 30 * time.Minute,
	}

	tests := []struct {
		name        string
		lastFetch   time.Time
		expectedMin int
		expectedMax int
		description string
	}{
		{
			name:        "very old feed",
			lastFetch:   time.Now().Add(-25 * time.Hour),
			expectedMin: 75, // 50 base + 30 for >24h
			expectedMax: 85,
			description: "feeds not updated in >24h get high priority",
		},
		{
			name:        "moderately old feed",
			lastFetch:   time.Now().Add(-8 * time.Hour),
			expectedMin: 60, // 50 base + 15 for >6h
			expectedMax: 70,
			description: "feeds not updated in >6h get medium priority",
		},
		{
			name:        "somewhat old feed",
			lastFetch:   time.Now().Add(-3 * time.Hour),
			expectedMin: 50, // 50 base + 5 for >2h
			expectedMax: 60,
			description: "feeds not updated in >2h get slight priority boost",
		},
		{
			name:        "recent feed",
			lastFetch:   time.Now().Add(-1 * time.Hour),
			expectedMin: 45, // 50 base, no bonus
			expectedMax: 55,
			description: "recently updated feeds get base priority",
		},
		{
			name:        "very recent feed",
			lastFetch:   time.Now().Add(-10 * time.Minute),
			expectedMin: 25, // 50 base - 20 for too recent
			expectedMax: 35,
			description: "very recently updated feeds get lower priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feed := database.Feed{
				ID:        1,
				LastFetch: tt.lastFetch,
			}

			priority := scheduler.calculateFeedPriority(feed)

			if priority < tt.expectedMin || priority > tt.expectedMax {
				t.Errorf("%s: expected priority between %d and %d, got %d",
					tt.description, tt.expectedMin, tt.expectedMax, priority)
			}
		})
	}
}

func TestFeedScheduler_CreateStaggeredSchedule(t *testing.T) {
	scheduler := &FeedScheduler{
		updateWindow: 1 * time.Hour,
		minInterval:  30 * time.Minute,
	}

	feeds := []database.Feed{
		{ID: 1, Title: "Feed 1", LastFetch: time.Now().Add(-2 * time.Hour)},
		{ID: 2, Title: "Feed 2", LastFetch: time.Now().Add(-4 * time.Hour)},
		{ID: 3, Title: "Feed 3", LastFetch: time.Now().Add(-1 * time.Hour)},
	}

	scheduled := scheduler.createStaggeredSchedule(feeds)

	if len(scheduled) != len(feeds) {
		t.Errorf("expected %d scheduled feeds, got %d", len(feeds), len(scheduled))
	}

	// Check that feeds are scheduled in the future
	now := time.Now()
	for i, sf := range scheduled {
		if sf.NextUpdate.Before(now) {
			t.Errorf("scheduled feed %d should be in the future", i)
		}

		if sf.Feed.ID == 0 {
			t.Errorf("scheduled feed %d should have valid feed data", i)
		}

		if sf.Priority == 0 {
			t.Errorf("scheduled feed %d should have priority calculated", i)
		}
	}

	// Check that feeds are sorted by next update time
	for i := 1; i < len(scheduled); i++ {
		prev := scheduled[i-1]
		curr := scheduled[i]

		if curr.NextUpdate.Before(prev.NextUpdate) {
			// If times are equal, check priority ordering
			if !curr.NextUpdate.Equal(prev.NextUpdate) {
				t.Errorf("feeds should be sorted by next update time")
			} else if curr.Priority > prev.Priority {
				t.Errorf("feeds with same update time should be sorted by priority (high to low)")
			}
		}
	}
}

func TestFeedScheduler_StartStop(t *testing.T) {
	scheduler := &FeedScheduler{
		updateWindow:    time.Hour,   // Need positive values for tickers
		cleanupInterval: time.Minute, // Need positive values for tickers
		stopChan:        make(chan struct{}),
	}

	// Test starting
	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	status := scheduler.GetSchedulerStatus()
	if !status.IsRunning {
		t.Error("scheduler should be running after Start()")
	}

	// Test starting when already running
	err = scheduler.Start()
	if err == nil {
		t.Error("Start should fail when already running")
	}

	// Test stopping
	scheduler.Stop()

	status = scheduler.GetSchedulerStatus()
	if status.IsRunning {
		t.Error("scheduler should not be running after Stop()")
	}

	// Test stopping when already stopped (should not panic)
	scheduler.Stop()
}

func TestFeedScheduler_GetSchedulerStatus(t *testing.T) {
	config := SchedulerConfig{
		UpdateWindow:    2 * time.Hour,
		MinInterval:     15 * time.Minute,
		MaxConcurrent:   5,
		CleanupInterval: 30 * time.Minute,
	}

	scheduler := NewFeedScheduler(nil, nil, config)

	status := scheduler.GetSchedulerStatus()

	if status.IsRunning {
		t.Error("scheduler should not be running initially")
	}

	if status.UpdateWindow != config.UpdateWindow {
		t.Errorf("expected UpdateWindow %v, got %v", config.UpdateWindow, status.UpdateWindow)
	}

	if status.MinInterval != config.MinInterval {
		t.Errorf("expected MinInterval %v, got %v", config.MinInterval, status.MinInterval)
	}

	if status.MaxConcurrent != config.MaxConcurrent {
		t.Errorf("expected MaxConcurrent %d, got %d", config.MaxConcurrent, status.MaxConcurrent)
	}

	if status.CleanupInterval != config.CleanupInterval {
		t.Errorf("expected CleanupInterval %v, got %v", config.CleanupInterval, status.CleanupInterval)
	}
}

// TestFeedScheduler_StopCompletesPromptly verifies that Stop() returns without
// deadlocking and that the WaitGroup goroutines (schedulerLoop + cleanupLoop)
// actually exit. Run with -race.
func TestFeedScheduler_StopCompletesPromptly(t *testing.T) {
	before := runtime.NumGoroutine()

	scheduler := &FeedScheduler{
		updateWindow:    time.Hour,
		cleanupInterval: time.Hour,
		stopChan:        make(chan struct{}),
	}

	if err := scheduler.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Give the two goroutines a moment to reach their select blocks.
	runtime.Gosched()

	done := make(chan struct{})
	go func() {
		scheduler.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return within 2s — possible deadlock or goroutine leak")
	}

	if scheduler.GetSchedulerStatus().IsRunning {
		t.Error("scheduler should not be running after Stop()")
	}

	// Allow goroutines to fully exit, then check for leaks.
	time.Sleep(10 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > before+2 {
		t.Errorf("possible goroutine leak: %d goroutines before Start, %d after Stop", before, after)
	}
}

// TestFeedScheduler_ConcurrentStop verifies that multiple concurrent Stop()
// calls don't panic, deadlock, or close stopChan twice.
func TestFeedScheduler_ConcurrentStop(t *testing.T) {
	scheduler := &FeedScheduler{
		updateWindow:    time.Hour,
		cleanupInterval: time.Hour,
		stopChan:        make(chan struct{}),
	}

	if err := scheduler.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	const callers = 10
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			scheduler.Stop()
		}()
	}

	close(start)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent Stop() calls did not complete within 5s")
	}

	if scheduler.GetSchedulerStatus().IsRunning {
		t.Error("scheduler should not be running after concurrent Stop()")
	}
}
