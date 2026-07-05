#!/bin/bash
#
# Bounded post-promote health watch.
#
# Polls Cloud Monitoring for a fixed window after a production traffic
# migration, watching the same signals as monitoring/alert-policies.yaml
# (5xx rate, instance count, Datastore read/write rate, egress) plus p95
# latency. Exits 1 the moment any signal breaches its threshold for that
# signal's configured duration, so the caller can trigger an auto-rollback
# without waiting out the rest of the window. Exits 0 if the whole window
# passes clean.
#
# Requires the caller to already be authenticated (gcloud auth) as a
# principal with roles/monitoring.viewer.
#
# Usage: scripts/post-promote-health-watch.sh <project-id>
#   env overrides: WATCH_SECONDS (default 900), POLL_INTERVAL_SECONDS (default 60)

set -u

PROJECT="${1:-}"
if [ -z "$PROJECT" ]; then
  echo "Usage: $0 <project-id>" >&2
  exit 2
fi

WATCH_SECONDS="${WATCH_SECONDS:-900}"
POLL_INTERVAL_SECONDS="${POLL_INTERVAL_SECONDS:-60}"

# Thresholds mirrored from monitoring/alert-policies.yaml (same filter,
# aligner, reducer, and duration as the deployed alert policy). p95 latency
# has no corresponding alert policy today; 3000ms is a conservative default
# roughly 5x the app's typical baseline (~600ms) — revisit if that baseline
# drifts.
declare -A FILTER=(
  [5xx]='resource.type="gae_app" AND metric.type="appengine.googleapis.com/http/server/response_count" AND metric.label.response_code>=500 AND metric.label.response_code<600'
  [latency_p95]='resource.type="gae_app" AND metric.type="appengine.googleapis.com/http/server/response_latencies"'
  [instances]='resource.type="gae_app" AND metric.type="appengine.googleapis.com/system/instance_count"'
  [ds_read]='resource.type="datastore_database" AND metric.type="datastore.googleapis.com/api/request_count" AND metric.label.api_method="Lookup"'
  [ds_write]='resource.type="datastore_database" AND metric.type="datastore.googleapis.com/api/request_count" AND (metric.label.api_method="Put" OR metric.label.api_method="Commit")'
  [egress]='resource.type="gae_app" AND metric.type="appengine.googleapis.com/system/network/sent_bytes_count"'
)
declare -A ALIGNER=(
  [5xx]=ALIGN_RATE
  [latency_p95]=ALIGN_PERCENTILE_95
  [instances]=ALIGN_MEAN
  [ds_read]=ALIGN_RATE
  [ds_write]=ALIGN_RATE
  [egress]=ALIGN_RATE
)
declare -A REDUCER=(
  [5xx]=REDUCE_SUM
  [latency_p95]=REDUCE_MEAN
  [instances]=REDUCE_SUM
  [ds_read]=REDUCE_SUM
  [ds_write]=REDUCE_SUM
  [egress]=REDUCE_SUM
)
declare -A THRESHOLD=(
  [5xx]=0.1
  [latency_p95]=3000
  [instances]=5
  [ds_read]=16.67
  [ds_write]=8.33
  [egress]=1677721.6
)
# duration each policy requires the breach to sustain, from alert-policies.yaml
declare -A DURATION_SECONDS=(
  [5xx]=180
  [latency_p95]=180
  [instances]=600
  [ds_read]=300
  [ds_write]=300
  [egress]=300
)

declare -A BREACH_STREAK=(
  [5xx]=0 [latency_p95]=0 [instances]=0 [ds_read]=0 [ds_write]=0 [egress]=0
)

query_latest() {
  local key="$1"
  # Cloud Monitoring ingestion lags real time by up to a few minutes for
  # App Engine/Datastore metrics, so the lookback window must be wider than
  # the poll interval or every query comes back empty. points[] is newest
  # first, so points[0] is still the freshest data available regardless of
  # how much of the window is empty.
  local end start
  end="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  start="$(date -u -d "@$(( $(date +%s) - 300 ))" +%Y-%m-%dT%H:%M:%SZ)"

  local response
  response=$(curl -s -G "https://monitoring.googleapis.com/v3/projects/${PROJECT}/timeSeries" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    --data-urlencode "filter=${FILTER[$key]}" \
    --data-urlencode "interval.startTime=${start}" \
    --data-urlencode "interval.endTime=${end}" \
    --data-urlencode "aggregation.alignmentPeriod=${POLL_INTERVAL_SECONDS}s" \
    --data-urlencode "aggregation.perSeriesAligner=${ALIGNER[$key]}" \
    --data-urlencode "aggregation.crossSeriesReducer=${REDUCER[$key]}")

  echo "$response" | jq -r '.timeSeries[0].points[0].value.doubleValue // .timeSeries[0].points[0].value.int64Value // empty'
}

elapsed=0
while [ "$elapsed" -lt "$WATCH_SECONDS" ]; do
  ACCESS_TOKEN="$(gcloud auth print-access-token)"

  for key in "${!FILTER[@]}"; do
    value=$(query_latest "$key")
    if [ -z "$value" ]; then
      echo "[health-watch] ${key}: no data point this poll (skipping)"
      continue
    fi

    breached=$(awk -v v="$value" -v t="${THRESHOLD[$key]}" 'BEGIN { print (v > t) ? 1 : 0 }')
    if [ "$breached" = "1" ]; then
      BREACH_STREAK[$key]=$(( BREACH_STREAK[$key] + POLL_INTERVAL_SECONDS ))
      echo "[health-watch] ${key}: ${value} > threshold ${THRESHOLD[$key]} (sustained ${BREACH_STREAK[$key]}s / needs ${DURATION_SECONDS[$key]}s)"
    else
      echo "[health-watch] ${key}: ${value} <= threshold ${THRESHOLD[$key]}, OK"
      BREACH_STREAK[$key]=0
    fi

    if [ "${BREACH_STREAK[$key]}" -ge "${DURATION_SECONDS[$key]}" ]; then
      echo "::error::Health watch breach: ${key} exceeded threshold ${THRESHOLD[$key]} for ${DURATION_SECONDS[$key]}s (latest value: ${value})" >&2
      exit 1
    fi
  done

  sleep "$POLL_INTERVAL_SECONDS"
  elapsed=$(( elapsed + POLL_INTERVAL_SECONDS ))
done

echo "[health-watch] ${WATCH_SECONDS}s window passed clean, no thresholds breached"
exit 0
