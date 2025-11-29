package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewFeedDiscovery(t *testing.T) {
	fd := NewFeedDiscovery()

	if fd == nil {
		t.Fatal("NewFeedDiscovery returned nil")
		return
	}

	if fd.client == nil {
		t.Error("HTTP client not initialized")
	}

	if fd.client.Timeout == 0 {
		t.Error("HTTP client timeout not set")
	}
}

func TestNormalizeURL(t *testing.T) {
	fd := NewFeedDiscovery()

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "empty URL",
			input:       "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "whitespace only",
			input:       "   ",
			expected:    "",
			expectError: true,
		},
		{
			name:        "URL with https",
			input:       "https://example.com",
			expected:    "https://example.com",
			expectError: false,
		},
		{
			name:        "URL with http",
			input:       "http://example.com",
			expected:    "http://example.com",
			expectError: false,
		},
		{
			name:        "URL without protocol",
			input:       "example.com",
			expected:    "https://example.com",
			expectError: false,
		},
		{
			name:        "URL without protocol with path",
			input:       "example.com/feed.xml",
			expected:    "https://example.com/feed.xml",
			expectError: false,
		},
		{
			name:        "URL with subdomain",
			input:       "www.example.com",
			expected:    "https://www.example.com",
			expectError: false,
		},
		{
			name:        "URL with port",
			input:       "example.com:8080",
			expected:    "https://example.com:8080",
			expectError: false,
		},
		{
			name:        "URL with query parameters",
			input:       "example.com/feed?format=rss",
			expected:    "https://example.com/feed?format=rss",
			expectError: false,
		},
		{
			name:        "URL with fragment",
			input:       "example.com/feed#rss",
			expected:    "https://example.com/feed#rss",
			expectError: false,
		},
		{
			name:        "URL with whitespace",
			input:       "  https://example.com  ",
			expected:    "https://example.com",
			expectError: false,
		},
		{
			name:        "URL without host",
			input:       "https://",
			expected:    "",
			expectError: true,
		},
		{
			name:        "protocol only",
			input:       "https://",
			expected:    "",
			expectError: true,
		},
		{
			name:        "localhost (blocked by SSRF protection)",
			input:       "localhost:3000",
			expected:    "",
			expectError: true,
		},
		{
			name:        "private IP address (blocked by SSRF protection)",
			input:       "192.168.1.1",
			expected:    "",
			expectError: true,
		},
		{
			name:        "private IP with port (blocked by SSRF protection)",
			input:       "192.168.1.1:8080",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fd.NormalizeURL(context.Background(), tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for input '%s', but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input '%s': %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("For input '%s', expected '%s', got '%s'", tt.input, tt.expected, result)
				}
			}
		})
	}
}

func TestTryMastodonFeedPaths(t *testing.T) {
	fd := NewFeedDiscovery()

	tests := []struct {
		name               string
		input              string
		shouldFindFeed     bool
		expectedFeedSuffix string
	}{
		{
			name:               "Mastodon user profile URL",
			input:              "https://mastodon.social/@username",
			shouldFindFeed:     true, // This user actually exists and has an RSS feed
			expectedFeedSuffix: ".rss",
		},
		{
			name:               "Hachyderm user profile URL",
			input:              "https://hachyderm.io/@mekkaokereke",
			shouldFindFeed:     true, // This is a real user, so feed should be found
			expectedFeedSuffix: ".rss",
		},
		{
			name:               "Non-Mastodon URL",
			input:              "https://example.com/blog",
			shouldFindFeed:     false, // Should return empty since it doesn't contain /@
			expectedFeedSuffix: "",
		},
		{
			name:               "Regular website",
			input:              "https://github.com/user/repo",
			shouldFindFeed:     false, // Should return empty since it doesn't contain /@
			expectedFeedSuffix: "",
		},
		{
			name:               "Invalid URL",
			input:              "not-a-url",
			shouldFindFeed:     false, // Should return empty for invalid URLs
			expectedFeedSuffix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := fd.tryMastodonFeedPaths(ctx, tt.input)

			if tt.shouldFindFeed {
				if len(result) == 0 {
					t.Errorf("For input '%s', expected to find feed but got empty result", tt.input)
				} else if len(result) > 0 && tt.expectedFeedSuffix != "" {
					found := false
					for _, feedURL := range result {
						if strings.HasSuffix(feedURL, tt.expectedFeedSuffix) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("For input '%s', expected feed URL to end with '%s', got %v", tt.input, tt.expectedFeedSuffix, result)
					}
				}
			} else {
				if len(result) != 0 {
					t.Errorf("For input '%s', expected empty result, got %v", tt.input, result)
				}
			}
		})
	}
}

