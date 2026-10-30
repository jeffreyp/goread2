# Caching Strategy

Describes GoRead2's HTTP caching (for static assets) and application-level caching (for reducing database queries).

## Table of Contents

- [Philosophy](#philosophy)
- [HTTP Caching](#http-caching)
- [Application-Level Caching](#application-level-caching)
- [Related Documentation](#related-documentation)

## Philosophy

Only cache what's safe and provides real value.

## HTTP Caching

### What Is Cached

**CSS and JavaScript (1 year)**: `Cache-Control: public, max-age=31536000`. Templates reference every CSS and JS file through a versioned URL, so a changed file always gets a new URL and browsers never use a stale copy.

**Other static files (10 minutes)**: images and favicons under `/static`, with `Cache-Control: public, max-age=600`. These URLs are not versioned, so a short lifetime keeps changes visible soon after a deploy.

**Everything else (no cache)**:
- API endpoints
- HTML pages
- Authentication endpoints
- User data

Caching dynamic content creates more problems than it solves: users see stale data, changes don't appear immediately, and inconsistent behavior across environments becomes hard to debug. HTML pages in particular stay uncached, because they carry the current asset versions.

### Versioned CSS and JS

`internal/assets` computes a SHA-256 hash of each file's content and templates call it through the `asset` template function:

```html
<script src="{{asset "js/app.min.js"}}"></script>
<!-- renders as /static/js/app.min.js?v=26ff6ce12ceb -->
```

Hashes are computed on first use and cached until the process restarts. If a file cannot be read, for example because the frontend has not been built, `asset` logs the error and returns the URL without a version.

`app.js` loads `modals.min.js` lazily on first use. `index.html` passes the versioned URL in a `data-modals-src` attribute on the `app.min.js` script tag, and `app.js` falls back to `./modals.js` when the attribute is absent, as in the Jest tests.

### Implementation

In production, App Engine serves `/static` directly and sets the expirations in `app.yaml` (see [deployment.md](deployment.md#static-file-handlers)). The local server sets the same headers in a middleware in `main.go`:

```go
if strings.HasPrefix(path, "/static/") {
    if strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".js") {
        c.Header("Cache-Control", "public, max-age=31536000")
    } else {
        c.Header("Cache-Control", "public, max-age=600")
    }
    c.Header("Vary", "Accept-Encoding")
}
```

### When Static Files Change

Rebuilding the frontend changes the hash of every modified file. The new versions take effect after a server restart locally, and after the next deploy in production. A browser with the old page open keeps using the old files until it reloads the page.

### Testing

```bash
# List the versioned asset URLs on the home page
curl -s http://localhost:8080/ | grep -o '/static/[^"]*?v=[0-9a-f]*'

# Check the cache header on a versioned asset (use a URL from the list above)
curl -I 'http://localhost:8080/static/js/app.min.js?v=26ff6ce12ceb'
# Cache-Control: public, max-age=31536000

# Check that API responses are not cached
curl -I http://localhost:8080/api/feeds
# Should NOT see Cache-Control header
```

### What Is Not Done

- No ETags for HTTP responses (adds complexity, buffers responses)
- No stale-while-revalidate (confusing behavior)
- No API caching (too risky for user data)

## Application-Level Caching

GoRead2 implements in-memory caching for expensive queries to reduce database costs, especially for Google Cloud Datastore. All caches are thread-safe and automatically expire.

### Caches in Use

#### Unread Count Cache (15 minutes TTL)

**Purpose:** Cache per-user unread article counts to avoid repeated database queries.

**Location:** `internal/cache/unread_cache.go`

**How it works:**
- Caches unread counts per user and feed
- **Incremental updates:** When marking articles read/unread, the cache is updated instantly
- Automatically invalidated on subscribe/unsubscribe operations
- TTL: 15 minutes, longer than the web client's 5-minute unread-count poll so that polls are served from the cache. Each cache miss on Datastore costs one read per recent article and one per matching `UserArticle`, roughly 10,000 reads for a typical account. Counts for newly fetched articles can lag by up to the TTL.
- **Max size:** 10,000 users (configured in `feed_service.go`). When full, the entry closest to expiry is evicted to make room for a new user.
- **Background cleanup:** `Start(ctx)` runs a periodic sweep that drops expired entries; `FeedService.Start(ctx)` calls it, wired from `main.go` so cleanup stops cleanly on shutdown.

**Benefits:**
- Avoids expensive COUNT queries on every page load
- Provides instant feedback when marking articles read
- Reduces database reads by ~70% for unread counts

**Example:**
```go
// Check cache first
if counts, hit := fs.unreadCache.Get(userID); hit {
    return counts, nil
}

// Cache miss - fetch from database
counts, err := fs.db.GetUserUnreadCounts(userID)
if err == nil {
    fs.unreadCache.Set(userID, counts)
}
return counts, err
```

**Invalidation triggers:**
- Subscribe to feed
- Unsubscribe from feed
- Batch article operations

#### Feed List Cache (20 minutes TTL)

**Purpose:** Cache the list of all user-subscribed feeds to reduce expensive `GetAllUserFeeds()` queries.

**Location:** `internal/cache/feed_list_cache.go`

**How it works:**
- Caches the complete list of feeds that have at least one subscriber
- Used during hourly feed refresh operations
- Automatically invalidated when users subscribe/unsubscribe
- TTL: 20 minutes
- **Max size:** 50,000 feeds (configured in `feed_service.go`). If the feed list exceeds this limit, `Set` is a no-op and the cache is bypassed.
- **Background cleanup:** like the unread cache, `Start(ctx)` runs a periodic sweep and is wired into `FeedService.Start(ctx)` for graceful shutdown.

**Benefits:**
- Reduces `GetAllUserFeeds()` queries from 48/day to ~3/day
- Saves ~$50-100/month in Datastore costs
- No impact on UX; new feed articles load immediately on subscribe

**Example:**
```go
// Check cache first
allUserFeeds, cached := fs.feedListCache.Get()
if !cached {
    // Cache miss - fetch from database
    var err error
    allUserFeeds, err = fs.db.GetAllUserFeeds()
    if err == nil {
        fs.feedListCache.Set(allUserFeeds)
    }
}
```

**Invalidation triggers:**
- User subscribes to a feed
- User unsubscribes from a feed

#### User Feeds Cache (5 minutes TTL, Datastore only)

**Purpose:** Cache each user's own subscribed feed list, which `GetUserArticlesPaginated` reads on every page of every "All Articles" or single-feed request.

**Location:** `internal/database/user_feeds_cache.go`

**How it works:**
- Caches the per-user result of `GetUserFeeds`, keyed by user ID
- Lives inside `DatastoreDB` rather than `internal/cache`, since `internal/cache` imports `internal/database` for the `Feed` type and a database-layer cache can't import it back without a cycle
- Automatically invalidated by `SubscribeUserToFeed` and `UnsubscribeUserFromFeed`
- TTL: 5 minutes, a safety net for drift between App Engine instances rather than the primary invalidation mechanism
- **Max size:** 10,000 users. When full, the entry closest to expiry is evicted to make room for a new user.
- Only implemented for `DatastoreDB`; the SQLite-backed `DB` used in local development and `schema_bench_test.go` benchmarks queries directly on every call.

**Benefits:**
- Removes a `UserFeed` query plus a `Feed` `GetMulti` from every paginated articles request for users who page through more than one screen of articles

**Invalidation triggers:**
- User subscribes to a feed
- User unsubscribes from a feed

### Cache Design Principles

1. **Copy on read/write:** All caches return copies to prevent external modification
2. **Thread-safe:** Uses sync.RWMutex for concurrent access
3. **TTL-based expiration:** Automatic expiration prevents stale data
4. **Explicit invalidation:** Critical operations invalidate caches immediately
5. **Graceful degradation:** Cache misses fall back to database queries
6. **Bounded size:** Configurable max-size limits prevent unbounded memory growth; `SetMaxUsers` / `SetMaxFeeds` configure the cap, and `0` means unlimited (the default before limits are applied)

### Cost Savings

- **Unread count cache:** ~70% reduction in count queries
- **Feed list cache:** ~45 fewer `GetAllUserFeeds()` queries/day
- **Total estimated savings:** $50-150/month in database costs

### Testing

All caches have test coverage including basic get/set operations, TTL expiration, concurrent access, invalidation, copy safety, and max-size/eviction behavior.

```bash
go test ./internal/cache/...
```

### Monitoring

Check cache statistics:
```go
// Unread cache stats
stats := unreadCache.GetStats()
fmt.Printf("Cached users: %d/%d, Total feeds: %d, Hits: %d, Misses: %d, HitRate: %.2f\n",
    stats.CachedUsers, stats.MaxUsers, stats.TotalFeeds, stats.Hits, stats.Misses, stats.HitRate)

// Feed list cache stats
flStats := feedListCache.GetStats()
fmt.Printf("Cached feeds: %d/%d, Is valid: %v, Hits: %d, Misses: %d, HitRate: %.2f\n",
    flStats.CachedFeeds, flStats.MaxFeeds, flStats.IsValid, flStats.Hits, flStats.Misses, flStats.HitRate)

// Session cache stats
sessionStats := sm.GetCacheStats()
fmt.Printf("Sessions: %d active, Hits: %d, Misses: %d, HitRate: %d%%\n",
    sessionStats["active"], sessionStats["hits"], sessionStats["misses"], sessionStats["hit_rate"])
```

## Related Documentation

- [Performance & Cost Optimization](performance.md) - Caching in the context of overall cost strategy
- [Troubleshooting Guide](troubleshooting.md) - Common issues
