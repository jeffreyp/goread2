package integration

import (
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/test/helpers"
)

// silenceLog discards package-level log output (e.g. the CSRF manager's
// "no secret configured" warning) for the duration of a benchmark. `go test`
// merges the test binary's stdout and stderr into one stream, so left alone
// these log lines land mid-line in the tab-separated "go test -bench" output
// and make it unparseable by benchstat.
func silenceLog(b *testing.B) {
	b.Helper()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(os.Stderr) })
}

// setupPerfFixture builds a database with numFeeds feeds (all subscribed to
// user) and numArticles articles on the first feed. Shared by the benchmarks
// below so each one measures only the operation named, not fixture setup.
func setupPerfFixture(b *testing.B, numFeeds, numArticles int) (testServer *helpers.TestServer, user *database.User, firstFeed *database.Feed) {
	b.Helper()
	silenceLog(b)

	helpers.SetupTestEnv(b)
	b.Cleanup(func() { helpers.CleanupTestEnv(b) })

	testServer = helpers.SetupTestServer(b)
	user = helpers.CreateTestUser(b, testServer.DB, "perf-baseline", "baseline@example.com", "Baseline User")

	for i := 0; i < numFeeds; i++ {
		feed := helpers.CreateTestFeed(b, testServer.DB, "Perf Feed "+strconv.Itoa(i), "https://perf"+strconv.Itoa(i)+".com/feed", "Performance feed")
		if err := testServer.DB.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
			b.Fatalf("Failed to subscribe: %v", err)
		}
		if i == 0 {
			firstFeed = feed
		}
	}

	for i := 0; i < numArticles; i++ {
		helpers.CreateTestArticle(b, testServer.DB, firstFeed.ID, "Article "+strconv.Itoa(i), "https://perf.com/article"+strconv.Itoa(i))
	}

	return testServer, user, firstFeed
}

// BenchmarkGetUserFeeds100 measures querying a user's full feed list once
// they're subscribed to 100 feeds.
func BenchmarkGetUserFeeds100(b *testing.B) {
	testServer, user, _ := setupPerfFixture(b, 100, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := testServer.DB.GetUserFeeds(user.ID); err != nil {
			b.Fatalf("Failed to get feeds: %v", err)
		}
	}
}

// BenchmarkGetUserArticlesPaginated measures a paginated article query against
// a feed with 1000 articles.
func BenchmarkGetUserArticlesPaginated(b *testing.B) {
	testServer, user, _ := setupPerfFixture(b, 1, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := testServer.DB.GetUserArticlesPaginated(user.ID, 50, "", false); err != nil {
			b.Fatalf("Failed to get articles: %v", err)
		}
	}
}

// BenchmarkGetUserUnreadCounts measures computing per-feed unread counts
// across 100 subscribed feeds.
func BenchmarkGetUserUnreadCounts(b *testing.B) {
	testServer, user, _ := setupPerfFixture(b, 100, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := testServer.DB.GetUserUnreadCounts(user.ID); err != nil {
			b.Fatalf("Failed to get unread counts: %v", err)
		}
	}
}

// BenchmarkConcurrentReads measures 10 users concurrently reading their own
// feeds, articles, and unread counts against a shared feed.
func BenchmarkConcurrentReads(b *testing.B) {
	silenceLog(b)

	helpers.SetupTestEnv(b)
	b.Cleanup(func() { helpers.CleanupTestEnv(b) })

	testServer := helpers.SetupTestServer(b)

	const numUsers = 10
	users := make([]*database.User, numUsers)
	for i := 0; i < numUsers; i++ {
		users[i] = helpers.CreateTestUser(b, testServer.DB, "concurrent-"+strconv.Itoa(i), "concurrent"+strconv.Itoa(i)+"@example.com", "Concurrent User "+strconv.Itoa(i))
	}

	feed := helpers.CreateTestFeed(b, testServer.DB, "Shared Feed", "https://shared.com/feed", "Shared feed for concurrent test")
	for _, user := range users {
		if err := testServer.DB.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
			b.Fatalf("Failed to subscribe user: %v", err)
		}
	}
	for i := 0; i < 50; i++ {
		helpers.CreateTestArticle(b, testServer.DB, feed.ID, "Concurrent Article "+strconv.Itoa(i), "https://shared.com/article"+strconv.Itoa(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(numUsers)
		for _, user := range users {
			go func(u *database.User) {
				defer wg.Done()
				_, _ = testServer.DB.GetUserFeeds(u.ID)
				_, _ = testServer.DB.GetUserArticlesPaginated(u.ID, 50, "", false)
				_, _ = testServer.DB.GetUserUnreadCounts(u.ID)
			}(user)
		}
		wg.Wait()
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
				// Mark articles as read
				_ = testServer.DB.MarkUserArticleRead(u.ID, i+1, true)
				wg.Done()
			}
		}(user)
	}

	wg.Wait()
	duration = time.Since(start)
	t.Logf("✅ %d concurrent mark-as-read operations completed in %v", numUsers*10, duration)
}
