#!/usr/bin/env bash
# Invariant/consistency test for concurrent payments against the same account.
#
# Fires many concurrent, differently-shaped transactions (varying amount,
# direction and destination) against one shared "hot" account and asserts
# that the core system invariants never break, even under heavy contention:
#
#   1. No account balance is ever negative.
#   2. Total money in the closed set of test accounts is conserved
#      (nothing created or destroyed by racing transactions).
#   3. Each account's balance matches its seeded balance plus the sum of
#      its own ledger entries (no lost updates / double-spends).
#   4. The number of HTTP 201 responses matches what was actually
#      committed to the database (payments + ledger row counts).
#
# Truncates ledger/payments/accounts first, like seed_accounts.sh.
#
# Usage:
#   ./scripts/invariant_test.sh [hot_balance] [num_other_accounts] [requests] [parallelism]
#
# Examples:
#   ./scripts/invariant_test.sh                  # balance=300, 20 accounts, 400 reqs, 50 parallel
#   ./scripts/invariant_test.sh 100 10 800 100   # tighter balance, higher contention

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/docker-compose.yaml"

HOT_BALANCE="${1:-300}"
NUM_OTHERS="${2:-20}"
NUM_REQUESTS="${3:-400}"
PARALLELISM="${4:-50}"
OTHER_BALANCE="${OTHER_BALANCE:-1000}"

DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-payment_service}"
BASE_URL="${BASE_URL:-http://localhost:8080}"

for arg_name in HOT_BALANCE NUM_OTHERS NUM_REQUESTS PARALLELISM; do
  if ! [[ "${!arg_name}" =~ ^[0-9]+$ ]]; then
    echo "$arg_name must be a positive integer, got: ${!arg_name}" >&2
    exit 1
  fi
done

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

psql_exec() {
  docker compose -f "$COMPOSE_FILE" exec -T db psql -q -U "$DB_USER" -d "$DB_NAME" "$@"
}

echo "==> Starting stack (if not already up)..."
docker compose -f "$COMPOSE_FILE" up -d db web >/dev/null

echo "==> Waiting for postgres..."
until docker compose -f "$COMPOSE_FILE" exec -T db pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; do
  sleep 1
done

echo "==> Waiting for API..."
until curl -s -o /dev/null -w '' "$BASE_URL/metrics" 2>/dev/null; do
  sleep 1
done

echo "==> Truncating ledger, payments and accounts..."
psql_exec -c "TRUNCATE TABLE ledger, payments, accounts RESTART IDENTITY CASCADE;" >/dev/null

echo "==> Seeding 1 hot account (balance=$HOT_BALANCE) and $NUM_OTHERS other accounts (balance=$OTHER_BALANCE each)..."
HOT_ID=$(psql_exec -v hot_balance="$HOT_BALANCE" -tA <<'SQL'
INSERT INTO accounts (id, balance) VALUES (gen_random_uuid(), :hot_balance) RETURNING id;
SQL
)
psql_exec -v n="$NUM_OTHERS" -v ob="$OTHER_BALANCE" >/dev/null <<'SQL'
INSERT INTO accounts (id, balance) SELECT gen_random_uuid(), :ob FROM generate_series(1, :n);
SQL

mapfile -t OTHER_IDS < <(psql_exec -tAc "SELECT id FROM accounts WHERE id <> '$HOT_ID';")

TOTAL_INITIAL=$(( HOT_BALANCE + NUM_OTHERS * OTHER_BALANCE ))
echo "    hot account: $HOT_ID"
echo "    total money in the system: $TOTAL_INITIAL"

echo "==> Building $NUM_REQUESTS concurrent, mixed-direction requests against the hot account..."
REQ_FILE="$WORKDIR/requests.tsv"
RESULTS_DIR="$WORKDIR/results"
mkdir -p "$RESULTS_DIR"
: > "$REQ_FILE"

for i in $(seq 1 "$NUM_REQUESTS"); do
  other_id="${OTHER_IDS[$((RANDOM % NUM_OTHERS))]}"
  amount_cents=$(( (RANDOM % 4900) + 100 )) # 1.00 - 49.99
  amount=$(printf '%d.%02d' $((amount_cents / 100)) $((amount_cents % 100)))
  # Skewed toward debits so the hot account actually gets driven towards
  # zero and concurrent requests really do race for the last of its funds,
  # instead of credits quietly keeping it topped up.
  if (( RANDOM % 10 < 7 )); then
    printf '%s\t%s\t%s\t%s\n' "$i" "$HOT_ID" "$other_id" "$amount" >>"$REQ_FILE"   # debit: hot -> other
  else
    printf '%s\t%s\t%s\t%s\n' "$i" "$other_id" "$HOT_ID" "$amount" >>"$REQ_FILE"   # credit: other -> hot
  fi
done

