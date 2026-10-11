#!/usr/bin/env bash
# Wait until <base-url>/health reports <version>, i.e. the new build is live.
# Usage: scripts/wait_for_version.sh <base-url> <version> <timeout-seconds>
set -euo pipefail

base_url="${1%/}"
want="$2"
timeout="${3:-900}"
deadline=$(( $(date +%s) + timeout ))

while true; do
  body="$(curl -fsS --max-time 10 "$base_url/health" 2>/dev/null || true)"
  version="$(echo "$body" | jq -r '.version // empty' 2>/dev/null || true)"
  status="$(echo "$body" | jq -r '.status // empty' 2>/dev/null || true)"
  if [[ "$version" == "$want" && "$status" == "OK" ]]; then
    echo "live: version $version, database reachable"
    exit 0
  fi
  if (( $(date +%s) > deadline )); then
    echo "::error::$base_url did not report version $want within ${timeout}s (last: ${body:-no response})"
    exit 1
  fi
  echo "waiting: live version '${version:-none}', status '${status:-none}'"
  sleep 15
done
