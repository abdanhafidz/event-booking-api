#!/usr/bin/env bash
# Wait until a fixture ticket type is back at full stock. PT-05 leaves its
# bookings unpaid, so the sweeper releases them when the 10-minute hold ends;
# the next run waits for that instead of re-seeding.
# Usage: scripts/perf/wait_for_stock.sh <base-url> <event.properties> <fixture-key> <timeout-seconds>
set -euo pipefail

base="${1%/}"
props="$2"
key="$3"
timeout="${4:-720}"
prop() { grep "^$1=" "$props" | cut -d= -f2 | tr -d '\r'; }

event_id="$(prop "${key}_event_id")"
ticket_type="$(prop "${key}_ticket_type_id")"
want="$(prop "${key}_tickets")"
deadline=$(( $(date +%s) + timeout ))

while true; do
  available="$(curl -fsS --max-time 10 "$base/v1/events/$event_id" \
    | jq -r --arg tt "$ticket_type" '.data.ticket_types[] | select(.id == $tt) | .available' || echo "?")"
  if [[ "$available" == "$want" ]]; then
    echo "$key stock is full: $available/$want"
    exit 0
  fi
  if (( $(date +%s) > deadline )); then
    echo "::error::$key stock still $available/$want after ${timeout}s; earlier holds were not released"
    exit 1
  fi
  echo "waiting for $key stock: $available/$want available"
  sleep 20
done
