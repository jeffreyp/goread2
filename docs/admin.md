# Admin Guide

Managing users, admin access, free months, and subscriptions in GoRead2, through the admin CLI and the admin API, and reading the audit log they write.

## Table of Contents

- [Granting the First Admin](#granting-the-first-admin)
- [Admin Token System](#admin-token-system)
- [Admin Script Usage](#admin-script-usage)
- [Access Levels](#access-levels)
- [Audit Logging](#audit-logging)
- [Admin API](#admin-api)
- [Usage Statistics](#usage-statistics)
- [Database Schema](#database-schema)
- [Troubleshooting](#troubleshooting)
- [Related Documentation](#related-documentation)

## Granting the First Admin

The recommended way to create the first admin is `INITIAL_ADMIN_EMAILS`, a comma-separated list of email addresses. In production it is the Secret Manager secret `initial-admin-emails` (see [configuration.md](configuration.md#administration)); locally it is an exported variable:

```bash
export INITIAL_ADMIN_EMAILS="admin@example.com"
make dev
```

A new account whose email is in the list, compared without regard to case, is made an admin when sign-in creates it, provided Google reports the email as verified. The local sign-in of `make dev-login` never grants admin this way. At startup, each instance also grants admin to every existing account with a listed email.

The sign-in grant also restores admin access if the last admin deletes their own account: signing in again creates a new account, which is made an admin if its email is listed. An existing account is not promoted at sign-in, so revoking admin from a listed account lasts until the next instance restart.

Without `INITIAL_ADMIN_EMAILS`, set the flag on the user record directly. Locally:

```bash
sqlite3 goread2.db "UPDATE users SET is_admin = 1 WHERE email = 'admin@example.com';"
```

In production, edit the `User` entity's `is_admin` property in the Google Cloud Console, under Firestore in Datastore mode. `gcloud` has no command that edits an entity.

## Admin Token System

The admin CLI authenticates with `ADMIN_TOKEN`, which must match a token stored in the database:

- A token is 64 lowercase hexadecimal characters, 32 random bytes. Any other value is rejected before the database lookup.
- The database stores only the token's SHA-256 hash, with a description, its creation and last-use times, and whether it has been revoked.
- Every command except `create-token` validates the token against the database and records its use.

A server outside App Engine also reads an `ADMIN_TOKEN`, for the `X-Admin-Token` header on cron and task requests. That value may be any string and is unrelated to CLI tokens (see [configuration.md](configuration.md#administration)).

### Creating the First Token

`create-token` accepts any non-empty `ADMIN_TOKEN`, so the first token can be created without one. When the database holds no tokens, it refuses unless at least one admin user exists. When tokens already exist, it warns and asks for confirmation before creating another.

```bash
ADMIN_TOKEN="bootstrap" ./admin.sh create-token "Initial setup"
```

The command prints the new token once. Export it for later commands:

```bash
export ADMIN_TOKEN="<token printed by create-token>"
```

### Listing and Revoking Tokens

```bash
./admin.sh list-tokens          # ID, description, creation and last-use times, status
./admin.sh revoke-token <id>    # revoke by the ID from list-tokens
```

Revoke a token that may have been exposed, and create a separate token, with a descriptive name, for each person or machine that needs one.

## Admin Script Usage

`admin.sh` wraps `go run cmd/admin/main.go` with shorter command names. Both read `ADMIN_TOKEN` from the shell, and every command requires a valid token, including those that only read data. The CLI uses the local SQLite file unless `GOOGLE_CLOUD_PROJECT` is set, in which case it uses that project's Datastore with the Application Default Credentials from `gcloud auth application-default login`:

```bash
GOOGLE_CLOUD_PROJECT=goread-467200 ./admin.sh list
```

| `admin.sh` | `cmd/admin` | Effect |
|------------|-------------|--------|
| `list` | `list-users` | Lists every user: ID, email, name, subscription status, admin flag, free months, and join date |
| `info <email>` | `user-info <email>` | Shows one user's account and subscription details |
| `admin <email> on` or `off` | `set-admin <email> true` or `false` | Grants or revokes admin access |
| `grant <email> <months>` | `grant-months <email> <months>` | Adds free months. Available only when `SUBSCRIPTION_ENABLED=true`. |
| `fix-sub <email>` | `fix-subscription <email>` | Fetches the user's subscription from Stripe by its stored ID and updates the status, for a user whose webhook was missed |
| `set-sub-id <email> <id>` | `set-subscription-id <email> <id>` | Stores a Stripe subscription ID on the user, for a subscription created without one |
| `create-token`, `list-tokens`, `revoke-token` | Same | Manage admin tokens, as above |
| None | `audit-logs` | Shows the audit log (see [Audit Logging](#audit-logging)) |
| None | `stats` | Prints usage statistics (see [Usage Statistics](#usage-statistics)) |

## Access Levels

Whether a user can add feeds depends on these checks, in order. The first that applies decides. The subscription system gates only adding feeds and importing OPML; every user can read the feeds already subscribed.

| Level | Set by | Feed limit | Ends | Sidebar label | Account page label |
|-------|--------|-----------|------|---------------|--------------------|
| Subscriptions disabled | `SUBSCRIPTION_ENABLED=false` | None | Never | Unlimited Access | Unlimited Access |
| Admin | `is_admin` | None | When revoked | Admin Access | Admin + GoRead2 Pro with a subscription, otherwise Admin (Unlimited) |
| Pro | An active Stripe subscription | None | When Stripe reports the subscription ended | GoRead2 Pro | GoRead2 Pro |
| Free months | `free_months_remaining` above 0 | None | Never | Free Months | Status: free_months |
| Trial | New account | 20 | 30 days after sign-up | Free Trial | Free Trial |
| Trial expired | Trial end date passed | No new feeds | When the user subscribes | Trial Expired | Trial Expired |

A user whose subscription is `past_due`, `unpaid`, or `cancelled`, without admin access or free months, is limited to 20 feeds.

Admin access affects feed limits and the `/admin` API only. It does not change billing: an admin who subscribes is billed like any other user, and the account page still offers a subscription to an admin without one.

**Free months** do not expire. `grant-months` adds to `free_months_remaining`, nothing ever decreases it, and the CLI rejects a negative number of months. Access granted this way lasts until the field is set to 0 directly in the database.

## Audit Logging

Granting or revoking admin access and granting free months, through the CLI or the admin API, write an audit log entry. Viewing a user's details through the admin API also writes one. Each entry records:

- the time and the operation: `grant_admin`, `revoke_admin`, `grant_free_months`, or `view_user_info`
- the acting admin's user ID and email. CLI entries record the email `CLI_ADMIN`, user ID 0, and the IP address `CLI`.
- the target user's ID and email
- the operation's details, as JSON, and the IP address of the request
- the result, `success` or `failure`, and the error message of a failure

To read the log from the CLI, newest first:

```bash
go run cmd/admin/main.go audit-logs                         # the latest 20 entries
go run cmd/admin/main.go audit-logs --limit 100
go run cmd/admin/main.go audit-logs --operation grant_admin
```

`GET /admin/audit-logs` returns the same entries as JSON, with filters for the acting admin, the target user, and the operation (see [api.md](api.md#admin-endpoints)).

### Audit Log Retention

Audit log entries are stored in the `audit_logs` table on SQLite and as `AuditLog` entities on Datastore. Entries themselves are never deleted, but two rules limit the personal data they hold:

- The monthly data retention run blanks the IP address of entries older than 12 months (see [privacy.md](privacy.md#data-retention)).
- Deleting an account replaces the user's email with `deleted-user-<id>` in every entry that names it, and clears the IP address of actions the user performed as an admin (see [privacy.md](privacy.md#account-deletion)).

Do not copy audit logs out of GoRead2 for archiving: a copy keeps IP addresses and the emails of deleted users past these limits, contrary to the privacy policy.

## Admin API

The `/admin` endpoints offer user lookup, admin and free-month grants, the audit log, and usage statistics to a signed-in admin. They require an admin session, and `POST` requests also require the `X-CSRF-Token` header. `GET /admin/users` is not implemented and returns `501`; use the CLI to list users. [api.md](api.md#admin-endpoints) documents each endpoint.

## Usage Statistics

GoRead2 computes usage statistics from data it already stores: the `User`, `UserActivity`, and `UserFeed` records. No third-party analytics service or cookie is involved. Admins can view them in two ways:

- `GET /admin/stats` returns the statistics as JSON to an admin session.
- `go run cmd/admin/main.go stats` prints them as a report.

The statistics are:

- **Signups**: new accounts per day for the last 30 days and per week, starting on Monday, for the last 12 weeks. Dates are in UTC.
- **Active users**: users whose last recorded activity falls within the last 1, 7, or 30 days. Activity is written at most once a day per user, so the daily figure can undercount by up to a day.
- **Subscriptions**: non-admin users counted by state: trialing, trial expired, active, past due (including unpaid), and cancelled. Admins are counted separately and excluded from every other figure. Users with granted free months are also counted on their own, whatever their state.
- **Trial-to-paid conversion**: the share of users whose trial has ended who have ever subscribed, meaning their state is active, past due, unpaid, or cancelled.
- **Feeds**: the number of distinct feeds with at least one subscriber, the total number of subscriptions, the mean and median feeds per user, and the distribution of users across the ranges 0, 1-5, 6-10, 11-20, 21-50, and 51+ feeds.

Computing the statistics reads every `User`, `UserActivity`, and `UserFeed` record. Each instance reuses the result for 15 minutes, so the `generated_at` field of the response can be up to 15 minutes old. The CLI always computes fresh statistics.

A daily cron job also records these statistics as Cloud Monitoring custom metrics, which the usage dashboard charts over time. See [monitoring.md](monitoring.md#usage-metrics).

The response has this shape, shown with illustrative values:

```json
{
  "generated_at": "2026-11-04T15:00:00Z",
  "total_users": 120,
  "signups_by_day": [{"date": "2026-10-06", "count": 1}, "..."],
  "signups_by_week": [{"date": "2026-08-17", "count": 4}, "..."],
  "active_users": {"daily": 18, "weekly": 41, "monthly": 77},
  "subscriptions": {
    "admins": 1,
    "trialing": 22,
    "trial_expired": 60,
    "active": 30,
    "past_due": 2,
    "cancelled": 5,
    "other": 0,
    "free_months": 3,
    "conversion_rate": 0.381
  },
  "feeds": {
    "active_feeds": 640,
    "subscriptions": 1530,
    "mean_per_user": 12.75,
    "median_per_user": 9,
    "per_user": [{"bucket": "0", "count": 8}, {"bucket": "1-5", "count": 25}, "..."]
  }
}
```

## Database Schema

The SQLite definitions of the tables behind admin operations, from `internal/database/schema.go`. Datastore stores the same fields as `User`, `AdminToken`, and `AuditLog` entities.

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

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md#admin-issues) for admin token errors, user lookup failures, and database lock issues.

## Related Documentation

- [API Reference](api.md#admin-endpoints) - The admin endpoints
- [Configuration Reference](configuration.md#administration) - `ADMIN_TOKEN` and `INITIAL_ADMIN_EMAILS`
- [Privacy and Personal Data](privacy.md) - Account deletion and audit log retention
- [Stripe Setup](stripe.md) - Subscriptions and billing
- [Troubleshooting Guide](troubleshooting.md) - Common issues
