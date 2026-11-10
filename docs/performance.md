# Performance and Caching

How GoRead2 keeps Datastore reads, instance time, and page latency low, and the HTTP and in-memory caches it uses to do so. [monitoring.md](monitoring.md) covers the dashboards, budget, and alerts that track the resulting cost.

## Table of Contents

- [Feed Refresh](#feed-refresh)
- [Database Queries](#database-queries)
- [Client Loading](#client-loading)
- [HTTP Caching](#http-caching)
- [In-Memory Caches](#in-memory-caches)
- [Testing Performance](#testing-performance)
- [Troubleshooting](#troubleshooting)
- [Related Documentation](#related-documentation)

## Feed Refresh

App Engine Cron starts a refresh of every subscribed feed every 2 hours, through the `/tasks/refresh-feeds` Cloud Task (see [deployment.md](deployment.md#async-task-processing-cloud-tasks)). `FeedScheduler.RefreshFeedsStaggered` runs it:

- **Only due feeds are fetched.** `FeedService.shouldCheckFeed` skips a feed checked too recently. A feed with a known update frequency is due once half its average interval has passed since the last check. Otherwise the interval depends on when the feed last had new content: 30 minutes within the last week, 1 hour within the last month, and 6 hours after that. A feed that has never had new content is checked hourly. Because cron runs every 2 hours, every interval shorter than that means "on every run".
- **Update frequency adapts.** Each fetch that finds new content updates the feed's average update interval as a weighted average, 70% the previous value and 30% the latest interval.
- **Unchanged feeds cost little.** Each fetch sends the `ETag` and `Last-Modified` values from the feed's previous response as `If-None-Match` and `If-Modified-Since`. A `304 Not Modified` response skips downloading and parsing the feed.
- **Fetches are spread and bounded.** The scheduler spreads fetches across `SCHEDULER_UPDATE_WINDOW`, runs at most `SCHEDULER_MAX_CONCURRENT` at once, and limits the requests sent to each feed domain (see [configuration.md](configuration.md#feed-fetching)).

At the end, the scheduler logs `Feed refresh complete: checked=<n>, skipped=<n>, had_new_content=<n>`.

A user's manual refresh, `POST /api/feeds/refresh`, fetches only that user's feeds and applies the same due check.

## Database Queries

- **Bounded queries.** Every article query is limited to one feed or one user, through methods such as `GetArticles(feedID)` and `GetUserArticles(userID)`. No code path scans every article.
- **Cursor pagination.** Article lists page with a cursor that encodes the last article's publish time and ID, so a deep page costs the same as the first. On SQLite, `GetUserArticlesPaginated` in `schema.go` positions with a `WHERE` clause. On Datastore, `getUserArticlesPaginated` in `datastore.go` runs one projection query per feed for publish times at or below the cursor, fetching `limit + 1` references from each. A feed that fills its window may have older articles, so references are trusted only down to the frontier, the newest of those feeds' oldest publish times. Articles exactly at the frontier are fetched by an equality query, because feeds with day-only or missing dates give many articles the same timestamp; that query reads at most 500 articles per feed, since feed content is external and could otherwise force an unbounded read on every page. Unread-only pages that skip read articles continue in further rounds below the frontier, capped at 1,000 scanned articles or 10 rounds per request; at the cap the page is returned short, with a cursor.
- **Concurrent projections.** `projectArticleRefs` runs the per-feed projection queries in parallel, at most `maxConcurrentFeedProjections` (25) at a time, so a user with 50 feeds waits for about two rounds of queries per page.
- **Deferred unsubscribe cleanup.** `UnsubscribeUserFromFeed` deletes only the `UserFeed` record. The daily `/cron/cleanup-orphaned-articles` job, through `CleanupOrphanedUserArticles`, later deletes `UserArticle` records for feeds the user no longer follows, once they are more than 7 days old. Article lists exclude unsubscribed feeds in the meantime.
- **Batch writes.** On subscribing, `BatchSetUserArticleStatus` writes the user's state for the feed's articles in one call rather than one write per article.
- **Indexes.** The composite indexes that Datastore queries need are in `index.yaml`, deployed with each staging deploy.
- **Slow query logging.** Datastore operations that take longer than 200 ms are logged by `logSlowQuery` in `datastore.go`. The SQLite implementation does not log slow queries.

## Client Loading

- **Content on demand.** List endpoints return articles without their `content`. The web app fetches the full content from `GET /api/articles/:id` when an article is opened, then prefetches the previous and next articles so that `j` and `k` show content without waiting. A prefetch still in flight when its article is opened is reused. The native app does the same (see [ios.md](ios.md#data-loading)).
- **Parallel startup.** `GoReadApp.init` in `web/static/js/app.js` requests `/api/feeds`, `/api/feeds/unread-counts`, and the first page of All Articles at once. The All Articles entry is static markup in `index.html`, so its articles do not wait for the feed list.
- **Article limit on subscribing.** Subscribing to a feed marks at most `max_articles_on_feed_add` of its most recent articles as unread for the user, 100 by default. Each user can change the limit on the account page, and 0 removes it.

## HTTP Caching

**CSS and JavaScript** are cached for a year (`Cache-Control: public, max-age=31536000`). Templates reference every CSS and JS file through a versioned URL, so a changed file always gets a new URL and browsers never use a stale copy.

**Other static files**, such as images and favicons under `/static`, are cached for 10 minutes (`Cache-Control: public, max-age=600`). These URLs are not versioned, so a short lifetime keeps changes visible soon after a deploy.

**API responses, HTML pages, and authentication endpoints** are not cached. HTML pages carry the current asset versions, so caching them would serve outdated asset URLs. The server sends no ETags and no `stale-while-revalidate` directives.

In production, App Engine serves `/static` directly and sets these expirations in `app.yaml` (see [deployment.md](deployment.md#static-file-handlers)). The local server sets the same headers in a middleware in `main.go`.

### Versioned CSS and JS

`internal/assets` computes a SHA-256 hash of each file's content, and templates call it through the `asset` template function:

```html
<script src="{{asset "js/app.min.js"}}"></script>
<!-- renders as /static/js/app.min.js?v=<hash> -->
```

Hashes are computed on first use and cached until the process restarts. If a file cannot be read, for example because the frontend has not been built, `asset` logs the error and returns the URL without a version.

`app.js` loads `modals.min.js` lazily on first use. `index.html` passes the versioned URL in a `data-modals-src` attribute on the `app.min.js` script tag, and `app.js` falls back to `./modals.js` when the attribute is absent, as in the Jest tests.

Rebuilding the frontend changes the hash of every modified file. The new versions take effect after a server restart locally, and after the next deploy in production. A browser with the old page open keeps using the old files until it reloads the page.

To check the headers locally:

```bash
# List the versioned asset URLs on the home page
curl -s http://localhost:8080/ | grep -o '/static/[^"]*?v=[0-9a-f]*'

# A versioned asset, using a URL from the list above, is cached for a year
curl -I 'http://localhost:8080/static/js/app.min.js?v=<hash>'

# API responses carry no Cache-Control header
curl -I http://localhost:8080/api/feeds
```

## In-Memory Caches

Each App Engine instance keeps its own caches; instances share nothing but the database (see [architecture.md](architecture.md#instances-and-in-memory-state)). All of the caches below are safe for concurrent use, return copies rather than shared values, expire entries after a fixed lifetime, and fall back to the database on a miss.

| Cache | Location | Lifetime | Size limit | Invalidated by |
|-------|----------|----------|------------|----------------|
| Sessions | `SessionManager` in `internal/auth/session.go` | 10 minutes, set by `SESSION_CACHE_TTL` | None | Sign-out and session revocation on the same instance |
| Unread counts | `internal/cache/unread_cache.go` | 15 minutes | 10,000 users | Read-state changes, which update counts in place, and subscribing or unsubscribing |
| Feed list | `internal/cache/feed_list_cache.go` | 20 minutes | 50,000 feeds | Subscribing or unsubscribing |
| Each user's feeds, Datastore only | `internal/database/user_feeds_cache.go` | 5 minutes | 10,000 users | `SubscribeUserToFeed` and `UnsubscribeUserFromFeed` |
| Per request | `internal/middleware/request_cache.go` | One API request | None | Feed mutations within the request |
| Usage statistics | `StatsService` | 15 minutes | One entry | Expiry only |

- **Sessions.** A revoked session can stay valid on another instance until its cache entry expires, which is why account deletion runs a second pass after 15 minutes (see [privacy.md](privacy.md#account-deletion)).
- **Unread counts.** The lifetime is longer than the web client's 5-minute unread-count poll, so polls are served from the cache. A miss on Datastore costs one read per recent article and one per matching `UserArticle`. Counts for newly fetched articles can lag by up to the lifetime. When the cache is full, the entry closest to expiry is evicted.
- **Feed list.** Holds every feed with at least one subscriber, for `FeedService.RefreshFeeds`. The cron refresh goes through `FeedScheduler`, which reads the feed list directly, so in production this cache is not read. If the list exceeds the size limit, the cache is bypassed.
- **Each user's feeds.** `GetUserArticlesPaginated` reads the user's feed list on every page, and this cache removes a `UserFeed` query and a `Feed` read from each. It lives in `internal/database` rather than `internal/cache`, because `internal/cache` imports `internal/database`. The 5-minute lifetime bounds drift between instances. The SQLite implementation has no such cache.
- **Per request.** Holds the signed-in user's feeds for the length of one API request, so handlers that need them more than once read them once.

The unread count and feed list caches start background sweeps of expired entries from `FeedService.Start`, which stop when the server shuts down. `go test ./internal/cache/...` covers expiry, concurrency, invalidation, copying, and eviction.

## Testing Performance

Go benchmarks in `internal/database/schema_bench_test.go`, such as `BenchmarkGetUserArticlesPaginatedFirstPage` and `BenchmarkGetUserUnreadCounts`, measure the main queries. A `benchmark` job in CI fails when one regresses (see [testing.md](testing.md#ci-benchmark-regression-gate-scriptscheck-benchmark-regressionsh)).

```bash
go test -bench=. ./internal/database/... ./internal/cache/...

# Memory profile, one package per run
go test -run '^$' -bench . -benchmem -memprofile /tmp/goread2-mem.prof ./internal/database/
go tool pprof -top -sample_index=alloc_space /tmp/goread2-mem.prof
```

`go test ./test/integration -run TestConcurrentUserOperations` exercises concurrent operations by several users.

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md#performance-issues) for slow pages, and [Feed Issues](troubleshooting.md#feed-issues) for refresh problems.

## Related Documentation

- [Architecture](architecture.md) - Instances, caches, and background work in context
- [Monitoring](monitoring.md) - Dashboards, the budget, and cost alerts
- [Configuration Reference](configuration.md#feed-fetching) - The feed fetching variables
- [Testing Guide](testing.md) - The benchmark regression gate
