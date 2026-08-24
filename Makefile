.PHONY: up down seed reset-metrics load test clean

COUNT ?= 2000
MIN_BALANCE ?= 1000
MAX_BALANCE ?= 100000
VUS ?= 50
STEADY_DURATION ?= 30s
BASE_URL ?= http://localhost:8080

## Start the docker-compose stack (db + web).
up:
	docker compose up -d

## Stop the stack, keeping the db volume.
down:
	docker compose down

## Truncate ledger/payments/accounts and seed $(COUNT) fresh accounts.
seed:
	./scripts/seed_accounts.sh $(COUNT) $(MIN_BALANCE) $(MAX_BALANCE)

## Restart web (clears its in-process Prometheus counters: go_sql_*,
## http_request_duration_seconds, goroutines, ...) and reset Postgres's
## stats collector (pg_stat_database, pg_stat_statements), so the next load
## test starts from zero instead of mixing in leftovers from a previous run.
reset-metrics:
	docker compose up -d --force-recreate web
	@echo "==> Waiting for web to be ready..."
	@for i in $$(seq 1 30); do \
		curl -sf $(BASE_URL)/metrics > /dev/null 2>&1 && exit 0; \
		sleep 1; \
	done; \
	echo "web did not become ready in time" >&2; exit 1
	docker compose exec -T db psql -U postgres -d payment_service -c "SELECT pg_stat_reset(); SELECT pg_stat_statements_reset();" > /dev/null

## Reset metrics, reseed accounts and run the k6 load test against POST /payments.
load: reset-metrics
	BASE_URL=$(BASE_URL) MIN_BALANCE=$(MIN_BALANCE) MAX_BALANCE=$(MAX_BALANCE) \
		./scripts/run_load_test.sh $(VUS) $(STEADY_DURATION) $(COUNT)

## Alias for load - reset metrics + reseed + run the full load test.
test: load

## Stop the stack and wipe the db volume (fully clean slate).
clean:
	docker compose down -v
