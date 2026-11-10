# Privacy and Personal Data

The personal data GoRead2 stores, where it is kept, and how it is exported, deleted, and retained.

`web/templates/privacy.html`, served at `/privacy`, is the privacy policy that users read. This document describes the implementation behind it. A change to the data stored, the cookies set, the third parties involved, or a retention period must update both in the same commit.

## Table of Contents

- [Personal Data Stored](#personal-data-stored)
- [Cookies and Browser Storage](#cookies-and-browser-storage)
- [Data Export](#data-export)
- [Account Deletion](#account-deletion)
- [Data Retention](#data-retention)
- [Requests That Need the Maintainer](#requests-that-need-the-maintainer)
- [Related Documentation](#related-documentation)

## Personal Data Stored

In the database, Datastore in production and SQLite locally:

| Record | Personal data | Purpose |
|--------|---------------|---------|
| `User` | Google ID, email, name, profile picture URL, creation date, subscription status and dates, Stripe subscription ID, admin flag, free months, reader settings, and the time a deletion was requested | The account |
| `UserFeed` | The feeds the user subscribes to | Subscriptions |
| `UserArticle` | The articles the user has read or starred | Reading state |
| `Session` | A random session ID with its creation and expiry times | Sign-in |
| `UserActivity` | The last time the account was used, recorded at most once a day | Inactive-account retention |
| `AuditLog` | The emails of the acting admin and the target user, the admin's IP address, the operation, and its result | The admin action trail (see [admin.md](admin.md#audit-logging)) |
| `AccountTombstone` | An HMAC of a deleted account's Google ID, with the deletion time and an optional expiry | Preventing a second free trial |

`Feed` and `Article` records are shared between users and hold no personal data.

Outside the database:

- **Stripe** holds a customer for each user who has started checkout, with the user's email, name, and a `user_id` metadata field, along with the subscription and payment records. Card details are entered on Stripe's checkout page and never reach GoRead2 (see [stripe.md](stripe.md)).
- **Cloud Logging** holds App Engine request logs, which include IP addresses and user agents, and the server's application logs, whose `SECURITY:` lines include IP addresses and user IDs.

## Cookies and Browser Storage

The web frontend sets only the following cookies and browser storage entries, none of which is used for tracking. The privacy policy lists the same entries and must be updated when one is added or removed.

| Name in production | Name on a local server | Type | Set by | Lifetime | Purpose |
|--------------------|------------------------|------|--------|----------|---------|
| `session_id` | `session_id_local` | Cookie, strictly necessary | `internal/auth/session.go` | 7 days, renewed on use | Authenticated session |
| `oauth_state` | `oauth_state_local` | Cookie, strictly necessary | `internal/handlers/auth_handler.go` | 10 minutes | OAuth CSRF state, cleared on callback |
| `tabletHelpDismissed` | Same | `localStorage`, functional | `web/static/js/app.js` | Persistent | Hides the tablet help hint once dismissed |

GoRead2 uses no third-party analytics, advertising, or tracking. Some content loads directly from other servers, which receive the visitor's IP address and browser details but set no cookies:

- **Google Fonts** serves the Inter stylesheet on every page.
- **cdn.jsdelivr.net** serves DOMPurify on every page.
- **Google** serves the signed-in user's profile picture.
- **Feed publishers** serve the images and media inside articles. GoRead2's servers fetch the feeds themselves, so publishers do not learn which feeds a user subscribes to.

## Data Export

`GET /api/account/export` returns everything in the database tables above that belongs to the user as one JSON file: the account, last-activity time, subscriptions, sessions without their IDs, audit log entries in which the user is the subject or the acting admin, and every article with a read or starred state. The account page offers it under "Your Data", as does the Settings screen of the iOS and macOS apps. It is limited to 3 exports in quick succession, refilling at 1 per minute. [api.md](api.md#get-apiaccountexport) gives the file format.

`GET /api/feeds/export` exports the user's subscriptions as OPML, which other RSS readers can import.

## Account Deletion

`DELETE /api/account` permanently deletes the signed-in user's account. The request must pass CSRF validation and repeat the account email as `confirm_email`, compared case-insensitively. The handler queues the deletion before revoking the user's sessions, so a queueing failure leaves the user signed in and able to retry; it then revokes every session and clears the cookie.

In production, once the deletion is queued, the handler also records a `deletion_requested_at` timestamp on the `User` record (`UserDataService.RequestDeletion`). This timestamp does two things: it lets the data retention run retry a deletion that both Cloud Tasks passes failed to complete (see [Data Retention](#data-retention) below), and it makes `SignInGoogleUser` refuse a re-sign-in on the Google identity while the deletion is in flight, returning `ErrAccountDeletionPending` instead of creating a session. Signing back in through the normal flow after the account is actually gone is unaffected, since deletion removes the `User` record.

`UserDataService.DeleteAccount` (`internal/services/user_data_service.go`) runs the deletion in this order, and every step is idempotent:

1. Delete the user's Stripe customers, which cancels any subscription (see [stripe.md](stripe.md#account-deletion)). If Stripe is not configured and the subscription is still live, the deletion stops here with an error.
2. Write the account tombstone
3. Delete `UserArticle`, `UserFeed`, and `Session` records in batches of 500, then the `UserActivity` record that tracks the account's last activity for retention
4. Pseudonymize audit log entries: the user's email becomes `deleted-user-<id>`, and the IP address of actions the user performed as an admin is cleared. Entries are kept for the admin trail.
5. Delete the `User` record last, so a partial failure leaves an account that a retry can find and finish

Shared `Feed` and `Article` records are not touched. Other App Engine instances can keep serving a deleted user from their in-memory session cache for up to its 10-minute TTL, so a second pass runs 15 minutes later to remove anything those requests recreated, including Stripe customers, which it finds by their `user_id` metadata (see [deployment.md](deployment.md#async-task-processing-cloud-tasks)).

**Account tombstones** stop a deleted account from starting a second free trial by signing in again. A tombstone is an `AccountTombstone` entity keyed by HMAC-SHA256 of the Google ID under `ACCOUNT_TOMBSTONE_KEY`, holding only the deletion time and an optional expiry. A self-requested deletion writes a tombstone that never expires. A deletion for inactivity writes one that expires 24 months later, so a returning user is offered a new trial after that. Writing a tombstone never shortens an existing one: a permanent tombstone stays permanent, and otherwise the later expiry is kept. An expired tombstone no longer counts at sign-in. A plain hash would not protect the numeric Google ID from brute force. On sign-in, a new account whose Google ID has an unexpired tombstone is created with its trial already ended. Without the key, deletion still works but writes no tombstone, and the sign-in check is skipped. The key is cached once read; a Secret Manager error other than a missing secret is not cached, so the next deletion or sign-in retries the lookup.

## Data Retention

`RetentionService` (`internal/services/retention_service.go`) applies the following policy on the first of each month through `/cron/apply-retention`, and applies the pending-deletion rule alone every day through `/cron/retry-pending-deletions`:

- **Inactive accounts**: an account is deleted, through the same path as `DELETE /api/account`, once it has had no activity for 24 months and its subscription status is `trial`, `cancelled`, or `expired`. Paying accounts (`active`, `past_due`, `unpaid`), admins, and accounts with granted free months are never deleted for inactivity. No warning is sent before deletion. The account's tombstone expires 24 months after the deletion (see [Account Deletion](#account-deletion)).
- **Pending self-requested deletions**: any account with a `deletion_requested_at` timestamp older than one hour is deleted, regardless of subscription status. This is the backstop for a `DELETE /api/account` deletion whose Cloud Tasks passes both failed (see [Account Deletion Failures](deployment.md#account-deletion-failures), the operator runbook). Because it runs daily, a paying subscriber whose deletion failed during an outage remains subscribed for about a day at most.
- **Account tombstones**: deleted once expired. Tombstones from self-requested deletions never expire.
- **Audit log IP addresses**: blanked on entries older than 12 months; the rest of each entry is kept. Each run reads only entries that still have an IP address, so an entry that a failed run missed is cleared by the next successful one.
- **Request logs**: Cloud Logging's `_Default` bucket keeps App Engine request logs, including IP addresses and user agents, for 30 days.

Activity is a `UserActivity` record per user holding the last time any session for that account was created or used. `SessionManager` writes it at sign-in and then at most once a day per user on each instance, rather than on every request. An account with no `UserActivity` record is not judged by its creation date: the first retention run records it as active as of that run, so it can be deleted for inactivity no sooner than 24 months later.

A failed deletion is logged and counted; the run continues with other accounts and then returns an error so that Cloud Tasks retries it. Each retry repeats only the work still outstanding.

## Requests That Need the Maintainer

The privacy policy invites users to ask for corrections, and for the deletion of an account they can no longer sign in to. The admin CLI has no command that edits a user's account details or deletes an account, so these requests are handled by hand. For an account the user cannot sign in to, re-enqueue the deletion task for the account's ID as described in [Account Deletion Failures](deployment.md#account-deletion-failures), which runs the same deletion as `DELETE /api/account`.

## Related Documentation

- [Security](security.md) - The controls that protect this data
- [API Reference](api.md) - `GET /api/account/export` and `DELETE /api/account`
- [Stripe Setup](stripe.md#account-deletion) - How deletion removes Stripe customers
- [Deployment Guide](deployment.md#async-task-processing-cloud-tasks) - The Cloud Tasks queue that runs deletions
- [Admin Guide](admin.md#audit-logging) - Reading the audit log
