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
- [Testing in Production](#testing-in-production)
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

GitHub Actions authenticates to GCP via Workload Identity Federation (WIF): keyless, no service account JSON key stored in GitHub. This is the foundation for the automated deploy workflows (staging/prod pipelines are tracked separately in the gr-f6v epic); this section documents the trust relationship itself.

**Resources created** (one-time setup, project `goread-467200`):
- Workload Identity Pool: `github-actions-pool` (location `global`)
- OIDC Provider: `github-provider`, issuer `https://token.actions.githubusercontent.com`, attribute condition restricting it to `assertion.repository == 'jeffreyp/goread2'`; no other repo can assume this identity
- Service account: `cicd-deploy@goread-467200.iam.gserviceaccount.com`, granted `roles/appengine.deployer`, `roles/cloudbuild.builds.editor`, `roles/storage.admin`, `roles/secretmanager.secretAccessor` at the project level, plus `roles/iam.serviceAccountUser` on `goread-467200@appspot.gserviceaccount.com` specifically (App Engine deploys require the deploying identity to be able to act as the App Engine default service account; easy to miss, deploys fail without it)
- The pool is bound to the service account via `roles/iam.workloadIdentityUser`, scoped to the `attribute.repository/jeffreyp/goread2` principal set; only workflow runs from this exact repo can impersonate it
- Two roles were added after this initial grant, each discovered by hitting the missing permission live rather than anticipated up front: `roles/appengine.serviceAdmin` (see the Production Deploys section's "Two real bugs" below: traffic migration between two *different* versions needs more than `appengine.deployer`) and `roles/monitoring.viewer` (see Auto-Rollback below, added 2026-07-05 so the post-promote health watch can read Cloud Monitoring time series)
- `roles/cloudscheduler.admin` and `roles/datastore.indexAdmin` let the staging workflow deploy `cron.yaml` and `index.yaml`

**GitHub Actions repository variables** (not secrets; the provider path and SA email aren't sensitive on their own):
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

**Configuration** (repo `jeffreyp/goread2`):
- Environment: `production`
- Required reviewer: `jeffreyp` (GitHub user id 548089)
- Deployment branch policy: restricted to `main` only; no other branch can target this environment
- **Branch protection on `main` was deliberately NOT enabled**, even though the originating issue (gr-onn) asked for it. Jeffrey pushes directly to `main` without PRs; GitHub's required-status-checks branch protection blocks pushes for commits that don't already have a passing check run, which effectively forces a PR-based workflow. Skipped to preserve the existing direct-push workflow; revisit only if the team moves to a PR-based flow.

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

- **Trigger**: `workflow_run` for the `Tests` workflow, `types: [completed]`, gated on `github.event.workflow_run.conclusion == 'success'` and `branches: [main]`; deploy never runs if CI failed.
- **Version name**: `staging-<short-sha>`, e.g. `staging-a1b2c3d`, deployed with `--no-promote` (no production traffic).
- **cron.yaml / index.yaml**: deployed on every run. Both deploys are idempotent, so a run that fails is repaired by the next successful one, with no need to track which commit last deployed them.
- **Job summary**: prints the version name and full staging URL (`https://<version>-dot-goread-467200.uc.r.appspot.com`) so a reviewer knows where to click through and test.
- All actions are pinned to commit SHA (not floating tags like `@v4`) per supply-chain hardening feedback from a security review; see the workflow file's inline `# vX` comments for the corresponding version.
- **BUILD_VERSION** is injected into `app.yaml` at deploy time (via `sed`, right before the deploy step) since App Engine standard has no `--set-env-vars` deploy flag and the file has no other templating mechanism.
- **Cleanup**: after deploying, stale `staging-*` versions are deleted, keeping the one just deployed plus the single most-recently-created other one; `staging-*` versions would otherwise accumulate one per push indefinitely. Restricted to versions with `traffic_split=0`: an older `staging-<sha>` can be the version currently serving 100% of production traffic, since `deploy-prod.yml` (below) promotes by migrating traffic directly onto a `staging-<sha>` version rather than redeploying under a new name. This guard is what stops the very next push from deleting live production. The extra kept slot is normally the production version just demoted by the last promotion, so `rollback.yml` still has a version to target for one more staging deploy cycle after a promotion.
- **Smoke check**: immediately after both versions deploy, `scripts/smoke-check.sh` (see below) runs against the `staging-<sha>` URL and fails the job if it doesn't pass, catching a broken deploy before a human ever clicks through it.
- **Concurrency**: `concurrency: { group: deploy-staging, cancel-in-progress: false }` serializes runs. Back-to-back commits to `main` each spawn their own `Tests` run and therefore their own `deploy-staging` run, with no guarantee the older commit's run finishes first. Without this guard, an older commit's run finishing *after* a newer commit's run would delete the newer commit's just-deployed `staging-<sha>` version in the cleanup step above, silently reverting staging. `cancel-in-progress` is `false` rather than `true` because killing a `gcloud app deploy` mid-flight can leave a version half-created, which is worse than a queued run waiting its turn.

**Bugs hit and fixed while first bringing this workflow up:**
1. `$GITHUB_SHA` is **not** the triggering commit for `workflow_run` events; it's whatever the default branch tip happens to be when the job executes. Two runs that fired close together both computed the same `staging-<sha>` name from `$GITHUB_SHA` and collided (`ABORTED: operation already in progress`). Fixed by deriving the short SHA from `github.event.workflow_run.head_sha` instead. Note this was a same-commit name collision; the separate, cross-commit ordering race described in **Concurrency** above wasn't fixed until the `concurrency:` group was added.
2. The first real deploy 503'd, see the Stripe placeholder note above; `deploy-staging.yml` deploys `app.yaml` directly with no `envsubst` step, so any lingering `${VAR}`-style placeholder deploys as a literal, broken string.

### Human OAuth Login Testing on Staging (fixed `staging` version)

The production approval gate (gr-euu) requires a human to actually log in and click through the staging UI before promoting: automated smoke checks alone don't prove the UI is usable. But Google OAuth requires an exact, pre-registered redirect URI, and the per-SHA `staging-<sha>` hostname changes on every deploy, so it can never complete a login round-trip.

To solve this, `deploy-staging.yml` deploys the **same build twice**:
- `staging-<sha>` (`--no-promote`): the promotion candidate that `deploy-prod.yml` migrates to production.
- `staging` (`--no-promote`, fixed name, overwritten on every merge): exists only so a human has a stable URL to log into: `https://staging-dot-goread-467200.uc.r.appspot.com`.

Both redirect URIs are registered on the **same** Google OAuth client (Google Cloud Console → APIs & Services → Credentials → the existing OAuth 2.0 Client ID → Authorized redirect URIs):
- `https://goreadapp.com/auth/callback` (production, unchanged)
- `https://staging-dot-goread-467200.uc.r.appspot.com/auth/callback` (staging; **one-time manual console step**, not automatable via this pipeline)

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
3. `cleanup`: deletes stale `prod-*` versions, keeping only the new current one and the captured previous one. Scoped to the legacy `prod-<timestamp>` naming from the old manual `make deploy-prod` path (two such versions exist from before this pipeline existed); going forward, promoted versions are pruned by `deploy-staging.yml`'s own cleanup instead (see above), since they keep their `staging-<sha>` name.

**Critical detail**: this uses `gcloud app versions migrate`, not a redeploy. The exact binary a reviewer clicked through on staging is what serves production: no rebuild step, no chance of drift between what was tested and what ships.

Immediately after migrating traffic, `scripts/smoke-check.sh` (see below) runs against `https://goreadapp.com` and fails the job if any assertion fails. Unlike the earlier design, this is no longer the last word; see **Auto-Rollback** below for what happens next when it fails.

**Naming decision (2026-07-05)**: a promoted version keeps its `staging-<sha>` name permanently in production. It is never renamed to `prod-*`. This was a deliberate choice, confirmed with Jeffrey after the first live promotion: App Engine Standard versions are immutable and this app deploys from source (`gcloud app versions describe` shows no pre-built `deployment.container` reference to reuse), so the only way to get a `prod-<timestamp>`-named version live would be to rebuild the same commit under a new name, reintroducing exactly the rebuild step this design exists to avoid, for a purely cosmetic naming benefit. `traffic_split=1.0` is what actually identifies the live production version; the name is not load-bearing anywhere in tooling (`deploy-staging.yml`'s cleanup guard checks `traffic_split=0`, not a name pattern).

**Two real bugs hit while first bringing this up live** (both invisible until an actual cross-version promotion was attempted, not the same-version no-op that "verified" rollback.yml earlier):
1. `cicd-deploy`'s original IAM grant (`roles/appengine.deployer` only) doesn't include `appengine.services.update`, required to actually shift traffic between two *different* versions (as opposed to `services.get`/`services.list`, which `deployer` does include). Fixed by granting `roles/appengine.serviceAdmin` to the service account. This also silently affected `rollback.yml`: its "verified live" test only ever migrated a version to itself, which never exercises this permission.
2. `gcloud app versions migrate` requires the *target* version to have App Engine warmup requests enabled before it can gain traffic from 0%: `INVALID_ARGUMENT: Warmup requests must be enabled for all versions that will gain additional traffic`. Fixed by adding `inbound_services: [warmup]` to `app.yaml` (applies to every version, staging and prod alike) plus a trivial `GET /_ah/warmup` handler in `main.go` that returns 200. Same blind spot as above: a same-version migrate never triggers this check since the target already has traffic.

Both fixes are load-bearing for `rollback.yml` too, not just this workflow: a real rollback to a previously-deployed (zero-traffic) version would have hit the identical two failures.

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

`gcloud app versions migrate` shifts 100% of traffic immediately. Before this safety net existed, a smoke check failure only turned the workflow red; the broken version stayed fully live until a human noticed and manually ran `rollback.yml`. `deploy-prod.yml`'s `deploy-prod` job now closes that gap with two checks, either of which auto-triggers a rollback:

1. **Smoke check** (`scripts/smoke-check.sh`, see above): runs immediately after the traffic migration.
2. **Post-promote health watch** (`scripts/post-promote-health-watch.sh`): runs immediately after the smoke check passes. Polls Cloud Monitoring every 60s for a 15-minute window (`WATCH_SECONDS=900`, `POLL_INTERVAL_SECONDS=60`), watching the same signals as five of the seven policies in `monitoring/alert-policies.yaml` (all but Datastore Entity Read Spike and Account Deletion Task Failed): 5xx rate, App Engine instance count, Datastore read/write rate, and network egress, plus p95 latency (`appengine.googleapis.com/http/server/response_latencies`), which has no corresponding alert policy today; a 3000ms threshold was chosen as a conservative ~5x multiple of the app's typical baseline (~600ms at the time this was written) and should be revisited if that baseline drifts. Each signal must breach its threshold for that signal's alert-policy `duration` (e.g. 5xx needs 180s sustained, instance count needs 600s) before it counts; a single noisy poll doesn't trigger anything. Queries the Cloud Monitoring REST API directly via `curl` + an access token (`gcloud monitoring time-series list` does not exist as a CLI command, unlike `policies`/`channels`/`dashboards`), since `jq` and `curl` are preinstalled on `ubuntu-latest` and this avoids scripting a second SDK. Each query looks back a fixed 300s window and takes the newest point (`points[0]`, API returns newest-first) rather than a window tied to the poll interval. App Engine/Datastore metric ingestion lags real time by a few minutes, and a narrower lookback came back empty on every poll in testing.
   - Requires `roles/monitoring.viewer` on `cicd-deploy@goread-467200.iam.gserviceaccount.com` (granted 2026-07-05; not part of the original WIF setup in the CI/CD Authentication section above, since nothing previously needed to *read* Monitoring data (only alerting policies needed write access).

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
Look for the `staging-<sha>` version with `traffic_split: 0.00` from before the bad deploy, per the [naming decision](#automated-staging-deploys-githubworkflowsdeploy-stagingyml) above; that version is the rollback target. There are no `prod-*` versions anymore except two legacy ones left over from before this pipeline existed.

**Triggering a rollback:**
```bash
gh workflow run rollback.yml --repo jeffreyp/goread2 -f version=staging-a1b2c3d
```
Or via the GitHub Actions UI: Actions → Rollback → Run workflow, entering the version name.

The workflow authenticates via the same WIF setup as deploy-staging, runs `gcloud app versions migrate VERSION --service=default`, and prints a confirmation with the version name to the job summary. This is instant (no rebuild) since it's re-pointing traffic at an already-deployed artifact.

Before migrating, it captures whichever version currently holds `traffic_split=1.0` (for the job summary). Its cleanup step, like `deploy-prod.yml`'s, is scoped to the legacy `prod-*` naming from the old manual `make deploy-prod` path; promoted `staging-<sha>` versions are pruned by `deploy-staging.yml`'s own cleanup instead (see the naming decision above).

## Apple Release Pipelines

The native iOS, iPadOS, and macOS apps are released through separate GitHub Actions workflows, which do not affect the App Engine deployment. See the [Apple release guide](apple-releases.md).

## Google App Engine Configuration

### Environment Variables Setup

**Important**: For Google App Engine deployments, environment variables should be configured in Google Secret Manager for security, not hardcoded in `app.yaml`.

**Secret Reference Convention**: The application supports a `_secret:` prefix for environment variables to explicitly trigger Secret Manager lookups. For example, setting `GOOGLE_CLIENT_ID=_secret:my-client-id` will fetch the secret from Google Secret Manager. This convention is consistent across all credentials (OAuth and Stripe) and prevents accidental conflicts with actual secret values.

**CSRF_SECRET, INITIAL_ADMIN_EMAILS, ACCOUNT_TOMBSTONE_KEY, and the four Stripe variables** all follow the same pattern as `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`: fetched from Secret Manager at runtime (secret names `csrf-secret`, `initial-admin-emails`, `account-tombstone-key`, `stripe-secret-key`, `stripe-publishable-key`, `stripe-webhook-secret`, `stripe-price-id`) and absent from `app.yaml` entirely. The server reads `ADMIN_TOKEN` only outside App Engine, so an App Engine deployment needs no `admin-token` secret (see [Environment Variables](#environment-variables)).

`ACCOUNT_TOMBSTONE_KEY` is the HMAC key for deleted-account tombstones, which stop a deleted account's Google ID from starting a second free trial (see [security.md](security.md#account-deletion)). Without it, account deletion still works but writes no tombstone, and sign-in skips the check. Do not rotate it: tombstones written under the old key stop matching.

The Stripe placeholders were removed 2026-07-04 while debugging why the first automated staging deploy (gr-rfd) 503'd: `deploy-staging.yml` deploys `app.yaml` directly with no `envsubst` step, so the old `${STRIPE_SECRET_KEY}`-style placeholders were being deployed as literal, unresolved strings. `secrets.GetStripeCredentials()` read that literal garbage from the env var (non-empty, so it never fell through to Secret Manager) and failed config validation. Same root cause `make substitute-secrets` existed to paper over for manual deploys. `app.yaml` now has zero `${VAR}` placeholders; the manual `make deploy-dev`/`deploy-prod`/`substitute-secrets` Makefile targets were removed once the GitHub Actions pipeline (staging/prod deploy workflows documented above) fully replaced them; see [Deployment Steps](#deployment-steps) below.

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

This is the actual production `app.yaml`. It carries no OAuth or Stripe values at all, only the Secret Manager secret *names* to look up; `CSRF_SECRET`, `INITIAL_ADMIN_EMAILS`, `ACCOUNT_TOMBSTONE_KEY`, and the four Stripe variables aren't listed here because they're fetched from Secret Manager under fixed default names with no `app.yaml` entry needed at all (see the Secret Reference Convention note above).

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

The first handler serves CSS and JS with a one-year expiration. Templates reference these files with a `?v=` content hash, so a deploy that changes a file also changes its URL (see [caching.md](caching.md#versioned-css-and-js)). The `go126` runtime always makes static files readable by the app, which reads them to compute the hashes. The second handler serves every other static file, such as images and favicons, with App Engine's default 10-minute expiration.

### Instance Shutdown

App Engine sends `SIGTERM` before it stops an instance. On `SIGTERM` or `SIGINT`, `main.go` stops accepting connections, gives in-flight requests up to 10 seconds to finish, and then returns, which runs deferred cleanup such as closing the Cloud Tasks client. Background work started with the signal context, such as the feed service, stops at the same time. Locally, a second Ctrl-C during the drain stops the process at once.

### .gcloudignore

`.gcloudignore`, at the repository root, controls which files `gcloud app deploy` uploads. Besides development artifacts, it excludes `ios/`, the native iOS app, which is not part of the App Engine service.

### cron.yaml Configuration

```yaml
cron:
- description: "Refresh RSS feeds"
  url: /cron/refresh-feeds
  schedule: every 2 hours
  target: default
  retry_parameters:
    min_backoff_seconds: 10
    max_backoff_seconds: 300
    max_doublings: 3

- description: "Cleanup expired sessions"
  url: /cron/cleanup-sessions
  schedule: every 24 hours
  target: default
  retry_parameters:
    min_backoff_seconds: 10
    max_backoff_seconds: 300
    max_doublings: 3

- description: "Cleanup orphaned user articles"
  url: /cron/cleanup-orphaned-articles
  schedule: every 24 hours
  target: default
  retry_parameters:
    min_backoff_seconds: 10
    max_backoff_seconds: 300
    max_doublings: 3

- description: "Apply data retention policy (delete inactive unpaid accounts, expire audit log IPs)"
  url: /cron/apply-retention
  schedule: 1 of month 09:00
  target: default
  retry_parameters:
    min_backoff_seconds: 10
    max_backoff_seconds: 300
    max_doublings: 3

- description: "Retry self-requested account deletions that Cloud Tasks gave up on"
  url: /cron/retry-pending-deletions
  schedule: every day 10:00
  target: default
  retry_parameters:
    min_backoff_seconds: 10
    max_backoff_seconds: 300
    max_doublings: 3

- description: "Record usage statistics as Cloud Monitoring custom metrics"
  url: /cron/record-usage-metrics
  schedule: every day 00:30
  target: default
  retry_parameters:
    min_backoff_seconds: 10
    max_backoff_seconds: 300
    max_doublings: 3
```

`/cron/apply-retention` applies the data retention policy described in [security.md](security.md#data-retention). `/cron/retry-pending-deletions` applies only the part of that policy that retries failed self-requested deletions, daily rather than monthly. `/cron/record-usage-metrics` records the usage statistics as Cloud Monitoring custom metrics, described in [monitoring.md](monitoring.md#usage-metrics); it runs in the cron request rather than as a Cloud Task.

### Deployment Steps

1. **Configure OAuth redirect URI:**
   Update OAuth configuration with production URL:
   `https://your-app.appspot.com/auth/callback`

2. **Set up secrets in Google Secret Manager:**
   `app.yaml` has no `${VAR}` placeholders; every credential (OAuth, CSRF, initial admin emails, tombstone key, Stripe) is fetched from Secret Manager at runtime. See [Setting up Google Secret Manager](#setting-up-google-secret-manager) above to create the secrets once per project.

3. **Initialize App Engine:**
   ```bash
   gcloud app create --region=us-central1
   ```

4. **Deploy:**
   There is no manual deploy command; deploys happen through the GitHub Actions pipeline described earlier in this doc:
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

- **Production**: Google Cloud Datastore (automatically detected)
- **Multi-user entities**: Users, UserFeeds, UserArticles
- **User isolation**: All queries filtered by authenticated user ID
- **Scalability**: Handles multiple concurrent users efficiently

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

Account deletion uses the same queue. `DELETE /api/account` enqueues `/tasks/delete-account?user_id=<id>` twice: once for immediate dispatch, and once delayed by 15 minutes. The delayed pass removes any rows recreated after the first pass by requests on sessions still held in another instance's session cache, whose TTL is 10 minutes. Both passes run the same idempotent deletion, so either one can finish a deletion the other left incomplete.

Each task is routed to the App Engine version that enqueued it, using the `GAE_SERVICE` and `GAE_VERSION` values App Engine sets on every instance (`internal/services/cloud_tasks.go`). A deletion requested on the zero-traffic `staging` version therefore runs `/tasks/delete-account` on `staging`, not on the production version, which may not yet serve a newly added task route. Cron requests always arrive at the default version, so the tasks they enqueue run there as well. The 15-minute delayed deletion pass uses the same routing, so it runs the same handler code as the immediate pass. If its version is deleted before it dispatches, the pass fails and is dropped after the queue's retries. The fixed `staging` version is never deleted, and a `staging-<sha>` version is deleted only after two later staging deploys (see [Automated Staging Deploys](#automated-staging-deploys-githubworkflowsdeploy-stagingyml)).

This routing has not been tested on a deployed version. A unit test confirms that each task request names the enqueuing service and version, but no task enqueued on `staging` has been observed running there. Account deletion is the only user action that enqueues a task, so the test requires deleting a second Google account on staging. Until that test is run, the behavior when a task's version has been deleted is also unconfirmed.

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

The retry flags mirror `cron.yaml`'s `retry_parameters` instead of Cloud Tasks' default (up to 100 attempts backing off to a full hour), which would keep hammering a broken task handler far longer than useful.

The App Engine default service account (`goread-467200@appspot.gserviceaccount.com`) needs `roles/cloudtasks.enqueuer` to create tasks on this queue:

```bash
gcloud projects add-iam-policy-binding goread-467200 \
  --member="serviceAccount:goread-467200@appspot.gserviceaccount.com" \
  --role="roles/cloudtasks.enqueuer"
```

**Local development and any environment where `GAE_ENV` isn't `standard`**: the app never creates a Cloud Tasks client (it would otherwise need credentials and a real queue to enqueue against), so the cron handlers fall back to doing the work in-process, the same as before Cloud Tasks was introduced. The `/tasks/*` worker endpoints still exist locally but are only reachable by calling them directly with admin authentication, matching `VerifyCronRequest`'s non-GAE fallback.

Queue name and location default to `cron-tasks` and `us-central1`; override with `CLOUD_TASKS_QUEUE` and `CLOUD_TASKS_LOCATION` if a deployment uses different values.

### Account Deletion Failures

Each failed attempt logs `Account deletion task failed for user <id>`, which triggers the `Account Deletion Task Failed` alert (see [monitoring.md](monitoring.md#7-account-deletion-task-failed)). With the queue's retry settings above, each pass gives up after about 2.5 minutes, so a deletion fails permanently only if Stripe or Datastore is unavailable during both passes. The user has already received a success response and been signed out, and cannot sign back in to retry, since the account carries a `deletion_requested_at` timestamp (see [security.md](security.md#account-deletion)). The daily `/cron/retry-pending-deletions` job retries any account in that state once it is more than an hour old, so the deletion completes without manual action within a day. If that retry also fails, it logs `Pending account deletion retry failed for user <id>`, which triggers the `Pending Account Deletion Retry Failed` alert (see [monitoring.md](monitoring.md#8-pending-account-deletion-retry-failed)). To complete a deletion before the next daily retry, re-enqueue it as described below.

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

The server reads every variable from the process environment. A variable with a Secret Manager name is also fetched from Google Secret Manager under that name when it is unset or holds a `_secret:` reference, which requires `GOOGLE_CLOUD_PROJECT`. On App Engine these values are stored in Secret Manager, not in `app.yaml` (see [Google App Engine Configuration](#google-app-engine-configuration)).

### Required Variables

| Variable | Secret Manager name | Description |
|----------|---------------------|-------------|
| `GOOGLE_CLIENT_ID` | `google-client-id` | OAuth 2.0 client ID from the Google Cloud Console |
| `GOOGLE_CLIENT_SECRET` | `google-client-secret` | OAuth 2.0 client secret from the Google Cloud Console |
| `GOOGLE_REDIRECT_URL` | None | OAuth callback URL, which must match a redirect URI registered on the OAuth client |
| `CSRF_SECRET` | `csrf-secret` | Base64-encoded secret of at least 32 bytes for CSRF tokens. In production, which means App Engine or `ENVIRONMENT=production`, the server exits at startup if it is missing, malformed, or too short. Elsewhere the server generates a random secret at each start when it is unset. |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GOOGLE_CLOUD_PROJECT` | Unset | Google Cloud project ID. When set, the server uses Datastore instead of the local SQLite file and can read Secret Manager. |
| `STAGING_REDIRECT_URL` | Unset | OAuth callback URL for the fixed `staging` version, selected by the request's `Host` header |
| `SECRET_CLIENT_ID_NAME` | `google-client-id` | Secret Manager name to read the OAuth client ID from |
| `SECRET_CLIENT_SECRET_NAME` | `google-client-secret` | Secret Manager name to read the OAuth client secret from |
| `GIN_MODE` | `debug` | Gin framework mode. Production sets `release`. |
| `PORT` | `8080` | Server port |
| `ENVIRONMENT` | Unset | Set to `production` to get production behavior outside App Engine: Secure cookies, the production cookie names, and a mandatory `CSRF_SECRET` |
| `SESSION_CACHE_TTL` | `10m` | In-memory session cache duration, as a Go duration such as `5m` or `1h` |
| `SUBSCRIPTION_ENABLED` | `false` | Enables the subscription system (see [feature-flags.md](feature-flags.md)) |
| `INITIAL_ADMIN_EMAILS` | Unset | Comma-separated emails granted admin privileges (see [admin.md](admin.md)). Secret Manager name `initial-admin-emails`. |
| `ADMIN_TOKEN` | Unset | Has two unrelated uses, described below |
| `ACCOUNT_TOMBSTONE_KEY` | Unset | HMAC key for deleted-account tombstones (see [security.md](security.md#account-deletion)). Secret Manager name `account-tombstone-key`. |
| `ALLOWED_ORIGIN` | Unset | The single origin allowed to make cross-origin API requests (see [security.md](security.md#cross-origin-requests)) |
| `CSP_REPORT_ONLY` | `false` | Set to `true` to send the Content Security Policy in report-only mode (see [security.md](security.md#content-security-policy)) |
| `CLOUD_TASKS_QUEUE` | `cron-tasks` | Cloud Tasks queue name (see [Async Task Processing](#async-task-processing-cloud-tasks)) |
| `CLOUD_TASKS_LOCATION` | `us-central1` | Cloud Tasks queue location |
| `RATE_LIMIT_REQUESTS_PER_MINUTE` | `120` | Maximum feed fetches per minute sent to a single feed domain |
| `RATE_LIMIT_BURST_SIZE` | `30` | Feed fetches to a single domain allowed in a burst above the per-minute rate |
| `SCHEDULER_UPDATE_WINDOW` | `15m` | Window across which a feed refresh spreads its fetches, as a Go duration |
| `SCHEDULER_MIN_INTERVAL` | `5m` | Minimum time between two fetches of the same feed. It must not exceed `SCHEDULER_UPDATE_WINDOW`. |
| `SCHEDULER_MAX_CONCURRENT` | `10` | Maximum concurrent feed fetches during a refresh |
| `SCHEDULER_CLEANUP_INTERVAL` | `1h` | Interval of the scheduler's background cleanup loop. The server does not start that loop, so the value has no effect. |

`ADMIN_TOKEN` is read in two places that do not share a value:

- **Admin CLI**: `cmd/admin` and `admin.sh` read it from the environment of the shell that runs them and validate it against the admin tokens stored in the database. The value must be a token printed by `create-token` (see [admin.md](admin.md#admin-token-system)).
- **Server outside App Engine**: the cron and task endpoints require an admin session and an `X-Admin-Token` header equal to the server's `ADMIN_TOKEN`, which may be any string. The server also accepts the value from the Secret Manager secret `admin-token`. If the value is unset, these endpoints reject every request. On App Engine the server never reads it, because the endpoints check the `X-Appengine-Cron` and `X-AppEngine-QueueName` headers instead.

`make validate-config` also reads `VALIDATE_STRICT`. Set to `true`, it makes missing OAuth credentials and a missing `GOOGLE_CLOUD_PROJECT` errors, as a check before a production deployment.

### Stripe Variables

The server needs these when `SUBSCRIPTION_ENABLED=true`, and while any subscription is still live (see [feature-flags.md](feature-flags.md#when-subscriptions-are-disabled)).

| Variable | Secret Manager name | Description |
|----------|---------------------|-------------|
| `STRIPE_SECRET_KEY` | `stripe-secret-key` | Stripe secret key for API calls |
| `STRIPE_PUBLISHABLE_KEY` | `stripe-publishable-key` | Stripe publishable key for the frontend |
| `STRIPE_WEBHOOK_SECRET` | `stripe-webhook-secret` | Webhook signing secret for signature verification |
| `STRIPE_PRICE_ID` | `stripe-price-id` | Stripe price ID for the Pro subscription |

## Security Considerations

### Authentication Security

- **OAuth 2.0**: Industry-standard Google OAuth integration
- **Session security**: HTTP-only cookies with secure flags
- **CSRF protection**: Built-in protection against cross-site requests
- **No password storage**: Leverages Google's authentication infrastructure

### Data Isolation

- **User separation**: Complete isolation of user data in database
- **Query filtering**: All database queries filtered by authenticated user ID
- **Session management**: Secure session creation, validation, and cleanup
- **API protection**: All endpoints require valid authentication

### Production Security

```yaml
# app.yaml security headers
handlers:
- url: /.*
  script: auto
  secure: always
  http_headers:
    Strict-Transport-Security: "max-age=31536000; includeSubDomains"
    X-Content-Type-Options: "nosniff"
    X-Frame-Options: "DENY"
    X-XSS-Protection: "1; mode=block"
```

## Monitoring and Maintenance

### Health Checks

```bash
# App Engine logs
gcloud app logs tail -s default
```

### Performance Optimization

1. **Caching**: Session, feed, and static asset caching
2. **Database optimization**: Proper indexes and connection pooling
3. **Scaling**: Horizontal scaling with load balancers
4. **Cleanup**: Regular cleanup of expired sessions and old articles

## Testing in Production

### Build and Test Locally

```bash
# Run complete build and test suite
make all

# Run tests only
make test

# Validate configuration
make validate-config
```

### Deployment Testing

```bash
# Test OAuth flow
curl -I https://your-domain.com/auth/login

# Test API endpoints (requires authentication)
curl -H "Cookie: session=..." https://your-domain.com/api/feeds
```

### Load Testing

```bash
# Install and run load testing
npm install -g artillery
artillery quick --count 10 --num 50 https://your-domain.com
```

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md#deployment-issues) for OAuth, session, database, and App Engine deployment issues.

## Cost Optimization

- **Instance management**: Use automatic scaling with min 0 instances
- **Async cron processing**: Cron jobs dispatch through Cloud Tasks rather than running in the cron request itself, so instances aren't held open for the duration of a feed refresh; see [Async Task Processing](#async-task-processing-cloud-tasks)
- **Datastore usage**: Monitor read/write operations
- **Bandwidth**: Cache RSS feeds to reduce external requests
- **Free tier**: Leverage GAE free tier limits

See [performance.md](performance.md) for the cost-driven optimizations already implemented.

## Related Documentation

- [Stripe Setup](stripe.md) - Configuring subscription payments
- [iOS App](ios.md) - Local development and on-device testing for the native client
- [Apple Release Guide](apple-releases.md) - TestFlight and notarized DMG releases
- [Monitoring Setup](monitoring.md) - Dashboards, alerts, and cost tracking
- [Security Guidelines](security.md) - Security controls and best practices
- [Troubleshooting Guide](troubleshooting.md) - Common issues