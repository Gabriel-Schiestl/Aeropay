#!/usr/bin/env bash
# Seeds the `accounts` table with N random accounts for stress/load testing.
#
# Usage:
#   ./scripts/seed_accounts.sh [count] [min_balance] [max_balance]
#
# Examples:
#   ./scripts/seed_accounts.sh            # 1000 accounts, balance 0-10000
#   ./scripts/seed_accounts.sh 100000     # 100k accounts
#   ./scripts/seed_accounts.sh 5000 0 100 # 5000 accounts, balance 0-100

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/docker-compose.yaml"

COUNT="${1:-1000}"
MIN_BALANCE="${2:-0}"
MAX_BALANCE="${3:-10000}"

DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-payment_service}"

if ! [[ "$COUNT" =~ ^[0-9]+$ ]]; then
  echo "count must be a positive integer, got: $COUNT" >&2
  exit 1
fi

echo "Starting stack (if not already up)..."
docker compose -f "$COMPOSE_FILE" up -d >/dev/null

echo "Waiting for postgres to be ready..."
until docker compose -f "$COMPOSE_FILE" exec -T db pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; do
  sleep 1
done

echo "Waiting for app migrations to create the accounts table..."
until [ "$(docker compose -f "$COMPOSE_FILE" exec -T db psql -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT to_regclass('accounts');")" = "accounts" ]; do
  sleep 1
done

echo "Seeding $COUNT accounts (balance range: $MIN_BALANCE-$MAX_BALANCE)..."
docker compose -f "$COMPOSE_FILE" exec -T db psql -U "$DB_USER" -d "$DB_NAME" \
  -v count="$COUNT" -v min_balance="$MIN_BALANCE" -v max_balance="$MAX_BALANCE" <<'SQL'
INSERT INTO accounts (id, balance)
SELECT
    gen_random_uuid(),
    round((:min_balance + random() * (:max_balance - :min_balance))::numeric, 4)
FROM generate_series(1, :count);
SQL

echo "Done. Current account count:"
docker compose -f "$COMPOSE_FILE" exec -T db psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM accounts;"
