#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Default values
SERVER_ADDRESS="http://localhost:8080"
SCENARIO=""

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -s|--scenario)
      SCENARIO="$2"
      shift 2
      ;;
    -a|--address)
      SERVER_ADDRESS="$2"
      shift 2
      ;;
    -h|--help)
      echo "Usage: $0 [-s|--scenario <1|2>] [-a|--address <server_address>]"
      echo ""
      echo "Options:"
      echo "  -s, --scenario <1|2>  Scenario to run (1 or 2)"
      echo "  -a, --address <url>   Server address (default: http://localhost:8080)"
      echo "  -h, --help            Show this help message"
      echo ""
      echo "Scenarios:"
      echo "  1: Batch upload of 5 counters and 5 gauges"
      echo "  2: Fill database with random metrics, then batch upload"
      exit 0
      ;;
    *)
      # Backward compatibility: first positional argument is server address
      SERVER_ADDRESS="$1"
      shift
      ;;
  esac
done

if ! HEY_BIN=$(which hey); then
  echo "Please install hey (https://github.com/rakyll/hey)"
  exit 1
fi

echo "Using hey at: $HEY_BIN"
echo "Server address: $SERVER_ADDRESS"

workers_num=10
test_duration="30s"
request_timeout_seconds="20"

run_scenario_1() {
  echo """
  === Running Scenario 1: Batch upload of 5 counters and 5 gauges ===
  """

  $HEY_BIN \
      -c "$workers_num" \
      -z "$test_duration" \
      -t "$request_timeout_seconds" \
      -m POST \
      -T "application/json" \
      -D "$SCRIPT_DIR/scenario1_data.json" \
      -disable-compression \
      "$SERVER_ADDRESS/updates/"
}

run_scenario_2() {
  echo """
  === Running Scenario 2: Fill database with random metrics ===
  === then batch upload of 5 counters and 5 gauges ===
  """

  echo "Filling database with random metrics..."
  n=1000
  for ((i=0; i<n; i++)); do
      curl -s -X POST \
          -H "Content-Type: application/json" \
          -d "{\"id\":\"counter_$i\",\"type\":\"counter\",\"delta\":$RANDOM}" \
           "$SERVER_ADDRESS/update/" >/dev/null

      curl -s -X POST \
          -H "Content-Type: application/json" \
          -d "{\"id\":\"gauge_$i\",\"type\":\"gauge\",\"value\":$RANDOM}" \
           "$SERVER_ADDRESS/update/" >/dev/null

      # Progress indicator
      if ((i % 100 == 0)); then
        echo "Progress: $i/$n"
      fi
  done

  echo "Starting load test..."
  $HEY_BIN \
      -c "$workers_num" \
      -z "$test_duration" \
      -t "$request_timeout_seconds" \
      -m POST \
      -T "application/json" \
      -D "$SCRIPT_DIR/scenario1_data.json" \
      -disable-compression \
      "$SERVER_ADDRESS/updates/"
}

# Run scenarios
case $SCENARIO in
  1)
    run_scenario_1
    ;;
  2)
    run_scenario_2
    ;;
  "")
    echo "No scenario specified. Parameter -s|--scenario is required."
    exit 1
    ;;
  *)
    echo "Error: Invalid scenario '$SCENARIO'."
    exit 1
    ;;
esac

echo ""
echo "Load test completed!"
