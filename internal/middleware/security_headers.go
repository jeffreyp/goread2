package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders returns middleware that sets HTTP security headers on all responses.
//
// CSP defaults to report-only mode. Set CSP_ENFORCE=true to block violations.
// In report-only mode, browsers log violations to the console but don't block
// resources, letting you audit what would break before enforcing.
func SecurityHeaders() gin.HandlerFunc {
	isProduction := os.Getenv("GAE_ENV") == "standard"
	enforceCSP := os.Getenv("CSP_ENFORCE") == "true"

	// Content-Security-Policy directives:
	// - script-src: 'unsafe-inline' required for Google Analytics bootstrap snippet
	// - style-src: 'unsafe-inline' required for inline styles in templates and article content
	// - img-src: http: and https: because RSS article content has images from arbitrary domains
	// - connect-src: GA4 uses regional subdomains (region1.google-analytics.com, etc.)
	csp := "default-src 'self'; " +
		"script-src 'self' 'unsafe-inline' https://www.googletagmanager.com https://cdn.jsdelivr.net; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: http: https:; " +
		"connect-src 'self' https://*.google-analytics.com https://*.analytics.google.com https://*.googletagmanager.com; " +
		"font-src 'self' https://fonts.gstatic.com; " +
		"frame-ancestors 'self'; " +
		"base-uri 'self'; " +
		"form-action 'self'"

	cspHeader := "Content-Security-Policy-Report-Only"
	if enforceCSP {
		cspHeader = "Content-Security-Policy"
	}

	return func(c *gin.Context) {
		// Prevent clickjacking — browsers will refuse to render in iframe unless same origin
		c.Header("X-Frame-Options", "SAMEORIGIN")

		// Prevent MIME-type sniffing — stops browsers interpreting JSON/text as executable
		c.Header("X-Content-Type-Options", "nosniff")

		// Control referrer — send origin only to cross-origin, full URL to same-origin
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Restrict browser features — disable APIs this app never uses
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=()")

		// Content Security Policy (report-only by default, set CSP_ENFORCE=true to block)
		c.Header(cspHeader, csp)

		// HSTS — force HTTPS for all future visits (only in production over TLS)
		if isProduction {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
