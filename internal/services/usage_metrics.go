package services

import (
	"context"
	"fmt"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// usageMetricPrefix is the Cloud Monitoring metric type prefix for the
// usage statistics GoRead2 records daily.
const usageMetricPrefix = "custom.googleapis.com/goread2/"

// MetricPoint is one gauge value. Double selects DoubleValue over IntValue.
type MetricPoint struct {
	Type        string            `json:"type"` // relative to usageMetricPrefix
	Labels      map[string]string `json:"labels,omitempty"`
	IntValue    int64             `json:"int_value,omitempty"`
	DoubleValue float64           `json:"double_value,omitempty"`
	Double      bool              `json:"double,omitempty"`
}

func intPoint(metricType string, value int, labels map[string]string) MetricPoint {
	return MetricPoint{Type: metricType, Labels: labels, IntValue: int64(value)}
}

func doublePoint(metricType string, value float64, labels map[string]string) MetricPoint {
	return MetricPoint{Type: metricType, Labels: labels, DoubleValue: value, Double: true}
}

// UsageMetricPoints converts stats into the gauge values recorded in Cloud
// Monitoring. The subscriptions/users states, including admin, partition all
// users, so a stacked chart of them sums to users/total.
func UsageMetricPoints(stats *AdminStats) []MetricPoint {
	sub := stats.Subscriptions
	points := []MetricPoint{
		intPoint("users/total", stats.TotalUsers, nil),
		intPoint("users/active", stats.ActiveUsers.Daily, map[string]string{"window": "day"}),
		intPoint("users/active", stats.ActiveUsers.Weekly, map[string]string{"window": "week"}),
		intPoint("users/active", stats.ActiveUsers.Monthly, map[string]string{"window": "month"}),
		intPoint("users/free_months", sub.FreeMonths, nil),
	}
	// The previous UTC day is the last complete one.
	if n := len(stats.SignupsByDay); n >= 2 {
		points = append(points, intPoint("users/signups_previous_day", stats.SignupsByDay[n-2].Count, nil))
	}
	for _, state := range []struct {
		name  string
		count int
	}{
		{"admin", sub.Admins},
		{"trialing", sub.Trialing},
		{"trial_expired", sub.TrialExpired},
		{"active", sub.Active},
		{"past_due", sub.PastDue},
		{"cancelled", sub.Cancelled},
		{"other", sub.Other},
	} {
		points = append(points, intPoint("subscriptions/users", state.count, map[string]string{"state": state.name}))
	}
	return append(points,
		doublePoint("subscriptions/conversion_rate", sub.ConversionRate, nil),
		intPoint("feeds/active", stats.Feeds.ActiveFeeds, nil),
		intPoint("feeds/subscriptions", stats.Feeds.Subscriptions, nil),
		doublePoint("feeds/per_user", stats.Feeds.MeanPerUser, map[string]string{"statistic": "mean"}),
		doublePoint("feeds/per_user", stats.Feeds.MedianPerUser, map[string]string{"statistic": "median"}),
	)
}

// MetricWriter records gauge values observed at a single time.
type MetricWriter interface {
	WriteGauges(ctx context.Context, points []MetricPoint, at time.Time) error
}

// UsageMetricsService records the admin usage statistics as metrics.
type UsageMetricsService struct {
	stats  *StatsService
	writer MetricWriter
}

// NewUsageMetricsService returns a service that writes to writer. A nil
// writer makes Record compute the points without writing them, which is how
// GoRead2 runs outside App Engine.
func NewUsageMetricsService(stats *StatsService, writer MetricWriter) *UsageMetricsService {
	return &UsageMetricsService{stats: stats, writer: writer}
}

// Record computes fresh statistics and writes them. It returns the points and
// whether they were written.
func (s *UsageMetricsService) Record(ctx context.Context) ([]MetricPoint, bool, error) {
	stats, err := s.stats.Refresh()
	if err != nil {
		return nil, false, err
	}
	points := UsageMetricPoints(stats)
	if s.writer == nil {
		return points, false, nil
	}
	if err := s.writer.WriteGauges(ctx, points, stats.GeneratedAt); err != nil {
		return points, false, fmt.Errorf("failed to write usage metrics: %w", err)
	}
	return points, true, nil
}

// CloudMonitoringWriter writes gauges as Cloud Monitoring custom metrics on
// the global monitored resource. Cloud Monitoring creates each metric
// descriptor on its first write.
type CloudMonitoringWriter struct {
	client    *monitoring.MetricClient
	projectID string
}

// NewCloudMonitoringWriter creates a client using application default
// credentials for the project in GOOGLE_CLOUD_PROJECT.
func NewCloudMonitoringWriter(ctx context.Context) (*CloudMonitoringWriter, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT not set")
	}
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating cloud monitoring client: %w", err)
	}
	return &CloudMonitoringWriter{client: client, projectID: projectID}, nil
}

// Close releases the underlying gRPC connection.
func (w *CloudMonitoringWriter) Close() error {
	return w.client.Close()
}

func (w *CloudMonitoringWriter) WriteGauges(ctx context.Context, points []MetricPoint, at time.Time) error {
	return w.client.CreateTimeSeries(ctx, timeSeriesRequest(w.projectID, points, at))
}

// timeSeriesRequest builds one CreateTimeSeries request. The API accepts up
// to 200 time series per request, well above the number of usage points.
func timeSeriesRequest(projectID string, points []MetricPoint, at time.Time) *monitoringpb.CreateTimeSeriesRequest {
	interval := &monitoringpb.TimeInterval{EndTime: timestamppb.New(at)}
	resource := &monitoredres.MonitoredResource{
		Type:   "global",
		Labels: map[string]string{"project_id": projectID},
	}
	series := make([]*monitoringpb.TimeSeries, len(points))
	for i, p := range points {
		value := &monitoringpb.TypedValue{Value: &monitoringpb.TypedValue_Int64Value{Int64Value: p.IntValue}}
		if p.Double {
			value = &monitoringpb.TypedValue{Value: &monitoringpb.TypedValue_DoubleValue{DoubleValue: p.DoubleValue}}
		}
		series[i] = &monitoringpb.TimeSeries{
			Metric:     &metric.Metric{Type: usageMetricPrefix + p.Type, Labels: p.Labels},
			Resource:   resource,
			MetricKind: metric.MetricDescriptor_GAUGE,
			Points:     []*monitoringpb.Point{{Interval: interval, Value: value}},
		}
	}
	return &monitoringpb.CreateTimeSeriesRequest{
		Name:       "projects/" + projectID,
		TimeSeries: series,
	}
}
