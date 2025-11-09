# Cloud Monitoring Configuration

This directory contains Cloud Monitoring dashboards and alerting policies for GoRead2 cost tracking.

## Files

- `cost-dashboard.json` - Dashboard configuration showing Datastore, App Engine, and bandwidth metrics
- `alerting-policies.json` - Alerting policy definitions for cost spike detection
- `deploy-dashboard.sh` - Script to deploy the dashboard to Cloud Monitoring
- `deploy-alerts.sh` - Script to deploy alerting policies to Cloud Monitoring

## Quick Start

```bash
# Deploy everything
make deploy-monitoring

# Or deploy individually
make deploy-monitoring-dashboard
make deploy-monitoring-alerts
```

## Manual Deployment

```bash
# Set your GCP project
gcloud config set project YOUR_PROJECT_ID

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
