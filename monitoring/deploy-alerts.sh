#!/bin/bash
# Deploy Cloud Monitoring alerting policies for GoRead2 cost tracking

set -e
set -o pipefail

POLICIES_FILE="monitoring/alert-policies.yaml"

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

# gcloud alpha monitoring policies create only accepts one policy per
# invocation, but $POLICIES_FILE is a multi-document YAML file (policies
# separated by "---"). Split it into one temp file per policy first.
SPLIT_DIR=$(mktemp -d)
trap 'rm -rf "$SPLIT_DIR"' EXIT

awk -v dir="$SPLIT_DIR" '
  /^---[[:space:]]*$/ { doc++; next }
  doc > 0 { print > (dir "/policy-" doc ".yaml") }
' "$POLICIES_FILE"

POLICY_FILES=("$SPLIT_DIR"/policy-*.yaml)

if [ ! -e "${POLICY_FILES[0]}" ]; then
    echo "❌ Error: No policies found in $POLICIES_FILE"
    exit 1
fi

echo "Creating alerting policies..."

FAILURES=0

for policy_file in "${POLICY_FILES[@]}"; do
    POLICY_NAME=$(grep -m1 '^displayName:' "$policy_file" | sed -E 's/^displayName:[[:space:]]*"?([^"]*)"?$/\1/')
    echo "  Creating policy: $POLICY_NAME"

    if gcloud alpha monitoring policies create --policy-from-file="$policy_file" 2>&1 | tee /tmp/alert-deploy.log; then
        echo "  ✅ Created: $POLICY_NAME"
    else
        if grep -q "already exists" /tmp/alert-deploy.log; then
            echo "  ⚠️  Policy already exists: $POLICY_NAME"
        else
            echo "  ❌ Failed to create: $POLICY_NAME"
            cat /tmp/alert-deploy.log
            FAILURES=$((FAILURES + 1))
        fi
    fi
done

if [ "$FAILURES" -gt 0 ]; then
    echo ""
    echo "❌ $FAILURES polic$([ "$FAILURES" -eq 1 ] && echo y || echo ies) failed to deploy"
    rm -f /tmp/alert-deploy.log
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
