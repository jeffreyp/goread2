#!/bin/bash
# Deploy Cloud Monitoring alerting policies for GoRead2 cost tracking

set -e

POLICIES_FILE="monitoring/alerting-policies.json"

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

echo "🚀 Deploying alerting policies to project: $PROJECT_ID"

# Check if policies file exists
if [ ! -f "$POLICIES_FILE" ]; then
    echo "❌ Error: Policies file not found: $POLICIES_FILE"
    exit 1
fi

# Check if jq is installed (needed for JSON processing)
if ! command -v jq &> /dev/null; then
    echo "⚠️  Warning: jq is not installed. Installing policies will be done as a batch."
    echo "To install jq: brew install jq (macOS) or apt-get install jq (Linux)"
fi

# Read the policies array and create each one
echo "Creating alerting policies..."
POLICY_COUNT=0

if command -v jq &> /dev/null; then
    # Use jq to process each policy individually
    jq -c '.policies[]' "$POLICIES_FILE" | while read -r policy; do
        POLICY_COUNT=$((POLICY_COUNT + 1))
        POLICY_NAME=$(echo "$policy" | jq -r '.displayName')
        echo "  Creating policy: $POLICY_NAME"

        # Write individual policy to temp file
        echo "$policy" > /tmp/policy-temp.json

        # Create the policy
        if gcloud alpha monitoring policies create --policy-from-file=/tmp/policy-temp.json 2>&1 | tee /tmp/alert-deploy.log; then
            echo "  ✅ Created: $POLICY_NAME"
        else
            if grep -q "already exists" /tmp/alert-deploy.log; then
                echo "  ⚠️  Policy already exists: $POLICY_NAME"
            else
                echo "  ❌ Failed to create: $POLICY_NAME"
                cat /tmp/alert-deploy.log
            fi
        fi

        rm -f /tmp/policy-temp.json
    done
else
    echo "❌ Error: jq is required to deploy individual policies"
    echo "Please install jq first."
    exit 1
fi

echo ""
echo "✅ Alerting policies deployment complete!"
echo ""
echo "View your alerting policies at:"
echo "https://console.cloud.google.com/monitoring/alerting/policies?project=$PROJECT_ID"
echo ""
echo "Note: You may need to configure notification channels for these alerts:"
echo "https://console.cloud.google.com/monitoring/alerting/notifications?project=$PROJECT_ID"

rm -f /tmp/alert-deploy.log
