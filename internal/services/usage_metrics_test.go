package services

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"google.golang.org/genproto/googleapis/api/metric"
)

func TestUsageMetricPoints(t *testing.T) {
	stats := &AdminStats{
		TotalUsers:   20,
		SignupsByDay: []DatedCount{{"2026-11-02", 9}, {"2026-11-03", 4}, {"2026-11-04", 1}},
		ActiveUsers:  ActiveUserStats{Daily: 3, Weekly: 7, Monthly: 12},
		Subscriptions: SubscriptionStats{
			Admins: 1, Trialing: 5, TrialExpired: 6, Active: 4, PastDue: 2, Cancelled: 1, Other: 1, FreeMonths: 3,
			ConversionRate: 0.5,
		},
		Feeds: FeedStats{ActiveFeeds: 40, Subscriptions: 90, MeanPerUser: 4.5, MedianPerUser: 3},
	}

	got := map[string]MetricPoint{}
	stateSum := int64(0)
	for _, p := range UsageMetricPoints(stats) {
		key := p.Type
		for _, k := range []string{"window", "state", "statistic"} {
			if v, ok := p.Labels[k]; ok {
				key += "{" + v + "}"
			}
		}
		if _, dup := got[key]; dup {
			t.Errorf("Duplicate point %s", key)
		}
		got[key] = p
		if p.Type == "subscriptions/users" {
			stateSum += p.IntValue
		}
	}

	wantInts := map[string]int64{
		"users/total":                        20,
		"users/active{day}":                  3,
		"users/active{week}":                 7,
		"users/active{month}":                12,
		"users/free_months":                  3,
		"users/signups_previous_day":         4,
		"subscriptions/users{admin}":         1,
		"subscriptions/users{trialing}":      5,
		"subscriptions/users{trial_expired}": 6,
		"subscriptions/users{active}":        4,
		"subscriptions/users{past_due}":      2,
		"subscriptions/users{cancelled}":     1,
		"subscriptions/users{other}":         1,
		"feeds/active":                       40,
		"feeds/subscriptions":                90,
	}
	wantDoubles := map[string]float64{
		"subscriptions/conversion_rate": 0.5,
		"feeds/per_user{mean}":          4.5,
		"feeds/per_user{median}":        3,
	}
	if len(got) != len(wantInts)+len(wantDoubles) {
		t.Errorf("Got %d points, want %d: %v", len(got), len(wantInts)+len(wantDoubles), got)
	}
	for key, want := range wantInts {
		if p, ok := got[key]; !ok || p.Double || p.IntValue != want {
			t.Errorf("%s = %+v (found %v), want int %d", key, p, ok, want)
		}
	}
	for key, want := range wantDoubles {
		if p, ok := got[key]; !ok || !p.Double || p.DoubleValue != want {
			t.Errorf("%s = %+v (found %v), want double %v", key, p, ok, want)
		}
	}
	if stateSum != int64(stats.TotalUsers) {
		t.Errorf("subscriptions/users states sum to %d, want users/total %d", stateSum, stats.TotalUsers)
	}
}

func TestTimeSeriesRequest(t *testing.T) {
	at := time.Date(2026, 11, 4, 0, 30, 0, 0, time.UTC)
	req := timeSeriesRequest("proj", []MetricPoint{
		intPoint("users/active", 3, map[string]string{"window": "day"}),
		doublePoint("subscriptions/conversion_rate", 0.25, nil),
	}, at)

	if req.Name != "projects/proj" || len(req.TimeSeries) != 2 {
		t.Fatalf("Unexpected request: %v", req)
	}
	first, second := req.TimeSeries[0], req.TimeSeries[1]
	if first.Metric.Type != "custom.googleapis.com/goread2/users/active" ||
		!reflect.DeepEqual(first.Metric.Labels, map[string]string{"window": "day"}) {
		t.Errorf("Unexpected metric %v", first.Metric)
	}
	if first.Resource.Type != "global" || first.Resource.Labels["project_id"] != "proj" {
		t.Errorf("Unexpected resource %v", first.Resource)
	}
	if first.MetricKind != metric.MetricDescriptor_GAUGE {
		t.Errorf("MetricKind = %v, want GAUGE", first.MetricKind)
	}
	point := first.Points[0]
	if point.Value.GetInt64Value() != 3 || !point.Interval.EndTime.AsTime().Equal(at) || point.Interval.StartTime != nil {
		t.Errorf("Unexpected point %v", point)
	}
	if v := second.Points[0].Value.GetDoubleValue(); v != 0.25 {
		t.Errorf("Double value = %v, want 0.25", v)
	}
}

type fakeMetricWriter struct {
	points []MetricPoint
	at     time.Time
	err    error
}

func (w *fakeMetricWriter) WriteGauges(_ context.Context, points []MetricPoint, at time.Time) error {
	w.points, w.at = points, at
	return w.err
}

func TestUsageMetricsServiceRecord(t *testing.T) {
	now := time.Date(2026, 11, 4, 0, 30, 0, 0, time.UTC)
	newService := func(db *mockDBStats, writer MetricWriter) *UsageMetricsService {
		stats := NewStatsService(db)
		stats.now = func() time.Time { return now }
		return NewUsageMetricsService(stats, writer)
	}

	t.Run("writes fresh stats", func(t *testing.T) {
		db := &mockDBStats{}
		writer := &fakeMetricWriter{}
		service := newService(db, writer)
		if _, err := service.stats.GetStats(); err != nil { // warm the cache
			t.Fatal(err)
		}
		points, written, err := service.Record(context.Background())
		if err != nil || !written {
			t.Fatalf("Record = written %v, err %v", written, err)
		}
		if db.infoCalls != 2 {
			t.Errorf("Expected Record to bypass the cache, got %d queries", db.infoCalls)
		}
		if !reflect.DeepEqual(writer.points, points) || !writer.at.Equal(now) {
			t.Errorf("Writer got %d points at %v, want %d at %v", len(writer.points), writer.at, len(points), now)
		}
	})

	t.Run("nil writer computes without writing", func(t *testing.T) {
		points, written, err := newService(&mockDBStats{}, nil).Record(context.Background())
		if err != nil || written || len(points) == 0 {
			t.Errorf("Record = %d points, written %v, err %v", len(points), written, err)
		}
	})

	t.Run("writer error", func(t *testing.T) {
		boom := errors.New("boom")
		_, written, err := newService(&mockDBStats{}, &fakeMetricWriter{err: boom}).Record(context.Background())
		if !errors.Is(err, boom) || written {
			t.Errorf("Record = written %v, err %v; want wrapped boom", written, err)
		}
	})

	t.Run("stats error", func(t *testing.T) {
		boom := errors.New("boom")
		writer := &fakeMetricWriter{}
		_, written, err := newService(&mockDBStats{infoErr: boom}, writer).Record(context.Background())
		if !errors.Is(err, boom) || written || writer.points != nil {
			t.Errorf("Record = written %v, err %v, wrote %v", written, err, writer.points)
		}
	})
}
