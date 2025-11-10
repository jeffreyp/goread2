package auth

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetSecureClientIP_AppEngine(t *testing.T) {
	// Set App Engine environment
	originalEnv := os.Getenv("GAE_ENV")
	if err := os.Setenv("GAE_ENV", "standard"); err != nil {
		t.Fatalf("Failed to set GAE_ENV: %v", err)
	}
	defer func() { _ = os.Setenv("GAE_ENV", originalEnv) }()

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Appengine-User-Ip", "203.0.113.1")
	c.Request.RemoteAddr = "127.0.0.1:12345"

	// Test App Engine IP extraction
	ip := GetSecureClientIP(c)
	if ip != "203.0.113.1" {
		t.Errorf("Expected IP from X-Appengine-User-Ip, got %s", ip)
	}
}

func TestGetSecureClientIP_Local(t *testing.T) {
	// Ensure we're not in App Engine mode
	originalEnv := os.Getenv("GAE_ENV")
	if err := os.Setenv("GAE_ENV", ""); err != nil {
		t.Fatalf("Failed to set GAE_ENV: %v", err)
	}
	defer func() { _ = os.Setenv("GAE_ENV", originalEnv) }()

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		remoteAddr    string
		expectedIP    string
		xForwardedFor string
	}{
		{
			name:       "IPv4 with port",
			remoteAddr: "192.0.2.1:54321",
			expectedIP: "192.0.2.1",
		},
		{
			name:       "IPv4 without port",
			remoteAddr: "192.0.2.1",
			expectedIP: "192.0.2.1",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[2001:db8::1]:8080",
			expectedIP: "2001:db8::1",
		},
		{
			name:          "Ignores X-Forwarded-For (spoofing protection)",
			remoteAddr:    "192.0.2.1:54321",
			expectedIP:    "192.0.2.1",
			xForwardedFor: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			c.Request.RemoteAddr = tt.remoteAddr

			if tt.xForwardedFor != "" {
				c.Request.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			ip := GetSecureClientIP(c)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestValidateClientIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Public IPs (should be valid)
		{"Public IPv4 - Google DNS", "8.8.8.8", true},
		{"Public IPv4 - Cloudflare", "1.1.1.1", true},
		{"Public IPv4 - Example", "93.184.216.34", true},

		// Private IPs (should be invalid)
		{"Private IPv4 - 10.x", "10.0.0.1", false},
		{"Private IPv4 - 192.168.x", "192.168.1.1", false},
		{"Private IPv4 - 172.16.x", "172.16.0.1", false},
		{"Private IPv4 - 172.31.x", "172.31.255.255", false},

		// Loopback (should be invalid)
		{"Loopback IPv4", "127.0.0.1", false},
		{"Loopback IPv4 alternate", "127.1.2.3", false},

		// Link-local (should be invalid)
		{"Link-local IPv4", "169.254.1.1", false},
		{"Link-local IPv4 - metadata", "169.254.169.254", false},

		// Invalid IPs
		{"Invalid IP", "not-an-ip", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateClientIP(tt.ip)
			if result != tt.expected {
				t.Errorf("ValidateClientIP(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestSanitizeIPForLogging(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "IPv4 address",
			ip:       "192.168.1.100",
			expected: "192.168.1.xxx",
		},
		{
			name:     "Public IPv4",
			ip:       "8.8.8.8",
			expected: "8.8.8.xxx",
		},
		{
			name:     "IPv6 address",
			ip:       "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expected: "2001:0db8:85a3:...",
		},
		{
			name:     "Short IPv6",
			ip:       "2001:db8::1",
			expected: "2001:db8::1",
		},
		{
			name:     "Malformed IP",
			ip:       "not-an-ip",
			expected: "not-an-ip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeIPForLogging(tt.ip)
			if result != tt.expected {
				t.Errorf("SanitizeIPForLogging(%s) = %s, want %s", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestGetSecureClientIP_PreventsSpoofing(t *testing.T) {
	// Ensure we're not in App Engine mode
	originalEnv := os.Getenv("GAE_ENV")
	if err := os.Setenv("GAE_ENV", ""); err != nil {
		t.Fatalf("Failed to set GAE_ENV: %v", err)
	}
	defer func() { _ = os.Setenv("GAE_ENV", originalEnv) }()

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test request with spoofed headers
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.RemoteAddr = "203.0.113.1:12345"          // Real client IP
	c.Request.Header.Set("X-Forwarded-For", "10.0.0.1") // Spoofed private IP
	c.Request.Header.Set("X-Real-IP", "192.168.1.1")    // Spoofed private IP

	// Should return RemoteAddr, not spoofed headers
	ip := GetSecureClientIP(c)
	if ip != "203.0.113.1" {
		t.Errorf("Expected real client IP from RemoteAddr, got %s (spoofing not prevented!)", ip)
	}

	// Verify it didn't use spoofed headers
	if ip == "10.0.0.1" || ip == "192.168.1.1" {
		t.Error("Function used spoofed IP headers instead of RemoteAddr!")
	}
}

func BenchmarkGetSecureClientIP(b *testing.B) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.RemoteAddr = "192.0.2.1:54321"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetSecureClientIP(c)
	}
}

func BenchmarkValidateClientIP(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateClientIP("8.8.8.8")
	}
}
