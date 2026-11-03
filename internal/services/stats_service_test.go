package services

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

func TestComputeStats(t *testing.T) {
	now := time.Date(2026, 11, 4, 15, 0, 0, 0, time.UTC) // a Wednesday
	days := func(n int) time.Time { return now.AddDate(0, 0, -n) }
	future := now.AddDate(0, 0, 10)

	infos := []database.UserRetentionInfo{
		// 1: trialing, signed up today, seen an hour ago
		{UserID: 1, SubscriptionStatus: "trial", CreatedAt: now.Add(-2 * time.Hour), TrialEndsAt: future, LastSeenAt: now.Add(-time.Hour)},
		// 2: trial ended, seen 3 days ago
		{UserID: 2, SubscriptionStatus: "trial", CreatedAt: days(40), TrialEndsAt: days(10), LastSeenAt: days(3)},
		// 3: active subscriber, seen 20 days ago
		{UserID: 3, SubscriptionStatus: "active", CreatedAt: days(60), TrialEndsAt: days(30), LastSeenAt: days(20)},
		// 4: cancelled, never seen
		{UserID: 4, SubscriptionStatus: "cancelled", CreatedAt: days(100), TrialEndsAt: days(70)},
		// 5: past due with free months
		{UserID: 5, SubscriptionStatus: "past_due", CreatedAt: days(2), TrialEndsAt: future, FreeMonthsRemaining: 2},
		// 6: admin, excluded from the funnel but counted as active
		{UserID: 6, SubscriptionStatus: "trial", IsAdmin: true, CreatedAt: days(400), TrialEndsAt: days(370), LastSeenAt: days(1).Add(time.Minute)},
		// 7: explicit expired status, seen 31 days ago
		{UserID: 7, SubscriptionStatus: "expired", CreatedAt: days(9), TrialEndsAt: days(1), LastSeenAt: days(31)},
		// 8: unknown status
		{UserID: 8, SubscriptionStatus: "mystery", CreatedAt: days(30), TrialEndsAt: days(0)},
	}
	subs := []database.UserFeed{
		{UserID: 1, FeedID: 100},
		{UserID: 2, FeedID: 100}, {UserID: 2, FeedID: 101}, {UserID: 2, FeedID: 102},
		{UserID: 2, FeedID: 103}, {UserID: 2, FeedID: 104}, {UserID: 2, FeedID: 105},
		{UserID: 3, FeedID: 101},
		{UserID: 99, FeedID: 200}, // deleted user: ignored
	}

	stats := computeStats(infos, subs, now)

	if stats.TotalUsers != 8 {
		t.Errorf("TotalUsers = %d, want 8", stats.TotalUsers)
	}
	if want := (ActiveUserStats{Daily: 2, Weekly: 3, Monthly: 4}); stats.ActiveUsers != want {
		t.Errorf("ActiveUsers = %+v, want %+v", stats.ActiveUsers, want)
	}

	wantSub := SubscriptionStats{
		Admins: 1, Trialing: 1, TrialExpired: 2, Active: 1, PastDue: 1, Cancelled: 1, Other: 1, FreeMonths: 1,
		ConversionRate: 3.0 / 5.0, // active, past due, cancelled of 2 expired + 3 paid
	}
	if stats.Subscriptions != wantSub {
		t.Errorf("Subscriptions = %+v, want %+v", stats.Subscriptions, wantSub)
	}

	if len(stats.SignupsByDay) != statsSignupDays {
		t.Fatalf("len(SignupsByDay) = %d, want %d", len(stats.SignupsByDay), statsSignupDays)
	}
	if first, last := stats.SignupsByDay[0], stats.SignupsByDay[statsSignupDays-1]; first.Date != "2026-10-06" || last.Date != "2026-11-04" {
		t.Errorf("SignupsByDay spans %s to %s, want 2026-10-06 to 2026-11-04", first.Date, last.Date)
	}
	dayCounts := map[string]int{}
	for _, c := range stats.SignupsByDay {
		if c.Count > 0 {
			dayCounts[c.Date] = c.Count
		}
	}
	// User 8 signed up exactly 30 days ago, before the first day in range.
	if want := map[string]int{"2026-11-04": 1, "2026-11-02": 1, "2026-10-26": 1}; !reflect.DeepEqual(dayCounts, want) {
		t.Errorf("SignupsByDay non-zero = %v, want %v", dayCounts, want)
	}

	if len(stats.SignupsByWeek) != statsSignupWeeks {
		t.Fatalf("len(SignupsByWeek) = %d, want %d", len(stats.SignupsByWeek), statsSignupWeeks)
	}
	weekCounts := map[string]int{}
	for _, c := range stats.SignupsByWeek {
		if c.Count > 0 {
			weekCounts[c.Date] = c.Count
		}
	}
	// Weeks start on Monday; 2026-11-02 is the Monday of now's week.
	if want := map[string]int{"2026-11-02": 2, "2026-10-26": 1, "2026-10-05": 1, "2026-09-21": 1, "2026-08-31": 1}; !reflect.DeepEqual(weekCounts, want) {
		t.Errorf("SignupsByWeek non-zero = %v, want %v", weekCounts, want)
	}

	feeds := stats.Feeds
	if feeds.ActiveFeeds != 6 || feeds.Subscriptions != 8 {
		t.Errorf("ActiveFeeds, Subscriptions = %d, %d; want 6, 8", feeds.ActiveFeeds, feeds.Subscriptions)
	}
	if feeds.MeanPerUser != 1.0 || feeds.MedianPerUser != 0 {
		t.Errorf("Mean, median = %v, %v; want 1, 0", feeds.MeanPerUser, feeds.MedianPerUser)
	}
	wantBuckets := []BucketCount{{"0", 5}, {"1-5", 2}, {"6-10", 1}, {"11-20", 0}, {"21-50", 0}, {"51+", 0}}
	if !reflect.DeepEqual(feeds.PerUser, wantBuckets) {
		t.Errorf("PerUser = %v, want %v", feeds.PerUser, wantBuckets)
	}
}

