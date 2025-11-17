package services

import (
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestNewURLValidator(t *testing.T) {
	validator := NewURLValidator()

	if validator == nil {
		t.Fatal("NewURLValidator returned nil")
		return
	}

	if validator.AllowedSchemes == nil {
		t.Fatal("AllowedSchemes map is nil")
	}

	if !validator.AllowedSchemes["http"] {
		t.Error("http scheme should be allowed")
	}

	if !validator.AllowedSchemes["https"] {
		t.Error("https scheme should be allowed")
	}

	if validator.AllowedSchemes["file"] {
		t.Error("file scheme should not be allowed")
	}

	if len(validator.BlockedNetworks) == 0 {
		t.Error("BlockedNetworks should not be empty")
	}
}

func TestValidateURL_ValidURLs(t *testing.T) {
	validator := NewURLValidator()

	validURLs := []string{
		"https://example.com",
		"http://example.com",
		"https://example.com:8080",
		"https://example.com/path",
		"https://example.com/path?query=value",
		"https://www.example.com",
		"https://google.com/feed.xml",
		"http://feeds.feedburner.com/example",
	}

	for _, url := range validURLs {
		t.Run(url, func(t *testing.T) {
			err := validator.ValidateURL(url)
			if err != nil {
				t.Errorf("ValidateURL(%q) returned error: %v", url, err)
			}
		})
	}
}

func TestValidateURL_InvalidSchemes(t *testing.T) {
	validator := NewURLValidator()

	invalidSchemes := []string{
		"file:///etc/passwd",
		"ftp://example.com",
		"gopher://example.com",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
	}

	for _, url := range invalidSchemes {
		t.Run(url, func(t *testing.T) {
			err := validator.ValidateURL(url)
			if err == nil {
				t.Errorf("ValidateURL(%q) should have returned an error for invalid scheme", url)
			}
			if !strings.Contains(err.Error(), "scheme") && !strings.Contains(err.Error(), "not allowed") {
				t.Errorf("Error message should mention scheme, got: %v", err)
			}
		})
	}
}

func TestValidateURL_LoopbackAddresses(t *testing.T) {
	validator := NewURLValidator()

	loopbackURLs := []string{
		"http://127.0.0.1",
		"http://127.0.0.1:8080",
		"http://127.1.2.3",
		"http://localhost", // This should fail DNS resolution to 127.0.0.1
		"http://[::1]",     // IPv6 loopback
	}

	for _, url := range loopbackURLs {
		t.Run(url, func(t *testing.T) {
			err := validator.ValidateURL(url)
			if err == nil {
				t.Errorf("ValidateURL(%q) should have blocked loopback address", url)
			}
			if !strings.Contains(err.Error(), "blocked") {
				t.Errorf("Error should mention 'blocked', got: %v", err)
			}
		})
	}
}

func TestValidateURL_PrivateNetworks(t *testing.T) {
	validator := NewURLValidator()

	privateURLs := []string{
		// Class A private network (10.0.0.0/8)
		"http://10.0.0.1",
		"http://10.255.255.255",
		// Class B private network (172.16.0.0/12)
		"http://172.16.0.1",
		"http://172.31.255.255",
		// Class C private network (192.168.0.0/16)
		"http://192.168.0.1",
		"http://192.168.255.255",
	}

	for _, url := range privateURLs {
		t.Run(url, func(t *testing.T) {
			err := validator.ValidateURL(url)
			if err == nil {
				t.Errorf("ValidateURL(%q) should have blocked private network address", url)
			}
			if !strings.Contains(err.Error(), "blocked") {
				t.Errorf("Error should mention 'blocked', got: %v", err)
			}
		})
	}
}

func TestValidateURL_LinkLocalAddresses(t *testing.T) {
	validator := NewURLValidator()

	linkLocalURLs := []string{
		// IPv4 link-local (169.254.0.0/16) - AWS/GCP metadata service
		"http://169.254.169.254",
		"http://169.254.169.254/latest/meta-data/",
		"http://169.254.0.1",
		// IPv6 link-local (fe80::/10)
		"http://[fe80::1]",
		"http://[fe80::abcd:1234]",
	}

	for _, url := range linkLocalURLs {
		t.Run(url, func(t *testing.T) {
			err := validator.ValidateURL(url)
			if err == nil {
				t.Errorf("ValidateURL(%q) should have blocked link-local address", url)
			}
			if !strings.Contains(err.Error(), "blocked") {
				t.Errorf("Error should mention 'blocked', got: %v", err)
			}
		})
	}
}

func TestValidateURL_MulticastAddresses(t *testing.T) {
	validator := NewURLValidator()

	multicastURLs := []string{
		// IPv4 multicast (224.0.0.0/4)
		"http://224.0.0.1",
		"http://239.255.255.255",
		// IPv6 multicast (ff00::/8)
		"http://[ff00::1]",
		"http://[ff02::1]",
	}

	for _, url := range multicastURLs {
		t.Run(url, func(t *testing.T) {
			err := validator.ValidateURL(url)
			if err == nil {
				t.Errorf("ValidateURL(%q) should have blocked multicast address", url)
			}
		})
	}
}

func TestValidateURL_EmptyOrInvalidURLs(t *testing.T) {
	validator := NewURLValidator()

	invalidURLs := []struct {
		url     string
		errText string
	}{
		{"", "invalid"},
		{"not-a-url", "invalid"},
		{"http://", "host"},
		{"://example.com", "scheme"},
	}

	for _, tc := range invalidURLs {
		t.Run(tc.url, func(t *testing.T) {
			err := validator.ValidateURL(tc.url)
			if err == nil {
				t.Errorf("ValidateURL(%q) should have returned an error", tc.url)
			}
		})
	}
}

func TestIsBlockedIP(t *testing.T) {
	validator := NewURLValidator()

	tests := []struct {
		ip      string
		blocked bool
	}{
		// Should be blocked
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true},
		{"224.0.0.1", true},
		{"::1", true},
		{"fe80::1", true},
		{"fc00::1", true},
		// Should not be blocked (public IPs)
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"93.184.216.34", false}, // example.com
	}

	for _, tc := range tests {
		t.Run(tc.ip, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tc.ip)
			}

			err := validator.isBlockedIP(ip)
			if tc.blocked && err == nil {
				t.Errorf("IP %s should be blocked", tc.ip)
			}
			if !tc.blocked && err != nil {
				t.Errorf("IP %s should not be blocked, got error: %v", tc.ip, err)
			}
		})
	}
}

