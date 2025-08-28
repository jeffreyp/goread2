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

	// First check if the input URL itself is already a valid feed
	if fd.isValidFeed(normalizedURL) {
		return []string{normalizedURL}, nil
	}

	// Try common feed paths (faster and more reliable than checking the main URL)
	commonFeeds := fd.tryCommonFeedPaths(normalizedURL)
	if len(commonFeeds) > 0 {
		return commonFeeds, nil
	}

	// If no common paths worked, try to discover feeds from the page
	feedURLs, err := fd.discoverFeedsFromHTML(normalizedURL)
	if err != nil {
		
		// If everything fails, return some educated guesses for both HTTP and HTTPS
		var guessedFeeds []string
		if parsedURL, err := url.Parse(normalizedURL); err == nil {
			schemes := []string{"https", "http"}
			paths := []string{"/feed", "/feed.xml", "/rss.xml", "/atom.xml"}
			
			for _, scheme := range schemes {
				baseSchemeHost := scheme + "://" + parsedURL.Host
				for _, path := range paths {
					guessedFeeds = append(guessedFeeds, baseSchemeHost+path)
				}
			}
		}
		
		return guessedFeeds, nil
	}

	// Return discovered feeds from HTML parsing
	if len(feedURLs) > 0 {
		return feedURLs, nil
	}

	return nil, fmt.Errorf("no feeds found for %s", normalizedURL)
}



// discoverFeedsFromHTML parses HTML to find feed links
func (fd *FeedDiscovery) discoverFeedsFromHTML(urlStr string) ([]string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Try both HTTPS and HTTP
	schemes := []string{"https", "http"}
	host := parsedURL.Host
	
	for _, scheme := range schemes {
		testURL := scheme + "://" + host
		
		// Create a context with timeout for this request
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		
		req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
		if err != nil {
			cancel()
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoRead/2.0)")
		
		resp, err := fd.client.Do(req)
		if err != nil {
			cancel()
			continue
		}
		
		if resp.StatusCode != 200 {
			_ = resp.Body.Close()
			cancel()
			continue
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()
		
		if err != nil {
			continue
		}

		// Successfully got HTML, now extract feed links
		return fd.extractFeedLinksFromHTML(string(body), testURL)
	}
	
	return nil, fmt.Errorf("unable to fetch HTML from %s using HTTP or HTTPS", host)
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

	// Try both HTTP and HTTPS schemes
	schemes := []string{"https", "http"}
	host := parsedURL.Host

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
	
	// Try each scheme (HTTPS first, then HTTP)
	for _, scheme := range schemes {
		baseSchemeHost := scheme + "://" + host
		
		for _, path := range commonPaths {
			feedURL := baseSchemeHost + path
			
			// Quick check if URL returns 200 with short timeout
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			req, err := http.NewRequestWithContext(ctx, "HEAD", feedURL, nil)
			if err != nil {
				cancel()
				continue
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoRead/2.0)")
			
			resp, err := fd.client.Do(req)
			cancel()
			
			if err != nil {
				continue
			}
			
			if resp != nil {
				_ = resp.Body.Close()
				if resp.StatusCode == 200 {
					// For performance, trust that HEAD requests to feed paths are valid
					// This avoids additional validation requests in production
					validFeeds = append(validFeeds, feedURL)
					// Stop after finding the first working feed for faster discovery
					break
				}
			}
		}
		
		// If we found feeds with the first scheme (HTTPS), don't try HTTP
		if len(validFeeds) > 0 {
			break
		}
	}

	return validFeeds
}

// isValidFeed checks if a given URL is a valid RSS/Atom feed
func (fd *FeedDiscovery) isValidFeed(feedURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoRead/2.0)")
	
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return false
	}
	
	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "xml") || 
	   strings.Contains(contentType, "rss") || 
	   strings.Contains(contentType, "atom") {
		return true
	}
	
	// If content type is not indicative, check the first few bytes of content
	body := make([]byte, 1024)
	n, err := resp.Body.Read(body)
	if err != nil && err != io.EOF {
		return false
	}
	
	content := string(body[:n])
	content = strings.ToLower(strings.TrimSpace(content))
	
	// Check for XML declaration and RSS/Atom root elements
	return strings.Contains(content, "<?xml") && 
		   (strings.Contains(content, "<rss") || 
		    strings.Contains(content, "<feed") ||
		    strings.Contains(content, "<atom"))
}