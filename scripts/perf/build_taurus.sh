#!/usr/bin/env bash
# Build the files uploaded to BlazeMeter: the JMeter plan, the users CSV and a
# Taurus config (bzt-test.yml) carrying the target host, the seeded IDs and the
# scenario's load properties. JSON is valid YAML, so the config is written as JSON.
# Usage: scripts/perf/build_taurus.sh <scenario> <base-url> <seed-dir> <out-dir>
set -euo pipefail

scenario="$1"
base_url="${2%/}"
seed_dir="$3"
out="$4"
mkdir -p "$out"

protocol="${base_url%%://*}"
host="$(echo "$base_url" | sed -E 's#^[a-z]+://##; s#[:/].*$##')"
port=443
[[ "$protocol" == "http" ]] && port=80

cp perf/jmeter/booking.jmx "$out/booking.jmx"
cp "$seed_dir/users.csv" "$out/users.csv"

seeded="$(grep -v '^\s*$' "$seed_dir/event.properties" | tr -d '\r' | jq -R 'split("=") | {(.[0]): .[1]}' | jq -s 'add')"
load="$(jq --arg s "$scenario" '.[$s].properties | with_entries(.value |= tostring)' perf/scenarios.json)"

jq -n \
  --arg protocol "$protocol" --arg host "$host" --arg port "$port" --arg scenario "$scenario" \
  --argjson seeded "$seeded" --argjson load "$load" '{
    execution: [{ executor: "jmeter", scenario: "booking" }],
    scenarios: {
      booking: {
        script: "booking.jmx",
        properties: ({ protocol: $protocol, host: $host, port: $port, scenario: $scenario, users_csv: "users.csv" }
                     + $seeded + $load)
      }
    },
    modules: { jmeter: { version: "5.6.3" } }
  }' > "$out/bzt-test.yml"

echo "BlazeMeter files for $scenario:"
ls -l "$out"
