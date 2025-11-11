package integration

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/test/helpers"
)

// TestPerformanceBaseline establishes performance baselines (not a benchmark, but records timing)
func TestPerformanceBaseline(t *testing.T) {
	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)
	user := helpers.CreateTestUser(t, testServer.DB, "perf-baseline", "baseline@example.com", "Baseline User")

	// Test 1: Create 100 feeds
	start := time.Now()
	for i := 0; i < 100; i++ {
		feed := helpers.CreateTestFeed(t, testServer.DB, "Perf Feed "+strconv.Itoa(i), "https://perf"+strconv.Itoa(i)+".com/feed", "Performance feed")
		err := testServer.DB.SubscribeUserToFeed(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}
	}
	duration := time.Since(start)
	t.Logf("✅ Created and subscribed to 100 feeds in %v (avg: %v per feed)", duration, duration/100)

	// Test 2: Query all feeds
	start = time.Now()
	feeds, err := testServer.DB.GetUserFeeds(user.ID)
	duration = time.Since(start)
	if err != nil {
		t.Fatalf("Failed to get feeds: %v", err)
	}
	t.Logf("✅ Queried %d feeds in %v", len(feeds), duration)

	// Test 3: Create 1000 articles
	if len(feeds) > 0 {
		start = time.Now()
		for i := 0; i < 1000; i++ {
			helpers.CreateTestArticle(t, testServer.DB, feeds[0].ID, "Article "+strconv.Itoa(i), "https://perf.com/article"+strconv.Itoa(i))
		}
		duration = time.Since(start)
		t.Logf("✅ Created 1000 articles in %v (avg: %v per article)", duration, duration/1000)

		// Test 4: Query articles with pagination
		start = time.Now()
		result, err := testServer.DB.GetUserArticlesPaginated(user.ID, 50, "", false)
		duration = time.Since(start)
		if err != nil {
			t.Fatalf("Failed to get articles: %v", err)
		}
		t.Logf("✅ Queried %d articles (paginated) in %v", len(result.Articles), duration)

		// Test 5: Calculate unread counts
		start = time.Now()
		counts, err := testServer.DB.GetUserUnreadCounts(user.ID)
		duration = time.Since(start)
		if err != nil {
			t.Fatalf("Failed to get unread counts: %v", err)
		}
		t.Logf("✅ Calculated unread counts for %d feeds in %v", len(counts), duration)
	}
}

// TestConcurrentUserOperations tests concurrent operations by multiple users
func TestConcurrentUserOperations(t *testing.T) {
	t.Parallel()

	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)

	// Create 10 users
	numUsers := 10
	users := make([]*database.User, numUsers)
	for i := 0; i < numUsers; i++ {
		users[i] = helpers.CreateTestUser(t, testServer.DB, "concurrent-"+strconv.Itoa(i), "concurrent"+strconv.Itoa(i)+"@example.com", "Concurrent User "+strconv.Itoa(i))
	}

	// Create a shared feed
	feed := helpers.CreateTestFeed(t, testServer.DB, "Shared Feed", "https://shared.com/feed", "Shared feed for concurrent test")

	// Subscribe all users to the feed
	for _, user := range users {
		err := testServer.DB.SubscribeUserToFeed(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe user: %v", err)
		}
	}

	// Add articles to the feed
	for i := 0; i < 50; i++ {
		helpers.CreateTestArticle(t, testServer.DB, feed.ID, "Concurrent Article "+strconv.Itoa(i), "https://shared.com/article"+strconv.Itoa(i))
	}

	// Test concurrent read operations
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(numUsers)

	for _, user := range users {
		go func(u *database.User) {
			defer wg.Done()
			// Each user queries their feeds
			_, _ = testServer.DB.GetUserFeeds(u.ID)
			// Each user queries their articles
			_, _ = testServer.DB.GetUserArticlesPaginated(u.ID, 50, "", false)
			// Each user gets unread counts
			_, _ = testServer.DB.GetUserUnreadCounts(u.ID)
		}(user)
	}

	wg.Wait()
	duration := time.Since(start)
	t.Logf("✅ %d users performed concurrent operations in %v (avg: %v per user)", numUsers, duration, duration/time.Duration(numUsers))

	// Test concurrent write operations
	start = time.Now()
	wg.Add(numUsers * 10) // Each user marks 10 articles

	for _, user := range users {
		go func(u *database.User) {
			for i := 0; i < 10; i++ {
				defer wg.Done()
				// Mark articles as read
				_ = testServer.DB.MarkUserArticleRead(u.ID, i+1, true)
			}
		}(user)
	}

	wg.Wait()
	duration = time.Since(start)
	t.Logf("✅ %d concurrent mark-as-read operations completed in %v", numUsers*10, duration)
}