fire_one() {
  local idx="$1" from="$2" to="$3" amount="$4"
  local status
  status=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/payments" \
    -H 'Content-Type: application/json' \
    -d "{\"amount\":$amount,\"currency\":\"USD\",\"from\":\"$from\",\"to\":\"$to\"}")
  echo "$status" >"$RESULTS_DIR/$idx"
}
export -f fire_one
export BASE_URL RESULTS_DIR

echo "==> Firing requests with parallelism=$PARALLELISM..."
xargs -a "$REQ_FILE" -P "$PARALLELISM" -n 4 -- bash -c 'fire_one "$@"' _

echo "==> Tallying HTTP results..."
TOTAL_SENT=$(ls "$RESULTS_DIR" | wc -l)
SUCCESS_COUNT=$( { grep -l '^201$' "$RESULTS_DIR"/* 2>/dev/null || true; } | wc -l)
REJECTED_COUNT=$( { grep -l '^400$' "$RESULTS_DIR"/* 2>/dev/null || true; } | wc -l)
SERVER_ERROR_COUNT=$( { grep -lE '^5[0-9]{2}$' "$RESULTS_DIR"/* 2>/dev/null || true; } | wc -l)
OTHER_COUNT=$(( TOTAL_SENT - SUCCESS_COUNT - REJECTED_COUNT - SERVER_ERROR_COUNT ))

echo "    sent=$TOTAL_SENT success(201)=$SUCCESS_COUNT rejected(400)=$REJECTED_COUNT 5xx=$SERVER_ERROR_COUNT other=$OTHER_COUNT"

FAILED=0

echo "==> Checking invariant 1: no negative balances..."
MIN_BALANCE=$(psql_exec -tAc "SELECT min(balance) FROM accounts;")
if (( $(echo "$MIN_BALANCE < 0" | bc -l) )); then
  echo "    FAIL: minimum balance across accounts is $MIN_BALANCE (negative!)"
  FAILED=1
else
  echo "    OK: minimum balance across accounts is $MIN_BALANCE"
fi

echo "==> Checking invariant 2: money conservation (no funds created/destroyed)..."
TOTAL_FINAL=$(psql_exec -tAc "SELECT sum(balance) FROM accounts;")
if [[ "$(echo "$TOTAL_FINAL == $TOTAL_INITIAL" | bc -l)" != "1" ]]; then
  echo "    FAIL: total balance is $TOTAL_FINAL, expected $TOTAL_INITIAL"
  FAILED=1
else
  echo "    OK: total balance unchanged at $TOTAL_FINAL"
fi

echo "==> Checking invariant 3: each account's balance matches seed + its ledger entries..."
MISMATCHES=$(psql_exec -v hot_id="$HOT_ID" -v hot_balance="$HOT_BALANCE" -v other_balance="$OTHER_BALANCE" -tA <<'SQL'
SELECT count(*) FROM (
  SELECT a.id,
         a.balance,
         (CASE WHEN a.id = :'hot_id' THEN :hot_balance ELSE :other_balance END)
           + COALESCE(SUM(l.amount), 0) AS expected
  FROM accounts a
  LEFT JOIN ledger l ON l.account = a.id
  GROUP BY a.id, a.balance
) t
WHERE t.balance <> t.expected;
SQL
)
if [[ "$MISMATCHES" != "0" ]]; then
  echo "    FAIL: $MISMATCHES account(s) have balance != seed + sum(ledger entries)"
  FAILED=1
else
  echo "    OK: every account's balance is fully explained by its ledger entries"
fi

echo "==> Checking invariant 4: HTTP successes match committed rows..."
PAYMENTS_COUNT=$(psql_exec -tAc "SELECT count(*) FROM payments;")
LEDGER_COUNT=$(psql_exec -tAc "SELECT count(*) FROM ledger;")
if [[ "$PAYMENTS_COUNT" != "$SUCCESS_COUNT" ]]; then
  echo "    FAIL: $SUCCESS_COUNT HTTP 201s but $PAYMENTS_COUNT payments rows"
  FAILED=1
elif [[ "$LEDGER_COUNT" != "$(( SUCCESS_COUNT * 2 ))" ]]; then
  echo "    FAIL: expected $(( SUCCESS_COUNT * 2 )) ledger rows (2 per payment), got $LEDGER_COUNT"
  FAILED=1
else
  echo "    OK: $SUCCESS_COUNT payments, $LEDGER_COUNT ledger rows"
fi

if (( SERVER_ERROR_COUNT > 0 )); then
  echo "==> WARNING: $SERVER_ERROR_COUNT request(s) got an unexpected 5xx (see $RESULTS_DIR before it's cleaned up)"
fi

echo
if (( FAILED == 0 )); then
  echo "PASS: invariants held under $NUM_REQUESTS concurrent mixed transactions."
  exit 0
else
  echo "FAIL: one or more invariants were violated."
  exit 1
fi
