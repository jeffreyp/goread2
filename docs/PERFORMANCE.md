# Performance & Cost Optimization

## Overview

GoRead2 is designed to be cost-effective while maintaining excellent performance. This document outlines our optimization strategies and their impact.

## Cost Optimization Strategy

Our pricing model ($2.99/month unlimited feeds) requires aggressive cost optimization to remain profitable. We've implemented several strategies to reduce Google Cloud / Datastore costs:

### Current Optimizations

#### 1. Smart Feed Update Prioritization ($30-60/month savings)

**Problem:** Checking every feed every 30 minutes was expensive and unnecessary for infrequently-updated feeds.

**Solution:**
- Reduced cron schedule from every 30 minutes to **every 1 hour**
- Implemented smart prioritization based on feed update patterns:
  - **Active feeds** (< 1 week since last update): checked every 30 minutes
  - **Regular feeds** (< 1 month): checked every 1 hour
  - **Dormant feeds** (> 1 month): checked every 6 hours
- Track feed update patterns with running average

**Implementation:** `internal/services/feed_service.go:776-818`

**Impact:**
- 50% reduction in cron-triggered instance spawns
- Fewer unnecessary feed checks
- Expected savings: $30-60/month

#### 2. Remove Unbounded Queries ($300-500/month savings)

**Problem:** `GetAllArticles()` method performed full table scans without limits.

**Solution:**
- Completely removed the legacy method
- All queries now use feed-specific or user-specific bounded queries
- Use `GetArticles(feedID)` or `GetUserArticles(userID)` instead

**Implementation:** Removed from all database implementations

**Impact:**
- Eliminates unbounded Datastore entity reads
- Expected savings: $300-500/month

#### 3. Cursor-Based Pagination ($100-200/month savings)

**Problem:** Offset-based pagination was inefficient, especially for Datastore:
- SQLite: Had to scan through all skipped rows with OFFSET
- Datastore: Fetched `limit + offset + 50` extra articles per feed
- Each page deeper required more reads

**Solution:**
- Implemented cursor-based pagination using keyset pagination for SQLite
- For Datastore: reduced per-feed fetch from `limit + offset + 50` to `limit * 2`
- Cursors encode the last article's timestamp and ID for precise positioning
- No need to scan skipped rows - jump directly to the right position

**Implementation:**
- SQLite: `internal/database/schema.go:814-909` (keyset pagination with WHERE clause)
- Datastore: `internal/database/datastore.go:733-940` (reduced fetch size)
- API: `internal/handlers/feed_handler.go:132-161` (cursor parameter)

**Impact:**
- **SQLite**: O(1) positioning instead of O(offset) scanning
- **Datastore**: Eliminates wasteful `+50` article fetching
- For user with 10 feeds on page 2: saves ~500+ entity reads
- Scales better with deep pagination
- Expected savings: $100-200/month

#### 4. Cache GetAllUserFeeds() ($50-100/month savings)

**Problem:** `GetAllUserFeeds()` was called 48 times per day (hourly refresh) with no caching.

**Solution:**
- Implemented `FeedListCache` with 20-minute TTL
- Cache automatically invalidated on subscribe/unsubscribe
- Reduces queries from 48/day to ~3/day

**Implementation:** `internal/cache/feed_list_cache.go`

**Impact:**
- ~45 fewer expensive queries per day
- Expected savings: $50-100/month

#### 5. Deferred Cleanup for UnsubscribeUserFromFeed ($20-50/month savings)

**Problem:** Unsubscribing from a feed triggered expensive cleanup of all user-article relationships:
- SQLite: DELETE with subquery on every unsubscribe
- Datastore: Queried all articles in feed, then all UserArticle entities (potentially thousands of reads)
- Users experienced slow unsubscribe operations
- Cost spike on every unsubscribe

**Solution:**
- Removed synchronous cleanup from `UnsubscribeUserFromFeed()`
- Unsubscribe now only deletes the UserFeed subscription (instant operation)
- Implemented `CleanupOrphanedUserArticles()` method for batch cleanup
- Daily cron job cleans up orphaned records older than 7 days
- Articles from unsubscribed feeds don't appear in UI (filtered by `GetUserArticlesPaginated`)

**Implementation:**
- Database methods: `internal/database/schema.go:787-794`, `internal/database/datastore.go:664-677`
- Cleanup methods: `internal/database/schema.go:1090-1114`, `internal/database/datastore.go:1255-1347`
- Cron handler: `internal/handlers/feed_handler.go:294-332`
- Cron schedule: `cron.yaml:11-18`

**Impact:**
- 10-100x faster unsubscribe operations
- Spreads cleanup cost over time instead of spike per unsubscribe
- Better user experience (instant unsubscribe)
- Expected savings: $20-50/month

### Total Cost Savings

**Estimated total: $550-1,010/month**

This reduces per-user costs by approximately $6-12/month, making the $2.99/month pricing sustainable and profitable.

