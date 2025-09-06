package services

import (
	"testing"
)

func TestNewFeedDiscovery(t *testing.T) {
	fd := NewFeedDiscovery()

	if fd == nil {
		t.Fatal("NewFeedDiscovery returned nil")
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
			input:       "blog.example.com",
			expected:    "https://blog.example.com",
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
			name:        "localhost",
			input:       "localhost:3000",
			expected:    "https://localhost:3000",
			expectError: false,
		},
		{
			name:        "IP address",
			input:       "192.168.1.1",
			expected:    "https://192.168.1.1",
			expectError: false,
		},
		{
			name:        "IP with port",
			input:       "192.168.1.1:8080",
			expected:    "https://192.168.1.1:8080",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fd.NormalizeURL(tt.input)

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