func TestValidateAndNormalizeURL(t *testing.T) {
	validator := NewURLValidator()

	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		// Valid URLs that should be normalized
		{"example.com", "https://example.com", false},
		{"google.com", "https://google.com", false},
		{"http://example.com", "http://example.com", false},
		{"https://example.com", "https://example.com", false},
		// Invalid URLs
		{"", "", true},
		{"   ", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, err := validator.ValidateAndNormalizeURL(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ValidateAndNormalizeURL(%q) should have returned an error", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAndNormalizeURL(%q) returned unexpected error: %v", tc.input, err)
				}
				if result != tc.expected {
					t.Errorf("ValidateAndNormalizeURL(%q) = %q, want %q", tc.input, result, tc.expected)
				}
			}
		})
	}
}

func TestCreateSecureHTTPClient(t *testing.T) {
	validator := NewURLValidator()
	client := validator.CreateSecureHTTPClient(30)

	if client == nil {
		t.Fatal("CreateSecureHTTPClient returned nil")
		return
	}

	if client.Timeout == 0 {
		t.Error("HTTP client should have a timeout set")
	}

	if client.CheckRedirect == nil {
		t.Error("HTTP client should have a CheckRedirect function")
	}

	if client.Transport == nil {
		t.Error("HTTP client should have a custom transport")
	}
}

func TestSecureHTTPClient_RedirectLimit(t *testing.T) {
	validator := NewURLValidator()

	// Create a mock request chain that exceeds the redirect limit
	var requests []*http.Request
	for i := 0; i < 11; i++ {
		req, _ := http.NewRequest("GET", "https://example.com", nil)
		requests = append(requests, req)
	}

	client := validator.CreateSecureHTTPClient(30)

	// Test that CheckRedirect blocks after 10 redirects
	testReq, _ := http.NewRequest("GET", "https://example.com", nil)
	err := client.CheckRedirect(testReq, requests[:10])
	if err == nil {
		t.Error("CheckRedirect should block after 10 redirects")
	}
	if !strings.Contains(err.Error(), "redirect") {
		t.Errorf("Error should mention redirects, got: %v", err)
	}
}

func TestSSRFProtection_AWSMetadata(t *testing.T) {
	validator := NewURLValidator()

	// Common AWS/GCP metadata endpoints
	metadataURLs := []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://169.254.169.254/computeMetadata/v1/",
		"http://169.254.169.254",
	}

	for _, url := range metadataURLs {
		t.Run(url, func(t *testing.T) {
			err := validator.ValidateURL(url)
			if err == nil {
				t.Errorf("ValidateURL(%q) should block metadata endpoint", url)
			}
			if !strings.Contains(err.Error(), "blocked") {
				t.Errorf("Error should mention 'blocked', got: %v", err)
			}
		})
	}
}

func TestSSRFProtection_InternalServices(t *testing.T) {
	validator := NewURLValidator()

	// Common internal service ports
	internalServices := []string{
		"http://127.0.0.1:6379",   // Redis
		"http://127.0.0.1:5432",   // PostgreSQL
		"http://127.0.0.1:3306",   // MySQL
		"http://127.0.0.1:27017",  // MongoDB
		"http://192.168.1.1:8080", // Internal web service
		"http://10.0.0.5:9200",    // Elasticsearch
		"http://172.16.0.10:8500", // Consul
	}

	for _, url := range internalServices {
		t.Run(url, func(t *testing.T) {
			err := validator.ValidateURL(url)
			if err == nil {
				t.Errorf("ValidateURL(%q) should block internal service", url)
			}
		})
	}
}

func TestValidateURL_IPv6UniqueLocal(t *testing.T) {
	validator := NewURLValidator()

	// IPv6 unique local addresses (fc00::/7) - equivalent to IPv4 private addresses
	uniqueLocalURLs := []string{
		"http://[fc00::1]",
		"http://[fd00::1]",
		"http://[fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff]",
	}

	for _, url := range uniqueLocalURLs {
		t.Run(url, func(t *testing.T) {
			err := validator.ValidateURL(url)
			if err == nil {
				t.Errorf("ValidateURL(%q) should block IPv6 unique local address", url)
			}
		})
	}
}