func TestComputeStatsEmpty(t *testing.T) {
	stats := computeStats(nil, nil, time.Now())
	if stats.TotalUsers != 0 || stats.Subscriptions.ConversionRate != 0 || stats.Feeds.MeanPerUser != 0 || stats.Feeds.MedianPerUser != 0 {
		t.Errorf("Expected zero stats, got %+v", stats)
	}
	if len(stats.Feeds.PerUser) != len(feedBuckets)+1 {
		t.Errorf("Expected %d empty buckets, got %v", len(feedBuckets)+1, stats.Feeds.PerUser)
	}
}

func TestMedianEvenCount(t *testing.T) {
	if got := median(map[int]int{1: 1, 2: 4, 3: 2, 4: 10}); got != 3 {
		t.Errorf("median = %v, want 3", got)
	}
}

// mockDBStats serves GetStats's two queries and counts them. It embeds
// database.Database so that any other call panics.
type mockDBStats struct {
	database.Database
	infoErr   error
	subsErr   error
	infoCalls int
	subsCalls int
}

func (m *mockDBStats) GetUserRetentionInfo() ([]database.UserRetentionInfo, error) {
	m.infoCalls++
	return []database.UserRetentionInfo{{UserID: 1, SubscriptionStatus: "active"}}, m.infoErr
}

func (m *mockDBStats) GetAllUserFeedSubscriptions() ([]database.UserFeed, error) {
	m.subsCalls++
	return []database.UserFeed{{UserID: 1, FeedID: 1}}, m.subsErr
}

func TestStatsServiceCaches(t *testing.T) {
	db := &mockDBStats{}
	service := NewStatsService(db)
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	first, err := service.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	now = now.Add(statsCacheTTL - time.Second)
	if second, err := service.GetStats(); err != nil || second != first {
		t.Errorf("Expected the cached stats within the TTL, got %p (err %v), want %p", second, err, first)
	}
	if db.infoCalls != 1 || db.subsCalls != 1 {
		t.Errorf("Expected one query of each kind, got %d and %d", db.infoCalls, db.subsCalls)
	}

	now = now.Add(time.Second)
	third, err := service.GetStats()
	if err != nil || third == first || !third.GeneratedAt.Equal(now) {
		t.Errorf("Expected fresh stats after the TTL, got %+v (err %v)", third, err)
	}
	if db.infoCalls != 2 || db.subsCalls != 2 {
		t.Errorf("Expected two queries of each kind, got %d and %d", db.infoCalls, db.subsCalls)
	}
}

func TestStatsServiceErrors(t *testing.T) {
	boom := errors.New("boom")
	for name, db := range map[string]*mockDBStats{
		"users":         {infoErr: boom},
		"subscriptions": {subsErr: boom},
	} {
		t.Run(name, func(t *testing.T) {
			service := NewStatsService(db)
			if _, err := service.GetStats(); !errors.Is(err, boom) {
				t.Fatalf("Expected wrapped error, got %v", err)
			}
			// A failure is not cached.
			db.infoErr, db.subsErr = nil, nil
			if _, err := service.GetStats(); err != nil {
				t.Errorf("Expected success after the failure cleared, got %v", err)
			}
		})
	}
}
