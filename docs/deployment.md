# Deployment Guide

Guide for deploying GoRead2 to Google App Engine, the only supported production environment, and for the automated CI/CD pipeline that promotes builds to it.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [CI/CD Authentication (GitHub Actions → GCP)](#cicd-authentication-github-actions--gcp)
- [Production Approval Gate (GitHub Environment)](#production-approval-gate-github-environment)
- [Automated Staging Deploys](#automated-staging-deploys-githubworkflowsdeploy-stagingyml)
- [Production Deploys](#production-deploys-githubworkflowsdeploy-prodyml)
- [Post-Deploy Smoke Check](#post-deploy-smoke-check-scriptssmoke-checksh)
- [Auto-Rollback](#auto-rollback-post-promote-safety-net)
- [Rollback](#rollback-githubworkflowsrollbackyml)
- [Apple Release Pipelines](#apple-release-pipelines)
- [Google App Engine Configuration](#google-app-engine-configuration)
- [Async Task Processing (Cloud Tasks)](#async-task-processing-cloud-tasks)
- [Environment Variables](#environment-variables)
- [Security Considerations](#security-considerations)
- [Monitoring and Maintenance](#monitoring-and-maintenance)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Cost Optimization](#cost-optimization)
- [Related Documentation](#related-documentation)

## Overview

GoRead2 deploys to Google App Engine only. All deployment methods require:
- Google OAuth 2.0 configuration
- Google Cloud Datastore, the multi-user production database
- Session management
- Production security considerations

## Prerequisites

### Google Cloud Setup

1. **Google Cloud Project**
   - Create a new Google Cloud Project or use existing one
   - Enable the following APIs:
     - App Engine Admin API (for GAE deployment)
     - Cloud Datastore API (for production database)
     - Cloud Build API (for deployment)

2. **Google OAuth 2.0 Setup**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Navigate to APIs & Services → Credentials
   - Create OAuth 2.0 Client ID
   - Configure OAuth consent screen
   - Set authorized redirect URIs for the deployment

3. **Install Google Cloud SDK**
   ```bash
   # Download and install from: https://cloud.google.com/sdk/docs/install
   curl https://sdk.cloud.google.com | bash
   exec -l $SHELL
   gcloud init
   ```

4. **Authentication**
   ```bash
   gcloud auth login
   gcloud config set project YOUR_PROJECT_ID
   ```

## CI/CD Authentication (GitHub Actions → GCP)

GitHub Actions authenticates to GCP via Workload Identity Federation (WIF): keyless, with no service account JSON key stored in GitHub.

**Resources** (project `goread-467200`):
- Workload Identity Pool: `github-actions-pool` (location `global`)
- OIDC Provider: `github-provider`, issuer `https://token.actions.githubusercontent.com`, with an attribute condition restricting it to `assertion.repository == 'jeffreyp/goread2'`; no other repository can assume this identity
- Service account: `cicd-deploy@goread-467200.iam.gserviceaccount.com`, granted at the project level:
  - `roles/appengine.deployer` and `roles/appengine.serviceAdmin`, for deploying versions and migrating traffic between them
  - `roles/cloudbuild.builds.editor` and `roles/storage.admin`, for the Cloud Build step a deploy runs through
  - `roles/secretmanager.secretAccessor`, for reading deploy-time secrets
  - `roles/cloudscheduler.admin` and `roles/datastore.indexAdmin`, for deploying `cron.yaml` and `index.yaml`
  - `roles/monitoring.viewer`, so the post-promote health watch can read Cloud Monitoring time series

  plus `roles/iam.serviceAccountUser` on `goread-467200@appspot.gserviceaccount.com` specifically, since an App Engine deploy requires the deploying identity to act as the App Engine default service account.
- The pool is bound to the service account via `roles/iam.workloadIdentityUser`, scoped to the `attribute.repository/jeffreyp/goread2` principal set; only workflow runs from this exact repository can impersonate it.

**GitHub Actions repository variables** (not secrets; the provider path and service account email aren't sensitive on their own):
- `WIF_PROVIDER` = `projects/1022472352583/locations/global/workloadIdentityPools/github-actions-pool/providers/github-provider`
- `CICD_SERVICE_ACCOUNT` = `cicd-deploy@goread-467200.iam.gserviceaccount.com`

**Usage in a workflow**:
```yaml
permissions:
  contents: read
  id-token: write   # required for WIF

steps:
  - uses: google-github-actions/auth@v2
    with:
      workload_identity_provider: ${{ vars.WIF_PROVIDER }}
      service_account: ${{ vars.CICD_SERVICE_ACCOUNT }}
  - uses: google-github-actions/setup-gcloud@v2
  - run: gcloud app deploy ...
```

## Production Approval Gate (GitHub Environment)

A GitHub Environment named `production` provides the human approval gate for production deploys. Any workflow job that declares `environment: production` pauses and waits for an approval before running.

**Configuration** (repository `jeffreyp/goread2`):
- Environment: `production`
- Required reviewer: `jeffreyp`
- Deployment branch policy: restricted to `main` only; no other branch can target this environment
- Branch protection on `main` is not enabled. Pushes go directly to `main` without pull requests, and GitHub's required-status-checks branch protection blocks a push whose commit lacks an already-passing check run, which would force a PR-based workflow.

**Usage in a workflow**:
```yaml
jobs:
  deploy-prod:
    runs-on: ubuntu-latest
    environment: production   # pauses here for approval
    steps:
      - run: gcloud app versions migrate ...
```

## Automated Staging Deploys (`.github/workflows/deploy-staging.yml`)

Every push to `main` that passes the `Tests` workflow automatically deploys to App Engine as a new, zero-traffic version, safe to run unattended because of `--no-promote`.

- **Trigger**: `workflow_run` for the `Tests` workflow, `types: [completed]`, gated on `github.event.workflow_run.conclusion == 'success'` and `branches: [main]`; the deploy never runs if CI failed.
- **Version name**: `staging-<short-sha>`, for example `staging-a1b2c3d`, deployed with `--no-promote` (no production traffic). The short SHA comes from `github.event.workflow_run.head_sha`, not `$GITHUB_SHA`, which for a `workflow_run` event is whichever commit the default branch tip happens to be when the job runs, not the triggering commit.
- **`cron.yaml` / `index.yaml`**: deployed on every run. Both deploys are idempotent, so a run that fails is repaired by the next successful one.
- **Job summary**: prints the version name and the full staging URL (`https://<version>-dot-goread-467200.uc.r.appspot.com`) so a reviewer knows where to click through and test.
- All actions are pinned to a commit SHA, not a floating tag such as `@v4`; see the workflow file's inline `# vX` comments for the corresponding version.
- `BUILD_VERSION` is injected into `app.yaml` at deploy time (via `sed`, immediately before the deploy step), since App Engine standard has no `--set-env-vars` deploy flag and the file has no other templating mechanism.
- **Cleanup**: after deploying, stale `staging-*` versions are deleted, keeping the one just deployed plus the single most-recently-created other one; otherwise `staging-*` versions would accumulate one per push indefinitely. This is scoped to versions with `traffic_split=0`, since an older `staging-<sha>` can be the version currently serving all production traffic: `deploy-prod.yml` (below) promotes by migrating traffic directly onto a `staging-<sha>` version rather than redeploying under a new name. The extra kept slot is normally the version just demoted by the last promotion, so `rollback.yml` still has a target for one more staging-deploy cycle after a promotion.
- **Smoke check**: immediately after both versions deploy, `scripts/smoke-check.sh` (see below) runs against the `staging-<sha>` URL and fails the job if it doesn't pass, catching a broken deploy before a human ever clicks through it.
- **Concurrency**: `concurrency: { group: deploy-staging, cancel-in-progress: false }` serializes runs. Back-to-back commits to `main` each spawn their own `Tests` run and therefore their own `deploy-staging` run, with no guarantee the older commit's run finishes first; without this guard, an older commit's run finishing after a newer commit's run could delete the newer commit's just-deployed version in the cleanup step above, silently reverting staging. `cancel-in-progress` is `false` rather than `true` because killing a `gcloud app deploy` mid-flight can leave a version half-created, which is worse than a queued run waiting its turn.

### Human OAuth Login Testing on Staging (fixed `staging` version)

The production approval gate requires a human to actually log in and click through the staging UI before promoting: automated smoke checks alone don't prove the UI is usable. But Google OAuth requires an exact, pre-registered redirect URI, and the per-SHA `staging-<sha>` hostname changes on every deploy, so it can never complete a login round trip.

To solve this, `deploy-staging.yml` deploys the **same build twice**:
- `staging-<sha>` (`--no-promote`): the promotion candidate that `deploy-prod.yml` migrates to production.
- `staging` (`--no-promote`, fixed name, overwritten on every merge): exists only so a human has a stable URL to log into: `https://staging-dot-goread-467200.uc.r.appspot.com`.

Both redirect URIs are registered on the **same** Google OAuth client (Google Cloud Console → APIs & Services → Credentials → the existing OAuth 2.0 Client ID → Authorized redirect URIs):
- `https://goreadapp.com/auth/callback` (production, unchanged)
- `https://staging-dot-goread-467200.uc.r.appspot.com/auth/callback` (staging; a one-time manual console step, not automatable via this pipeline)

`internal/auth/auth.go` then picks the matching redirect URL per-request based on the incoming `Host` header (`GOOGLE_REDIRECT_URL` for production, `STAGING_REDIRECT_URL` for staging; both set in `app.yaml` and deployed identically to every App Engine version). See [authentication.md](authentication.md#host-aware-redirect-url-staging-support) for the implementation and its security rationale.

**Accepted tradeoff**: staging shares the production Datastore. Cron only ever targets the default (production) serving version, and staging writes are intentional reviewer actions on the same binary that's about to be promoted anyway, so catching a data-mutating bug on staging beats catching it in prod. **Caveat**: do not exercise Stripe subscription flows on staging, since it shares live Stripe keys and webhooks point at the production domain only.

## Production Deploys (`.github/workflows/deploy-prod.yml`)

Promotes an already-deployed, manually-tested staging version to production. Manual trigger only; never runs automatically.

**Trigger**: `workflow_dispatch` with an optional `version` input. If omitted, defaults to the most recently created `staging-<sha>` version (the fixed `staging` testing version is never a promotion target).

```bash
gh workflow run deploy-prod.yml --repo jeffreyp/goread2                       # promote latest staging-<sha>
gh workflow run deploy-prod.yml --repo jeffreyp/goread2 -f version=staging-a1b2c3d  # promote a specific one
```

**Jobs**:
1. `resolve-version`: figures out which version to promote (default or explicit input) and validates its format. Runs before the approval gate so the reviewer's job summary can show the concrete version, not a placeholder.
2. `deploy-prod`: declares `environment: production`, pausing for human approval (see the Production Approval Gate section above). After approval: captures whichever version currently holds `traffic_split=1.0` (for the cleanup job and for the record), then runs `gcloud app versions migrate` to shift 100% of traffic to the resolved version.
3. `cleanup`: deletes stale `prod-*` versions, keeping only the new current one and the captured previous one. Promoted `staging-<sha>` versions are pruned by `deploy-staging.yml`'s own cleanup instead (see above), since a promotion keeps its `staging-<sha>` name rather than being renamed to `prod-*`.

**Critical detail**: this uses `gcloud app versions migrate`, not a redeploy. The exact binary a reviewer clicked through on staging is what serves production: no rebuild step, no chance of drift between what was tested and what ships.

Immediately after migrating traffic, `scripts/smoke-check.sh` (see below) runs against `https://goreadapp.com` and fails the job if any assertion fails; see **Auto-Rollback** below for what happens next when it fails.

A promoted version keeps its `staging-<sha>` name permanently in production and is never renamed to `prod-*`. App Engine Standard versions are immutable and this app deploys from source, so the only way to get a `prod-<timestamp>`-named version live would be to rebuild the same commit under a new name, reintroducing the rebuild step this design exists to avoid, for a purely cosmetic naming benefit. `traffic_split=1.0` is what actually identifies the live production version; no tooling keys off the name (`deploy-staging.yml`'s cleanup guard checks `traffic_split=0`, not a name pattern).

Migrating traffic between two different versions, rather than a same-version no-op, requires `roles/appengine.serviceAdmin` in addition to `roles/appengine.deployer` (see CI/CD Authentication above), and the target version must have App Engine warmup requests enabled before it can gain traffic from 0%: `app.yaml` declares `inbound_services: [warmup]` for every version, staging and production alike, and `main.go` serves `GET /_ah/warmup` to satisfy it. Both requirements also apply to `rollback.yml`: a rollback to a previously-deployed, zero-traffic version exercises the identical permission and the identical warmup check.

**Concurrency**: `deploy-prod.yml` and `rollback.yml` share one `concurrency: { group: production-deploy, cancel-in-progress: false }` group. Both run `gcloud app versions migrate` against the same production traffic split, so if a rollback were dispatched while a promotion is still mid-flight (e.g. during the 15-minute post-promote health watch), racing `migrate` calls could leave production pointed at either version nondeterministically. The shared group forces one to fully finish (including auto-rollback dispatch, if triggered) before the other starts.

## Post-Deploy Smoke Check (`scripts/smoke-check.sh`)

Unauthenticated HTTP assertions run against a base URL after every staging and production deploy (see the two workflow sections above). Exits non-zero, failing the calling workflow job, if any assertion fails.

```bash
./scripts/smoke-check.sh https://staging-a1b2c3d-dot-goread-467200.uc.r.appspot.com
```

**Assertions:**
- `GET /`: 200, body contains `GoRead` (app started, templates loaded)
- `GET /privacy`: 200
- `GET /api/feeds`: 401 with a JSON error body (auth middleware ran, database responded)
- `GET /auth/login`: 200 with a JSON `auth_url` pointing at `accounts.google.com` (OAuth config loaded). Note this is a JSON response, not a server-side redirect; the handler hands the URL to the frontend to redirect the browser itself, so there's no `Location` header or 302 to check.
- `GET /static/js/app.min.js`: 200, `Content-Type: application/javascript` (frontend build present)
- `GET /static/css/styles.min.css`: 200
- `Strict-Transport-Security` header present on `/`: only set when `GAE_ENV=standard` (see `internal/middleware/security_headers.go`), so this assertion only passes when run against a real App Engine deployment, not a local dev server
- `X-Content-Type-Options: nosniff` header present on `/`
- `GET /auth/smoke-login`: 404 (confirms no backdoor auth endpoint is enabled)

## Auto-Rollback (post-promote safety net)

`gcloud app versions migrate` shifts 100% of traffic immediately, so `deploy-prod.yml`'s `deploy-prod` job pairs the migration with two checks, either of which auto-triggers a rollback:

1. **Smoke check** (`scripts/smoke-check.sh`, see above): runs immediately after the traffic migration.
2. **Post-promote health watch** (`scripts/post-promote-health-watch.sh`): runs immediately after the smoke check passes. Polls Cloud Monitoring every 60s for a 15-minute window (`WATCH_SECONDS=900`, `POLL_INTERVAL_SECONDS=60`), watching the same signals as five of the seven policies in `monitoring/alert-policies.yaml` (all but Datastore Entity Read Spike and Account Deletion Task Failed): 5xx rate, App Engine instance count, Datastore read/write rate, and network egress, plus p95 latency (`appengine.googleapis.com/http/server/response_latencies`), which has no corresponding alert policy; its threshold, 3000ms, is set conservatively above the app's typical latency. Each signal must breach its threshold for that signal's alert-policy `duration` (e.g. 5xx needs 180s sustained, instance count needs 600s) before it counts; a single noisy poll doesn't trigger anything. Queries the Cloud Monitoring REST API directly via `curl` plus an access token (`gcloud monitoring time-series list` does not exist as a CLI command, unlike `policies`/`channels`/`dashboards`). Each query looks back a fixed 300s window and takes the newest point (`points[0]`, the API returns newest-first) rather than a window tied to the poll interval, since App Engine and Datastore metric ingestion lags real time by a few minutes and a narrower lookback can come back empty.
   - Requires `roles/monitoring.viewer` on `cicd-deploy@goread-467200.iam.gserviceaccount.com` (see CI/CD Authentication above).

**On failure of either check**: the job dispatches `rollback.yml` via `gh workflow run rollback.yml -f version=$PREVIOUS_VERSION` (the version captured as live *before* this promotion, from the same `deploy-prod` job), fails loudly with a `::error::` annotation and a job summary entry, and does not attempt to wait for the rollback to complete; check the Actions tab for its progress. Requires `actions: write` on the job's `GITHUB_TOKEN` (added as a job-level `permissions` override, since the workflow-level default is `contents: read` / `id-token: write` only). If no previous version was captured (e.g. a first-ever promotion with nothing yet live), auto-rollback is skipped with an error instructing manual intervention, since there is nothing to roll back to.

This is a safety net, not a replacement for watching the deploy: the health watch adds up to 15 minutes to every production promotion, and a human should still confirm the rollback (once dispatched) actually restored service.

## Rollback (`.github/workflows/rollback.yml`)

One-click rollback: shifts 100% of production traffic to a previously-deployed version instantly, without needing local `gcloud` credentials.

**Finding available version names:**
```bash
gcloud app versions list --service=default --project=goread-467200 \
  --format="table(version.id,traffic_split,version.createTime)" \
  --sort-by="~version.createTime"
```
Look for the `staging-<sha>` version with `traffic_split: 0.00` from before the bad deploy; a promoted version keeps this name rather than being renamed (see [Production Deploys](#production-deploys-githubworkflowsdeploy-prodyml) above), so that version is the rollback target.

**Triggering a rollback:**
```bash
gh workflow run rollback.yml --repo jeffreyp/goread2 -f version=staging-a1b2c3d
```
Or via the GitHub Actions UI: Actions → Rollback → Run workflow, entering the version name.

The workflow authenticates via the same WIF setup as `deploy-staging.yml`, runs `gcloud app versions migrate VERSION --service=default`, and prints a confirmation with the version name to the job summary. This is instant (no rebuild) since it's re-pointing traffic at an already-deployed artifact.

Before migrating, it captures whichever version currently holds `traffic_split=1.0` (for the job summary). Its cleanup step, like `deploy-prod.yml`'s, targets stale `prod-*` versions; promoted `staging-<sha>` versions are pruned by `deploy-staging.yml`'s own cleanup instead (see the naming discussion under Production Deploys above).

## Apple Release Pipelines

The native iOS, iPadOS, and macOS apps are released through separate GitHub Actions workflows, which do not affect the App Engine deployment. See the [Apple release guide](apple-releases.md).

## Google App Engine Configuration

### Environment Variables Setup

On App Engine, every secret is stored in Google Secret Manager, never in `app.yaml`. The server fetches the OAuth credentials, `CSRF_SECRET`, `INITIAL_ADMIN_EMAILS`, `ACCOUNT_TOMBSTONE_KEY`, and the four Stripe variables at runtime under fixed secret names. [configuration.md](configuration.md#how-values-are-resolved) describes how values are resolved and lists each secret's name. `app.yaml` itself carries no `${VAR}`-style placeholders; it is deployed directly, with no substitution step.

#### Setting up Google Secret Manager

1. **Enable the Secret Manager API:**
   ```bash
   gcloud services enable secretmanager.googleapis.com
   ```

2. **Create secrets for each environment variable:**
   ```bash
   # OAuth configuration
   echo -n "your-oauth-client-id" | gcloud secrets create google-client-id --data-file=-
   echo -n "your-oauth-client-secret" | gcloud secrets create google-client-secret --data-file=-

   # CSRF secret (REQUIRED for production)
   openssl rand -base64 32 | gcloud secrets create csrf-secret --data-file=-

   # Initial admin bootstrap emails (optional)
   echo -n "admin@example.com" | gcloud secrets create initial-admin-emails --data-file=-

   # Deleted-account tombstone key (recommended; never rotate)
   openssl rand -base64 32 | tr -d '\n' | gcloud secrets create account-tombstone-key --data-file=-

   # Stripe configuration (if using subscriptions)
   echo -n "sk_live_your-secret-key" | gcloud secrets create stripe-secret-key --data-file=-
   echo -n "pk_live_your-publishable-key" | gcloud secrets create stripe-publishable-key --data-file=-
   echo -n "whsec_your-webhook-secret" | gcloud secrets create stripe-webhook-secret --data-file=-
   echo -n "price_your-price-id" | gcloud secrets create stripe-price-id --data-file=-
   ```

3. **Grant App Engine access to secrets:**
   ```bash
   PROJECT_ID=$(gcloud config get-value project)

   gcloud secrets add-iam-policy-binding google-client-id \
       --member="serviceAccount:${PROJECT_ID}@appspot.gserviceaccount.com" \
       --role="roles/secretmanager.secretAccessor"

   gcloud secrets add-iam-policy-binding google-client-secret \
       --member="serviceAccount:${PROJECT_ID}@appspot.gserviceaccount.com" \
       --role="roles/secretmanager.secretAccessor"

   gcloud secrets add-iam-policy-binding csrf-secret \
       --member="serviceAccount:${PROJECT_ID}@appspot.gserviceaccount.com" \
       --role="roles/secretmanager.secretAccessor"

   gcloud secrets add-iam-policy-binding initial-admin-emails \
       --member="serviceAccount:${PROJECT_ID}@appspot.gserviceaccount.com" \
       --role="roles/secretmanager.secretAccessor"

   gcloud secrets add-iam-policy-binding account-tombstone-key \
       --member="serviceAccount:${PROJECT_ID}@appspot.gserviceaccount.com" \
       --role="roles/secretmanager.secretAccessor"

   # Repeat for other secrets (Stripe, etc.)...
   ```

### app.yaml Configuration

[`app.yaml`](../app.yaml) sets the OAuth secret names, the production and staging redirect URLs, and the subscription flag; everything else it needs, including `CSRF_SECRET`, `INITIAL_ADMIN_EMAILS`, `ACCOUNT_TOMBSTONE_KEY`, and the four Stripe variables, is fetched from Secret Manager under fixed default names with no `app.yaml` entry at all (see [Environment Variables Setup](#environment-variables-setup) above). The excerpt below is the `env_variables` block; the handlers, scaling, and warmup settings covered elsewhere in this section are omitted:

```yaml
runtime: go126

env_variables:
  GIN_MODE: release
  GOOGLE_REDIRECT_URL: "https://goreadapp.com/auth/callback"
  STAGING_REDIRECT_URL: "https://staging-dot-goread-467200.uc.r.appspot.com/auth/callback"
  GOOGLE_CLOUD_PROJECT: "goread-467200"
  SECRET_CLIENT_ID_NAME: "google-client-id"
  SECRET_CLIENT_SECRET_NAME: "google-client-secret"
  SUBSCRIPTION_ENABLED: "true"
```

`SECRET_CLIENT_ID_NAME`/`SECRET_CLIENT_SECRET_NAME` override which Secret Manager secret name to fetch the OAuth credentials from; omit them to fall back to the defaults `google-client-id`/`google-client-secret`.

`app.yaml` also sets `instance_class: F1` with `automatic_scaling.min_instances: 0`, and declares `inbound_services: [warmup]`, required for [Production Deploys](#production-deploys-githubworkflowsdeploy-prodyml) to shift traffic onto a new version.

### Static File Handlers

App Engine serves `/static` directly from `web/static` through two handlers in `app.yaml`, so these requests never reach the Go app:

```yaml
handlers:
- url: /static/(.*\.(css|js))$
  static_files: web/static/\1
  upload: web/static/.*\.(css|js)$
  expiration: 365d
  secure: always

- url: /static
  static_dir: web/static
  secure: always
```

The first handler serves CSS and JS with a one-year expiration. Templates reference these files with a `?v=` content hash, so a deploy that changes a file also changes its URL (see [performance.md](performance.md#versioned-css-and-js)). The second handler serves every other static file, such as images and favicons, with App Engine's default 10-minute expiration.

### Instance Shutdown

App Engine sends `SIGTERM` before it stops an instance. On `SIGTERM` or `SIGINT`, `main.go` stops accepting connections, gives in-flight requests up to 10 seconds to finish, and then returns, which runs deferred cleanup such as closing the Cloud Tasks client. Background work started with the signal context, such as the feed service, stops at the same time. Locally, a second Ctrl-C during the drain stops the process at once.

### .gcloudignore

`.gcloudignore`, at the repository root, controls which files `gcloud app deploy` uploads. Besides development artifacts, it excludes `ios/`, the native iOS app, which is not part of the App Engine service.

### cron.yaml Configuration

`cron.yaml` schedules six jobs, all targeting the `default` service with the same retry policy (`min_backoff_seconds: 10`, `max_backoff_seconds: 300`, `max_doublings: 3`):

| Schedule | Endpoint | Purpose |
|----------|----------|---------|
| Every 2 hours | `/cron/refresh-feeds` | Refresh subscribed RSS feeds |
| Every 24 hours | `/cron/cleanup-sessions` | Delete expired sessions |
| Every 24 hours | `/cron/cleanup-orphaned-articles` | Delete orphaned per-user article state |
| Monthly, 1st at 09:00 | `/cron/apply-retention` | Apply the data retention policy; see [privacy.md](privacy.md#data-retention) |
| Daily at 10:00 | `/cron/retry-pending-deletions` | Retry self-requested account deletions Cloud Tasks gave up on; see [privacy.md](privacy.md#account-deletion) |
| Daily at 00:30 | `/cron/record-usage-metrics` | Record usage statistics as Cloud Monitoring custom metrics; see [monitoring.md](monitoring.md#usage-metrics) |

See [`cron.yaml`](../cron.yaml) for the full configuration. Four of the six (refresh-feeds, apply-retention, retry-pending-deletions, and cleanup-orphaned-articles) dispatch through Cloud Tasks (see [Async Task Processing](#async-task-processing-cloud-tasks) below); cleanup-sessions and record-usage-metrics run directly in the cron request.

### Deployment Steps

1. **Configure OAuth redirect URI**: add the production URL, `https://your-app.appspot.com/auth/callback`, to the OAuth client's authorized redirect URIs.

2. **Create secrets in Google Secret Manager**: `app.yaml` has no `${VAR}` placeholders; every credential (OAuth, CSRF, initial admin emails, tombstone key, Stripe) is fetched from Secret Manager at runtime. See [Setting up Google Secret Manager](#setting-up-google-secret-manager) above; this is done once per project.

3. **Initialize App Engine:**
   ```bash
   gcloud app create --region=us-central1
   ```

4. **Deploy**: there is no manual deploy command; deploys happen through the GitHub Actions pipeline described earlier in this doc:
   - Push to `main` → `deploy-staging.yml` automatically deploys a zero-traffic `staging-<sha>` version and deploys `cron.yaml` and `index.yaml` (see [Automated Staging Deploys](#automated-staging-deploys-githubworkflowsdeploy-stagingyml)).
   - Click through the staging URL, then promote with `gh workflow run deploy-prod.yml` (see [Production Deploys](#production-deploys-githubworkflowsdeploy-prodyml)); this pauses for human approval, migrates traffic, then runs the smoke check and 15-minute health watch automatically.
   - To roll back, see [Rollback](#rollback-githubworkflowsrollbackyml).

5. **Verify deployment:**
   ```bash
   # Open application
   gcloud app browse

   # Check logs for any configuration issues
   gcloud app logs tail -s default
   ```

### Database Configuration (App Engine)

- **Production database**: Google Cloud Datastore, selected automatically when `GAE_ENV=standard`.
- **Entities**: `User`, `UserFeed`, and `UserArticle` are per-user; `Feed` and `Article` are shared across users. See [architecture.md](architecture.md#data-model) for the full data model.
- **User isolation**: every query is filtered by the authenticated user's ID.

### Point-in-Time Recovery

The production `(default)` database in `goread-467200` has point-in-time recovery (PITR) enabled, with a 7-day version retention period. Any state of the database within the last 7 days, at minute granularity, can be recovered. This covers mistakes such as a wrong account deletion (`DELETE /api/account` or the retention run) or a bad migration. It has no scheduled backups, so nothing older than 7 days can be restored. PITR is billed for the storage of the retained versions.

To check the setting and the earliest recoverable time:

```bash
gcloud firestore databases describe --database='(default)' --project=goread-467200 \
  --format='yaml(pointInTimeRecoveryEnablement,versionRetentionPeriod,earliestVersionTime)'
```

To recover, clone the database as it was at a given minute into a new database. The clone does not modify `(default)`:

```bash
gcloud firestore databases clone --project=goread-467200 \
  --source-database='projects/goread-467200/databases/(default)' \
  --snapshot-time='2026-11-08T10:20:00Z' \
  --destination-database=restore-20261108
```

The application always reads `(default)`, so recovered entities must be copied from the clone back into `(default)` with the Datastore client or console. There is no tool for this in the repository. Delete the clone once the copy is verified. To restore the whole database, copy every kind back, or delete `(default)` and clone the recovered database into a new `(default)`; the second option takes the application offline until the clone completes. No restore has yet been performed against production.

## Async Task Processing (Cloud Tasks)

The `/cron/refresh-feeds`, `/cron/cleanup-orphaned-articles`, `/cron/apply-retention`, and `/cron/retry-pending-deletions` cron handlers enqueue their work as Cloud Tasks instead of running it in the request that App Engine Cron triggers. This keeps the cron request itself under 100ms, so App Engine doesn't hold an instance open for the duration of a feed refresh, and Cloud Tasks retries the work independently of cron's own retry policy if it fails.

Each cron handler enqueues a task targeting a corresponding worker endpoint on the same App Engine service:

- `/cron/refresh-feeds` enqueues `/tasks/refresh-feeds`
- `/cron/cleanup-orphaned-articles` enqueues `/tasks/cleanup-orphaned-articles`
- `/cron/apply-retention` enqueues `/tasks/apply-retention`
- `/cron/retry-pending-deletions` enqueues `/tasks/retry-pending-deletions`

Account deletion uses the same queue. `DELETE /api/account` enqueues `/tasks/delete-account?user_id=<id>` twice: once for immediate dispatch, and once delayed by 15 minutes. The delayed pass removes any rows recreated after the first pass by requests on sessions still held in another instance's session cache, whose TTL is 10 minutes. Both passes run the same idempotent deletion, so either one can finish a deletion the other left incomplete. [privacy.md](privacy.md#account-deletion) describes what the deletion removes.

Each task is routed to the App Engine version that enqueued it, using the `GAE_SERVICE` and `GAE_VERSION` values App Engine sets on every instance (`internal/services/cloud_tasks.go`). A deletion requested on the zero-traffic `staging` version therefore runs `/tasks/delete-account` on `staging`, not on the production version. Cron requests always arrive at the default version, so the tasks they enqueue run there as well. The fixed `staging` version is never deleted, and a `staging-<sha>` version is deleted only after two later staging deploys (see [Automated Staging Deploys](#automated-staging-deploys-githubworkflowsdeploy-stagingyml) above).

Tasks are dispatched using Cloud Tasks' `AppEngineHttpRequest` target. App Engine attaches an `X-AppEngine-QueueName` header to genuine task dispatches and strips that header from any external request that tries to set it, the same protection `X-Appengine-Cron` gives the cron endpoints; the task worker endpoints check for its presence (`internal/auth.VerifyTaskRequest`) rather than requiring a separate signature check.

**One-time setup** (project `goread-467200`, matching the App Engine region). This has already been done for the production project; these commands are for standing up a new project or environment:

```bash
gcloud services enable cloudtasks.googleapis.com
gcloud tasks queues create cron-tasks --location=us-central1 \
  --max-attempts=5 \
  --min-backoff=10s \
  --max-backoff=300s \
  --max-doublings=3
```

The retry flags mirror `cron.yaml`'s `retry_parameters` instead of Cloud Tasks' default (up to 100 attempts backing off to a full hour), which would keep retrying a broken task handler far longer than useful.

The App Engine default service account (`goread-467200@appspot.gserviceaccount.com`) needs `roles/cloudtasks.enqueuer` to create tasks on this queue:

```bash
gcloud projects add-iam-policy-binding goread-467200 \
  --member="serviceAccount:goread-467200@appspot.gserviceaccount.com" \
  --role="roles/cloudtasks.enqueuer"
```

**Local development and any environment where `GAE_ENV` isn't `standard`**: the app never creates a Cloud Tasks client, since that would need credentials and a real queue to enqueue against, so the cron handlers run the work directly in the request instead. The `/tasks/*` worker endpoints still exist locally but are only reachable by calling them directly with admin authentication, matching `VerifyCronRequest`'s non-GAE fallback.

Queue name and location default to `cron-tasks` and `us-central1`; override with `CLOUD_TASKS_QUEUE` and `CLOUD_TASKS_LOCATION` if a deployment uses different values.

### Account Deletion Failures

Each failed attempt logs `Account deletion task failed for user <id>`, which triggers the `Account Deletion Task Failed` alert (see [monitoring.md](monitoring.md#7-account-deletion-task-failed)). With the queue's retry settings above, each pass gives up after about 2.5 minutes, so a deletion fails permanently only if Stripe or Datastore is unavailable during both passes. The user has already received a success response and been signed out, and cannot sign back in to retry, since the account carries a `deletion_requested_at` timestamp (see [privacy.md](privacy.md#account-deletion)). The daily `/cron/retry-pending-deletions` job retries any account in that state once it is more than an hour old, so the deletion completes without manual action within a day. If that retry also fails, it logs `Pending account deletion retry failed for user <id>`, which triggers the `Pending Account Deletion Retry Failed` alert (see [monitoring.md](monitoring.md#8-pending-account-deletion-retry-failed)). To complete a deletion before the next daily retry, re-enqueue it as described below.

When the alert fires, check whether the account still exists about 20 minutes after the request, once both passes have run, by looking for the ID in the first column of the admin CLI's user list (see [admin.md](admin.md#admin-script-usage)):

```bash
./admin.sh list
```

If the account is still listed, re-enqueue the deletion once the failing dependency has recovered. The task is idempotent, so running it again is safe:

```bash
gcloud tasks create-app-engine-task --project=goread-467200 \
  --queue=cron-tasks --location=us-central1 \
  --method=POST --relative-uri="/tasks/delete-account?user_id=<id>"
```

Then confirm in the Stripe dashboard that no customer with the user's email remains.

## Environment Variables

[configuration.md](configuration.md) lists every variable the server reads, with its default and Secret Manager name.

## Security Considerations

[security.md](security.md) and [authentication.md](authentication.md) cover authentication, CSRF, CORS, input validation, and session security.

HTTP security headers are set by Go middleware, not an `app.yaml` `http_headers` block (`app.yaml` has none): `internal/middleware/security_headers.go` sets `Content-Security-Policy` (or `Content-Security-Policy-Report-Only` when `CSP_REPORT_ONLY=true`; see [security.md](security.md#content-security-policy)), `X-Frame-Options: SAMEORIGIN`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, and a restrictive `Permissions-Policy`. `Strict-Transport-Security` is added only when `GAE_ENV=standard`, since HSTS should not be sent over a non-TLS local connection.

## Monitoring and Maintenance

```bash
gcloud app logs tail -s default
```

See [monitoring.md](monitoring.md) for dashboards, alert policies, the cost budget, and the usage-metrics cron job.

## Testing

`make test` runs the full suite locally; see [testing.md](testing.md) for what it covers. `make validate-config` checks that required environment variables and secrets are present before a deploy. Against an already-deployed version, `scripts/smoke-check.sh` (see [Post-Deploy Smoke Check](#post-deploy-smoke-check-scriptssmoke-checksh) above) is what actually runs; there is no separate manual checklist beyond what it already asserts.

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md#deployment-issues) for OAuth, session, database, and App Engine deployment issues.

## Cost Optimization

- **Instance scaling**: `app.yaml` sets `instance_class: F1` with `automatic_scaling.min_instances: 0`, so no instance runs, and nothing is billed, while there's no traffic.
- **Async cron processing**: four of the six cron jobs dispatch through Cloud Tasks rather than running in the cron request itself, so instances aren't held open for the duration of a feed refresh or a retention sweep; see [Async Task Processing](#async-task-processing-cloud-tasks) above.
- **Conditional feed fetches**: feed refreshes send `If-None-Match`/`If-Modified-Since` and skip downloading and parsing on a `304`; see [performance.md](performance.md#feed-refresh).

See [performance.md](performance.md) for the cost-driven optimizations already implemented and [monitoring.md](monitoring.md) for the cost dashboard and budget alert.

## Related Documentation

- [Stripe Setup](stripe.md) - Configuring subscription payments
- [Configuration Reference](configuration.md) - Every environment variable the server reads
- [Architecture](architecture.md) - How the deployed pieces fit together, including the data model
- [Privacy and Personal Data](privacy.md) - Account deletion and data retention
- [Apple Apps](apple-apps.md) - Local development and on-device testing for the native app
- [Apple Release Guide](apple-releases.md) - TestFlight and notarized DMG releases
- [Monitoring Setup](monitoring.md) - Dashboards, alerts, and cost tracking
- [Security Guidelines](security.md) - Security controls and best practices
- [Troubleshooting Guide](troubleshooting.md) - Common issues
