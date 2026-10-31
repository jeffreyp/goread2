#!/bin/bash
#
# Runs a command against a local Cloud Datastore emulator and always shuts the
# emulator down afterwards, whether the command passes, fails, or is
# interrupted.
#
# Usage: scripts/with-datastore-emulator.sh COMMAND [ARGS...]
#   e.g. scripts/with-datastore-emulator.sh ./test.sh
#        scripts/with-datastore-emulator.sh go test ./internal/database/ -run TestDatastore
#   env: DATASTORE_EMULATOR_PORT (default 8081)
#
# The emulator runs with --no-store-on-disk and --consistency=1.0, the same
# flags CI uses. The command's exit status is passed through.

set -u

if [ "$#" -eq 0 ]; then
  echo "Usage: $0 COMMAND [ARGS...]" >&2
  exit 2
fi

PORT="${DATASTORE_EMULATOR_PORT:-8081}"
HOST="localhost:${PORT}"

if ! command -v gcloud >/dev/null 2>&1; then
  echo "❌ gcloud is not installed; see docs/testing.md (Datastore Emulator)" >&2
  exit 1
fi

# Refuse to share a port with another emulator: reusing it would hide a leaked
# process, and shutting it down would stop something this script did not start.
if listener=$(lsof -nP -iTCP:"$PORT" -sTCP:LISTEN 2>/dev/null | tail -n +2) && [ -n "$listener" ]; then
  echo "❌ Port $PORT is already in use:" >&2
  echo "$listener" >&2
  echo "Stop that process, or set DATASTORE_EMULATOR_PORT to a free port." >&2
  exit 1
fi

# The emulator needs a Java 21+ JRE. Homebrew's openjdk is keg-only, so put it
# on PATH when it is installed.
if command -v brew >/dev/null 2>&1; then
  jdk="$(brew --prefix openjdk 2>/dev/null)/bin"
  [ -x "$jdk/java" ] && PATH="$jdk:$PATH"
fi

LOG="$(mktemp -t datastore-emulator)"
EMULATOR_PID=""

cleanup() {
  status=$?
  trap - EXIT INT TERM
  if [ -n "$EMULATOR_PID" ]; then
    echo "🛑 Stopping Datastore emulator on $HOST"
    curl -s -m 5 -X POST "http://${HOST}/shutdown" >/dev/null 2>&1
    for _ in 1 2 3 4 5 6 7 8 9 10; do
      kill -0 "$EMULATOR_PID" 2>/dev/null || break
      sleep 1
    done
    # Fall back to killing the whole process group: gcloud starts a wrapper
    # script, which starts the Java server.
    if kill -0 "$EMULATOR_PID" 2>/dev/null; then
      kill -TERM -- "-$EMULATOR_PID" 2>/dev/null
      sleep 2
      kill -KILL -- "-$EMULATOR_PID" 2>/dev/null
    fi
    wait "$EMULATOR_PID" 2>/dev/null
  fi
  rm -f "$LOG"
  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Job control gives the background emulator its own process group, so cleanup
# can signal gcloud, its wrapper, and the Java server together.
set -m
gcloud beta emulators datastore start \
  --project=goread2-local-test \
  --host-port="$HOST" \
  --no-store-on-disk \
  --consistency=1.0 >"$LOG" 2>&1 &
EMULATOR_PID=$!
set +m

echo "⏳ Starting Datastore emulator on $HOST"
ready=""
for _ in $(seq 1 60); do
  if ! kill -0 "$EMULATOR_PID" 2>/dev/null; then
    echo "❌ Datastore emulator exited during startup:" >&2
    cat "$LOG" >&2
    exit 1
  fi
  if [ "$(curl -s -m 2 "http://${HOST}/" 2>/dev/null)" = "Ok" ]; then
    ready=1
    break
  fi
  sleep 1
done
if [ -z "$ready" ]; then
  echo "❌ Datastore emulator did not respond on $HOST within 60s:" >&2
  cat "$LOG" >&2
  exit 1
fi
echo "✅ Datastore emulator ready on $HOST"

export DATASTORE_EMULATOR_HOST="$HOST"
"$@"
