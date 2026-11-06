# Troubleshooting Guide

Known failures and their fixes for GoRead2, by area. Error text in quotes is what the application returns or logs.

## Table of Contents

- [Authentication Issues](#authentication-issues)
- [Feed Issues](#feed-issues)
- [Database Issues](#database-issues)
- [Subscription Issues](#subscription-issues)
- [Admin Issues](#admin-issues)
- [Performance Issues](#performance-issues)
- [Deployment Issues](#deployment-issues)
- [Monitoring Issues](#monitoring-issues)
- [Development Issues](#development-issues)
- [Logging and Debugging](#logging-and-debugging)
- [Getting Help](#getting-help)
- [Related Documentation](#related-documentation)

## Authentication Issues

See [authentication.md](authentication.md) for how sign-in and sessions work.

### "OAuth configuration error" at Startup

The server exits at startup when it cannot load the OAuth credentials. Run `make validate-config`, which reports whether the required credentials are available without printing them. Locally, the credentials come from `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and `GOOGLE_REDIRECT_URL`. On App Engine, they come from Secret Manager. See [deployment.md](deployment.md#setting-up-google-secret-manager).

### Google Rejects the Redirect URI

`GOOGLE_REDIRECT_URL` must exactly match an authorized redirect URI on the OAuth client in Google Cloud Console, including scheme, host, port, and path, with no trailing slash:

- Local: `http://localhost:8080/auth/callback`
- Production: `https://goreadapp.com/auth/callback`

Per-commit staging URLs (`staging-<sha>-dot-...`) cannot complete a sign-in. Use the fixed `staging` version, as described in [authentication.md](authentication.md#host-aware-redirect-url-staging-support).

### Google Blocks the Account at Consent

While the OAuth consent screen is in testing mode, only listed test users can sign in. Add the account under APIs & Services → OAuth consent screen, or publish the consent screen.

### "The OAuth state parameter is not valid"

The state cookie was missing or did not match at the callback. This happens when the callback opens in a different browser or private window from the one that started sign-in, or when the browser blocks the cookie. Start sign-in again in one window, with cookies allowed for the site.

### "The OAuth state parameter has expired or has already been used"

The state lasts 10 minutes and works once, so a slow sign-in or a reloaded callback URL produces this error. Start sign-in again.

### Signed Out, or the Session Does Not Persist

Sessions last 7 days from their last use. Check the following:

- **Secure cookie over HTTP**: with `ENVIRONMENT=production` set, the session cookie is marked Secure, so a browser on plain `http://localhost` discards it. Unset the variable for local development.
- **Revoked session**: signing out, and account deletion, revoke sessions on the server.
- **Browser settings**: a private window, or a setting that clears cookies, ends the session on the browser's side.

Local and production use different cookie names, so being signed in to one does not affect the other. See [authentication.md](authentication.md#environment-isolation).

### "Invalid CSRF token" or "CSRF token required"

State-changing requests must send the token from `GET /auth/me` in the `X-CSRF-Token` header. Without `CSRF_SECRET`, a local server generates a new secret at each start, which invalidates tokens issued earlier. Reload the page to fetch a new token, or set `CSRF_SECRET` locally to keep tokens valid across restarts.

## Feed Issues

### Adding a Feed Fails

`POST /api/feeds` returns an `error_code` that identifies the cause:

| `error_code` | Status | Meaning | Fix |
|--------------|--------|---------|-----|
| `invalid_url` | 400 | The URL is malformed | Correct the URL |
| `ssrf_blocked` | 400 | The URL points at a private or local address | Use a publicly reachable URL. This also applies to feed servers on localhost |
| `feed_not_found` | 404 | The page exists but advertises no feed | Enter the direct feed URL |
| `feed_timeout` | 408 | Discovery and fetching took more than 15 seconds | Try again later, or use the direct feed URL |
| `invalid_feed_format` | 422 | The response is not valid RSS, Atom, or RDF, or exceeds 10 MB | Inspect the response, shown below |
| `network_error` | 502 | DNS failure, connection failure, or an HTTP error status | Check that the site responds; it may block automated requests |

To inspect a feed directly:

```bash
curl -I "https://example.com/feed.xml"
curl -s "https://example.com/feed.xml" | xmllint --noout -
```

### New Articles Appear Late

This is usually expected, for these reasons:

- `/cron/refresh-feeds` runs every 2 hours in production (`cron.yaml`).
- Each run skips feeds that were checked recently, and rarely updated feeds are checked least often. See [performance.md](performance.md).
- Unread counts for newly fetched articles can lag by up to 15 minutes. See [caching.md](caching.md#unread-count-cache-15-minutes-ttl).

To check that refreshes are running in production:

```bash
gcloud app logs read -s default --limit 1000 | grep -i "refresh"
```

To force a refresh locally, press `r` in the web app, and check `last_fetch` in the database:

```bash
sqlite3 goread2.db "SELECT title, last_fetch FROM feeds;"
```

### OPML Import Fails

| Symptom | Cause | Fix |
|---------|-------|-----|
| "The file exceeds the maximum allowed size of 10 MB." | File over 10 MB | Split the file |
| "Import would exceed your feed limit of 20 feeds." | Trial account limit. The response's `imported_count` gives the number imported before the limit | Subscribe, or import fewer feeds |
| "An internal error occurred while importing." | Malformed XML, nesting deeper than 50 levels, or more than 50,000 elements | Validate with `xmllint --noout subscriptions.opml`, and check the server log |

## Database Issues

Local development uses SQLite (`goread2.db`), and production uses Google Cloud Datastore. The commands in this section apply only to SQLite. See [setup.md](setup.md#database-configuration).

### "Database is locked"

Another process is writing to the same SQLite file, usually a second server or an admin command. Stop the other process and retry:

```bash
pgrep -fl goread2
pkill goread2
```

### Corrupted Local Database

```bash
sqlite3 goread2.db "PRAGMA integrity_check;"
```

If the check fails, move the file aside and restart the server, which creates an empty database. This loses all local data:

```bash
mv goread2.db goread2_backup.db
```

## Subscription Issues

See [stripe.md](stripe.md) for setup and webhook configuration.

### Subscription Routes Return 404

The checkout, portal, Stripe config, and webhook routes are registered only when `SUBSCRIPTION_ENABLED=true`. See [feature-flags.md](feature-flags.md).

### Stripe Configuration Errors

Check the configuration with:

```bash
go run cmd/setup-stripe/main.go validate
```

A successful run prints the publishable key, which is not secret. To check that a secret is set without printing it:

```bash
[ -n "$STRIPE_SECRET_KEY" ] && echo set || echo unset
```

A price ID that does not exist in the current Stripe mode makes checkout fail. Test and live mode have separate prices and keys, so check that both come from the same mode. To create a price, run `go run cmd/setup-stripe/main.go create-product` and set `STRIPE_PRICE_ID` to the printed ID.

### Feed Limit Not Enforced

The limit does not apply when `SUBSCRIPTION_ENABLED` is off, or to admins, active subscribers, and accounts with granted free months. Check the account with `./admin.sh info user@example.com`.

### Webhook Failures

Check delivery attempts in the Stripe Dashboard under Developers → Webhooks. The endpoint returns:

| Response | Meaning | Fix |
|----------|---------|-----|
| 400 "Webhook signature verification failed." | `STRIPE_WEBHOOK_SECRET` does not match this endpoint | Copy the signing secret for this endpoint. The Stripe CLI uses a different secret from the Dashboard endpoint |
| 500 "The webhook endpoint is not properly configured." | `STRIPE_WEBHOOK_SECRET` is unset | Set the secret |
| 404 | `SUBSCRIPTION_ENABLED` is off | Enable the flag |

An unsigned test request should return the 400 response, which confirms that the endpoint is reachable:

```bash
curl -i -X POST https://example.com/webhooks/stripe -d '{}'
```

### Real Cards Fail in Production

Test cards work only with test keys. Production needs live keys (`sk_live_`, `pk_live_`) and a live-mode price and webhook. The Stripe Dashboard's logs show the reason for a declined payment.

## Admin Issues

See [admin.md](admin.md) for the admin token system and CLI reference.

### Admin Token Errors

| Message | Cause | Fix |
|---------|-------|-----|
| "ADMIN_TOKEN environment variable must be set" | The variable is unset | Set it to a token from `create-token` |
| "ADMIN_TOKEN must be exactly 64 lowercase hex characters (32 bytes)" | The format is wrong | Use a token from `create-token` |
| "Invalid ADMIN_TOKEN - token not found in database or inactive" | The token was revoked, belongs to another database, or was not created by `create-token` | Create a new token. A value from `openssl rand` is not valid, because tokens must exist in the database |
| "No admin users found in database" | Bootstrap needs an existing admin | Sign in once, then set the account as admin. See [admin.md](admin.md#initial-setup-bootstrap) |
| "Admin tokens already exist in the database" | A warning when creating another token | Confirm with `y` if authorized |

### User Not Found

The account must exist, so the user must have signed in at least once. Check the spelling with `./admin.sh list`.

### Admin Changes Not Reflected

Each instance caches the session and its user record for up to 10 minutes, the default `SESSION_CACHE_TTL`. A change to admin status or free months can therefore take that long to appear. Restart the local server to see it at once.

### Admins See Upgrade Prompts

Admin and free-month accounts may still see Stripe UI elements. Admin status overrides subscription requirements, so these prompts can be ignored.

## Performance Issues

See [performance.md](performance.md) for the optimizations in place.

### Slow Pages

Find the slow request in the browser's DevTools Network tab. Datastore operations over 200 ms are logged as warnings:

```bash
gcloud app logs read -s default --limit 1000 | grep -i "slow"
```

### High Memory Usage

```bash
# Resident memory of a locally built server (make build && ./goread2)
ps -o pid,rss,command -p "$(pgrep -x goread2)"

# Production: App Engine logs a message when an instance exceeds its
# memory limit and is shut down
gcloud app logs read -s default --limit 1000 | grep -i "memory limit"

# Allocation profile from the database benchmarks
go test -run '^$' -bench . -benchmem -memprofile /tmp/goread2-mem.prof ./internal/database/
go tool pprof -top -sample_index=alloc_space /tmp/goread2-mem.prof
```

The server does not register the `net/http/pprof` handlers, so there is no live heap profile endpoint.

## Deployment Issues

GoRead2 deploys only to Google App Engine, through GitHub Actions. See [deployment.md](deployment.md) for the pipeline.

### A Deploy Workflow Fails

Check the failed job in the Actions tab first, and then the application logs:

```bash
gh run list --repo jeffreyp/goread2 --limit 5
gcloud app logs tail -s default
```

A version that deploys but returns 503 usually failed configuration validation at startup, which the logs report. Check that the secrets it needs exist in Secret Manager, and that `app.yaml` contains no unresolved `${VAR}` placeholders.

## Monitoring Issues

See [monitoring.md](monitoring.md) for the dashboards and alerting setup.

### Dashboard Not Showing Data

Metrics Explorer shows whether a metric exists. Datastore metrics can take a few minutes to appear after a deploy. Usage dashboard charts stay empty until the first daily cron run.

### Alerts Not Firing

Check the policy's status and notification channels in Cloud Console, and use Metrics Explorer to confirm that the threshold was exceeded.

### Permission Errors

Creating or editing monitoring resources requires `roles/monitoring.dashboardEditor` for dashboards and `roles/monitoring.alertPolicyEditor` for alerting policies.

## Development Issues

### Tests Fail in CI

The `Tests` workflow runs separate jobs, so find the failed job in the Actions tab before looking at test logic. See [testing.md](testing.md#cicd-integration) for what each job runs.

### Tests Fail Locally

Run the suite through `make test`, which sets the environment variables the tests need:

```bash
make test
go test -v -run TestName ./internal/services/
```

Datastore-backed tests skip without an emulator. `make test-datastore` runs them. See [testing.md](testing.md#datastore-emulator-internalservicesadmin_token_datastore_testgo).

### The App Does Not Render

Minified JS and CSS are not checked in. Without them, templates point at files that do not exist, and the server logs an error when it computes their versions. Build them and restart the server:

```bash
npm install
make build-frontend
```

Then check the browser console for errors, and confirm that the assets load:

```bash
curl -I http://localhost:8080/static/js/app.min.js
```

## Logging and Debugging

GoRead2 logs to stdout and has no log file. Locally, logs appear in the terminal running the server. In production:

```bash
gcloud app logs tail -s default
```

The `request_id` field on a 401 or 403 response correlates a client-reported failure with the server logs.

To test the database connection and admin token together, run `go run cmd/admin/main.go list-users` with a valid `ADMIN_TOKEN`.

## Getting Help

Report bugs at [GitHub Issues](https://github.com/jeffreyp/goread2/issues). GitHub issues are public, so do not report security vulnerabilities there. See [security.md](security.md#reporting-security-issues).

**Do not include secrets or personal data in a report.** Remove OAuth secrets, Stripe keys, admin tokens, `CSRF_SECRET`, session cookies, and user email addresses from anything pasted. Do not paste the output of `env`, and if a secret is posted by accident, rotate it.

Include:

- The steps to reproduce the failure, and the error message
- The environment: local development, staging, or production
- The build version shown in the footer of the help modal (`?`), or the output of `git rev-parse --short HEAD` for a local build
- The output of `go version`
- The names of the relevant variables that are set, without their values:

  ```bash
  env | cut -d= -f1 | grep -E '^(GOOGLE|STRIPE|SUBSCRIPTION)' | sort
  ```

## Related Documentation

- [Setup Guide](setup.md) - Installation and configuration
- [Deployment Guide](deployment.md) - Production deployment
- [Admin Guide](admin.md) - User management
- [API Reference](api.md) - API documentation
