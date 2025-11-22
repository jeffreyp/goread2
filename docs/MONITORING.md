# Cloud Monitoring Setup

This document describes the Cloud Monitoring infrastructure for tracking GoRead2 costs and performance metrics.

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

### Cost Tracking Dashboard (`monitoring/dashboard-cost-tracking.json`) - RECOMMENDED

This dashboard tracks operational metrics that drive costs:

**Note**: App Engine Standard environment has limited HTTP metrics available. Request count and response code metrics are not available through Cloud Monitoring for App Engine Standard runtime.

#### Dashboard Widgets

1. **Datastore Read Operations** - Read operations per second (cost: $0.06 per 100K entities)
2. **Datastore Write Operations** - Write operations per second (cost: $0.18 per 100K entities)
3. **App Engine Instance Hours** - Number of running instances (instance-hours = primary cost driver)
4. **Network Egress** - Outbound bandwidth usage (cost: $0.12/GB beyond free tier)
5. **Datastore Read/Write Entities** - Total entity counts
6. **Request Latency (p95)** - Performance indicator
7. **Datastore Operations by Type** - Breakdown of all operation types
8. **Datastore Operations Summary** - Overall Datastore activity

**Deploy:**
```bash
gcloud monitoring dashboards create --config-from-file=monitoring/dashboard-cost-tracking.json
```

### Legacy Dashboard (`monitoring/cost-dashboard.json`)

Older dashboard configuration with similar metrics. Use `dashboard-cost-tracking.json` instead for the latest version.

## Viewing Actual Billing Costs

Since billing metrics are not available in Cloud Monitoring dashboards, use these methods to track actual costs:

### Option 1: GCP Billing Reports (Recommended)

Go to: https://console.cloud.google.com/billing/reports

This shows:
- Daily/monthly cost trends
- Cost breakdown by service (App Engine, Datastore, etc.)
- Cost breakdown by SKU
- Forecasted costs

Filter by specific services:
- **App Engine**: Instance hours, bandwidth
- **Cloud Datastore**: Entity reads, writes, deletes, storage

### Option 2: BigQuery Billing Export (Advanced)

For programmatic access to billing data:
1. Enable [Cloud Billing export to BigQuery](https://cloud.google.com/billing/docs/how-to/export-data-bigquery)
2. Query billing data using SQL
3. Create custom dashboards with Data Studio

### Option 3: Cloud Console Datastore Stats

For Datastore-specific metrics: https://console.cloud.google.com/datastore/stats

- Entity counts by kind
- Storage size
- Index sizes
- **Note**: Stats update once per day

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
- **Auto-close**: 15 minutes
- **Critical**: May indicate production outage

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

#### Step 1: Set Up Notification Channels

Before deploying alerts, create notification channels to receive alerts:

```bash
# Email notification
gcloud alpha monitoring channels create \
  --display-name="GoRead2 Alerts Email" \
  --type=email \
  --channel-labels=email_address=your-email@example.com

# Slack notification (recommended)
gcloud alpha monitoring channels create \
  --display-name="GoRead2 Alerts Slack" \
  --type=slack \
  --channel-labels=url=https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# List channels to get IDs
gcloud alpha monitoring channels list
```

#### Step 2: Configure Alert Policies

Edit `monitoring/alert-policies.yaml` and add your notification channel IDs to the `notificationChannels` array in each alert policy.

#### Step 3: Deploy Alerts

```bash
# Install alpha components if needed
gcloud components install alpha

# Deploy alerts
gcloud alpha monitoring policies create --policy-from-file=monitoring/alert-policies.yaml
```

**Note**: The YAML file contains multiple alert policies separated by `---`. You may need to split them into separate files or deploy them individually.

### Alternative: Use Deployment Scripts

If you have existing deployment scripts:

```bash
# Deploy dashboard (if script exists)
./monitoring/deploy-dashboard.sh

# Deploy alerting policies (if script exists)
./monitoring/deploy-alerts.sh
```

### Makefile Targets

```bash
# Deploy dashboard (if Makefile target exists)
make deploy-monitoring-dashboard

# Deploy alerting policies (if Makefile target exists)
make deploy-monitoring-alerts

# Deploy both (if Makefile target exists)
make deploy-monitoring
```

## Viewing Metrics

### Cloud Console
- **Dashboards**: https://console.cloud.google.com/monitoring/dashboards
- **Alerting Policies**: https://console.cloud.google.com/monitoring/alerting/policies
- **Metrics Explorer**: https://console.cloud.google.com/monitoring/metrics-explorer

### Updating Thresholds

The alert thresholds are configured for moderate usage. You may need to adjust them based on your actual traffic patterns:

1. Monitor the dashboard for a few days to establish baseline metrics
2. Edit `monitoring/alerting-policies.json` to update thresholds
3. Redeploy with `./monitoring/deploy-alerts.sh`

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
- Instance hours vary by instance class (configured: 1 CPU, 0.5GB RAM)
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
To update alerting policies:

1. Edit `monitoring/alerting-policies.json`
2. List existing policies to find the one to update:
   ```bash
   gcloud alpha monitoring policies list
   ```
3. Delete the old policy:
   ```bash
   gcloud alpha monitoring policies delete POLICY_ID
   ```
4. Redeploy:
   ```bash
   ./monitoring/deploy-alerts.sh
   ```

## Troubleshooting

### Dashboard Not Showing Data
- Verify that the app is deployed and receiving traffic
- Check that metrics are being generated in Metrics Explorer
- Datastore metrics may take a few minutes to appear after deployment

### Alerts Not Firing
- Confirm notification channels are configured
- Check alert policy status in Cloud Console
- Verify that thresholds are being exceeded using Metrics Explorer

### Permission Errors
Required IAM roles:
- `roles/monitoring.dashboardEditor` - Create/edit dashboards
- `roles/monitoring.alertPolicyEditor` - Create/edit alerting policies

## Related Documentation

- [Performance Optimization](PERFORMANCE.md) - Performance tuning based on metrics
- [Deployment](DEPLOYMENT.md) - App deployment process
- [Google Cloud Monitoring Docs](https://cloud.google.com/monitoring/docs)
