# Configuration Reference

GoRead2 is configured entirely through environment variables. This page lists every variable the server and its command-line tools read, grouped by area. Topic guides link here rather than repeating these lists.

## Table of Contents

- [How Values Are Resolved](#how-values-are-resolved)
- [Authentication and Sessions](#authentication-and-sessions)
- [Security](#security)
- [Database and Google Cloud](#database-and-google-cloud)
- [Subscriptions and Stripe](#subscriptions-and-stripe)
- [Administration](#administration)
- [Feed Fetching](#feed-fetching)
- [Server and Deployment](#server-and-deployment)
- [Set by the Platform or Tooling](#set-by-the-platform-or-tooling)
- [Local Development](#local-development)
- [Related Documentation](#related-documentation)

## How Values Are Resolved

The server reads each variable from the process environment. It does not load `.env` files; `.env.example` describes the variables as a reference only.

A variable with a Secret Manager name in the tables below is a secret. When a secret's variable is unset, or its value starts with `_secret:`, the server fetches it from Google Secret Manager under that fixed name, which requires `GOOGLE_CLOUD_PROJECT`. Text after the `_secret:` prefix is ignored: `GOOGLE_CLIENT_ID=_secret:anything` fetches `google-client-id`. Only the two OAuth secret names can be changed, through `SECRET_CLIENT_ID_NAME` and `SECRET_CLIENT_SECRET_NAME`.

On App Engine, secrets are stored in Secret Manager and never appear in `app.yaml`. [deployment.md](deployment.md#setting-up-google-secret-manager) has the commands that create them, and [deployment.md](deployment.md#appyaml-configuration) shows the variables `app.yaml` does set.

Several behaviors differ in production, which means App Engine (`GAE_ENV=standard`) or `ENVIRONMENT=production`. These are noted where they apply.

At startup, the server and `make validate-config` print a warning for each environment variable the application does not recognize. The list of recognized variables is in `internal/config/validation.go` and must be updated when a variable is added.

## Authentication and Sessions

| Variable | Default | Required | Secret Manager name | Purpose |
|----------|---------|----------|---------------------|---------|
| `GOOGLE_CLIENT_ID` | None | Always | `google-client-id` | OAuth 2.0 client ID from the Google Cloud Console |
| `GOOGLE_CLIENT_SECRET` | None | Always | `google-client-secret` | OAuth 2.0 client secret |
| `GOOGLE_REDIRECT_URL` | None | Always | None | OAuth callback URL. It must match a redirect URI registered on the OAuth client. |
| `STAGING_REDIRECT_URL` | Unset | No | None | Callback URL for the fixed `staging` App Engine version, chosen by the request's `Host` header (see [authentication.md](authentication.md#host-aware-redirect-url-staging-support)) |
| `SECRET_CLIENT_ID_NAME` | `google-client-id` | No | None | Secret Manager name to read the OAuth client ID from |
| `SECRET_CLIENT_SECRET_NAME` | `google-client-secret` | No | None | Secret Manager name to read the OAuth client secret from |
| `SESSION_CACHE_TTL` | `10m` | No | None | How long each instance caches a session in memory, as a Go duration such as `5m` or `1h`. An invalid value logs a warning and uses the default. |

## Security

| Variable | Default | Required | Secret Manager name | Purpose |
|----------|---------|----------|---------------------|---------|
| `CSRF_SECRET` | Generated | In production | `csrf-secret` | Base64-encoded secret of at least 32 bytes for CSRF tokens. Generate one with `openssl rand -base64 32`. In production the server exits at startup if it is missing, malformed, or too short. Elsewhere the server generates a random secret at each start when it is unset, which invalidates CSRF tokens issued before the restart. |
| `ACCOUNT_TOMBSTONE_KEY` | Unset | No | `account-tombstone-key` | HMAC key for deleted-account tombstones, which stop a deleted account from starting a second free trial (see [security.md](security.md#account-deletion)). Without it, deletion writes no tombstone and sign-in skips the check. Never rotate it, since tombstones written under the old key stop matching. |
| `ALLOWED_ORIGIN` | Unset | No | None | The single origin allowed to make credentialed cross-origin API requests. Unset, no cross-origin access is allowed (see [security.md](security.md#cross-origin-requests)). |
| `CSP_REPORT_ONLY` | `false` | No | None | Set to `true` to send the Content Security Policy in report-only mode, in which browsers log violations without blocking them (see [security.md](security.md#content-security-policy)) |
| `ENVIRONMENT` | Unset | No | None | Set to `production` to get production behavior outside App Engine: Secure cookies, the production cookie names, a mandatory `CSRF_SECRET`, and no dev login |

## Database and Google Cloud

| Variable | Default | Required | Secret Manager name | Purpose |
|----------|---------|----------|---------------------|---------|
| `GOOGLE_CLOUD_PROJECT` | Unset | On App Engine | None | Google Cloud project ID. When set, the server uses Datastore; when unset, it uses the local SQLite file `goread2.db`. Also required to read Secret Manager, Cloud Tasks, and Cloud Monitoring. |
| `DATASTORE_EMULATOR_HOST` | Unset | No | None | Address of a Datastore emulator. Set by `scripts/with-datastore-emulator.sh` (see [testing.md](testing.md)). |

`internal/config/config.go` also reads `DATABASE_PATH`, but the database layer always opens `./goread2.db`, so the variable has no effect.

## Subscriptions and Stripe

| Variable | Default | Required | Secret Manager name | Purpose |
|----------|---------|----------|---------------------|---------|
| `SUBSCRIPTION_ENABLED` | `false` | No | None | Turns the subscription system on (see [Subscription Flag](#subscription-flag)) |
| `STRIPE_SECRET_KEY` | None | When subscriptions are enabled, or while any subscription is live | `stripe-secret-key` | Stripe secret key for API calls. Starts with `sk_`. |
| `STRIPE_PUBLISHABLE_KEY` | None | When subscriptions are enabled | `stripe-publishable-key` | Stripe publishable key for the frontend. Starts with `pk_`. |
| `STRIPE_WEBHOOK_SECRET` | None | When subscriptions are enabled | `stripe-webhook-secret` | Signing secret of the Stripe webhook endpoint. Starts with `whsec_`. |
| `STRIPE_PRICE_ID` | None | When subscriptions are enabled | `stripe-price-id` | Price ID of the GoRead2 Pro subscription. Starts with `price_`. |

### Subscription Flag

`SUBSCRIPTION_ENABLED` is read once at startup, so a change takes effect on the next restart or deployment. Production sets it to `true` in `app.yaml`. Values are compared without regard to case:

- `true`, `1`, `yes`, `on`, and `enabled` enable subscriptions.
- `false`, `0`, `no`, `off`, and `disabled` disable them.
- An empty or unset value disables them.

[stripe.md](stripe.md#subscription-flag) describes what the flag changes, and what to keep configured when it is turned off after users have subscribed.

## Administration

| Variable | Default | Required | Secret Manager name | Purpose |
|----------|---------|----------|---------------------|---------|
| `INITIAL_ADMIN_EMAILS` | Unset | No | `initial-admin-emails` | Comma-separated email addresses granted admin access at sign-in and at startup (see [admin.md](admin.md)) |
| `ADMIN_TOKEN` | Unset | For the admin CLI | `admin-token`, server only | Has two unrelated uses, described below |

`ADMIN_TOKEN` is read in two places that do not share a value:

- **Admin CLI**: `cmd/admin` and `admin.sh` read it from the environment of the shell that runs them and validate it against the admin tokens stored in the database. It must be a token printed by `create-token`; any other value is rejected (see [admin.md](admin.md#admin-token-system)).
- **Server outside App Engine**: the cron and task endpoints require an admin session and an `X-Admin-Token` header equal to the server's `ADMIN_TOKEN`, which may be any string. If it is unset, those endpoints reject every request. On App Engine the server never reads it, because those endpoints check App Engine's `X-Appengine-Cron` and `X-AppEngine-QueueName` headers instead, so an App Engine deployment needs no `admin-token` secret.

## Feed Fetching

These control the per-domain rate limit on outgoing feed fetches and the scheduler that spreads a feed refresh over time. Each must be positive; the server logs a warning at startup if one is not.

| Variable | Default | Purpose |
|----------|---------|---------|
| `RATE_LIMIT_REQUESTS_PER_MINUTE` | `120` | Maximum feed fetches per minute sent to one feed domain |
| `RATE_LIMIT_BURST_SIZE` | `30` | Fetches to one domain allowed in a burst above the per-minute rate |
| `SCHEDULER_UPDATE_WINDOW` | `15m` | Window across which a refresh spreads its fetches, as a Go duration |
| `SCHEDULER_MIN_INTERVAL` | `5m` | Minimum time between two fetches of the same feed. It must not exceed `SCHEDULER_UPDATE_WINDOW`. |
| `SCHEDULER_MAX_CONCURRENT` | `10` | Maximum concurrent fetches during a refresh |
| `SCHEDULER_CLEANUP_INTERVAL` | `1h` | Interval of the scheduler's background cleanup loop. The server does not start that loop, so the value has no effect. |

None of these is a secret or required.

## Server and Deployment

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `8080` | Port the server listens on. App Engine sets it. |
| `GIN_MODE` | `debug` | Gin framework mode, read by Gin itself. On App Engine the server always runs in `release` mode. |
| `CLOUD_TASKS_QUEUE` | `cron-tasks` | Cloud Tasks queue that cron work is enqueued on (see [deployment.md](deployment.md#async-task-processing-cloud-tasks)) |
| `CLOUD_TASKS_LOCATION` | `us-central1` | Location of that queue |
| `BUILD_VERSION` | `dev` | Version string shown by the home page. The staging deploy workflow adds it to `app.yaml`. |
| `VALIDATE_STRICT` | `false` | Read by `make validate-config`. Set to `true`, it treats missing OAuth credentials and a missing `GOOGLE_CLOUD_PROJECT` as errors, as a check before a production deployment. |

## Set by the Platform or Tooling

The server also reads these, which are not set by hand:

| Variable | Set by | Effect |
|----------|--------|--------|
| `GAE_ENV` | App Engine, as `standard` | Selects production behavior, including Cloud Tasks, App Engine cron and task header checks, and the trusted client IP header |
| `GAE_SERVICE`, `GAE_VERSION` | App Engine | Route each Cloud Task to the version that enqueued it |

## Local Development

A local server needs only the three OAuth variables. `make dev-login` fills in placeholder values for those, plus `ACCOUNT_TOMBSTONE_KEY` and `SUBSCRIPTION_ENABLED=false`, and replaces Google sign-in with a local form (see [testing.md](testing.md#local-dev-login-make-dev-login)). [setup.md](setup.md) covers the rest of a local installation, and [stripe.md](stripe.md) the Stripe test setup.

## Related Documentation

- [Deployment Guide](deployment.md) - Creating the Secret Manager secrets and the production `app.yaml`
- [Stripe Setup](stripe.md) - The subscription flag and Stripe configuration
- [Security Guidelines](security.md) - The controls the security variables configure
- [Setup Guide](setup.md) - Local installation
