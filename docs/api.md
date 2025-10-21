# API Reference

Complete reference for GoRead2's REST API endpoints.

## Overview

GoRead2 provides a RESTful API for managing feeds, articles, and user subscriptions. All endpoints require authentication via session cookies obtained through Google OAuth.

**Base URL**: `http://localhost:8080` (development) or `https://your-domain.com` (production)

**Authentication**: Session-based authentication with HTTP-only cookies

**CSRF Protection**: All state-changing operations (POST, PUT, DELETE) require a valid CSRF token in the `X-CSRF-Token` header

**Rate Limiting**:
- Auth endpoints: 10 requests/second (burst: 20)
- API endpoints: 30 requests/second (burst: 50)
- Returns `429 Too Many Requests` when limit exceeded

## Authentication Endpoints

### OAuth Flow

#### `GET /auth/login`
Initiate Google OAuth authentication flow.

**Response**: Redirects to Google OAuth consent screen

**Example**:
```bash
curl -L "http://localhost:8080/auth/login"
```

#### `GET /auth/callback`
OAuth callback handler (configured in Google Cloud Console).

**Parameters**:
- `code` (query) - Authorization code from Google
- `state` (query) - CSRF protection state parameter

**Response**: Redirects to main application with session cookie

#### `POST /auth/logout`
Logout and clear session.

**Response**:
```json
{
  "message": "Logged out successfully"
}
```

**Example**:
```bash
curl -X POST "http://localhost:8080/auth/logout" \
  -H "Cookie: session=your-session-cookie"
```

#### `GET /auth/me`
Get current authenticated user information and CSRF token.

**Response**:
```json
{
  "user": {
    "id": 1,
    "email": "user@example.com",
    "name": "John Doe",
    "avatar": "https://lh3.googleusercontent.com/...",
    "max_articles_on_feed_add": 100
  },
  "csrf_token": "your-csrf-token-here"
}
```

**Note**: The `csrf_token` must be included in the `X-CSRF-Token` header for all POST, PUT, and DELETE requests.

**Example**:
```bash
curl "http://localhost:8080/auth/me" \
  -H "Cookie: session=your-session-cookie"
```

## Feed Endpoints

All feed endpoints are user-specific and require authentication.

**Note**: All POST, PUT, and DELETE endpoints require the `X-CSRF-Token` header with a valid token obtained from `/auth/me`.

### `GET /api/feeds`
List user's subscribed feeds.

**Response**:
```json
[
  {
    "id": 1,
    "title": "Example Blog",
    "url": "https://example.com/feed.xml",
    "description": "An example RSS feed",
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z",
    "last_fetch": "2023-01-01T12:00:00Z"
  }
]
```

**Caching**: 5 minutes (`Cache-Control: private, max-age=300`)

**Example**:
```bash
curl "http://localhost:8080/api/feeds" \
  -H "Cookie: session=your-session-cookie"
```

### `POST /api/feeds`
Subscribe user to a new feed.

**Headers**:
- `X-CSRF-Token` (required) - CSRF token from `/auth/me`

**Request Body**:
```json
{
  "url": "https://example.com/feed.xml"
}
```

**Response** (201 Created):
```json
{
  "id": 2,
  "title": "New Feed",
  "url": "https://example.com/feed.xml",
  "description": "Feed description",
  "created_at": "2023-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:00:00Z",
  "last_fetch": "2023-01-01T00:00:00Z"
}
```

**Error Responses**:
- `400 Bad Request` - Invalid URL or missing field
- `402 Payment Required` - Feed limit reached (trial users)
- `403 Forbidden` - Invalid or missing CSRF token

**Example**:
```bash
curl -X POST "http://localhost:8080/api/feeds" \
  -H "Cookie: session=your-session-cookie" \
  -H "X-CSRF-Token: your-csrf-token" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/feed.xml"}'
```
- `409 Conflict` - Already subscribed to feed
- `500 Internal Server Error` - Feed fetch failed

**Example**:
```bash
curl -X POST "http://localhost:8080/api/feeds" \
  -H "Cookie: session=your-session-cookie" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/feed.xml"}'
```

### `DELETE /api/feeds/:id`
Unsubscribe user from a feed.

**Parameters**:
- `id` (path) - Feed ID

**Response**:
```json
{
  "message": "Feed removed from your subscriptions successfully"
}
```

**Error Responses**:
- `400 Bad Request` - Invalid feed ID
- `404 Not Found` - Feed not found or not subscribed
- `500 Internal Server Error` - Database error

