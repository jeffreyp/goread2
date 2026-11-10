# Architecture

How the parts of GoRead2 fit together: the clients, the Go server and its packages, the two database implementations, the per-instance caches, background work, and the data model.

## Table of Contents

- [Overview](#overview)
- [Clients](#clients)
- [Server](#server)
- [Persistence](#persistence)
- [Instances and In-Memory State](#instances-and-in-memory-state)
- [Background Work](#background-work)
- [External Services](#external-services)
- [Data Model](#data-model)
- [Related Documentation](#related-documentation)

## Overview

```
 Web frontend         iOS, iPadOS, macOS app
 (web/)               (ios/)
        \               /
         REST API, session cookie, CSRF header
                 |
     Go server on App Engine (main.go, internal/)
     one or more instances, each with its own caches
        |              |                 |
   Datastore     Secret Manager     Stripe, Google OAuth
        ^
        |
   App Engine Cron -> /cron/* -> Cloud Tasks -> /tasks/*
```

One Go binary serves the web pages, the JSON API, and the scheduled jobs. It runs on Google App Engine standard with automatic scaling from zero to ten instances, and stores everything in Google Cloud Datastore. Locally the same binary runs against a SQLite file.

## Clients

- **Web frontend**: Go templates in `web/templates/` render the pages, and the JavaScript in `web/static/js/` drives them through the API. `app.js` is the reader, `account.js` the account page, and `modals.js` the shared dialogs. `make build-frontend` minifies them; the minified files are not committed. [features.md](features.md) describes the interface.
- **Native app**: a SwiftUI app in `ios/` builds for iOS, iPadOS, and macOS from one target and uses the same API. It signs in through a one-time code exchange and sends the session token as the cookie the web uses (see [authentication.md](authentication.md#native-client-flow-ios-and-macos) and [ios.md](ios.md)).

Both clients authenticate with the session cookie and send the CSRF token from `GET /auth/me` on every mutating request. [api.md](api.md) documents the API.

## Server

`main.go` loads configuration, opens the database, builds the services, and registers the routes on a Gin router. Every request passes through gzip compression, the security headers, CORS, and the request body limit. The route groups then add their own middleware:

| Routes | Middleware | Purpose |
|--------|------------|---------|
| `/auth/*` | Rate limit | Sign-in, sign-out, the native code exchange, and `GET /auth/me` |
| `/api/*` | Rate limit, `RequireAuth`, request-scoped cache, CSRF | The JSON API used by both clients |
| `/admin/*` | `RequireAdmin`, CSRF | Admin operations (see [admin.md](admin.md)) |
| `/cron/*`, `/tasks/*` | App Engine header checks | Scheduled jobs and their Cloud Tasks workers |
| `/webhooks/stripe` | Rate limit, Stripe signature | Subscription events, registered only when subscriptions are enabled |
| `/`, `/account`, `/privacy`, `/subscription/*` | Page authentication where needed | HTML pages |

The code under `internal/` is split by layer:

| Package | Holds |
|---------|-------|
| `handlers` | HTTP handlers: parse the request, call services, write the response |
| `services` | Business logic: feed fetching and parsing, subscriptions and Stripe, account export and deletion, retention, audit logging, usage statistics, Cloud Tasks |
| `database` | The `Database` interface and its SQLite and Datastore implementations |
| `auth` | OAuth, sessions, CSRF tokens, rate limiters, client IP resolution, and the cron and task request checks |
| `middleware` | Security headers, CORS, body limits, the request-scoped cache, and the 404 handler |
| `cache` | The per-instance feed list and unread count caches |
| `config`, `secrets` | Environment variables and Secret Manager lookups (see [configuration.md](configuration.md)) |
| `assets` | Content hashes for the `?v=` query on CSS and JS URLs (see [caching.md](caching.md#versioned-css-and-js)) |

The command-line tools in `cmd/` use the same packages: `admin` for user management, `setup-stripe` for the Stripe product, `validate-config` for `make validate-config`, and `generate-test-articles` for test data.

## Persistence

Services reach storage only through `database.Database`, an interface in `internal/database/schema.go`. `database.InitDB` picks the implementation:

- **Datastore** (`DatastoreDB`, `datastore.go`) when `GOOGLE_CLOUD_PROJECT` is set. Production, staging, and the emulator-backed tests use it.
- **SQLite** (`DB`, `schema.go`) otherwise, in the file `./goread2.db`. Local development and most tests use it.

Both implementations must behave the same. The Datastore tests mirror the SQLite tests and run against the Datastore emulator (see [testing.md](testing.md)). The one exception to the interface is admin tokens, which `SubscriptionService` reads and writes directly through the Datastore client or SQL connection.

Staging and production share one Datastore database, `(default)` in project `goread-467200`, which has point-in-time recovery enabled (see [deployment.md](deployment.md#point-in-time-recovery)).

## Instances and In-Memory State

App Engine runs between zero and ten instances, and each keeps its own in-memory state. Nothing is shared between instances except the database, so a cached value can be stale on one instance after another changes it:

- **Sessions** are cached for 10 minutes by default (`SESSION_CACHE_TTL`). A revoked session can keep working on another instance until its cache entry expires, which is why account deletion runs a second pass after 15 minutes (see [privacy.md](privacy.md#account-deletion)).
- **Unread counts, the feed list, and each user's subscriptions** are cached with short lifetimes and updated in place on writes from the same instance. [caching.md](caching.md#caches-in-use) lists every cache with its lifetime and invalidation.
- **Rate limiters** count per instance, so the effective limit grows with the number of instances.
- **Usage statistics** are reused for 15 minutes.

On `SIGTERM`, which App Engine sends before stopping an instance, the server stops accepting connections and gives in-flight requests up to 10 seconds to finish.

## Background Work

GoRead2 runs no always-on background loop that does work on a schedule. All scheduled work starts from App Engine Cron, defined in `cron.yaml`:

1. App Engine Cron sends a request to a `/cron/*` endpoint on the default version.
2. On App Engine, the handler enqueues a Cloud Task for the matching `/tasks/*` endpoint and returns at once. Locally, without Cloud Tasks, it does the work in the request.
3. Cloud Tasks delivers the task, retrying it on failure, and the worker endpoint does the work.

The jobs refresh feeds every two hours, clean up expired sessions and orphaned article state daily, apply the retention policy monthly, retry failed account deletions daily, and record usage metrics daily. Account deletion uses the same queue. [deployment.md](deployment.md#async-task-processing-cloud-tasks) covers the queue and its routing.

A feed refresh fetches feeds concurrently, limits requests to each feed domain, spreads fetches over a window, and uses each feed's `ETag` and `Last-Modified` values for conditional requests (see [configuration.md](configuration.md#feed-fetching)). Users can also start a refresh of their own feeds from the clients.

## External Services

| Service | Used for |
|---------|----------|
| Google OAuth | Sign-in |
| Google Cloud Datastore | All persistent data |
| Google Secret Manager | OAuth, CSRF, Stripe, and other secrets (see [configuration.md](configuration.md#how-values-are-resolved)) |
| Cloud Tasks | Running scheduled jobs and account deletion |
| Cloud Logging and Cloud Monitoring | Logs, alerts, and usage metrics (see [monitoring.md](monitoring.md)) |
| Stripe | Checkout, the customer portal, and subscription webhooks (see [stripe.md](stripe.md)) |
| Feed publishers | The RSS and Atom feeds the server fetches |

## Data Model

Each SQLite table has a Datastore kind with the same fields. Datastore needs no schema; the composite indexes its queries use are defined in `index.yaml`. SQLite tables are created, and missing columns added, at startup by `migrateDatabase` in `schema.go`.

| Kind | SQLite table | Datastore key | Holds |
|------|--------------|---------------|-------|
| `User` | `users` | Numeric ID | One account: Google identity, subscription state, admin flag, free months, and settings |
| `Feed` | `feeds` | Numeric ID | One feed URL, shared by every subscriber, with its fetch state |
| `Article` | `articles` | Numeric ID | One article from a feed, shared by every subscriber |
| `UserFeed` | `user_feeds` | `<user_id>_<feed_id>` | A user's subscription to a feed |
| `UserArticle` | `user_articles` | `<user_id>_<article_id>` | A user's read and starred state for an article. A missing record means unread and unstarred. |
| `Session` | `sessions` | The session ID | A signed-in session and its expiry |
| `UserActivity` | `user_activity` | The user's ID | When the account was last used, for the retention policy |
| `AccountTombstone` | `account_tombstones` | HMAC of a deleted account's Google ID | A deleted account's trial history (see [privacy.md](privacy.md#account-deletion)) |
| `AdminToken` | `admin_tokens` | Numeric ID | The hash of an admin CLI token (see [admin.md](admin.md#admin-token-system)) |
| `AuditLog` | `audit_logs` | Numeric ID | One admin action (see [admin.md](admin.md#audit-logging)) |

Feeds and articles are global: adding a feed that another user already follows links the existing `Feed` instead of creating a second one, and each article is stored once. Everything specific to a user lives in the user-keyed kinds. [privacy.md](privacy.md#personal-data-stored) lists which of these hold personal data.

The SQLite definitions, from `CreateTables` in `internal/database/schema.go`:

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    google_id TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    avatar TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    subscription_status TEXT DEFAULT 'trial',  -- trial, active, past_due, unpaid, cancelled, expired
    subscription_id TEXT,                      -- Stripe subscription ID
    trial_ends_at DATETIME,
    last_payment_date DATETIME,
    next_billing_date DATETIME,
    is_admin BOOLEAN DEFAULT 0,
    free_months_remaining INTEGER DEFAULT 0,
    max_articles_on_feed_add INTEGER DEFAULT 100,
    deletion_requested_at DATETIME             -- set while a requested deletion is pending
);

CREATE TABLE feeds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    url TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_fetch DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_checked DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_had_new_content DATETIME DEFAULT CURRENT_TIMESTAMP,
    average_update_interval INTEGER DEFAULT 0,
    etag TEXT DEFAULT '',
    last_modified TEXT DEFAULT ''
);

CREATE TABLE articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_id INTEGER NOT NULL REFERENCES feeds (id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    url TEXT UNIQUE NOT NULL,
    content TEXT,
    description TEXT,
    author TEXT,
    published_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_feeds (
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    feed_id INTEGER NOT NULL REFERENCES feeds (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, feed_id)
);

CREATE TABLE user_articles (
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    article_id INTEGER NOT NULL REFERENCES articles (id) ON DELETE CASCADE,
    is_read BOOLEAN DEFAULT FALSE,
    is_starred BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (user_id, article_id)
);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);

CREATE TABLE user_activity (
    user_id INTEGER PRIMARY KEY,
    last_seen_at DATETIME NOT NULL
);

CREATE TABLE account_tombstones (
    id TEXT PRIMARY KEY,                       -- HMAC-SHA256 of the Google ID
    deleted_at DATETIME NOT NULL,
    expires_at DATETIME                        -- NULL for a tombstone that never expires
);

CREATE TABLE admin_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash TEXT UNIQUE NOT NULL,           -- SHA-256 of the token
    description TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT 1                -- 0 once revoked
);

CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    admin_user_id INTEGER,                     -- 0 for the CLI
    admin_email TEXT NOT NULL,                 -- CLI_ADMIN for the CLI
    operation_type TEXT NOT NULL,
    target_user_id INTEGER,
    target_user_email TEXT,
    operation_details TEXT,                    -- JSON
    ip_address TEXT,                           -- CLI for the CLI
    result TEXT NOT NULL,                      -- success or failure
    error_message TEXT
);
```

The schema in `schema.go` declares foreign keys as separate `FOREIGN KEY` clauses; they are shown inline here for brevity.

## Related Documentation

- [API Reference](api.md) - The endpoints both clients use
- [Configuration Reference](configuration.md) - Every environment variable
- [Deployment Guide](deployment.md) - App Engine, Cloud Tasks, and the deploy pipeline
- [Privacy and Personal Data](privacy.md) - What the data model holds about users
- [Caching Strategy](caching.md) - The per-instance caches in detail
- [Testing Guide](testing.md) - Running the suite against SQLite and Datastore
