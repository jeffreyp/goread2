package services

import (
	"fmt"
	"hash/fnv"
	"log"
	"sort"
	"sync"
	"time"

	"goread2/internal/database"
)

// FeedScheduler manages staggered feed updates to prevent DDoS attacks
type FeedScheduler struct {
	rateLimiter *DomainRateLimiter
	feedService *FeedService
	mu          sync.RWMutex
	isRunning   bool
	stopChan    chan struct{}

	// Configuration
	updateWindow    time.Duration // Time window to spread updates across
	minInterval     time.Duration // Minimum time between updates for same feed
	maxConcurrent   int           // Maximum concurrent feed updates
	cleanupInterval time.Duration // How often to cleanup old rate limiters
}

// SchedulerConfig holds configuration for the feed scheduler
type SchedulerConfig struct {
	UpdateWindow    time.Duration // Default: 6 hours
	MinInterval     time.Duration // Default: 30 minutes
	MaxConcurrent   int           // Default: 10
	CleanupInterval time.Duration // Default: 1 hour
}

// ScheduledFeed represents a feed scheduled for update
type ScheduledFeed struct {
	Feed      database.Feed
	NextUpdate time.Time
	Priority   int // Higher number = higher priority
}

// NewFeedScheduler creates a new feed scheduler
func NewFeedScheduler(feedService *FeedService, rateLimiter *DomainRateLimiter, config SchedulerConfig) *FeedScheduler {
	// Set sensible defaults
	if config.UpdateWindow <= 0 {
		config.UpdateWindow = 6 * time.Hour
	}
	if config.MinInterval <= 0 {
		config.MinInterval = 30 * time.Minute
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 10
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 1 * time.Hour
	}

	return &FeedScheduler{
		rateLimiter:     rateLimiter,
		feedService:     feedService,
		updateWindow:    config.UpdateWindow,
		minInterval:     config.MinInterval,
		maxConcurrent:   config.MaxConcurrent,
		cleanupInterval: config.CleanupInterval,
		stopChan:        make(chan struct{}),
	}
}

// Start begins the staggered feed update process
func (fs *FeedScheduler) Start() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	fs.isRunning = true
	go fs.schedulerLoop()
	go fs.cleanupLoop()

	log.Printf("Feed scheduler started with %v update window, %d max concurrent updates",
		fs.updateWindow, fs.maxConcurrent)
	return nil
}

// Stop stops the feed scheduler
func (fs *FeedScheduler) Stop() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !fs.isRunning {
		return
	}

	fs.isRunning = false
	close(fs.stopChan)
	log.Printf("Feed scheduler stopped")
}

// RefreshFeedsStaggered performs a staggered refresh of all feeds
func (fs *FeedScheduler) RefreshFeedsStaggered() error {
	// Get all feeds that need updating
	feeds, err := fs.getAllUniqueFeeds()
	if err != nil {
		return fmt.Errorf("failed to get feeds: %w", err)
	}

	if len(feeds) == 0 {
		log.Printf("No feeds to update")
		return nil
	}

	// Create scheduled feeds with staggered update times
	scheduledFeeds := fs.createStaggeredSchedule(feeds)

	log.Printf("Scheduling %d feeds for staggered updates over %v",
		len(scheduledFeeds), fs.updateWindow)

	// Process feeds according to schedule
	return fs.processScheduledFeeds(scheduledFeeds)
}

// schedulerLoop runs the continuous scheduler
func (fs *FeedScheduler) schedulerLoop() {
	ticker := time.NewTicker(fs.updateWindow)
	defer ticker.Stop()

	for {
		select {
		case <-fs.stopChan:
			return
		case <-ticker.C:
			if err := fs.RefreshFeedsStaggered(); err != nil {
				log.Printf("Scheduled feed refresh failed: %v", err)
			}
		}
	}
}

// cleanupLoop periodically cleans up old rate limiters
func (fs *FeedScheduler) cleanupLoop() {
	ticker := time.NewTicker(fs.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fs.stopChan:
			return
		case <-ticker.C:
			fs.rateLimiter.CleanupOldLimiters()
		}
	}
}

// getAllUniqueFeeds gets all unique feeds from both global and user feeds
func (fs *FeedScheduler) getAllUniqueFeeds() ([]database.Feed, error) {
	// Get all unique feeds from both global feeds and all user feeds
	globalFeeds, err := fs.feedService.GetFeeds()
	if err != nil {
		return nil, err
	}

	// Also get all user feeds to ensure we refresh feeds that users are subscribed to
	allUserFeeds, err := fs.feedService.db.GetAllUserFeeds()
	if err != nil {
		allUserFeeds = []database.Feed{}
	}

	// Combine and deduplicate feeds by URL
	feedMap := make(map[string]database.Feed)

	// Add global feeds
	for _, feed := range globalFeeds {
		feedMap[feed.URL] = feed
	}

	// Add user feeds (will overwrite if same URL, keeping most recent data)
	for _, feed := range allUserFeeds {
		feedMap[feed.URL] = feed
	}

	// Convert back to slice
	feeds := make([]database.Feed, 0, len(feedMap))
	for _, feed := range feedMap {
		feeds = append(feeds, feed)
	}

	return feeds, nil
}