**Example**:
```bash
curl -X DELETE "http://localhost:8080/api/feeds/1" \
  -H "Cookie: session=your-session-cookie"
```

### `POST /api/feeds/refresh`
Manually refresh all user's feeds.

**Response**:
```json
{
  "message": "Feeds refreshed successfully"
}
```

**Example**:
```bash
curl -X POST "http://localhost:8080/api/feeds/refresh" \
  -H "Cookie: session=your-session-cookie"
```

### `GET /api/feeds/unread-counts`
Get unread article counts for all user's feeds.

**Response**:
```json
{
  "1": 5,    // Feed ID 1 has 5 unread articles
  "2": 12,   // Feed ID 2 has 12 unread articles
  "3": 0     // Feed ID 3 has 0 unread articles
}
```

**Caching**: 10 seconds (`Cache-Control: private, max-age=10`)

**Example**:
```bash
curl "http://localhost:8080/api/feeds/unread-counts" \
  -H "Cookie: session=your-session-cookie"
```

### `POST /api/feeds/import`
Import feeds from OPML file.

**Headers**:
- `X-CSRF-Token` (required) - CSRF token from `/auth/me`

**Request**: Multipart form data with OPML file

**Parameters**:
- `opml` (file) - OPML file (max 10MB)

**Response**:
```json
{
  "message": "OPML imported successfully",
  "imported_count": 15
}
```

**Error Responses**:
- `400 Bad Request` - No file provided or file too large
- `402 Payment Required` - Import would exceed feed limit
- `403 Forbidden` - Invalid or missing CSRF token
- `500 Internal Server Error` - OPML parsing failed

**Example**:
```bash
curl -X POST "http://localhost:8080/api/feeds/import" \
  -H "Cookie: session=your-session-cookie" \
  -H "X-CSRF-Token: your-csrf-token" \
  -F "opml=@subscriptions.opml"
```

### `GET /api/feeds/export`
Export user's feeds to OPML format.

**Response**: OPML XML file download

**Headers**:
- `Content-Type: application/xml; charset=utf-8`
- `Content-Disposition: attachment; filename=goread2-subscriptions.opml`

**Response Body** (OPML XML):
```xml
<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>GoRead2 Subscriptions</title>
  </head>
  <body>
    <outline type="rss" text="Feed Title" title="Feed Title"
             xmlUrl="https://example.com/feed.xml"
             htmlUrl="https://example.com/feed.xml"/>
    <!-- More feeds... -->
  </body>
</opml>
```

**Error Responses**:
- `401 Unauthorized` - Not authenticated
- `500 Internal Server Error` - Export generation failed

**Example**:
```bash
curl "http://localhost:8080/api/feeds/export" \
  -H "Cookie: session=your-session-cookie" \
  -o subscriptions.opml
```

## Article Endpoints

### `GET /api/feeds/:id/articles`
Get articles for a specific feed.

**Parameters**:
- `id` (path) - Feed ID or "all" for all feeds

**Response**:
```json
[
  {
    "id": 1,
    "feed_id": 1,
    "title": "Article Title",
    "url": "https://example.com/article1",
    "content": "Full article content...",
    "description": "Article summary",
    "author": "John Author",
    "published_at": "2023-01-01T10:00:00Z",
    "created_at": "2023-01-01T10:30:00Z",
    "is_read": false,
    "is_starred": false
  }
]
```

**Special Cases**:
- Use `id=all` to get articles from all subscribed feeds
- Articles are ordered by `published_at` (newest first)
- `is_read` and `is_starred` are user-specific

**Example**:
```bash
# Get articles from specific feed
curl "http://localhost:8080/api/feeds/1/articles" \
  -H "Cookie: session=your-session-cookie"

# Get articles from all feeds
curl "http://localhost:8080/api/feeds/all/articles" \
  -H "Cookie: session=your-session-cookie"
```

### `POST /api/articles/:id/read`
Mark article as read or unread for current user.

**Parameters**:
- `id` (path) - Article ID

**Request Body**:
```json
{
  "is_read": true
}
```

**Response**:
```json
{
  "message": "Article updated successfully"
}
```

**Example**:
```bash
# Mark as read
curl -X POST "http://localhost:8080/api/articles/1/read" \
  -H "Cookie: session=your-session-cookie" \
  -H "Content-Type: application/json" \
  -d '{"is_read": true}'

# Mark as unread
curl -X POST "http://localhost:8080/api/articles/1/read" \
  -H "Cookie: session=your-session-cookie" \
  -H "Content-Type: application/json" \
  -d '{"is_read": false}'
```

