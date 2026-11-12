# Cloud Monitoring Configuration

This directory holds the Cloud Monitoring dashboards and alerting policies for GoRead2. See [docs/monitoring.md](../docs/monitoring.md) for what each one tracks, the alert thresholds, and how to deploy or update them.

## Files

- `dashboard-cost-tracking.json` - Dashboard with the 3 metrics App Engine Standard exposes: instance count, network egress, and p95 latency
- `dashboard-usage.json` - Usage dashboard charting the `custom.googleapis.com/goread2/` metrics recorded daily by `/cron/record-usage-metrics` (see [docs/monitoring.md](../docs/monitoring.md#usage-metrics))
- `alert-policies.yaml` - The eight alerting policy definitions (see [docs/monitoring.md](../docs/monitoring.md#alerting-policies))
- `deploy-dashboard.sh` - Deploys a dashboard; takes the dashboard file as an optional argument, defaulting to `dashboard-cost-tracking.json`
- `deploy-alerts.sh` - Deploys every policy in `alert-policies.yaml`, splitting it into one `gcloud` call per policy

```bash
make deploy-monitoring             # everything below
make deploy-monitoring-dashboard
make deploy-monitoring-usage-dashboard
make deploy-monitoring-alerts
```
