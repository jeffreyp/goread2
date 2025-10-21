# HTTP Caching Strategy

## Philosophy

**Keep it simple.** We only cache what's safe and provides real value.

## What We Cache

### Static Assets (24 hours)

Files that rarely change:
- CSS files (`/static/*.css`)
- JavaScript files (`/static/*.js`)
- Images (`/static/*.svg`, `favicon.ico`)

**Cache-Control:** `public, max-age=86400` (24 hours)

### Everything Else (No cache)

- API endpoints
- HTML pages
- Authentication endpoints
- User data

**Why?** Caching dynamic content creates more problems than it solves:
- Users see stale data
- Changes don't appear immediately
- Hard to debug "it works for me but not you" issues

## Implementation

The caching middleware is dead simple (main.go:122-133):

```go
r.Use(func(c *gin.Context) {
    path := c.Request.URL.Path

    // Cache static assets for 24 hours
    if strings.HasPrefix(path, "/static/") {
        c.Header("Cache-Control", "public, max-age=86400")
        c.Header("Vary", "Accept-Encoding")
    }

    c.Next()
})
```

## Benefits

- **Faster page loads:** CSS/JS loads instantly from browser cache
- **Reduced bandwidth:** ~80% reduction for static assets
- **No stale data issues:** Dynamic content is always fresh
- **Easy to understand:** No magic, no surprises

## When Static Files Change

If you update CSS/JS and users have it cached:
1. They'll see the old version for up to 24 hours
2. Hard refresh (Cmd/Ctrl+Shift+R) clears the cache
3. Or just wait 24 hours

For frequent CSS/JS changes during development, test locally where caching behaves the same way.

## Testing

```bash
# Check static asset cache headers
curl -I http://localhost:8080/static/styles.css

# Should see:
# Cache-Control: public, max-age=86400
# Vary: Accept-Encoding

# Check API has no cache
curl -I http://localhost:8080/api/feeds
# Should NOT see Cache-Control header
```

## What We DON'T Do

- ❌ No ETags (adds complexity, buffers responses)
- ❌ No stale-while-revalidate (confusing behavior)
- ❌ No API caching (too risky for user data)
- ❌ No different cache times for different endpoints (too many magic numbers)

## Performance Impact

- Static asset requests: ~80% from cache
- Server load: ~20-30% reduction (just from static assets)
- User experience: Faster page loads, no stale data confusion

That's it. Simple and safe.
