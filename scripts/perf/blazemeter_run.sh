#!/usr/bin/env bash
# Upload the test files to an existing BlazeMeter test, start it, wait for it to
# end, and download the aggregate report.
#
# One-time setup in BlazeMeter: create a performance test of type Taurus whose
# main file is bzt-test.yml (upload the files from build_taurus.sh once), choose
# the load location, and put its ID in the BLAZEMETER_TEST_ID variable. Each run
# re-uploads the same file names, which replaces them.
#
# Usage: scripts/perf/blazemeter_run.sh <files-dir> <timeout-minutes> <out-dir>
set -euo pipefail

files="$1"
timeout_min="$2"
out="$3"
api="https://a.blazemeter.com/api/v4"
auth="$BLAZEMETER_API_KEY_ID:$BLAZEMETER_API_KEY_SECRET"
test_id="$BLAZEMETER_TEST_ID"
mkdir -p "$out"

bzm() { curl -fsS --retry 3 -u "$auth" "$@"; }

for f in "$files"/*; do
  echo "uploading $(basename "$f")"
  bzm -F "file=@$f" "$api/tests/$test_id/files" > /dev/null
done

master="$(bzm -X POST "$api/tests/$test_id/start" | jq -r '.result.id')"
if [[ -z "$master" || "$master" == "null" ]]; then
  echo "::error::BlazeMeter did not start the test"
  exit 1
fi
report_url="https://a.blazemeter.com/app/#/masters/$master"
echo "$report_url" > "$out/report_url.txt"
echo "started BlazeMeter run $master: $report_url"

deadline=$(( $(date +%s) + timeout_min * 60 ))
while true; do
  status="$(bzm "$api/masters/$master/status" | jq -r '.result.status')"
  echo "$(date -u +%H:%M:%S) status: $status"
  [[ "$status" == "ENDED" ]] && break
  if (( $(date +%s) > deadline )); then
    bzm -X POST "$api/masters/$master/terminate" > /dev/null || true
    echo "::error::BlazeMeter run exceeded ${timeout_min} minutes and was terminated"
    exit 1
  fi
  sleep 30
done

bzm "$api/masters/$master" > "$out/master.json"
bzm "$api/masters/$master/reports/aggregatereport/data" > "$out/aggregate.json"

{
  echo "## BlazeMeter run: ${SCENARIO:-scenario}"
  echo
  echo "Report: $report_url"
  echo
} >> "${GITHUB_STEP_SUMMARY:-/dev/null}"