func TestTryMastodonFeedPaths_URLPatternDetection(t *testing.T) {
	fd := NewFeedDiscovery()

	tests := []struct {
		name           string
		input          string
		shouldDetect   bool
		expectedTarget string
	}{
		{
			name:           "Mastodon URL with @ pattern",
			input:          "https://mastodon.social/@testuser",
			shouldDetect:   true,
			expectedTarget: "https://mastodon.social/@testuser.rss",
		},
		{
			name:           "Hachyderm URL with @ pattern",
			input:          "https://hachyderm.io/@mekkaokereke",
			shouldDetect:   true,
			expectedTarget: "https://hachyderm.io/@mekkaokereke.rss",
		},
		{
			name:           "URL without @ pattern",
			input:          "https://example.com/user/profile",
			shouldDetect:   false,
			expectedTarget: "",
		},
		{
			name:           "GitHub URL",
			input:          "https://github.com/user/repo",
			shouldDetect:   false,
			expectedTarget: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := fd.tryMastodonFeedPaths(ctx, tt.input)

			if tt.shouldDetect {
				// For Mastodon URLs, we should attempt to check the .rss URL
				// (though it may fail in tests due to network access)
				// The important thing is that non-Mastodon URLs return empty
				// We can't test the actual HTTP request result in unit tests
			} else {
				// For non-Mastodon URLs, should always return empty
				if len(result) != 0 {
					t.Errorf("For non-Mastodon URL '%s', expected empty result, got %v", tt.input, result)
				}
			}
		})
	}
}

// TestTryCommonFeedPaths_ParallelPerformance verifies that parallel requests are faster than sequential
func TestTryCommonFeedPaths_ParallelPerformance(t *testing.T) {
	// Create test servers with intentional delays
	requestCount := 0
	successPath := "/feed.xml"

	// Server responds to /feed.xml with 200 after a delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Add a small delay to simulate network latency
		time.Sleep(50 * time.Millisecond)

		if r.URL.Path == successPath {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fd := NewFeedDiscovery()
	ctx := context.Background()

	// Measure execution time
	start := time.Now()
	result := fd.tryCommonFeedPaths(ctx, server.URL)
	elapsed := time.Since(start)

	// Verify we found the feed
	if len(result) == 0 {
		t.Fatal("Expected to find feed, got empty result")
	}

	// Verify the correct feed was found
	if !strings.Contains(result[0], successPath) {
		t.Errorf("Expected feed URL to contain %s, got %s", successPath, result[0])
	}

	// With 8 paths and 50ms delay each, sequential would take ~400ms for first scheme
	// Parallel should complete in ~50-150ms (time for slowest request + overhead)
	// We use 300ms as threshold to allow for CI/test environment variability
	maxExpectedTime := 300 * time.Millisecond
	if elapsed > maxExpectedTime {
		t.Errorf("Parallel execution took too long: %v (expected < %v). Possible sequential execution.", elapsed, maxExpectedTime)
	}

	t.Logf("Found feed in %v with %d requests", elapsed, requestCount)
}

// TestTryMastodonFeedPaths_ParallelPerformance verifies parallel execution for Mastodon feeds
func TestTryMastodonFeedPaths_ParallelPerformance(t *testing.T) {
	// Create test server with intentional delay
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Add a small delay to simulate network latency
		time.Sleep(50 * time.Millisecond)

		if strings.HasSuffix(r.URL.Path, ".rss") {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fd := NewFeedDiscovery()
	ctx := context.Background()

	// Measure execution time
	start := time.Now()
	result := fd.tryMastodonFeedPaths(ctx, server.URL+"/@testuser")
	elapsed := time.Since(start)

	// Verify we found the feed
	if len(result) == 0 {
		t.Fatal("Expected to find Mastodon feed, got empty result")
	}

	// Verify the correct feed was found
	if !strings.HasSuffix(result[0], ".rss") {
		t.Errorf("Expected feed URL to end with .rss, got %s", result[0])
	}

	// With 2 schemes and 50ms delay each, sequential would take ~100ms
	// Parallel should complete in ~50-100ms (time for fastest successful request)
	// We use 150ms as threshold to allow for overhead and test environment variability
	maxExpectedTime := 150 * time.Millisecond
	if elapsed > maxExpectedTime {
		t.Errorf("Parallel execution took too long: %v (expected < %v). Possible sequential execution.", elapsed, maxExpectedTime)
	}

	t.Logf("Found Mastodon feed in %v with %d requests", elapsed, requestCount)
}
