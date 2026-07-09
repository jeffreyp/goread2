# Cloud Monitoring Setup

Describes the Cloud Monitoring infrastructure for tracking GoRead2 costs and performance metrics.

## Table of Contents

- [Overview](#overview)
- [Dashboards](#dashboards)
- [Tracking Actual Costs and Usage](#tracking-actual-costs-and-usage)
- [Alerting Policies](#alerting-policies)
- [Post-Promote Health Watch](#post-promote-health-watch-a-second-consumer-of-these-signals)
- [Billing Budget](#billing-budget)
- [Deployment](#deployment)
- [Viewing Metrics](#viewing-metrics)
- [Cost Estimation](#cost-estimation)
- [Maintenance](#maintenance)
- [Troubleshooting](#troubleshooting)
- [Related Documentation](#related-documentation)

## Overview

GoRead2 uses Google Cloud Monitoring to track:
- Datastore read/write operations and data volumes
- App Engine instance counts and resource usage
- Bandwidth consumption
- Request latency and error rates

The monitoring setup helps identify cost optimization opportunities and detect unusual usage patterns early.

### Monitoring Costs

**Good news**: The metrics used in this dashboard are **free** under Google Cloud's monitoring pricing model.

- **Free tier**: First 150 MB of Monitoring data per month (includes all GCP service metrics)
- **GCP service metrics**: Datastore and App Engine metrics are included at no charge
- **Dashboards**: Free to create and view
- **Alerting policies**: First 100 alerting policy evaluations per month are free

The dashboard and alerts use only built-in GCP metrics (Datastore, App Engine), which fall under the free allotment. Custom metrics or extensive API calls would incur costs, but this setup uses neither.

**Estimated cost for this monitoring setup**: $0/month for typical usage.

## Dashboards

**Important Note on Billing Metrics**: GCP billing metrics (`billing.googleapis.com/cost/*`) are **not available** through the Cloud Monitoring Metrics API and cannot be used in custom dashboards. To view actual billing costs, use the [GCP Billing Reports page](https://console.cloud.google.com/billing/reports) in the Cloud Console.

The dashboards below track **operational metrics** (operations, instances, bandwidth) that correlate with costs but do not show actual dollar amounts.

### Cost Tracking Dashboard (`monitoring/dashboard-cost-tracking.json`)

**Important Limitation**: App Engine Standard environment exposes very limited metrics through Cloud Monitoring. Most HTTP and Datastore metrics are **not available**.

This minimal dashboard tracks the only 3 operational metrics available for App Engine Standard:

#### Dashboard Widgets

1. **App Engine Instance Count** - Number of running instances (instance-hours = primary cost driver)
   - Cost: Varies by instance class (F1/F2/F4/F4_1G)
   - Your config: F1, 256MB RAM

2. **Network Egress** - Outbound bandwidth usage
   - Cost: $0.12/GB (after free tier of 1GB/day)

3. **Request Latency (p95)** - 95th percentile response time
   - Performance indicator only, not directly tied to cost

**Metrics NOT Available for App Engine Standard:**
- ❌ Datastore read/write operations
- ❌ Datastore entity counts
- ❌ HTTP request counts
- ❌ HTTP response codes
- ❌ CPU/memory usage

**Deploy:**
```bash
gcloud monitoring dashboards create --config-from-file=monitoring/dashboard-cost-tracking.json
```

### Legacy Dashboard (`monitoring/cost-dashboard.json`)

Older dashboard configuration with similar metrics. Use `dashboard-cost-tracking.json` instead for the latest version.

## Tracking Actual Costs and Usage

Since most operational metrics are not available through Cloud Monitoring for App Engine Standard, use these alternative methods:

### For Billing/Costs

**GCP Billing Reports (Primary Method)**: https://console.cloud.google.com/billing/reports

View:
- Daily/monthly cost trends
- Cost breakdown by service (App Engine, Datastore, etc.)
- Cost breakdown by SKU (instance hours, bandwidth, Datastore operations, storage)
- Forecasted costs

Filter by:
- **App Engine**: Instance hours, outbound bandwidth
- **Cloud Datastore**: Entity reads, writes, deletes, stored data

### For Datastore Usage

**Cloud Console Datastore Stats**: https://console.cloud.google.com/datastore/stats

Shows:
- Entity counts by kind
- Storage size per kind
- Index sizes
- **Note**: Stats update once per day (not real-time)

### For HTTP Request Metrics

Since request count and response code metrics are not available in Cloud Monitoring, use:

**App Engine Logs**:
```bash
# View recent logs
gcloud app logs tail

# View logs for specific time period
gcloud app logs read --limit=100
```

Or use the [Logs Explorer](https://console.cloud.google.com/logs) to:
- Filter by HTTP status codes
- Count requests over time
- View request patterns
- Create log-based metrics (advanced)

## Alerting Policies

The alert configuration (`monitoring/alert-policies.yaml`) includes six policies that monitor for cost spikes and unusual patterns:

### 1. High Datastore Read Operations
- **Threshold**: >1000 reads/minute (~16.67/second) sustained for 5 minutes
- **Purpose**: Detect inefficient queries or traffic spike
- **Actions**: Check query patterns, review caching effectiveness
- **Auto-close**: 30 minutes

### 2. High Datastore Write Operations
- **Threshold**: >500 writes/minute (~8.33/second) sustained for 5 minutes
- **Purpose**: Identify excessive updates or inefficient batch operations
- **Actions**: Review batch operations, check for write loops
- **Auto-close**: 30 minutes

### 3. High App Engine Instance Count
- **Threshold**: >5 instances sustained for 10 minutes
- **Purpose**: Catch traffic spike or inefficient auto-scaling
- **Actions**: Review app.yaml scaling settings, check for slow requests
- **Note**: Max configured is 10 instances
- **Auto-close**: 30 minutes

### 4. High Network Egress (Bandwidth)
- **Threshold**: >100 MB/minute (~1.67 MB/second) sustained for 5 minutes
- **Purpose**: Detect excessive data transfer or potential data leak
- **Actions**: Check RSS feed sizes, review API response payloads
- **Auto-close**: 30 minutes

### 5. Datastore Entity Read Spike
- **Threshold**: >1000 entity reads/hour (baseline, adjust for your usage)
- **Purpose**: Catch query fanout or sudden traffic increase
- **Actions**: Investigate recent code changes, check for N+1 queries
- **Auto-close**: 1 hour
- **Note**: Adjust threshold based on your baseline after monitoring

### 6. High HTTP Error Rate (5xx)
- **Threshold**: >0.1 errors/second (>1% of typical traffic) sustained for 3 minutes
- **Purpose**: Detect application errors or resource exhaustion
- **Actions**: Check application logs immediately, review recent deployments
- **Auto-close**: 30 minutes (GCP enforces a 30-minute minimum; the originally documented 15 minutes was never a valid value and was corrected at deploy time)
- **Critical**: May indicate production outage

## Post-Promote Health Watch (a second consumer of these signals)

`scripts/post-promote-health-watch.sh`, invoked from `deploy-prod.yml` after every production promotion (see [deployment.md](deployment.md#auto-rollback-post-promote-safety-net)), polls the same five signals as policies 1, 2, 3, 4, and 6 above directly via the Cloud Monitoring REST API, plus p95 latency, which has no alert policy of its own. It mirrors each policy's filter, aligner, reducer, threshold, and duration so the two stay in sync; if you change a threshold or duration here, update the script's matching `THRESHOLD`/`DURATION_SECONDS` entries too; they are not read from `alert-policies.yaml` at runtime.

This is a separate mechanism from the alert policies, not a replacement: alert policies page a human at any time via the notification channel below; the health watch only runs in the ~15-minute window right after a promotion and auto-triggers `rollback.yml` on its own. Requires `roles/monitoring.viewer` on `cicd-deploy@goread-467200.iam.gserviceaccount.com` (added 2026-07-05); the alert policies never needed this since they're evaluated by GCP itself, not queried by a workflow.

## Billing Budget

Unlike the alert policies above (which track operational proxies, such as Datastore ops, egress, and instance count, because the Monitoring API can't see actual billing data), a Cloud Billing Budget is a hard dollar-based backstop that Google computes independently from real invoiced spend.

**Current configuration**: `GoRead2 Monthly Budget`, $20/month, calendar-month period, scoped to `projects/goread-467200` only. Alerts fire at 50%, 80%, and 100% of spend via the same email channel used by the operational alerts (`projects/goread-467200/notificationChannels/5854116738752263410`), plus GCP's default IAM recipients (billing account admins).

To adjust the amount or thresholds:
```bash
gcloud billing budgets list --billing-account=016C57-9F50ED-4C680E
gcloud billing budgets update BUDGET_ID --billing-account=016C57-9F50ED-4C680E --budget-amount=NEW_AMOUNTUSD
```

**Note**: the billing account also has a pre-existing `$10 Monthly Budget Alert` (thresholds 50/90/100/150%) that predates this setup and was not created as part of this work. Review whether to keep both or consolidate.

### Customizing Alert Thresholds

The default thresholds are conservative. To adjust:

1. **Monitor for 1-2 weeks** to establish your baseline
2. **Calculate thresholds**: Baseline average + (2 × standard deviation)
3. **Edit** `monitoring/alert-policies.yaml`
4. **Redeploy** alerts using gcloud command

Example threshold calculation:
```
Average reads: 500/minute
Std dev: 100
Threshold: 500 + (2 × 100) = 700 reads/minute
```

## Deployment

### Prerequisites
- `gcloud` CLI installed and configured
- Appropriate permissions on the GCP project (`roles/monitoring.editor` or `roles/monitoring.admin`)
- `jq` installed (optional, for alert deployment via script)

### Deploy Dashboard

#### Option 1: Using gcloud CLI (Recommended)

```bash
# Set your project
gcloud config set project YOUR_PROJECT_ID

# Deploy the working App Engine dashboard
gcloud monitoring dashboards create --config-from-file=monitoring/dashboard-appengine-simple.json

# Note: dashboard-cost-tracking.json contains Datastore metrics that don't work
# with App Engine Standard - use dashboard-appengine-simple.json instead
```

#### Option 2: Via Cloud Console

1. Go to [Cloud Monitoring Dashboards](https://console.cloud.google.com/monitoring/dashboards)
2. Click **Create Dashboard**
3. Click **JSON** in the top right corner
4. Paste the contents of `monitoring/dashboard-appengine-simple.json`
5. Click **Apply**

The dashboard will be created in Cloud Monitoring. If it already exists, update it manually or delete and recreate it.

### Deploy Alerting Policies

**Current state**: all 6 policies are deployed and notify `projects/goread-467200/notificationChannels/5854116738752263410` (email to jeffreyp07@gmail.com). The steps below are for adding a second channel (e.g. Slack) or redeploying from scratch in a new project.

**Unrelated pre-existing alert, disabled**: the project also had a separate, non-repo-tracked policy called "GoRead2 - Sudden Increase in Datastore Operations" (`alertPolicies/4972926877818684428`), likely a GCP console/suggested default predating this setup. It compared live traffic against a *forecasted* baseline and fired on a 100% increase. Since this app's baseline Datastore traffic is close to zero between the every-2-hour feed-refresh cron run, any normal cron burst looked like an "infinite % increase" and triggered it every cycle. It went unnoticed until the notification channel above was attached (2026-07-03), at which point it started emailing every ~2 hours for entirely routine cron activity. Disabled via `gcloud alpha monitoring policies update ... --no-enabled`; the repo's own `Datastore Entity Read Spike` policy (absolute >1000 reads/hour threshold) already covers this signal without the false-positive-prone forecast comparison.

#### Step 1: Set Up Notification Channels

```bash
# Email notification
gcloud alpha monitoring channels create \
  --display-name="GoRead2 Alerts Email" \
  --type=email \
  --channel-labels=email_address=your-email@example.com

# Slack notification (optional, additive)
gcloud alpha monitoring channels create \
  --display-name="GoRead2 Alerts Slack" \
  --type=slack \
  --channel-labels=url=https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# List channels to get IDs
gcloud alpha monitoring channels list
```

#### Step 2: Configure Alert Policies

`monitoring/alert-policies.yaml` already has the email channel ID in every policy's `notificationChannels` array. Add a second channel ID to the array (one per line) if you want alerts to also go to Slack.

#### Step 3: Deploy Alerts

```bash
# Install alpha components if needed
gcloud components install alpha

# gcloud alpha monitoring policies create only accepts ONE policy per invocation;
# the multi-document YAML (separated by ---) must be split into per-policy files first
# and each created individually. To update an already-deployed policy instead of
# creating a duplicate, use:
gcloud alpha monitoring policies update POLICY_NAME --add-notification-channels=CHANNEL_ID
```

**Gotcha hit during initial deploy**: `alertStrategy.autoClose` has a 30-minute (`"1800s"`) minimum. The original "High HTTP Error Rate" policy specified `"900s"` (15 minutes) and was rejected by the API until corrected.

### Alternative: Use Deployment Scripts

```bash
# Deploy dashboard
./monitoring/deploy-dashboard.sh

# Deploy alerting policies
./monitoring/deploy-alerts.sh
```

`monitoring/deploy-alerts.sh` reads `monitoring/alert-policies.yaml`, splits its six `---`-separated documents into per-policy temp files (`gcloud alpha monitoring policies create` only accepts one policy per invocation), and creates each in turn. Policies that already exist are reported and skipped, not updated; use `gcloud alpha monitoring policies update` (see [Step 3](#step-3-deploy-alerts) above) to change an existing policy's configuration.

### Makefile Targets

```bash
make deploy-monitoring-dashboard  # ./monitoring/deploy-dashboard.sh
make deploy-monitoring-alerts     # ./monitoring/deploy-alerts.sh
make deploy-monitoring            # both
```

## Viewing Metrics

### Cloud Console
- **Dashboards**: https://console.cloud.google.com/monitoring/dashboards
- **Alerting Policies**: https://console.cloud.google.com/monitoring/alerting/policies
- **Metrics Explorer**: https://console.cloud.google.com/monitoring/metrics-explorer

### Updating Thresholds

The alert thresholds are configured for moderate usage. You may need to adjust them based on your actual traffic patterns:

1. Monitor the dashboard for a few days to establish baseline metrics
2. Edit `monitoring/alert-policies.yaml` to update thresholds
3. Redeploy the changed policy with `gcloud alpha monitoring policies update POLICY_NAME ...` (see [Step 3](#step-3-deploy-alerts) above); `deploy-alerts.sh` only creates new policies and skips ones that already exist

Recommended adjustments:
- Increase thresholds if you get too many false positives
- Decrease thresholds for stricter monitoring
- Add new alerts for specific metrics that matter to your use case

## Cost Estimation

While the dashboard shows operational metrics, you can estimate costs using:

### Datastore Costs
- Read operations: $0.06 per 100,000 entities
- Write operations: $0.18 per 100,000 entities
- Storage: $0.108 per GB/month

### App Engine Costs
- Instance hours vary by instance class (configured: F1, 256MB RAM)
- Bandwidth: $0.12 per GB (first 1GB free per day)

### Example Calculation
If the dashboard shows:
- 50 Datastore reads/sec = 4.32M reads/day = ~$2.60/day
- 10 Datastore writes/sec = 864K writes/day = ~$1.55/day
- 2 instances running 24/7 = 48 instance-hours/day = ~$2.40/day (approximate)
- Total: ~$6.55/day or ~$197/month

## Maintenance

### Regular Reviews
- Check the dashboard weekly for unusual patterns
- Review alert notifications and adjust thresholds as needed
- Monitor for sustained increases in operations that might indicate optimization opportunities

### Dashboard Updates
To update the dashboard configuration:

1. Edit `monitoring/cost-dashboard.json`
2. Find your dashboard ID:
   ```bash
   gcloud monitoring dashboards list
   ```
3. Update the dashboard:
   ```bash
   gcloud monitoring dashboards update DASHBOARD_ID \
     --config-from-file=monitoring/cost-dashboard.json
   ```

### Alert Policy Updates
To update an existing alerting policy, edit `monitoring/alert-policies.yaml` and push the change with `gcloud alpha monitoring policies update` as described in [Updating Thresholds](#updating-thresholds) above. `monitoring/deploy-alerts.sh` only creates new policies from that file; it skips ones that already exist rather than updating them.

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md#monitoring-issues) for dashboard, alerting, and IAM permission issues.

## Related Documentation

- [Performance Optimization](performance.md) - Performance tuning based on metrics
- [Deployment](deployment.md) - App deployment process
- [Google Cloud Monitoring Docs](https://cloud.google.com/monitoring/docs)
