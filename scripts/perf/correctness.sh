#!/usr/bin/env bash
# Server-side correctness checks after a performance run. These are never
# calibrated: any violation fails the job.
#   1. Inventory invariant for every ticket type of the run's events
#   2. No booking with more than one successful payment
#   3. No PENDING_PAYMENT booking more than 15 s past its hold
#   4. PT-05: exactly flash_tickets active bookings and 0 available
# Usage: scripts/perf/correctness.sh <scenario> <event.properties>
set -euo pipefail

scenario="$1"
props="$2"
prop() { grep "^$1=" "$props" | cut -d= -f2 | tr -d '\r' || true; }

# Check both load and flash fixtures; the exact flash-sale count only after PT-05.
events="$(prop event_id),$(prop flash_event_id)"
flash_tt=""
flash_tickets="0"
if [[ "$scenario" == "PT-05" ]]; then
  flash_tt="$(prop flash_ticket_type_id)"
  flash_tickets="$(prop flash_tickets)"
fi

export PGHOST="$DB_HOST" PGPORT="${DB_PORT:-5432}" PGUSER="$DB_USER" PGPASSWORD="$DB_PASSWORD" \
       PGDATABASE="${DB_NAME:-postgres}" PGSSLMODE="${DB_SSLMODE:-require}"

psql -X -v ON_ERROR_STOP=1 -v events="$events" -v flash_tt="$flash_tt" -v flash_tickets="$flash_tickets" \
  -f scripts/perf/correctness.sql

echo "correctness checks passed"
{
  echo
  echo "Correctness checks: **passed** (inventory invariant, single payment, holds released$([[ -n "$flash_tt" ]] && echo ", flash sale exact"))."
} >> "${GITHUB_STEP_SUMMARY:-/dev/null}"
