#!/bin/bash
# Deploy Cloud Monitoring dashboard for GoRead2 cost tracking

set -e

DASHBOARD_FILE="monitoring/cost-dashboard.json"

# Check if gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo "❌ Error: gcloud CLI is not installed"
    echo "Please install it from: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Get project ID from gcloud config
PROJECT_ID=$(gcloud config get-value project 2>/dev/null)

if [ -z "$PROJECT_ID" ]; then
    echo "❌ Error: No project ID configured"
    echo "Please run: gcloud config set project YOUR_PROJECT_ID"
    exit 1
fi

echo "🚀 Deploying Cloud Monitoring dashboard to project: $PROJECT_ID"

# Check if dashboard file exists
if [ ! -f "$DASHBOARD_FILE" ]; then
    echo "❌ Error: Dashboard file not found: $DASHBOARD_FILE"
    exit 1
fi

# Create or update the dashboard
echo "Creating dashboard..."
gcloud monitoring dashboards create --config-from-file="$DASHBOARD_FILE" 2>&1 | tee /tmp/dashboard-deploy.log

# Check if dashboard already exists (creation failed)
if grep -q "already exists" /tmp/dashboard-deploy.log; then
    echo "Dashboard already exists. To update it, you need to:"
    echo "1. Find the dashboard ID from the Cloud Console"
    echo "2. Run: gcloud monitoring dashboards update DASHBOARD_ID --config-from-file=$DASHBOARD_FILE"
    echo ""
    echo "Or delete the existing dashboard first:"
    echo "  gcloud monitoring dashboards list"
    echo "  gcloud monitoring dashboards delete DASHBOARD_ID"
    exit 1
fi

echo "✅ Dashboard deployed successfully!"
echo ""
echo "View your dashboard at:"
echo "https://console.cloud.google.com/monitoring/dashboards?project=$PROJECT_ID"

rm -f /tmp/dashboard-deploy.log