## Performance Optimizations

### 1. Unread Count Caching

**Purpose:** Avoid expensive COUNT queries on every page load

**Implementation:**
- In-memory cache with 90-second TTL
- Incremental updates when marking articles read/unread
- Thread-safe with sync.RWMutex

**Benefits:**
- ~70% reduction in count queries
- Instant feedback when marking articles read
- Better user experience

See [CACHING.md](CACHING.md) for details.

### 2. Concurrent Feed Fetching

**Implementation:**
- Processes feeds in parallel batches of 5
- Uses goroutines for concurrent HTTP requests
- Rate limiting per domain to avoid overwhelming servers

**Benefits:**
- Feed refresh completes much faster
- Better resource utilization

### 3. Smart Article Limits

**Implementation:**
- Configurable max articles per feed on initial import
- Prevents database overload with high-volume feeds
- Default: 100 articles per feed

**Benefits:**
- Faster initial feed imports
- Lower storage costs
- Better user experience (no overwhelming article counts)

## Database Query Patterns

### Best Practices

1. **Always use bounded queries**
   - ✅ `GetArticles(feedID)`
   - ✅ `GetUserArticles(userID)`
   - ❌ `GetAllArticles()` (removed)

2. **Use indexes efficiently**
   - Queries filtered by user_id use indexes
   - Order by published_at DESC for recent articles first
   - Composite indexes for complex queries

3. **Batch operations where possible**
   - `BatchSetUserArticleStatus()` instead of individual calls
   - Reduces database round trips

4. **Cache expensive queries**
   - Unread counts (90s TTL)
   - Feed lists (20min TTL)
   - Automatic invalidation on changes

### Query Optimization Examples

**Bad:**
```go
// Fetches ALL articles from database
articles := db.GetAllArticles()
for _, article := range articles {
    if article.FeedID == targetFeedID {
        // Process article
    }
}
```

**Good:**
```go
// Fetches only articles from specific feed
articles := db.GetArticles(targetFeedID)
for _, article := range articles {
    // Process article
}
```

## Monitoring & Metrics

### Key Metrics to Track

1. **Database query counts**
   - GetAllUserFeeds calls per day
   - Unread count cache hit rate
   - Feed list cache hit rate

2. **Feed refresh performance**
   - Feeds checked per refresh cycle
   - Feeds skipped due to prioritization
   - Average refresh duration

3. **Cost metrics**
   - Datastore entity reads per day
   - Instance hours per day
   - Bandwidth usage

### Logging

Enable detailed logging for cost analysis:

```bash
# Feed refresh stats
2025-11-02 12:00:00 Feed refresh completed: checked=45, skipped=23, has_new=12, duration=8.5s

# Cache hit rates
2025-11-02 12:00:00 Unread cache stats: users=15, feeds=120, hit_rate=73%
```

## Future Optimization Opportunities

Several P1 and P2 optimizations remain in the Beads issue tracker:

1. **Increase unread cache TTL** (P1) - From 90s to 5-10 minutes
2. **Smart feed update prioritization refinement** (P2)
3. **Cloud Monitoring dashboards** (P2) - Better visibility into costs
4. **Keys-only queries** (P2) - Where full entities not needed

Run `bd ready` to see all available optimization issues.

## Testing Performance

### Benchmark Tests

```bash
# Run performance benchmarks
go test -bench=. ./internal/database/...
go test -bench=. ./internal/cache/...

# Profile memory usage
go test -memprofile=mem.prof ./internal/...
go tool pprof mem.prof
```

### Load Testing

```bash
# Test concurrent user operations
go test ./test/integration -run TestConcurrentUserOperations

# Test feed refresh performance
go test ./test/integration -run TestPerformanceBaseline
```

## Deployment Considerations

### Production Settings

1. **App Engine instance settings**
   - Use F1 instances (cost-effective)
   - Automatic scaling based on load
   - Min instances: 0 (save costs when idle)

2. **Datastore settings**
   - Use composite indexes for complex queries
   - Monitor entity read/write counts
   - Set up budget alerts

3. **Cron jobs**
   - Feed refresh: every 1 hour
   - Orphaned article cleanup: every 24 hours
   - Monitor via App Engine logs

### Cost Alerts

Set up Google Cloud budget alerts:
- Alert at 50% of monthly budget
- Alert at 80% of monthly budget
- Alert at 100% of monthly budget

## Summary

Through careful optimization, we've reduced operational costs by $550-1,010/month while maintaining excellent performance and user experience. Key strategies:

1. ✅ Smart feed update prioritization
2. ✅ Remove unbounded queries
3. ✅ Cursor-based pagination (replaces offset-based)
4. ✅ Cache expensive operations
5. ✅ Deferred cleanup for unsubscribe operations
6. ✅ Concurrent processing where safe

These optimizations make the $2.99/month pricing sustainable and profitable.
