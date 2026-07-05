#!/bin/bash
#
# Post-deploy unauthenticated smoke check.
#
# Verifies a freshly-deployed GoRead2 App Engine version is actually serving
# traffic correctly, without needing a logged-in session: the app started,
# static assets built, OAuth config loaded, security headers are present,
# and no backdoor auth endpoint got left enabled.
#
# Usage: scripts/smoke-check.sh <base-url>
#   e.g. scripts/smoke-check.sh https://staging-a1b2c3d-dot-goread-467200.uc.r.appspot.com

set -u

BASE_URL="${1:-}"
if [ -z "$BASE_URL" ]; then
  echo "Usage: $0 <base-url>" >&2
  exit 2
fi

# Strip any trailing slash so path concatenation below doesn't double up.
BASE_URL="${BASE_URL%/}"

FAILURES=0

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

pass() {
  echo -e "${GREEN}✓${NC} $1"
}

fail() {
  echo -e "${RED}✗${NC} $1" >&2
  FAILURES=$((FAILURES + 1))
}

# fetch <path> — GETs $BASE_URL<path> without following redirects, capturing
# headers and body into HEADERS_FILE/BODY_FILE and status into STATUS.
HEADERS_FILE="$(mktemp)"
BODY_FILE="$(mktemp)"
trap 'rm -f "$HEADERS_FILE" "$BODY_FILE"' EXIT

fetch() {
  local path="$1"
  STATUS=$(curl -s -o "$BODY_FILE" -D "$HEADERS_FILE" -w '%{http_code}' \
    --max-time 15 "${BASE_URL}${path}")
}

check_status() {
  local path="$1" expected="$2"
  if [ "$STATUS" = "$expected" ]; then
    pass "GET $path -> $STATUS"
  else
    fail "GET $path -> expected $expected, got $STATUS"
  fi
}

header_present() {
  local path="$1" header="$2"
  if grep -qi "^${header}:" "$HEADERS_FILE"; then
    pass "GET $path has header '$header'"
  else
    fail "GET $path missing header '$header'"
  fi
}

# GET / returns 200 with 'GoRead' in the body
fetch "/"
check_status "/" 200
if grep -q "GoRead" "$BODY_FILE"; then
  pass "GET / body contains 'GoRead'"
else
  fail "GET / body does not contain 'GoRead'"
fi

# GET /privacy returns 200
fetch "/privacy"
check_status "/privacy" 200

# GET /api/feeds returns 401 with JSON error body
fetch "/api/feeds"
check_status "/api/feeds" 401
if grep -qi '"error"' "$BODY_FILE"; then
  pass "GET /api/feeds body contains a JSON error"
else
  fail "GET /api/feeds body does not look like a JSON error: $(cat "$BODY_FILE")"
fi

# GET /auth/login returns 200 with a JSON auth_url pointing at
# accounts.google.com. (The handler hands the URL to the frontend to
# redirect the browser itself, rather than issuing a server-side 302.)
fetch "/auth/login"
check_status "/auth/login" 200
if grep -qi "accounts\.google\.com" "$BODY_FILE"; then
  pass "GET /auth/login auth_url points at accounts.google.com"
else
  fail "GET /auth/login body does not contain an accounts.google.com auth_url"
fi

# GET /static/js/app.min.js returns 200 with Content-Type application/javascript
fetch "/static/js/app.min.js"
check_status "/static/js/app.min.js" 200
if grep -qi "^content-type:.*javascript" "$HEADERS_FILE"; then
  pass "GET /static/js/app.min.js has javascript Content-Type"
else
  fail "GET /static/js/app.min.js Content-Type is not application/javascript"
fi

# GET /static/css/styles.min.css returns 200
fetch "/static/css/styles.min.css"
check_status "/static/css/styles.min.css" 200

# Strict-Transport-Security and X-Content-Type-Options headers (checked
# against / — set by middleware on every response)
fetch "/"
header_present "/" "Strict-Transport-Security"
header_present "/" "X-Content-Type-Options"

# GET /auth/smoke-login returns 404 (confirms no backdoor endpoint is enabled)
fetch "/auth/smoke-login"
check_status "/auth/smoke-login" 404

echo ""
if [ "$FAILURES" -eq 0 ]; then
  echo "Smoke check passed: ${BASE_URL}"
  exit 0
else
  echo "Smoke check FAILED: ${FAILURES} assertion(s) failed against ${BASE_URL}" >&2
  exit 1
fi