### `POST /api/articles/:id/star`
Toggle star status for article.

**Parameters**:
- `id` (path) - Article ID

**Response**:
```json
{
  "message": "Article starred status toggled"
}
```

**Example**:
```bash
curl -X POST "http://localhost:8080/api/articles/1/star" \
  -H "Cookie: session=your-session-cookie"
```

### `POST /api/articles/mark-all-read`
Mark all articles as read for the current user.

**Headers**:
- `X-CSRF-Token` (required) - CSRF token from `/auth/me`

**Response**:
```json
{
  "message": "All articles marked as read",
  "articles_count": 42
}
```

**Description**:
This endpoint marks all articles across all subscribed feeds as read for the authenticated user. The response includes the total count of articles that were marked as read.

**Error Responses**:
- `401 Unauthorized` - Not authenticated
- `403 Forbidden` - Invalid or missing CSRF token
- `500 Internal Server Error` - Database error

**Example**:
```bash
curl -X POST "http://localhost:8080/api/articles/mark-all-read" \
  -H "Cookie: session=your-session-cookie" \
  -H "X-CSRF-Token: your-csrf-token"
```

**Note**: This is an API-only endpoint with no corresponding UI button. It's useful for automation scripts or third-party integrations.

## Subscription Endpoints

These endpoints are only available when `SUBSCRIPTION_ENABLED=true`.

### `GET /api/subscription`
Get user's subscription information and limits.

**Response** (Subscription Enabled):
```json
{
  "status": "trial",
  "trial_ends_at": "2023-02-01T00:00:00Z",
  "feed_count": 15,
  "feed_limit": 20,
  "days_remaining": 25,
  "can_add_feeds": true,
  "subscription_id": null,
  "last_payment_date": null
}
```

**Response** (Subscription Disabled):
```json
{
  "status": "unlimited",
  "feed_count": 150,
  "feed_limit": -1,
  "can_add_feeds": true,
  "subscription_enabled": false
}
```

**Status Values**:
- `trial` - Free trial period
- `active` - Paid subscription
- `cancelled` - Cancelled subscription
- `expired` - Expired trial or subscription
- `unlimited` - Admin or subscription disabled

**Example**:
```bash
curl "http://localhost:8080/api/subscription" \
  -H "Cookie: session=your-session-cookie"
```

### `POST /api/subscription/checkout`
Create Stripe checkout session for subscription.

**Response**:
```json
{
  "checkout_url": "https://checkout.stripe.com/c/pay/cs_test_..."
}
```

**Example**:
```bash
curl -X POST "http://localhost:8080/api/subscription/checkout" \
  -H "Cookie: session=your-session-cookie"
```

### `POST /api/subscription/portal`
Create Stripe customer portal session for billing management.

**Response**:
```json
{
  "portal_url": "https://billing.stripe.com/p/session/test_..."
}
```

**Example**:
```bash
curl -X POST "http://localhost:8080/api/subscription/portal" \
  -H "Cookie: session=your-session-cookie"
```

### `GET /api/stripe/config`
Get Stripe configuration for frontend.

**Response**:
```json
{
  "publishable_key": "pk_test_...",
  "price_id": "price_..."
}
```

**Example**:
```bash
curl "http://localhost:8080/api/stripe/config" \
  -H "Cookie: session=your-session-cookie"
```

## Account Endpoints

### `GET /api/account/stats`
Get detailed account statistics.

**Response**:
```json
{
  "total_feeds": 25,
  "total_articles": 1250,
  "total_unread": 45,
  "active_feeds": 18,
  "subscription_info": {
    "status": "active",
    "feed_limit": -1
  },
  "feeds": [...]
}
```

**Example**:
```bash
curl "http://localhost:8080/api/account/stats" \
  -H "Cookie: session=your-session-cookie"
```

### `PUT /api/account/max-articles`
Update the maximum number of articles to import when adding a new feed.

**Request Body**:
```json
{
  "max_articles": 100
}
```

**Parameters**:
- `max_articles` (integer, required) - Maximum articles to import (0-10000, where 0 = unlimited)

**Response**:
```json
{
  "message": "Setting updated successfully",
  "max_articles": 100
}
```

