package services

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

const (
	// statsCacheTTL is how long StatsService reuses computed statistics, so
	// that repeated views do not scan every User, UserActivity, and UserFeed.
	statsCacheTTL    = 15 * time.Minute
	statsSignupDays  = 30
	statsSignupWeeks = 12
)

// AdminStats is a snapshot of usage statistics built from data GoRead2
// already stores. All dates are in UTC.
type AdminStats struct {
	GeneratedAt   time.Time         `json:"generated_at"`
	TotalUsers    int               `json:"total_users"`
	SignupsByDay  []DatedCount      `json:"signups_by_day"`
	SignupsByWeek []DatedCount      `json:"signups_by_week"`
	ActiveUsers   ActiveUserStats   `json:"active_users"`
	Subscriptions SubscriptionStats `json:"subscriptions"`
	Feeds         FeedStats         `json:"feeds"`
}

// DatedCount is a count for the day, or the week starting on Monday, that
// begins at Date.
type DatedCount struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Count int    `json:"count"`
}

// ActiveUserStats counts users with recorded activity in the last 1, 7, and
// 30 days. Activity is written at most once a day per user, so a user's
// recorded last-seen time can trail their latest request by up to a day and
// the daily figure undercounts accordingly.
type ActiveUserStats struct {
	Daily   int `json:"daily"`
	Weekly  int `json:"weekly"`
	Monthly int `json:"monthly"`
}

// SubscriptionStats counts non-admin users by subscription state. Admins are
// counted separately and excluded from every other field.
type SubscriptionStats struct {
	Admins       int `json:"admins"`
	Trialing     int `json:"trialing"`
	TrialExpired int `json:"trial_expired"`
	Active       int `json:"active"`
	PastDue      int `json:"past_due"` // past_due and unpaid
	Cancelled    int `json:"cancelled"`
	Other        int `json:"other"`
	// FreeMonths counts users with granted free months, whatever their status.
	FreeMonths int `json:"free_months"`
	// ConversionRate is the share of users whose trial has ended who have
	// ever subscribed (active, past due, unpaid, or cancelled). It is zero
	// when no trial has ended.
	ConversionRate float64 `json:"conversion_rate"`
}

// FeedStats describes feed subscriptions.
type FeedStats struct {
	// ActiveFeeds is the number of distinct feeds with at least one subscriber.
	ActiveFeeds   int     `json:"active_feeds"`
	Subscriptions int     `json:"subscriptions"`
	MeanPerUser   float64 `json:"mean_per_user"`
	MedianPerUser float64 `json:"median_per_user"`
	// PerUser is the distribution of feeds per user, in ascending buckets.
	PerUser []BucketCount `json:"per_user"`
}

// BucketCount is the number of users whose feed count falls in Bucket.
type BucketCount struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

// feedBuckets are the upper bounds (inclusive) of the feeds-per-user
// distribution; the last bucket is open-ended.
var feedBuckets = []int{0, 5, 10, 20, 50}

// StatsService computes AdminStats and caches the result for statsCacheTTL.
type StatsService struct {
	db  database.Database
	now func() time.Time

	mu       sync.Mutex
	cached   *AdminStats
	cachedAt time.Time
}

func NewStatsService(db database.Database) *StatsService {
	return &StatsService{db: db, now: time.Now}
}

// GetStats returns the cached statistics if they are fresh, and computes
// them otherwise. The returned value must not be modified.
func (s *StatsService) GetStats() (*AdminStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cached != nil && s.now().Sub(s.cachedAt) < statsCacheTTL {
		return s.cached, nil
	}
	return s.refreshLocked()
}

// Refresh computes the statistics regardless of the cache and caches the
// result. The returned value must not be modified.
func (s *StatsService) Refresh() (*AdminStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refreshLocked()
}

func (s *StatsService) refreshLocked() (*AdminStats, error) {
	now := s.now()
	infos, err := s.db.GetUserRetentionInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to load users: %w", err)
	}
	subs, err := s.db.GetAllUserFeedSubscriptions()
	if err != nil {
		return nil, fmt.Errorf("failed to load feed subscriptions: %w", err)
	}

	stats := computeStats(infos, subs, now)
	s.cached = stats
	s.cachedAt = now
	return stats, nil
}

