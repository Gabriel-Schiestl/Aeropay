#!/usr/bin/env bash
# Truncates accounts/payments/ledger and seeds `accounts` with N random rows
# for stress/load testing. Also exports the seeded account ids to
# scripts/k6/data/accounts.json so the k6 load script can pick real ids.
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
ACCOUNTS_FILE="$ROOT_DIR/scripts/k6/data/accounts.json"

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

echo "Truncating ledger, payments and accounts..."
docker compose -f "$COMPOSE_FILE" exec -T db psql -U "$DB_USER" -d "$DB_NAME" \
  -c "TRUNCATE TABLE ledger, payments, accounts RESTART IDENTITY CASCADE;"

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

mkdir -p "$(dirname "$ACCOUNTS_FILE")"
echo "Exporting seeded account ids to $ACCOUNTS_FILE..."
docker compose -f "$COMPOSE_FILE" exec -T db psql -U "$DB_USER" -d "$DB_NAME" \
  -tAc "SELECT coalesce(json_agg(id), '[]') FROM accounts;" > "$ACCOUNTS_FILE"
