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

## Dashboard

The cost tracking dashboard provides visualizations for:

### Datastore Metrics
- **Read Operations**: Number of Datastore Lookup operations per second
- **Write Operations**: Number of Put and Commit operations per second
- **Read Bytes**: Volume of data read from Datastore
- **Write Bytes**: Volume of data written to Datastore
- **Operations by Type**: Breakdown of all Datastore API methods

### App Engine Metrics
- **Instance Count**: Number of running instances (by state)
- **Request Count**: HTTP requests per second
- **Response Bytes**: Outbound bandwidth usage
- **Response Latency**: 95th percentile response time

## Alerting Policies

Five alerting policies monitor for cost spikes and unusual patterns:

### 1. High Datastore Read Operations
- **Threshold**: > 100 reads/second for 5 minutes
- **Purpose**: Detect inefficient queries or unexpected traffic

### 2. High Datastore Write Operations
- **Threshold**: > 50 writes/second for 5 minutes
- **Purpose**: Identify write-heavy workloads or potential issues

### 3. High App Engine Instance Count
- **Threshold**: > 8 instances for 10 minutes
- **Purpose**: Catch autoscaling issues (max configured is 10)

### 4. Sudden Increase in Datastore Operations
- **Threshold**: > 50% increase compared to previous hour
- **Purpose**: Catch unexpected spikes in usage

### 5. High Bandwidth Usage
- **Threshold**: > 10 MB/second for 5 minutes
- **Purpose**: Detect large responses or potential issues

## Deployment

### Prerequisites
- `gcloud` CLI installed and configured
- Appropriate permissions on the GCP project
- `jq` installed (for alert deployment)

### Deploy Dashboard

```bash
# Set your project
gcloud config set project YOUR_PROJECT_ID

# Deploy the dashboard
./monitoring/deploy-dashboard.sh
```

The dashboard will be created in Cloud Monitoring. If it already exists, you'll need to update it manually or delete and recreate it.

### Deploy Alerting Policies

```bash
# Deploy all alerting policies
./monitoring/deploy-alerts.sh
```

**Note**: After deploying alerts, you must configure notification channels (email, SMS, Slack, etc.) in the Cloud Console:
https://console.cloud.google.com/monitoring/alerting/notifications

### Makefile Targets

```bash
# Deploy dashboard
make deploy-monitoring-dashboard

# Deploy alerting policies
make deploy-monitoring-alerts

# Deploy both
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