// createStaggeredSchedule creates a staggered schedule for feed updates
func (fs *FeedScheduler) createStaggeredSchedule(feeds []database.Feed) []ScheduledFeed {
	scheduledFeeds := make([]ScheduledFeed, len(feeds))
	now := time.Now()

	for i, feed := range feeds {
		// Calculate staggered delay based on feed ID and last update
		delay := fs.calculateStaggeredDelay(feed.ID, feed.LastFetch)

		// Calculate priority based on update frequency and activity
		priority := fs.calculateFeedPriority(feed)

		scheduledFeeds[i] = ScheduledFeed{
			Feed:       feed,
			NextUpdate: now.Add(delay),
			Priority:   priority,
		}
	}

	// Sort by next update time, then by priority
	sort.Slice(scheduledFeeds, func(i, j int) bool {
		if scheduledFeeds[i].NextUpdate.Equal(scheduledFeeds[j].NextUpdate) {
			return scheduledFeeds[i].Priority > scheduledFeeds[j].Priority
		}
		return scheduledFeeds[i].NextUpdate.Before(scheduledFeeds[j].NextUpdate)
	})

	return scheduledFeeds
}

// calculateStaggeredDelay calculates when a feed should be updated
func (fs *FeedScheduler) calculateStaggeredDelay(feedID int, lastFetch time.Time) time.Duration {
	// Use feed ID hash to distribute evenly across update window
	hash := fnv.New32a()
	hash.Write([]byte(fmt.Sprintf("%d", feedID)))

	// Spread across the update window
	hashValue := hash.Sum32()
	delay := time.Duration(hashValue) % fs.updateWindow

	// Respect minimum intervals (don't hammer recently updated feeds)
	timeSinceUpdate := time.Since(lastFetch)
	if timeSinceUpdate < fs.minInterval {
		additionalDelay := fs.minInterval - timeSinceUpdate
		delay += additionalDelay
	}

	return delay
}

// calculateFeedPriority calculates priority for feed updates
func (fs *FeedScheduler) calculateFeedPriority(feed database.Feed) int {
	priority := 50 // Base priority

	// Higher priority for feeds that haven't been updated in a while
	timeSinceUpdate := time.Since(feed.LastFetch)
	if timeSinceUpdate > 24*time.Hour {
		priority += 30
	} else if timeSinceUpdate > 6*time.Hour {
		priority += 15
	} else if timeSinceUpdate > 2*time.Hour {
		priority += 5
	}

	// Lower priority for feeds that were just updated
	if timeSinceUpdate < fs.minInterval {
		priority -= 20
	}

	return priority
}

// processScheduledFeeds processes feeds according to their schedule
func (fs *FeedScheduler) processScheduledFeeds(scheduledFeeds []ScheduledFeed) error {
	semaphore := make(chan struct{}, fs.maxConcurrent)
	var wg sync.WaitGroup

	for _, scheduled := range scheduledFeeds {
		// Wait until it's time to update this feed
		delay := time.Until(scheduled.NextUpdate)
		if delay > 0 {
			select {
			case <-time.After(delay):
				// Time to update
			case <-fs.stopChan:
				return nil // Scheduler stopped
			}
		}

		// Acquire semaphore for concurrent limit
		select {
		case semaphore <- struct{}{}:
			// Got semaphore
		case <-fs.stopChan:
			return nil // Scheduler stopped
		}

		wg.Add(1)
		go func(feed database.Feed) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			fs.updateSingleFeed(feed)
		}(scheduled.Feed)
	}

	wg.Wait()
	return nil
}

// updateSingleFeed updates a single feed with rate limiting
func (fs *FeedScheduler) updateSingleFeed(feed database.Feed) {
	// Check rate limiting for this domain
	if !fs.rateLimiter.Allow(feed.URL) {
		log.Printf("Rate limited: skipping update for feed %d (%s)", feed.ID, feed.URL)
		return
	}

	log.Printf("Updating feed %d: %s", feed.ID, feed.Title)

	// Fetch and update the feed
	feedData, err := fs.feedService.fetchFeed(feed.URL)
	if err != nil {
		log.Printf("Failed to fetch feed %d (%s): %v", feed.ID, feed.URL, err)
		return
	}

	if err := fs.feedService.saveArticlesFromFeed(feed.ID, feedData); err != nil {
		log.Printf("Failed to save articles for feed %d: %v", feed.ID, err)
		return
	}

	if err := fs.feedService.db.UpdateFeedLastFetch(feed.ID, time.Now()); err != nil {
		log.Printf("Failed to update last fetch time for feed %d: %v", feed.ID, err)
	}

	log.Printf("Successfully updated feed %d: %s", feed.ID, feed.Title)
}

// GetSchedulerStatus returns the current status of the scheduler
func (fs *FeedScheduler) GetSchedulerStatus() SchedulerStatus {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	return SchedulerStatus{
		IsRunning:       fs.isRunning,
		UpdateWindow:    fs.updateWindow,
		MinInterval:     fs.minInterval,
		MaxConcurrent:   fs.maxConcurrent,
		CleanupInterval: fs.cleanupInterval,
	}
}

// SchedulerStatus holds the current status of the scheduler
type SchedulerStatus struct {
	IsRunning       bool          `json:"is_running"`
	UpdateWindow    time.Duration `json:"update_window"`
	MinInterval     time.Duration `json:"min_interval"`
	MaxConcurrent   int           `json:"max_concurrent"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}