# Cloud Monitoring Configuration

This directory contains Cloud Monitoring dashboards and alerting policies for GoRead2 cost tracking.

## Files

- `dashboard-cost-tracking.json` - Dashboard configuration with 10 widgets showing Datastore, App Engine, and bandwidth metrics
- `alert-policies.yaml` - Six alerting policy definitions for cost spike and error detection
- `cost-dashboard.json` - (Legacy) Original dashboard configuration
- `alerting-policies.json` - (Legacy) Original alerting policies
- `deploy-dashboard.sh` - Script to deploy the dashboard to Cloud Monitoring (if exists)
- `deploy-alerts.sh` - Script to deploy alerting policies to Cloud Monitoring (if exists)

## Quick Start

```bash
# Deploy everything
make deploy-monitoring

# Or deploy individually
make deploy-monitoring-dashboard
make deploy-monitoring-alerts
```

## Manual Deployment

### Using gcloud CLI (Recommended)

```bash
# Set your GCP project
gcloud config set project YOUR_PROJECT_ID

# Deploy dashboard
gcloud monitoring dashboards create --config-from-file=monitoring/dashboard-cost-tracking.json

# Deploy alerts (after setting up notification channels)
# Note: YAML file contains multiple policies separated by ---
# You may need to deploy each policy separately
gcloud alpha monitoring policies create --policy-from-file=monitoring/alert-policies.yaml
```

### Using deployment scripts (if available)

```bash
# Deploy dashboard
./monitoring/deploy-dashboard.sh

# Deploy alerts (requires jq)
./monitoring/deploy-alerts.sh
```

## Prerequisites

- `gcloud` CLI installed and authenticated
- Appropriate IAM permissions:
  - `roles/monitoring.dashboardEditor`
  - `roles/monitoring.alertPolicyEditor`
- `jq` installed (for alert deployment)

## Documentation

See [docs/MONITORING.md](../docs/MONITORING.md) for complete documentation including:
- Dashboard metrics explanation
- Alert threshold details
- Updating and maintaining monitoring resources
- Cost information (spoiler: it's free!)
