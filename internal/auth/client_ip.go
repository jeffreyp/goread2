package auth

import (
	"net"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetSecureClientIP returns the client's real IP address with validation
// to prevent IP spoofing attacks. This function:
// 1. On App Engine: Uses X-Appengine-User-Ip (trusted by Google)
// 2. On local: Uses RemoteAddr directly (no trust of X-Forwarded-For)
//
// This prevents attackers from spoofing their IP to bypass rate limiting
// or obscure their identity in audit logs.
func GetSecureClientIP(c *gin.Context) string {
	// On App Engine, trust X-Appengine-User-Ip which is set by Google's infrastructure
	// and cannot be spoofed by clients
	if os.Getenv("GAE_ENV") == "standard" {
		if ip := c.GetHeader("X-Appengine-User-Ip"); ip != "" {
			return ip
		}
	}

	// For local/self-hosted: use RemoteAddr which is the actual TCP connection IP
	// This cannot be spoofed by HTTP headers
	remoteAddr := c.Request.RemoteAddr

	// RemoteAddr is in format "IP:port", extract just the IP
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If parsing fails, return the raw RemoteAddr
		// This handles cases where there's no port (shouldn't happen with HTTP)
		return remoteAddr
	}

	return ip
}

// ValidateClientIP validates that an IP address is not from a private/internal range
// This is used as an additional security measure for sensitive operations
func ValidateClientIP(ip string) bool {
	// Parse the IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check if IP is from a private/internal range
	// These ranges should not be accessing the service from external networks
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16", // Link-local
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return false // Private IP
		}
	}

	return true // Public IP
}

// SanitizeIPForLogging sanitizes an IP address for safe logging
// Returns the IP with the last octet replaced with 'xxx' for privacy
func SanitizeIPForLogging(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		// IPv4: replace last octet
		return strings.Join(parts[:3], ".") + ".xxx"
	}

	// IPv6: only truncate if longer than 20 chars (full addresses)
	// Short compressed IPv6 (like 2001:db8::1) should be left as-is
	if strings.Contains(ip, ":") && len(ip) > 20 {
		colonCount := 0
		for i, ch := range ip {
			if ch == ':' {
				colonCount++
				if colonCount == 3 {
					// Return up to and including the third colon
					return ip[:i+1] + "..."
				}
			}
		}
	}

	// Malformed or short IPv6: return as-is
	return ip
}
