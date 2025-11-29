package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// FeedDiscovery handles URL normalization and feed discovery
type FeedDiscovery struct {
	client       *http.Client
	urlValidator *URLValidator
}

func NewFeedDiscovery() *FeedDiscovery {
	validator := NewURLValidator()
	return &FeedDiscovery{
		client:       validator.CreateSecureHTTPClient(30 * time.Second),
		urlValidator: validator,
	}
}

// NormalizeURL adds protocol if missing and validates the URL with SSRF protection
func (fd *FeedDiscovery) NormalizeURL(ctx context.Context, rawURL string) (string, error) {
	// Use the URL validator which includes SSRF protection
	normalizedURL, err := fd.urlValidator.ValidateAndNormalizeURL(ctx, rawURL)
	if err != nil {
		// Check if it's an SSRF protection error
		if strings.Contains(err.Error(), "SSRF protection") ||
			strings.Contains(err.Error(), "blocked network") ||
			strings.Contains(err.Error(), "not allowed") {
			return "", fmt.Errorf("%w: %v", ErrSSRFBlocked, err)
		}
		// Check if it's a DNS/network error
		if strings.Contains(err.Error(), "DNS lookup failed") ||
			strings.Contains(err.Error(), "no IP addresses found") {
			return "", fmt.Errorf("%w: %v", ErrNetworkError, err)
		}
		// Otherwise it's an invalid URL
		return "", fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	return normalizedURL, nil
}

// DiscoverFeedURL attempts to find feed URLs from a given URL
func (fd *FeedDiscovery) DiscoverFeedURL(ctx context.Context, inputURL string) ([]string, error) {
	normalizedURL, err := fd.NormalizeURL(ctx, inputURL)
	if err != nil {
		// Errors from NormalizeURL are already wrapped with custom types
		return nil, err
	}

	// First check if the input URL itself is already a valid feed
	if fd.isValidFeed(ctx, normalizedURL) {
		return []string{normalizedURL}, nil
	}

	// Check for Mastodon-style feeds first (e.g., https://mastodon.social/@username.rss)
	mastodonFeeds := fd.tryMastodonFeedPaths(ctx, normalizedURL)
	if len(mastodonFeeds) > 0 {
		return mastodonFeeds, nil
	}

	// Try common feed paths (faster and more reliable than checking the main URL)
	commonFeeds := fd.tryCommonFeedPaths(ctx, normalizedURL)
	if len(commonFeeds) > 0 {
		return commonFeeds, nil
	}

	// If no common paths worked, try to discover feeds from the page
	feedURLs, err := fd.discoverFeedsFromHTML(ctx, normalizedURL)
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

	// No feeds found at all
	return nil, fmt.Errorf("%w: %s", ErrFeedNotFound, normalizedURL)
}

// discoverFeedsFromHTML parses HTML to find feed links
func (fd *FeedDiscovery) discoverFeedsFromHTML(ctx context.Context, urlStr string) ([]string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Try both HTTPS and HTTP
	schemes := []string{"https", "http"}
	host := parsedURL.Host

	for _, scheme := range schemes {
		testURL := scheme + "://" + host

		req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoRead/2.0)")

		resp, err := fd.client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != 200 {
			_ = resp.Body.Close()
			continue
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

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

// tryCommonFeedPaths tries common feed paths for a website in parallel
func (fd *FeedDiscovery) tryCommonFeedPaths(ctx context.Context, baseURL string) []string {
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

	// Build all candidate URLs
	var candidateURLs []string
	for _, scheme := range schemes {
		baseSchemeHost := scheme + "://" + host
		for _, path := range commonPaths {
			candidateURLs = append(candidateURLs, baseSchemeHost+path)
		}
	}

	// Channel to receive the first valid feed URL
	resultChan := make(chan string, 1)
	doneChan := make(chan struct{})
	var wg sync.WaitGroup

	// Launch parallel requests for all candidate URLs
	for _, feedURL := range candidateURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			// Check if we already found a result
			select {
			case <-doneChan:
				return
			default:
			}

			// Quick check if URL returns 200
			req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoRead/2.0)")

			resp, err := fd.client.Do(req)
			if err != nil {
				return
			}

			if resp != nil {
				_ = resp.Body.Close()
				if resp.StatusCode == 200 {
					// Try to send result (non-blocking in case another goroutine won)
					select {
					case resultChan <- url:
					case <-doneChan:
					}
				}
			}
		}(feedURL)
	}

	// Wait for all goroutines in a separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Return the first valid feed found, or empty if none found
	select {
	case feedURL := <-resultChan:
		close(doneChan) // Signal other goroutines to stop
		if feedURL != "" {
			return []string{feedURL}
		}
		return nil
	case <-ctx.Done():
		close(doneChan)
		return nil
	}
}

// tryMastodonFeedPaths tries Mastodon-style RSS feeds (e.g., @username.rss) in parallel
func (fd *FeedDiscovery) tryMastodonFeedPaths(ctx context.Context, baseURL string) []string {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	// Check if this looks like a Mastodon user profile URL (contains @username pattern)
	path := parsedURL.Path
	if !strings.Contains(path, "/@") {
		return nil
	}

	// Try adding .rss to the end of the URL for Mastodon-style feeds
	schemes := []string{"https", "http"}

	// Build all candidate URLs
	var candidateURLs []string
	for _, scheme := range schemes {
		candidateURLs = append(candidateURLs, scheme+"://"+parsedURL.Host+path+".rss")
	}

	// Channel to receive the first valid feed URL
	resultChan := make(chan string, 1)
	doneChan := make(chan struct{})
	var wg sync.WaitGroup

	// Launch parallel requests for all candidate URLs
	for _, feedURL := range candidateURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			// Check if we already found a result
			select {
			case <-doneChan:
				return
			default:
			}

			// Quick check if URL returns 200
			req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoRead/2.0)")

			resp, err := fd.client.Do(req)
			if err != nil {
				return
			}

			if resp != nil {
				_ = resp.Body.Close()
				if resp.StatusCode == 200 {
					// Try to send result (non-blocking in case another goroutine won)
					select {
					case resultChan <- url:
					case <-doneChan:
					}
				}
			}
		}(feedURL)
	}

	// Wait for all goroutines in a separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Return the first valid feed found, or empty if none found
	select {
	case feedURL := <-resultChan:
		close(doneChan) // Signal other goroutines to stop
		if feedURL != "" {
			return []string{feedURL}
		}
		return nil
	case <-ctx.Done():
		close(doneChan)
		return nil
	}
}

// isValidFeed checks if a given URL is a valid RSS/Atom feed
func (fd *FeedDiscovery) isValidFeed(ctx context.Context, feedURL string) bool {
	// Validate URL for SSRF protection
	if err := fd.urlValidator.ValidateURL(ctx, feedURL); err != nil {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoRead/2.0)")

	// Use the secure HTTP client
	resp, err := fd.client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

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
