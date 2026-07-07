#!/usr/bin/env bash
# Fails (exit 1) if any benchmark's sec/op regressed by more than THRESHOLD
# percent vs BASELINE, per benchstat's own significance test (a change
# benchstat marks "~" is noise, not a regression, and is ignored regardless
# of the raw percentage). Only the sec/op table is checked — B/op and
# allocs/op deltas are informational only, printed but not gated on.
#
# Usage: check-benchmark-regression.sh <baseline-file> <current-file> [threshold-pct]
set -euo pipefail

BASELINE="${1:?usage: check-benchmark-regression.sh <baseline> <current> [threshold_pct]}"
CURRENT="${2:?usage: check-benchmark-regression.sh <baseline> <current> [threshold_pct]}"
THRESHOLD="${3:-20}"
BENCHSTAT_BIN="${BENCHSTAT_BIN:-benchstat}"

echo "=== benchstat: ${BASELINE} vs ${CURRENT} (regression threshold: ${THRESHOLD}%) ==="
"$BENCHSTAT_BIN" "$BASELINE" "$CURRENT"

csv="$("$BENCHSTAT_BIN" -format csv "$BASELINE" "$CURRENT" 2>/dev/null)"

# The CSV output has one blank-line-separated table per metric (sec/op,
# B/op, allocs/op in that order); only the first (sec/op) matters here.
sec_table="$(awk 'BEGIN{RS=""} NR==1' <<<"$csv")"

regressed="$(awk -F, -v threshold="$THRESHOLD" '
    NR <= 2 { next }                 # skip the two header rows
    $1 == "" || $1 == "geomean" { next }
    {
        vs = $6
        gsub(/^[ \t]+|[ \t]+$/, "", vs)
        if (vs == "~" || vs == "") next
        pct = vs
        gsub(/[+%]/, "", pct)
        if (pct + 0 > threshold + 0) {
            print "REGRESSION: " $1 " is " vs " slower than baseline (threshold " threshold "%)"
        }
    }
' <<<"$sec_table")"

if [[ -n "$regressed" ]]; then
    echo ""
    echo "$regressed"
    exit 1
fi

echo ""
echo "No benchmark regressed more than ${THRESHOLD}% vs baseline."