func computeStats(infos []database.UserRetentionInfo, subs []database.UserFeed, now time.Time) *AdminStats {
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	thisWeek := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7)) // Monday

	stats := &AdminStats{
		GeneratedAt:   now,
		TotalUsers:    len(infos),
		SignupsByDay:  datedCounts(today, statsSignupDays, 1),
		SignupsByWeek: datedCounts(thisWeek, statsSignupWeeks, 7),
	}
	firstDay := today.AddDate(0, 0, -(statsSignupDays - 1))
	firstWeek := thisWeek.AddDate(0, 0, -7*(statsSignupWeeks-1))

	feedsPerUser := make(map[int]int, len(infos))
	trialsEnded, converted := 0, 0
	for _, info := range infos {
		feedsPerUser[info.UserID] = 0

		created := info.CreatedAt.UTC()
		if !created.Before(firstDay) && !created.After(now) {
			stats.SignupsByDay[int(created.Sub(firstDay)/(24*time.Hour))].Count++
		}
		if !created.Before(firstWeek) && !created.After(now) {
			stats.SignupsByWeek[int(created.Sub(firstWeek)/(7*24*time.Hour))].Count++
		}

		if !info.LastSeenAt.IsZero() {
			since := now.Sub(info.LastSeenAt)
			if since <= 24*time.Hour {
				stats.ActiveUsers.Daily++
			}
			if since <= 7*24*time.Hour {
				stats.ActiveUsers.Weekly++
			}
			if since <= 30*24*time.Hour {
				stats.ActiveUsers.Monthly++
			}
		}

		sub := &stats.Subscriptions
		if info.IsAdmin {
			sub.Admins++
			continue
		}
		if info.FreeMonthsRemaining > 0 {
			sub.FreeMonths++
		}
		paid := false
		switch info.SubscriptionStatus {
		case "trial", "":
			if now.After(info.TrialEndsAt) {
				sub.TrialExpired++
				trialsEnded++
			} else {
				sub.Trialing++
			}
		case "expired":
			sub.TrialExpired++
			trialsEnded++
		case "active":
			sub.Active++
			paid = true
		case "past_due", "unpaid":
			sub.PastDue++
			paid = true
		case "cancelled":
			sub.Cancelled++
			paid = true
		default:
			sub.Other++
		}
		if paid {
			trialsEnded++
			converted++
		}
	}
	if trialsEnded > 0 {
		stats.Subscriptions.ConversionRate = float64(converted) / float64(trialsEnded)
	}

	activeFeeds := map[int]bool{}
	for _, sub := range subs {
		if _, ok := feedsPerUser[sub.UserID]; !ok {
			continue // subscription left behind by a deleted user
		}
		feedsPerUser[sub.UserID]++
		activeFeeds[sub.FeedID] = true
		stats.Feeds.Subscriptions++
	}
	stats.Feeds.ActiveFeeds = len(activeFeeds)
	stats.Feeds.PerUser = feedDistribution(feedsPerUser)
	if len(feedsPerUser) > 0 {
		stats.Feeds.MeanPerUser = float64(stats.Feeds.Subscriptions) / float64(len(feedsPerUser))
		stats.Feeds.MedianPerUser = median(feedsPerUser)
	}
	return stats
}

// datedCounts returns n zero counts for consecutive periods of stepDays days,
// ending with the period that starts at last.
func datedCounts(last time.Time, n, stepDays int) []DatedCount {
	counts := make([]DatedCount, n)
	for i := range counts {
		counts[i].Date = last.AddDate(0, 0, -stepDays*(n-1-i)).Format("2006-01-02")
	}
	return counts
}

func feedDistribution(feedsPerUser map[int]int) []BucketCount {
	buckets := make([]BucketCount, len(feedBuckets)+1)
	low := 0
	for i, high := range feedBuckets {
		if low == high {
			buckets[i].Bucket = fmt.Sprintf("%d", high)
		} else {
			buckets[i].Bucket = fmt.Sprintf("%d-%d", low, high)
		}
		low = high + 1
	}
	buckets[len(feedBuckets)].Bucket = fmt.Sprintf("%d+", low)

	for _, n := range feedsPerUser {
		i := sort.SearchInts(feedBuckets, n) // first bound >= n
		buckets[i].Count++
	}
	return buckets
}

func median(feedsPerUser map[int]int) float64 {
	counts := make([]int, 0, len(feedsPerUser))
	for _, n := range feedsPerUser {
		counts = append(counts, n)
	}
	sort.Ints(counts)
	mid := len(counts) / 2
	if len(counts)%2 == 1 {
		return float64(counts[mid])
	}
	return float64(counts[mid-1]+counts[mid]) / 2
}
