package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// FeedDiscovery handles URL normalization and feed discovery
type FeedDiscovery struct {
	client *http.Client
}

func NewFeedDiscovery() *FeedDiscovery {
	return &FeedDiscovery{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NormalizeURL adds protocol if missing and validates the URL
func (fd *FeedDiscovery) NormalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	// Add protocol if missing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		// Try HTTPS first
		rawURL = "https://" + rawURL
	}

	// Validate URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Host == "" {
		return "", fmt.Errorf("URL must have a host")
	}

	return parsedURL.String(), nil
}

// DiscoverFeedURL attempts to find feed URLs from a given URL
func (fd *FeedDiscovery) DiscoverFeedURL(inputURL string) ([]string, error) {
	normalizedURL, err := fd.NormalizeURL(inputURL)
	if err != nil {
		return nil, fmt.Errorf("URL normalization failed: %w", err)
	}

	// Try common feed paths first (faster and more reliable than checking the main URL)
	commonFeeds := fd.tryCommonFeedPaths(normalizedURL)
	if len(commonFeeds) > 0 {
		return commonFeeds, nil
	}

	// Skip checking if main URL is a feed directly since it can hang
	// Most websites won't serve feeds on their main page anyway

	// If no common paths worked, try to discover feeds from the page
	feedURLs, err := fd.discoverFeedsFromHTML(normalizedURL)
	if err != nil {
		
		// If everything fails, return some educated guesses without validation
		baseSchemeHost := normalizedURL
		if parsedURL, err := url.Parse(normalizedURL); err == nil {
			baseSchemeHost = parsedURL.Scheme + "://" + parsedURL.Host
		}
		
		guessedFeeds := []string{
			baseSchemeHost + "/feed",
			baseSchemeHost + "/feed.xml", 
			baseSchemeHost + "/rss.xml",
			baseSchemeHost + "/atom.xml",
		}
		
		return guessedFeeds, nil
	}

	// Return discovered feeds from HTML parsing
	if len(feedURLs) > 0 {
		return feedURLs, nil
	}

	return nil, fmt.Errorf("no feeds found for %s", normalizedURL)
}

// isFeedURL checks if a URL is likely a direct feed URL
func (fd *FeedDiscovery) isFeedURL(urlStr string) bool {
	
	// Create a shorter timeout context for this specific check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "HEAD", urlStr, nil)
	if err != nil {
		return false
	}
	
	resp, err := fd.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	contentType := resp.Header.Get("Content-Type")
	return strings.Contains(contentType, "xml") || 
		   strings.Contains(contentType, "rss") || 
		   strings.Contains(contentType, "atom")
}

// validateFeedURL checks if a URL actually contains valid RSS/Atom content
func (fd *FeedDiscovery) validateFeedURL(urlStr string) bool {
	// Create a context with timeout for validation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return false
	}
	
	resp, err := fd.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	// Read first 1KB to check for RSS/Atom structure
	buffer := make([]byte, 1024)
	n, err := resp.Body.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	content := string(buffer[:n])
	content = strings.ToLower(content)
	
	// Check for RSS/Atom indicators
	return strings.Contains(content, "<rss") || 
		   strings.Contains(content, "<feed") ||
		   strings.Contains(content, "<channel>") ||
		   strings.Contains(content, "xmlns=\"http://www.w3.org/2005/atom\"")
}

// discoverFeedsFromHTML parses HTML to find feed links
func (fd *FeedDiscovery) discoverFeedsFromHTML(urlStr string) ([]string, error) {
	
	// Create a context with timeout for this request
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := fd.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d when fetching %s", resp.StatusCode, urlStr)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return fd.extractFeedLinksFromHTML(string(body), urlStr)
}

// extractFeedLinksFromHTML uses regex to find feed links in HTML
func (fd *FeedDiscovery) extractFeedLinksFromHTML(html, baseURL string) ([]string, error) {
	var feedURLs []string

	// Simple regex to find any link tag with RSS/Atom type
	patterns := []string{
		`<link[^>]*type="application/rss\+xml"[^>]*href="([^"]*)"[^>]*>`,
		`<link[^>]*href="([^"]*)"[^>]*type="application/rss\+xml"[^>]*>`,
		`<link[^>]*type="application/atom\+xml"[^>]*href="([^"]*)"[^>]*>`, 
		`<link[^>]*href="([^"]*)"[^>]*type="application/atom\+xml"[^>]*>`,
	}

	baseURLParsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(`(?i)` + pattern)
		matches := regex.FindAllStringSubmatch(html, -1)
		
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				feedURL := match[1]
				
				// Convert relative URLs to absolute
				if strings.HasPrefix(feedURL, "/") {
					feedURL = baseURLParsed.Scheme + "://" + baseURLParsed.Host + feedURL
				} else if !strings.HasPrefix(feedURL, "http") {
					feedURL = baseURL + "/" + feedURL
				}
				feedURLs = append(feedURLs, feedURL)
			}
		}
	}

	return feedURLs, nil
}

// tryCommonFeedPaths tries common feed paths for a website
func (fd *FeedDiscovery) tryCommonFeedPaths(baseURL string) []string {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	baseSchemeHost := parsedURL.Scheme + "://" + parsedURL.Host

	commonPaths := []string{
		"/feed",
		"/feed.xml",
		"/rss",
		"/rss.xml",
		"/atom.xml",
		"/feeds/all.atom.xml",
		"/feeds/all.rss.xml",
		"/index.xml",
	}

	var validFeeds []string
	for _, path := range commonPaths {
		feedURL := baseSchemeHost + path
		
		// Quick check if URL returns 200 with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		req, err := http.NewRequestWithContext(ctx, "HEAD", feedURL, nil)
		if err != nil {
			cancel()
			continue
		}
		
		resp, err := fd.client.Do(req)
		cancel()
		
		if err != nil {
			continue
		}
		
		if resp != nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				// Do full validation to ensure it's actually a feed
				if fd.validateFeedURL(feedURL) {
					validFeeds = append(validFeeds, feedURL)
				}
			}
		}
	}

	return validFeeds
}