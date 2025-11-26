package services

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// dnsLookupTimeout is the maximum time to wait for DNS resolution
	// This prevents hanging on slow or unresponsive DNS servers
	dnsLookupTimeout = 5 * time.Second
)

// URLValidator provides SSRF protection for feed URL validation
type URLValidator struct {
	// AllowedSchemes restricts the URL schemes that can be used
	AllowedSchemes map[string]bool
	// BlockedNetworks contains IP ranges that should not be accessed
	BlockedNetworks []*net.IPNet
}

// NewURLValidator creates a new URL validator with secure defaults
func NewURLValidator() *URLValidator {
	validator := &URLValidator{
		AllowedSchemes: map[string]bool{
			"http":  true,
			"https": true,
		},
		BlockedNetworks: make([]*net.IPNet, 0),
	}

	// Add blocked IP ranges for SSRF protection
	blockedRanges := []string{
		// IPv4 Loopback
		"127.0.0.0/8",
		// IPv4 Private networks (RFC 1918)
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		// IPv4 Link-local
		"169.254.0.0/16",
		// IPv4 Multicast
		"224.0.0.0/4",
		// IPv4 Reserved
		"240.0.0.0/4",
		// IPv6 Loopback
		"::1/128",
		// IPv6 Link-local
		"fe80::/10",
		// IPv6 Unique local addresses
		"fc00::/7",
		// IPv6 Multicast
		"ff00::/8",
	}

	for _, cidr := range blockedRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			validator.BlockedNetworks = append(validator.BlockedNetworks, network)
		}
	}

	return validator
}

// ValidateURL validates a URL for SSRF protection
func (v *URLValidator) ValidateURL(ctx context.Context, rawURL string) error {
	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check scheme
	if !v.AllowedSchemes[parsedURL.Scheme] {
		return fmt.Errorf("URL scheme '%s' not allowed (only http/https permitted)", parsedURL.Scheme)
	}

	// Check host is present
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	// Extract hostname and port
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}

	// Check for IP address directly in URL
	if ip := net.ParseIP(hostname); ip != nil {
		if err := v.isBlockedIP(ip); err != nil {
			return err
		}
	}

	// Resolve DNS to check actual IP addresses with timeout
	// Create a context with timeout to prevent hanging on slow DNS servers
	dnsCtx, cancel := context.WithTimeout(ctx, dnsLookupTimeout)
	defer cancel()

	resolver := &net.Resolver{}
	ips, err := resolver.LookupIP(dnsCtx, "ip", hostname)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %s: %w", hostname, err)
	}

	if len(ips) == 0 {
		return fmt.Errorf("no IP addresses found for hostname %s", hostname)
	}

	// Check all resolved IPs
	for _, ip := range ips {
		if err := v.isBlockedIP(ip); err != nil {
			return fmt.Errorf("hostname %s resolves to blocked IP %s: %w", hostname, ip, err)
		}
	}

	return nil
}

// isBlockedIP checks if an IP address is in a blocked network range
func (v *URLValidator) isBlockedIP(ip net.IP) error {
	// Check against all blocked networks
	for _, network := range v.BlockedNetworks {
		if network.Contains(ip) {
			return fmt.Errorf("IP address %s is in blocked network range %s (SSRF protection)", ip, network)
		}
	}
	return nil
}

// CreateSecureHTTPClient creates an HTTP client with SSRF protection
func (v *URLValidator) CreateSecureHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Limit redirect chain length
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects (max 10)")
			}

			// Validate each redirect destination
			// Use request context for DNS lookups
			if err := v.ValidateURL(req.Context(), req.URL.String()); err != nil {
				return fmt.Errorf("redirect to blocked URL: %w", err)
			}

			return nil
		},
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			// Disable HTTP/2 for more predictable behavior
			ForceAttemptHTTP2: false,
		},
	}
}

// ValidateAndNormalizeURL validates a URL and normalizes it (adds https:// if needed)
func (v *URLValidator) ValidateAndNormalizeURL(ctx context.Context, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	// Add protocol if missing - default to https
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// Validate the URL
	if err := v.ValidateURL(ctx, rawURL); err != nil {
		return "", err
	}

	return rawURL, nil
}