**Example**:
```bash
curl -X PUT "http://localhost:8080/api/account/max-articles" \
  -H "Cookie: session=your-session-cookie" \
  -H "Content-Type: application/json" \
  -d '{"max_articles": 250}'
```

**Error Responses**:
- `400 Bad Request` - Invalid max_articles value (outside 0-10000 range)
- `401 Unauthorized` - Not authenticated
- `500 Internal Server Error` - Database error

## Webhook Endpoints

### `POST /webhooks/stripe`
Stripe webhook endpoint for subscription events.

**Headers**:
- `Stripe-Signature` - Webhook signature for verification

**Events Handled**:
- `checkout.session.completed`
- `customer.subscription.created`
- `customer.subscription.updated`
- `customer.subscription.deleted`
- `invoice.payment_succeeded`
- `invoice.payment_failed`

**Response**:
```json
{
  "received": true
}
```

**Note**: This endpoint is called by Stripe, not for direct API usage.

## Debug Endpoints

**⚠️ Admin Only**: All debug endpoints require admin privileges.

### `GET /api/debug/feeds/:id`
Debug information for feed subscription.

**Authentication**: Requires admin user

**Response**:
```json
{
  "user_id": 1,
  "feed_id": 1,
  "is_subscribed": true,
  "user_feeds_count": 15,
  "all_articles_count": 250,
  "user_articles_count": 250,
  "user_feeds": [...],
  "all_articles": [...],
  "user_articles": [...]
}
```

## Error Handling

### Standard Error Response

```json
{
  "error": "Error message describing what went wrong"
}
```

### HTTP Status Codes

- `200 OK` - Success
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `402 Payment Required` - Subscription required
- `403 Forbidden` - Access denied
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource already exists
- `500 Internal Server Error` - Server error

### Subscription-Specific Errors

When feed limits are reached:

```json
{
  "error": "You've reached the limit of 20 feeds for free users. Upgrade to Pro for unlimited feeds.",
  "limit_reached": true,
  "current_limit": 20
}
```

When trial has expired:

```json
{
  "error": "Your 30-day free trial has expired. Subscribe to continue using GoRead2.",
  "trial_expired": true
}
```

## Rate Limiting

Currently, no rate limiting is implemented, but consider implementing for production:

- **Authentication endpoints**: 5 requests per minute
- **Feed operations**: 10 requests per minute
- **Article operations**: 30 requests per minute
- **General API**: 100 requests per minute

## CORS Policy

CORS is configured to allow:
- **Origins**: Same origin only (no cross-origin requests)
- **Credentials**: Cookies included in same-origin requests
- **Headers**: Standard headers + `Content-Type`

## Session Management

### Session Cookies

- **Name**: `session`
- **Security**: HTTP-only, Secure (HTTPS), SameSite=Strict
- **Expiration**: 24 hours
- **Domain**: Same as application domain

### Session Validation

All protected endpoints validate:
1. Session cookie exists and is valid
2. Session hasn't expired
3. Associated user exists and is active
4. User context is properly set

## API Versioning

Currently using v1 (implicit). Future versions will use:
- **URL versioning**: `/api/v2/feeds`
- **Header versioning**: `Accept: application/vnd.goread.v2+json`
- **Backward compatibility**: v1 maintained during transition periods

## Examples and SDKs

### JavaScript Example

```javascript
// Authenticated API request
async function getFeeds() {
  const response = await fetch('/api/feeds', {
    credentials: 'include'  // Include session cookie
  });
  
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }
  
  return await response.json();
}

// Add new feed
async function addFeed(url) {
  const response = await fetch('/api/feeds', {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ url })
  });
  
  return await response.json();
}
```

### curl Examples

```bash
# Login and save session cookie
curl -c cookies.txt -L "http://localhost:8080/auth/login"

# Use session cookie for API requests
curl -b cookies.txt "http://localhost:8080/api/feeds"

# Add feed with session
curl -b cookies.txt -X POST "http://localhost:8080/api/feeds" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/feed.xml"}'
```

### Python Example

```python
import requests

# Create session to maintain cookies
session = requests.Session()

# Login (follow redirects)
login_response = session.get('http://localhost:8080/auth/login', allow_redirects=True)

# Make authenticated API calls
feeds_response = session.get('http://localhost:8080/api/feeds')
feeds = feeds_response.json()

# Add new feed
add_response = session.post('http://localhost:8080/api/feeds', 
                           json={'url': 'https://example.com/feed.xml'})
```

This API provides comprehensive access to all GoRead2 functionality while maintaining security and user data isolation.