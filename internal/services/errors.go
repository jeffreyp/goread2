package services

import "errors"

// Feed-related error types for better error handling and user experience
var (
	// ErrInvalidURL indicates the provided URL is malformed or invalid
	ErrInvalidURL = errors.New("invalid URL")

	// ErrFeedNotFound indicates no RSS/Atom feeds were found at the URL
	ErrFeedNotFound = errors.New("feed not found")

	// ErrFeedTimeout indicates the feed discovery/fetch exceeded the timeout
	ErrFeedTimeout = errors.New("feed timeout")

	// ErrInvalidFeedFormat indicates the feed has invalid XML or unsupported format
	ErrInvalidFeedFormat = errors.New("invalid feed format")

	// ErrNetworkError indicates network-level failures (DNS, connection, etc.)
	ErrNetworkError = errors.New("network error")

	// ErrSSRFBlocked indicates the URL was blocked by SSRF protection
	ErrSSRFBlocked = errors.New("SSRF protection")

	// ErrDatabaseError indicates a database operation failed
	ErrDatabaseError = errors.New("database error")

	// Existing subscription-related errors (already defined elsewhere, documented here for reference)
	// ErrFeedLimitReached - user has reached their feed limit
	// ErrTrialExpired - user's trial has expired
)

// ErrorDetails provides structured error information for API responses
type ErrorDetails struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"error"`
	Details   string `json:"details,omitempty"`
}

// Error codes for frontend consumption
const (
	ErrorCodeInvalidURL        = "invalid_url"
	ErrorCodeFeedNotFound      = "feed_not_found"
	ErrorCodeFeedTimeout       = "feed_timeout"
	ErrorCodeInvalidFormat     = "invalid_feed_format"
	ErrorCodeNetworkError      = "network_error"
	ErrorCodeSSRFBlocked       = "ssrf_blocked"
	ErrorCodeDatabaseError     = "database_error"
	ErrorCodeUnknown           = "unknown_error"
	ErrorCodeLimitReached      = "limit_reached"
	ErrorCodeTrialExpired      = "trial_expired"
	ErrorCodeAlreadySubscribed = "already_subscribed"
)

// GetErrorDetails extracts user-friendly error information from an error
func GetErrorDetails(err error) ErrorDetails {
	if err == nil {
		return ErrorDetails{}
	}

	// Check for wrapped errors
	switch {
	case errors.Is(err, ErrInvalidURL):
		return ErrorDetails{
			ErrorCode: ErrorCodeInvalidURL,
			Message:   "Please enter a valid website URL (e.g., 'example.com' or 'https://example.com/feed.xml')",
			Details:   err.Error(),
		}
	case errors.Is(err, ErrFeedNotFound):
		return ErrorDetails{
			ErrorCode: ErrorCodeFeedNotFound,
			Message:   "No RSS/Atom feeds found on this website. Try entering a direct feed URL instead, or check if the site provides feeds.",
			Details:   err.Error(),
		}
	case errors.Is(err, ErrFeedTimeout):
		return ErrorDetails{
			ErrorCode: ErrorCodeFeedTimeout,
			Message:   "The feed took too long to load. The site might be slow or temporarily unavailable. Please try again later.",
			Details:   err.Error(),
		}
	case errors.Is(err, ErrInvalidFeedFormat):
		return ErrorDetails{
			ErrorCode: ErrorCodeInvalidFormat,
			Message:   "The feed has an invalid format and couldn't be parsed. This might be a temporary issue or the feed may be broken.",
			Details:   err.Error(),
		}
	case errors.Is(err, ErrNetworkError):
		return ErrorDetails{
			ErrorCode: ErrorCodeNetworkError,
			Message:   "Unable to reach the website. Please check the URL and your internet connection, then try again.",
			Details:   err.Error(),
		}
	case errors.Is(err, ErrSSRFBlocked):
		return ErrorDetails{
			ErrorCode: ErrorCodeSSRFBlocked,
			Message:   "This URL cannot be accessed for security reasons. Please use a publicly accessible URL.",
			Details:   err.Error(),
		}
	case errors.Is(err, ErrDatabaseError):
		return ErrorDetails{
			ErrorCode: ErrorCodeDatabaseError,
			Message:   "A database error occurred while adding the feed. Please try again.",
			Details:   err.Error(),
		}
	case errors.Is(err, ErrFeedLimitReached):
		return ErrorDetails{
			ErrorCode: ErrorCodeLimitReached,
			Message:   "You've reached the limit of 20 feeds for free users. Upgrade to Pro for unlimited feeds.",
			Details:   err.Error(),
		}
	case errors.Is(err, ErrTrialExpired):
		return ErrorDetails{
			ErrorCode: ErrorCodeTrialExpired,
			Message:   "Your 30-day free trial has expired. Subscribe to continue using GoRead2.",
			Details:   err.Error(),
		}
	default:
		return ErrorDetails{
			ErrorCode: ErrorCodeUnknown,
			Message:   err.Error(),
			Details:   "",
		}
	}
}
