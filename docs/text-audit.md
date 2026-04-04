# GoRead2 User-Facing Text Audit

_Total strings found: 399_

This report captures all user-visible text in templates, JavaScript, and Go handlers.
Use it to:
- Identify tone/voice inconsistencies
- Prepare strings for localization (i18n)
- Spot redundant or confusing messages


---

## `internal/handlers/admin_handler.go`

_19 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 29 | Go: API response | List users API not yet implemented | `"error": "List users API not yet implemented",` |
| 38 | Go: API response | Email parameter required | `c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter required"})` |
| 47 | Go: API response | Invalid request body | `c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": ` |
| 54 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 61 | Go: API response | User not found | `c.JSON(http.StatusNotFound, gin.H{"error": "User not found", "details": err.Erro` |
| 67 | Go: API response | Cannot remove your own admin privileges | `c.JSON(http.StatusForbidden, gin.H{"error": "Cannot remove your own admin privil` |
| 88 | Go: API response | Failed to set admin status | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set admin statu` |
| 112 | Go: API response | Admin status updated successfully | `"message": "Admin status updated successfully",` |
| 127 | Go: API response | Email parameter required | `c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter required"})` |
| 136 | Go: API response | Invalid request body | `c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": ` |
| 143 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 150 | Go: API response | User not found | `c.JSON(http.StatusNotFound, gin.H{"error": "User not found", "details": err.Erro` |
| 171 | Go: API response | Failed to grant free months | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant free mont` |
| 191 | Go: API response | Free months granted successfully | `"message": "Free months granted successfully",` |
| 242 | Go: API response | Failed to get audit logs | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit logs"` |
| 257 | Go: API response | Email parameter required | `c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter required"})` |
| 264 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 271 | Go: API response | User not found | `c.JSON(http.StatusNotFound, gin.H{"error": "User not found", "details": err.Erro` |
| 278 | Go: API response | Failed to get subscription info | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subscriptio` |

---

## `internal/handlers/auth_handler.go`

_10 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 32 | Go: API response | Failed to generate state | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"` |
| 53 | Go: API response | Invalid state parameter | `c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})` |
| 59 | Go: API response | State parameter has expired or already been used | `c.JSON(http.StatusBadRequest, gin.H{"error": "State parameter has expired or alr` |
| 68 | Go: API response | Missing authorization code | `c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})` |
| 76 | Go: API response | Failed to authenticate | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate"})` |
| 83 | Go: API response | Failed to create session | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"` |
| 106 | Go: API response | Logged out successfully | `c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})` |
| 112 | Go: API response | Not authenticated | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})` |
| 174 | Go: API response | Admin authentication required | `c.JSON(http.StatusForbidden, gin.H{"error": "Admin authentication required"})` |
| 195 | Go: API response | Session cleanup completed | `c.JSON(http.StatusOK, gin.H{"message": "Session cleanup completed"})` |

---

## `internal/handlers/feed_handler.go`

_47 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 38 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 60 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 68 | Go: API response | You've reached the limit of 20 feeds for free users. Upgrade to Pro for unlimited feeds. | `"error":         "You've reached the limit of 20 feeds for free users. Upgrade t` |
| 76 | Go: API response | Your 30-day free trial has expired. Subscribe to continue using GoRead2. | `"error":         "Your 30-day free trial has expired. Subscribe to continue usin` |
| 131 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 138 | Go: API response | Invalid feed ID | `c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feed ID"})` |
| 146 | Go: API response | Feed removed from your subscriptions successfully | `c.JSON(http.StatusOK, gin.H{"message": "Feed removed from your subscriptions suc` |
| 152 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 191 | Go: API response | Invalid feed ID | `c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feed ID"})` |
| 207 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 214 | Go: API response | Invalid article ID | `c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})` |
| 234 | Go: API response | Article updated successfully | `c.JSON(http.StatusOK, gin.H{"message": "Article updated successfully"})` |
| 240 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 247 | Go: API response | Invalid article ID | `c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})` |
| 256 | Go: API response | Article starred status toggled | `c.JSON(http.StatusOK, gin.H{"message": "Article starred status toggled"})` |
| 262 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 273 | Go: API response | All articles marked as read | `"message":        "All articles marked as read",` |
| 294 | Go: API response | Admin authentication required | `c.JSON(http.StatusForbidden, gin.H{"error": "Admin authentication required"})` |
| 314 | Go: API response | Feed refresh started | `c.JSON(http.StatusAccepted, gin.H{"message": "Feed refresh started"})` |
| 335 | Go: API response | Feeds refreshed successfully | `c.JSON(http.StatusOK, gin.H{"message": "Feeds refreshed successfully"})` |
| 354 | Go: API response | Admin authentication required | `c.JSON(http.StatusForbidden, gin.H{"error": "Admin authentication required"})` |
| 373 | Go: API response | Cleanup completed successfully | `"message":       "Cleanup completed successfully",` |
| 381 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 388 | Go: API response | Invalid feed ID | `c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feed ID"})` |
| 395 | Go: API response | Failed to get user feeds | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user feeds"` |
| 402 | Go: API response | Failed to get all articles | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get all article` |
| 409 | Go: API response | Failed to get user articles | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user articl` |
| 438 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 444 | Go: API response | URL parameter required | `c.JSON(http.StatusBadRequest, gin.H{"error": "URL parameter required"})` |
| 451 | Go: API response | Database error | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details` |
| 459 | Go: API response | Article not found in database | `"message": "Article not found in database",` |
| 474 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 481 | Go: API response | Failed to get user feeds | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user feeds"` |
| 532 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 556 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 563 | Go: API response | No OPML file provided | `c.JSON(http.StatusBadRequest, gin.H{"error": "No OPML file provided"})` |
| 570 | Go: API response | File too large (max 10MB) | `c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 10MB)"})` |
| 577 | Go: API response | Failed to read OPML file | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read OPML file"` |
| 586 | Go: API response | Import would exceed your feed limit of 20 feeds. Upgrade to Pro for unlimited feeds. | `"error":          "Import would exceed your feed limit of 20 feeds. Upgrade to P` |
| 595 | Go: API response | Your 30-day free trial has expired. Subscribe to continue using GoRead2. | `"error":         "Your 30-day free trial has expired. Subscribe to continue usin` |
| 605 | Go: API response | OPML imported successfully | `"message":        "OPML imported successfully",` |
| 613 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 621 | Go: API response | Failed to generate OPML export | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OPML e` |
| 637 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 653 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 690 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 709 | Go: API response | Setting updated successfully | `"message":      "Setting updated successfully",` |

---

## `internal/handlers/payment_handler.go`

_13 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 30 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 50 | Go: API response | You already have an active subscription | `c.JSON(http.StatusConflict, gin.H{"error": "You already have an active subscript` |
| 75 | Go: API response | Error reading request body | `c.JSON(http.StatusBadRequest, gin.H{"error": "Error reading request body"})` |
| 84 | Go: API response | Webhook endpoint is not properly configured | `"error": "Webhook endpoint is not properly configured",` |
| 94 | Go: fmt.Sprintf | Webhook signature verification failed: %v | `c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Webhook signature veri` |
| 106 | Go: API response | Error parsing webhook JSON | `c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing webhook JSON"})` |
| 126 | Go: API response | Error parsing webhook JSON | `c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing webhook JSON"})` |
| 154 | Go: API response | Error updating subscription | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating subscripti` |
| 164 | Go: API response | Error parsing webhook JSON | `c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing webhook JSON"})` |
| 178 | Go: API response | Error handling subscription deletion | `c.JSON(http.StatusInternalServerError, gin.H{"error": "Error handling subscripti` |
| 195 | Go: API response | Authentication required | `c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})` |
| 222 | Go: API response | Subscription Successful - GoRead2 | `"title":      "Subscription Successful - GoRead2",` |
| 230 | Go: API response | Subscription Cancelled - GoRead2 | `"title": "Subscription Cancelled - GoRead2",` |

---

## `internal/services/errors.go`

_8 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 8 | Go: errors.New | invalid URL | `ErrInvalidURL = errors.New("invalid URL")` |
| 11 | Go: errors.New | feed not found | `ErrFeedNotFound = errors.New("feed not found")` |
| 14 | Go: errors.New | feed timeout | `ErrFeedTimeout = errors.New("feed timeout")` |
| 17 | Go: errors.New | invalid feed format | `ErrInvalidFeedFormat = errors.New("invalid feed format")` |
| 20 | Go: errors.New | network error | `ErrNetworkError = errors.New("network error")` |
| 23 | Go: errors.New | SSRF protection | `ErrSSRFBlocked = errors.New("SSRF protection")` |
| 26 | Go: errors.New | database error | `ErrDatabaseError = errors.New("database error")` |
| 29 | Go: errors.New | feed not modified | `ErrFeedNotModified = errors.New("feed not modified")` |

---

## `internal/services/feed_discovery.go`

_2 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 110 | Go: fmt.Errorf | invalid URL: %w | `return nil, fmt.Errorf("invalid URL: %w", err)` |
| 147 | Go: fmt.Errorf | unable to fetch HTML from %s using HTTP or HTTPS | `return nil, fmt.Errorf("unable to fetch HTML from %s using HTTP or HTTPS", host)` |

---

## `internal/services/feed_scheduler.go`

_2 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 79 | Go: fmt.Errorf | scheduler is already running | `return fmt.Errorf("scheduler is already running")` |
| 110 | Go: fmt.Errorf | failed to get feeds: %w | `return fmt.Errorf("failed to get feeds: %w", err)` |

---

## `internal/services/feed_service.go`

_14 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 180 | Go: fmt.Errorf | failed to fetch feed: %w | `return nil, fmt.Errorf("failed to fetch feed: %w", err)` |
| 197 | Go: fmt.Errorf | failed to insert feed: %w | `return nil, fmt.Errorf("failed to insert feed: %w", err)` |
| 201 | Go: fmt.Errorf | failed to save articles: %w | `return nil, fmt.Errorf("failed to save articles: %w", err)` |
| 416 | Go: fmt.Errorf | failed to get user articles: %w | `return 0, fmt.Errorf("failed to get user articles: %w", err)` |
| 425 | Go: fmt.Errorf | failed to mark articles as read: %w | `return 0, fmt.Errorf("failed to mark articles as read: %w", err)` |
| 477 | Go: fmt.Errorf | failed to get user %d: %w | `return fmt.Errorf("failed to get user %d: %w", userID, err)` |
| 483 | Go: fmt.Errorf | failed to get articles for feed %d: %w | `return fmt.Errorf("failed to get articles for feed %d: %w", feedID, err)` |
| 771 | Go: fmt.Sprintf | Failed to save article '%s': %v | `errors = append(errors, fmt.Sprintf("Failed to save article '%s': %v", article.T` |
| 790 | Go: fmt.Errorf | failed to save any articles from feed %d | `return 0, fmt.Errorf("failed to save any articles from feed %d", feedID)` |
| 1006 | Go: fmt.Errorf | failed to parse OPML: %w | `return 0, fmt.Errorf("failed to parse OPML: %w", err)` |
| 1039 | Go: fmt.Errorf | failed to parse OPML: %w | `return 0, fmt.Errorf("failed to parse OPML: %w", err)` |
| 1093 | Go: fmt.Errorf | failed to get user feeds: %w | `return nil, fmt.Errorf("failed to get user feeds: %w", err)` |
| 1121 | Go: fmt.Errorf | failed to marshal OPML: %w | `return nil, fmt.Errorf("failed to marshal OPML: %w", err)` |
| 1395 | Go: fmt.Errorf | failed to convert ISO-8859-1 to UTF-8: %w | `return nil, fmt.Errorf("failed to convert ISO-8859-1 to UTF-8: %w", err)` |

---

## `internal/services/payment_service.go`

_17 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 84 | Go: fmt.Errorf | required Stripe credential %s is not set | `return fmt.Errorf("required Stripe credential %s is not set", varName)` |
| 96 | Go: fmt.Errorf | failed to get user: %w | `return nil, fmt.Errorf("failed to get user: %w", err)` |
| 102 | Go: fmt.Errorf | user already has an active subscription | `return nil, fmt.Errorf("user already has an active subscription")` |
| 108 | Go: fmt.Errorf | failed to get/create customer: %w | `return nil, fmt.Errorf("failed to get/create customer: %w", err)` |
| 138 | Go: fmt.Errorf | failed to create checkout session: %w | `return nil, fmt.Errorf("failed to create checkout session: %w", err)` |
| 152 | Go: fmt.Sprintf | email:'%s' | `Query: fmt.Sprintf("email:'%s'", user.Email),` |
| 163 | Go: fmt.Errorf | failed to search for customer: %w | `return "", fmt.Errorf("failed to search for customer: %w", err)` |
| 178 | Go: fmt.Errorf | failed to create customer: %w | `return "", fmt.Errorf("failed to create customer: %w", err)` |
| 192 | Go: fmt.Errorf | failed to get subscription: %w | `return fmt.Errorf("failed to get subscription: %w", err)` |
| 201 | Go: fmt.Errorf | user_id not found in subscription metadata | `return fmt.Errorf("user_id not found in subscription metadata")` |
| 207 | Go: fmt.Errorf | invalid user_id in metadata: %w | `return fmt.Errorf("invalid user_id in metadata: %w", err)` |
| 242 | Go: fmt.Errorf | failed to update user subscription: %w | `return fmt.Errorf("failed to update user subscription: %w", err)` |
| 263 | Go: fmt.Errorf | failed to create product: %w | `return nil, fmt.Errorf("failed to create product: %w", err)` |
| 281 | Go: fmt.Errorf | failed to create price: %w | `return nil, fmt.Errorf("failed to create price: %w", err)` |
| 301 | Go: fmt.Errorf | failed to get user: %w | `return "", fmt.Errorf("failed to get user: %w", err)` |
| 307 | Go: fmt.Errorf | failed to get customer: %w | `return "", fmt.Errorf("failed to get customer: %w", err)` |
| 318 | Go: fmt.Errorf | failed to create customer portal session: %w | `return "", fmt.Errorf("failed to create customer portal session: %w", err)` |

---

## `internal/services/subscription_service.go`

_23 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 22 | Go: errors.New | feed limit reached for trial users | `ErrFeedLimitReached = errors.New("feed limit reached for trial users")` |
| 23 | Go: errors.New | trial period has expired | `ErrTrialExpired     = errors.New("trial period has expired")` |
| 209 | Go: fmt.Errorf | failed to generate random token: %w | `return "", fmt.Errorf("failed to generate random token: %w", err)` |
| 226 | Go: fmt.Errorf | failed to store admin token: %w | `return "", fmt.Errorf("failed to store admin token: %w", err)` |
| 241 | Go: fmt.Errorf | failed to store admin token in datastore: %w | `return "", fmt.Errorf("failed to store admin token in datastore: %w", err)` |
| 244 | Go: errors.New | admin token storage not supported for this database type | `return "", errors.New("admin token storage not supported for this database type"` |
| 301 | Go: fmt.Errorf | failed to query admin token: %w | `return false, fmt.Errorf("failed to query admin token: %w", err)` |
| 321 | Go: errors.New | admin token validation not supported for this database type | `return false, errors.New("admin token validation not supported for this database` |
| 332 | Go: fmt.Errorf | failed to query admin tokens: %w | `return nil, fmt.Errorf("failed to query admin tokens: %w", err)` |
| 348 | Go: fmt.Errorf | failed to scan admin token: %w | `return nil, fmt.Errorf("failed to scan admin token: %w", err)` |
| 361 | Go: fmt.Errorf | failed to query admin tokens: %w | `return nil, fmt.Errorf("failed to query admin tokens: %w", err)` |
| 379 | Go: errors.New | admin token listing not supported for this database type | `return nil, errors.New("admin token listing not supported for this database type` |
| 388 | Go: fmt.Errorf | failed to revoke admin token: %w | `return fmt.Errorf("failed to revoke admin token: %w", err)` |
| 393 | Go: fmt.Errorf | failed to check revoke result: %w | `return fmt.Errorf("failed to check revoke result: %w", err)` |
| 397 | Go: errors.New | admin token not found | `return errors.New("admin token not found")` |
| 411 | Go: errors.New | admin token not found | `return errors.New("admin token not found")` |
| 413 | Go: fmt.Errorf | failed to get admin token: %w | `return fmt.Errorf("failed to get admin token: %w", err)` |
| 418 | Go: errors.New | admin token not found | `return errors.New("admin token not found")` |
| 426 | Go: fmt.Errorf | failed to revoke admin token: %w | `return fmt.Errorf("failed to revoke admin token: %w", err)` |
| 432 | Go: errors.New | admin token revocation not supported for this database type | `return errors.New("admin token revocation not supported for this database type")` |
| 442 | Go: fmt.Errorf | failed to count admin tokens: %w | `return false, fmt.Errorf("failed to count admin tokens: %w", err)` |
| 454 | Go: fmt.Errorf | failed to count admin tokens: %w | `return false, fmt.Errorf("failed to count admin tokens: %w", err)` |
| 460 | Go: errors.New | admin token check not supported for this database type | `return false, errors.New("admin token check not supported for this database type` |

---

## `internal/services/url_validator.go`

_11 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 76 | Go: fmt.Errorf | invalid URL: %w | `return fmt.Errorf("invalid URL: %w", err)` |
| 81 | Go: fmt.Errorf | URL scheme '%s' not allowed (only http/https permitted) | `return fmt.Errorf("URL scheme '%s' not allowed (only http/https permitted)", par` |
| 86 | Go: fmt.Errorf | URL must have a host | `return fmt.Errorf("URL must have a host")` |
| 92 | Go: fmt.Errorf | URL must have a hostname | `return fmt.Errorf("URL must have a hostname")` |
| 110 | Go: fmt.Errorf | DNS lookup failed for %s: %w | `return fmt.Errorf("DNS lookup failed for %s: %w", hostname, err)` |
| 114 | Go: fmt.Errorf | no IP addresses found for hostname %s | `return fmt.Errorf("no IP addresses found for hostname %s", hostname)` |
| 120 | Go: fmt.Errorf | hostname %s resolves to blocked IP %s: %w | `return fmt.Errorf("hostname %s resolves to blocked IP %s: %w", hostname, ip, err` |
| 132 | Go: fmt.Errorf | IP address %s is in blocked network range %s (SSRF protection) | `return fmt.Errorf("IP address %s is in blocked network range %s (SSRF protection` |
| 145 | Go: fmt.Errorf | too many redirects (max 10) | `return fmt.Errorf("too many redirects (max 10)")` |
| 151 | Go: fmt.Errorf | redirect to blocked URL: %w | `return fmt.Errorf("redirect to blocked URL: %w", err)` |
| 174 | Go: fmt.Errorf | URL cannot be empty | `return "", fmt.Errorf("URL cannot be empty")` |

---

## `web/static/js/account.js`

_10 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 113 | JS: thrown error | Failed to load subscription info | `throw new Error('Failed to load subscription info');` |
| 338 | JS: dialog | Please enter a valid number between 0 and 10000 | `alert('Please enter a valid number between 0 and 10000');` |
| 344 | JS: textContent | Saving... | `saveButton.textContent = 'Saving...';` |
| 358 | JS: textContent | Saved! | `saveButton.textContent = 'Saved!';` |
| 360 | JS: textContent | Save | `saveButton.textContent = 'Save';` |
| 369 | JS: dialog | Failed to save setting. Please try again. | `alert('Failed to save setting. Please try again.');` |
| 370 | JS: textContent | Save | `saveButton.textContent = 'Save';` |
| 385 | JS: thrown error | Failed to load account stats | `throw new Error('Failed to load account stats');` |
| 486 | JS: dialog | Failed to start upgrade process. Please try again. | `alert('Failed to start upgrade process. Please try again.');` |
| 509 | JS: dialog | Failed to open subscription management. Please try again. | `alert('Failed to open subscription management. Please try again.');` |

---

## `web/static/js/app.js`

_45 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 33 | JS: notification | Connection restored | `this.showToast('Connection restored', ToastType.SUCCESS);` |
| 39 | JS: notification | No internet connection | `this.showToast('No internet connection', ToastType.WARNING);` |
| 58 | JS: textContent | Online | `indicator.textContent = 'Online';` |
| 61 | JS: textContent | Offline | `indicator.textContent = 'Offline';` |
| 149 | JS: textContent | Retry | `retryBtn.textContent = 'Retry';` |
| 161 | JS: textContent | Dismiss | `dismissBtn.textContent = 'Dismiss';` |
| 264 | JS: notification | Retry attempt ${attempts}/${maxRetries}... | `this.showToast(`Retry attempt ${attempts}/${maxRetries}...`, ToastType.INFO, 200` |
| 795 | JS: thrown error | HTTP ${feedsResponse.status} | `throw new Error(`HTTP ${feedsResponse.status}`);` |
| 814 | JS: notification | Invalid feed data received from server | `this.showError('Invalid feed data received from server');` |
| 822 | JS: ui string | loading feeds | `context: 'loading feeds'` |
| 840 | JS: textContent | Loading feeds... | `loadingDiv.textContent = 'Loading feeds...';` |
| 864 | JS: ui string | loading feeds | `context: 'loading feeds'` |
| 884 | JS: template literal | [data-feed-id="all"] | `const allElement = document.querySelector(`[data-feed-id="all"]`);` |
| 888 | JS: textContent | Articles | `document.getElementById('article-pane-title').textContent = 'Articles';` |
| 894 | JS: notification | Invalid feed data received from server | `this.showError('Invalid feed data received from server', ErrorType.SERVER);` |
| 972 | JS: ui string | Delete feed | `deleteButton.title = 'Delete feed';` |
| 1053 | JS: thrown error | Authentication required | `throw new Error('Authentication required');` |
| 1056 | JS: thrown error | HTTP ${response.status}: ${response.statusText} | `throw new Error(`HTTP ${response.status}: ${response.statusText}`);` |
| 1108 | JS: ui string | loading articles | `context: 'loading articles'` |
| 1152 | JS: textContent | Load More Articles | `button.textContent = 'Load More Articles';` |
| 1164 | JS: textContent | Loading... | `button.textContent = 'Loading...';` |
| 1171 | JS: textContent | Load More Articles | `button.textContent = 'Load More Articles';` |
| 1232 | JS: ui string | Star article | `data-article-id="${article.id}" title="Star article">★</button>` |
| 1324 | JS: ui string | Star article | `data-article-id="${article.id}" title="Star article">★</button>` |
| 1397 | JS: ui string | Star article | `data-article-id="${article.id}" title="Star article">★</button>` |
| 1805 | JS: thrown error | HTTP ${response.status} | `throw new Error(`HTTP ${response.status}`);` |
| 1821 | JS: notification | Failed to toggle star | `this.showError('Failed to toggle star');` |
| 1835 | JS: notification | Failed to toggle star: | `this.showError('Failed to toggle star: ' + error.message);` |
| 2135 | JS: ui string | adding feed | `context: 'adding feed'` |
| 2146 | JS: ui string | adding feed | `context: 'adding feed'` |
| 2165 | JS: dialog | Are you sure you want to remove this feed from your subscriptions? | `if (!confirm('Are you sure you want to remove this feed from your subscriptions?` |
| 2199 | JS: notification | Failed to delete feed: | `this.showError('Failed to delete feed: ' + error.message);` |
| 2211 | JS: textContent | Refreshing... | `refreshBtn.textContent = 'Refreshing...';` |
| 2228 | JS: ui string | refreshing feeds | `context: 'refreshing feeds'` |
| 2245 | JS: notification | Feeds refreshed successfully | `this.showSuccess('Feeds refreshed successfully');` |
| 2384 | JS: thrown error | Stripe not configured | `throw new Error('Stripe not configured');` |
| 2407 | JS: notification | Payment processing is not available. Please contact support. | `this.showError('Payment processing is not available. Please contact support.');` |
| 2409 | JS: notification | Failed to start upgrade process: | `this.showError('Failed to start upgrade process: ' + error.message);` |
| 2458 | JS: notification | Failed to start login process | `this.showError('Failed to start login process');` |
| 2461 | JS: notification | Login failed: | `this.showError('Login failed: ' + error.message);` |
| 2475 | JS: notification | Logout failed: | `this.showError('Logout failed: ' + error.message);` |
| 2767 | JS: textContent | Release to refresh | `textEl.textContent = 'Release to refresh';` |
| 2770 | JS: textContent | Pull to refresh | `textEl.textContent = 'Pull to refresh';` |
| 2802 | JS: textContent | Refreshing... | `this.indicator.querySelector('.pull-to-refresh-text').textContent = 'Refreshing.` |
| 2821 | JS: textContent | Pull to refresh | `this.indicator.querySelector('.pull-to-refresh-text').textContent = 'Pull to ref` |

---

## `web/static/js/modals.js`

_7 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 76 | JS: notification | Please select an OPML file | `this.app.showError('Please select an OPML file');` |
| 84 | JS: notification | File is too large (max 10MB) | `this.app.showError('File is too large (max 10MB)');` |
| 90 | JS: textContent | Importing... | `submitButton.textContent = 'Importing...';` |
| 119 | JS: ui string | Successfully imported ${result.imported_count} feed(s) from OPML file | `const message = `Successfully imported ${result.imported_count} feed(s) from OPM` |
| 128 | JS: notification | Imported ${error.imported_count} feed(s) before reaching your limit. | `this.app.showSuccess(`Imported ${error.imported_count} feed(s) before reaching y` |
| 144 | JS: notification | Failed to import OPML: | `this.app.showError('Failed to import OPML: ' + errorMessage);` |
| 147 | JS: notification | Failed to import OPML: | `this.app.showError('Failed to import OPML: ' + error.message);` |

---

## `web/templates/account.html`

_19 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 28 | text content | GoRead2 | `<a>` |
| 30 | text content | ← Back to Feeds | `<a>` |
| 37 | text content | Account Management | `<h1>` |
| 41 | text content | Profile | `<h2>` |
| 43 | text content | Loading profile... | `<div>` |
| 49 | text content | Subscription | `<h2>` |
| 51 | text content | Loading subscription info... | `<div>` |
| 57 | text content | Settings | `<h2>` |
| 59 | text content | Loading settings... | `<div>` |
| 65 | text content | Usage Statistics | `<h2>` |
| 67 | text content | Loading usage stats... | `<div>` |
| 73 | text content | Contact & Support | `<h2>` |
| 75 | text content | Need help or have questions? Get in touch with us: | `<p>` |
| 77 | text content | Twitter/X: | `<strong>` |
| 77 | text content | @GoReadApp2 | `<a>` |
| 89 | text content | Confirm Action | `<h2>` |
| 90 | text content | Are you sure you want to proceed? | `<p>` |
| 92 | text content | Confirm | `<button>` |
| 93 | text content | Cancel | `<button>` |

---

## `web/templates/index.html`

_65 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 36 | text content | Loading GoRead... | `<div>` |
| 43 | text content | GoRead2 | `<h1>` |
| 46 | text content | Refresh | `<button>` |
| 47 | text content | Add Feed | `<button>` |
| 48 | text content | Import OPML | `<button>` |
| 49 | text content | Help | `<button>` |
| 63 | text content | Feeds | `<h2>` |
| 67 | text content | All Articles | `<span>` |
| 77 | text content | Select a feed | `<h2>` |
| 80 | attribute: value | unread | `<input value=...>` |
| 80 | text content | Unread | `<input>` |
| 83 | attribute: value | all | `<input value=...>` |
| 83 | text content | All | `<input>` |
| 97 | text content | Welcome to GoRead2! | `<p>` |
| 98 | text content | Select a feed from the left panel to view articles, then click on an article to read it here. | `<p>` |
| 100 | text content | On tablet? | `<strong>` |
| 100 | text content | Tap the | `<strong>` |
| 100 | text content | button in the bottom-right corner to access your feeds and articles. | `<span>` |
| 107 | attribute: aria-label | Toggle article list | `<button aria-label=...>` |
| 112 | attribute: aria-label | View feeds | `<button aria-label=...>` |
| 116 | text content | Feeds | `<span>` |
| 118 | attribute: aria-label | View articles | `<button aria-label=...>` |
| 123 | text content | Articles | `<span>` |
| 125 | attribute: aria-label | View content | `<button aria-label=...>` |
| 130 | text content | Content | `<span>` |
| 139 | text content | Add RSS Feed | `<h2>` |
| 142 | text content | Website or Feed URL: | `<label>` |
| 143 | attribute: placeholder | example.com or https://example.com/feed.xml | `<input placeholder=...>` |
| 144 | text content | Enter a website domain (e.g., "slashdot.org") or direct feed URL | `<small>` |
| 147 | text content | Add Feed | `<button>` |
| 148 | text content | Cancel | `<button>` |
| 158 | text content | Help & Shortcuts | `<h2>` |
| 161 | text content | Getting Started | `<h3>` |
| 163 | text content | Add Feed: | `<strong>` |
| 163 | text content | Click "Add Feed" and enter a website URL or RSS feed URL | `<strong>` |
| 166 | text content | Import OPML: | `<strong>` |
| 166 | text content | Click "Import OPML" to import feeds from other RSS readers like Feedly, Inoreader, or NewsBlur | `<strong>` |
| 169 | text content | Navigation: | `<strong>` |
| 169 | text content | Use keyboard shortcuts or click to navigate between articles | `<strong>` |
| 174 | text content | Keyboard Shortcuts | `<h3>` |
| 176 | text content | Next article | `<span>` |
| 179 | text content | Previous article | `<span>` |
| 182 | text content | or | `<kbd>` |
| 182 | text content | Enter | `<kbd>` |
| 182 | text content | Open article in new tab | `<span>` |
| 185 | text content | Toggle read/unread | `<span>` |
| 188 | text content | Toggle star | `<span>` |
| 191 | text content | Refresh feeds | `<span>` |
| 196 | text content | Features | `<h3>` |
| 198 | text content | Personal Data: | `<strong>` |
| 198 | text content | Your read/starred status and feed subscriptions are private to you | `<strong>` |
| 201 | text content | Auto-Read: | `<strong>` |
| 201 | text content | Articles are automatically marked as read when you navigate away | `<strong>` |
| 204 | text content | Feed Discovery: | `<strong>` |
| 204 | text content | Enter any website URL and we'll try to find its RSS feed | `<strong>` |
| 209 | text content | Contact & Support | `<h3>` |
| 211 | text content | Need Help? | `<strong>` |
| 211 | text content | Follow us on | `<strong>` |
| 211 | text content | Twitter/X @GoReadApp2 | `<a>` |
| 211 | text content | for support and updates | `<a>` |
| 222 | text content | Import OPML File | `<h2>` |
| 225 | text content | OPML File: | `<label>` |
| 227 | text content | Select an OPML file exported from another RSS reader (max 10MB) | `<small>` |
| 230 | text content | Import Feeds | `<button>` |
| 231 | text content | Cancel | `<button>` |

---

## `web/templates/privacy.html`

_66 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 25 | text content | Privacy Policy | `<h1>` |
| 26 | text content | Last updated: September 15, 2025 | `<em>` |
| 28 | text content | Information We Collect | `<h2>` |
| 29 | text content | GoRead2 collects minimal personal information necessary to provide our RSS feed reading service and process authentication. | `<p>` |
| 31 | text content | Google Authentication | `<h2>` |
| 32 | text content | When you sign in with Google, we only access: | `<p>` |
| 34 | text content | Your email address (for account identification) | `<li>` |
| 35 | text content | Your name (for display purposes) | `<li>` |
| 36 | text content | Your profile picture (for display purposes) | `<li>` |
| 39 | text content | Data Storage | `<h2>` |
| 40 | text content | The only data we store is: | `<p>` |
| 42 | text content | Your RSS feed subscriptions | `<li>` |
| 43 | text content | Article read/unread status | `<li>` |
| 44 | text content | Article starred status | `<li>` |
| 45 | text content | Basic authentication information from Google | `<li>` |
| 46 | text content | Subscription status and payment history (for Pro subscribers) | `<li>` |
| 49 | text content | Payment Processing | `<h2>` |
| 50 | text content | For GoRead2 Pro subscriptions, we use Stripe to process payments securely. When you subscribe: | `<p>` |
| 52 | text content | Data shared with Stripe: | `<strong>` |
| 52 | text content | Your email address and name are shared with Stripe to create your customer account | `<strong>` |
| 53 | text content | Payment information: | `<strong>` |
| 53 | text content | All payment card details are handled directly by Stripe and never stored on our servers | `<strong>` |
| 54 | text content | Stripe's role: | `<strong>` |
| 54 | text content | Stripe acts as our payment processor and is subject to their own privacy policy | `<strong>` |
| 55 | text content | Data retention: | `<strong>` |
| 55 | text content | Stripe retains payment and customer data according to their data retention policies and legal requirements | `<strong>` |
| 57 | text content | For more information about how Stripe handles your data, please review | `<p>` |
| 57 | text content | Stripe's Privacy Policy | `<a>` |
| 59 | text content | Analytics and Usage Tracking | `<h2>` |
| 60 | text content | We use Google Analytics to understand how our service is used and to improve the user experience. Google Analytics collects: | `<p>` |
| 62 | text content | Usage data: | `<strong>` |
| 62 | text content | Page views, session duration, and navigation patterns | `<strong>` |
| 63 | text content | Technical information: | `<strong>` |
| 63 | text content | Browser type, device information, and IP address (anonymized) | `<strong>` |
| 64 | text content | Performance metrics: | `<strong>` |
| 64 | text content | Page load times and user interactions | `<strong>` |
| 66 | text content | This data is processed by Google and is subject to | `<p>` |
| 66 | text content | Google's Privacy Policy | `<a>` |
| 66 | text content | . No personally identifiable information from your account is shared with Google Analytics. | `<a>` |
| 67 | text content | Opting out: | `<strong>` |
| 67 | text content | You can opt out of Google Analytics tracking by installing the | `<strong>` |
| 67 | text content | Google Analytics Opt-out Browser Add-on | `<a>` |
| 67 | text content | or by using browser settings to block tracking scripts. | `<a>` |
| 69 | text content | Data Usage | `<h2>` |
| 70 | text content | We use your information solely to provide the RSS feed reading service and process subscription payments. Beyond our payment processor (Stripe) and analytics provider (Google Analytics), we do not share, sell, or distribute your personal information to other third parties. | `<p>` |
| 72 | text content | Data Retention | `<h2>` |
| 73 | text content | Your data is retained only as long as you have an active account. You may request deletion of your account and all associated data at any time. | `<p>` |
| 74 | text content | Subscription data: | `<strong>` |
| 74 | text content | If you have a paid subscription, some payment-related data may be retained by Stripe for legal and regulatory compliance even after account deletion. This includes transaction records required for tax purposes and fraud prevention. | `<strong>` |
| 76 | text content | Your Rights | `<h2>` |
| 77 | text content | You have the right to: | `<p>` |
| 79 | text content | Access: | `<strong>` |
| 79 | text content | Request a copy of the personal data we hold about you | `<strong>` |
| 80 | text content | Correction: | `<strong>` |
| 80 | text content | Request correction of inaccurate personal data | `<strong>` |
| 81 | text content | Deletion: | `<strong>` |
| 81 | text content | Request deletion of your account and associated data | `<strong>` |
| 82 | text content | Portability: | `<strong>` |
| 82 | text content | Export your RSS feed subscriptions and data | `<strong>` |
| 83 | text content | Subscription management: | `<strong>` |
| 83 | text content | Cancel your subscription at any time through your account settings or Stripe's customer portal | `<strong>` |
| 86 | text content | Contact | `<h2>` |
| 87 | text content | If you have any questions about this privacy policy or want to exercise your data rights, please contact us: | `<p>` |
| 89 | text content | Twitter/X: | `<strong>` |
| 89 | text content | @GoReadApp2 | `<a>` |
| 93 | text content | Back | `<a>` |

---

## `web/templates/subscription_cancel.html`

_10 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 20 | text content | Subscription Cancelled | `<h1>` |
| 23 | text content | Your subscription process was cancelled. No charges have been made to your account. | `<p>` |
| 25 | text content | You can still enjoy GoRead2 with the free plan: | `<p>` |
| 27 | text content | ✅ Up to 20 RSS feeds | `<li>` |
| 28 | text content | ✅ 30-day free trial | `<li>` |
| 29 | text content | ✅ Full article reading | `<li>` |
| 30 | text content | ✅ Article starring and read status | `<li>` |
| 33 | text content | Ready to upgrade? You can subscribe anytime to unlock unlimited feeds and premium features. | `<p>` |
| 37 | text content | Continue with Free Plan | `<a>` |
| 38 | text content | Try Subscribing Again | `<a>` |

---

## `web/templates/subscription_success.html`

_11 string(s)_

| Line | Category | Text | Context |
|------|----------|------|---------|
| 20 | text content | 🎉 Welcome to GoRead2 Pro! | `<h1>` |
| 23 | text content | Your subscription has been successfully activated! | `<strong>` |
| 25 | text content | You now have access to: | `<p>` |
| 27 | text content | ✅ Unlimited RSS feeds | `<li>` |
| 28 | text content | ✅ Priority support | `<li>` |
| 29 | text content | ✅ Advanced features | `<li>` |
| 30 | text content | ✅ No more feed limits | `<li>` |
| 33 | text content | You can now add as many RSS feeds as you want and enjoy the full GoRead2 experience. | `<p>` |
| 36 | text content | Session ID: {{.session_id}} | `<small>` |
| 41 | text content | Start Using GoRead2 Pro | `<a>` |
| 42 | text content | Manage Subscription | `<a>` |
