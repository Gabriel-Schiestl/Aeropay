#!/usr/bin/env bash
# Reseeds accounts (truncating accounts/payments/ledger first) and runs the
# k6 load test against POST /payments. Uses the local k6 binary if present,
# otherwise falls back to the official grafana/k6 Docker image on the same
# network as the `web` service.
#
# Usage:
#   ./scripts/run_load_test.sh [vus] [steady_duration] [accounts_count]
#
# Examples:
#   ./scripts/run_load_test.sh                 # 50 VUs, 30s steady, 2000 accounts
#   ./scripts/run_load_test.sh 200 2m           # 200 VUs, 2m steady
#   ./scripts/run_load_test.sh 200 2m 20000     # also reseed 20000 accounts

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/docker-compose.yaml"
K6_SCRIPT="$ROOT_DIR/scripts/k6/load_payments.js"

VUS="${1:-50}"
STEADY_DURATION="${2:-30s}"
ACCOUNTS_COUNT="${3:-2000}"

MIN_BALANCE="${MIN_BALANCE:-1000}"
MAX_BALANCE="${MAX_BALANCE:-100000}"

echo "==> Reseeding accounts (truncates ledger/payments/accounts first)..."
"$ROOT_DIR/scripts/seed_accounts.sh" "$ACCOUNTS_COUNT" "$MIN_BALANCE" "$MAX_BALANCE"

echo "==> Running k6 load test ($VUS VUs, ${STEADY_DURATION} steady)..."
if command -v k6 >/dev/null 2>&1; then
  BASE_URL="${BASE_URL:-http://localhost:8080}" \
  k6 run -e VUS="$VUS" -e STEADY_DURATION="$STEADY_DURATION" -e BASE_URL="${BASE_URL:-http://localhost:8080}" \
    "$K6_SCRIPT"
else
  echo "k6 not found locally, running via Docker (grafana/k6)..."
  NETWORK="$(docker compose -f "$COMPOSE_FILE" ps -q web | xargs -r docker inspect -f '{{range $k, $v := .NetworkSettings.Networks}}{{$k}}{{end}}')"
  if [ -z "$NETWORK" ]; then
    echo "Could not resolve the docker network for the 'web' service. Is 'docker compose up -d' running?" >&2
    exit 1
  fi
  docker run --rm -i \
    --network "$NETWORK" \
    -v "$ROOT_DIR/scripts/k6:/scripts" \
    -w /scripts \
    -e BASE_URL="${BASE_URL:-http://web:8080}" \
    -e VUS="$VUS" \
    -e STEADY_DURATION="$STEADY_DURATION" \
    grafana/k6 run /scripts/load_payments.js
fi
