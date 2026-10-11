#!/usr/bin/env bash
# Smoke test driven by the deployed OpenAPI document.
#   1. Every path and method in /swagger/doc.json is routed (no ROUTE_NOT_FOUND, no 5xx).
#   2. A journey on the fixed smoke fixture (5 tickets) behaves as documented.
#      Everything it books is released again, so the fixture is reusable and
#      never needs re-seeding.
# Usage: scripts/smoke_test.sh <base-url> <dir from "seed -export">
set -uo pipefail

base="${1%/}"
seed_dir="$2"
report="$seed_dir/smoke-results.md"
failures=0

prop() { grep "^$1=" "$seed_dir/event.properties" | cut -d= -f2 | tr -d '\r'; }
event_id="$(prop smoke_event_id)"
ticket_type="$(prop smoke_ticket_type_id)"
user="$(head -1 "$seed_dir/users.csv" | tr -d '\r')"

echo "| Check | Expected | Got | Result |" > "$report"
echo "| --- | --- | --- | --- |" >> "$report"

record() { # name expected got
  local result="pass"
  if [[ "$2" != "$3" ]]; then result="**FAIL**"; failures=$((failures + 1)); fi
  echo "| $1 | $2 | $3 | $result |" >> "$report"
  echo "$result  $1: expected $2, got $3"
}

# call METHOD PATH [IDEMPOTENCY_KEY] [BODY] -> sets CODE, BODY, HEADERS
call() {
  local args=(-s -o /tmp/smoke_body -D /tmp/smoke_headers -w '%{http_code}' -X "$1" "$base$2"
              -H "X-User-Id: $user" -H "Content-Type: application/json")
  [[ -n "${3:-}" ]] && args+=(-H "Idempotency-Key: $3")
  [[ -n "${4:-}" ]] && args+=(-d "$4")
  CODE="$(curl "${args[@]}")"
  BODY="$(cat /tmp/smoke_body)"
  HEADERS="$(cat /tmp/smoke_headers)"
}
field() { echo "$BODY" | jq -r "$1"; }

# ---------- 1. health and OpenAPI coverage ----------
call GET /health
record "GET /health" "200 OK" "$CODE $(field .status)"

spec="$(curl -fsS "$base/swagger/doc.json")" || { echo "::error::cannot fetch /swagger/doc.json"; exit 1; }
while IFS=$'\t' read -r method path; do
  concrete="${path//\{id\}/00000000-0000-0000-0000-000000000000}"
  call "${method^^}" "$concrete" "smoke-route-$RANDOM$RANDOM" '{}'
  code="$(field '.code // empty')"
  routed="routed"
  if [[ "$code" == "ROUTE_NOT_FOUND" || "$CODE" -ge 500 ]]; then routed="$CODE $code"; fi
  record "${method^^} $path is routed" "routed" "$routed"
done < <(echo "$spec" | jq -r '.paths | to_entries[] | .key as $p | .value | keys[] | [., $p] | @tsv')

# ---------- 2. journey on the smoke fixture ----------
available_now() {
  curl -fsS "$base/v1/events/$event_id" | jq -r --arg tt "$ticket_type" '.data.ticket_types[] | select(.id == $tt) | .available'
}
call GET "/v1/events/$event_id"
record "GET smoke event" "200" "$CODE"
start="$(available_now)"
# A run cancelled halfway may leave a hold for up to 10 minutes; 3 tickets is all this journey needs.
if (( start < 3 )); then
  record "Smoke stock available" ">= 3" "$start"
fi

key="smoke-book-${GITHUB_RUN_ID:-local}-$RANDOM"
call POST /v1/bookings "$key" "{\"ticket_type_id\":\"$ticket_type\",\"quantity\":2}"
booking="$(field .data.id)"
record "POST booking" "201 PENDING_PAYMENT" "$CODE $(field .data.status)"

call POST /v1/bookings "$key" "{\"ticket_type_id\":\"$ticket_type\",\"quantity\":2}"
replayed="$(echo "$HEADERS" | grep -i '^Idempotent-Replayed:' | tr -d '\r' | awk '{print $2}')"
record "Replay booking (same key)" "201 $booking true" "$CODE $(field .data.id) ${replayed:-false}"

call POST "/v1/bookings/$booking/pay" "smoke-pay-$RANDOM$RANDOM" '{"payment_token":"tok_success"}'
record "Pay tok_success" "200 CONFIRMED" "$CODE $(field .data.status)"

call POST /v1/bookings "smoke-book2-$RANDOM$RANDOM" "{\"ticket_type_id\":\"$ticket_type\",\"quantity\":1}"
declined="$(field .data.id)"
call POST "/v1/bookings/$declined/pay" "smoke-pay2-$RANDOM$RANDOM" '{"payment_token":"tok_decline"}'
record "Pay tok_decline releases tickets" "402 PAYMENT_DECLINED" "$CODE $(field .code)"

# At most 3 of the 5 tickets remain, so 6 can never fit.
call POST /v1/bookings "smoke-big-$RANDOM$RANDOM" "{\"ticket_type_id\":\"$ticket_type\",\"quantity\":6}"
record "Insufficient tickets" "409 INSUFFICIENT_TICKETS" "$CODE $(field .code)"

call POST "/v1/bookings/$booking/cancel"
record "Cancel confirmed booking" "200 CANCELLED" "$CODE $(field .data.status)"

record "Stock restored after journey" "$start" "$(available_now)"

{
  echo "## OpenAPI smoke test"
  echo
  cat "$report"
} >> "${GITHUB_STEP_SUMMARY:-/dev/null}"

if (( failures > 0 )); then
  echo "::error::$failures smoke check(s) failed"
  exit 1
fi
echo "all smoke checks passed"
