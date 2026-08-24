.PHONY: up down seed load test clean

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

## Reseed accounts and run the k6 load test against POST /payments.
load:
	BASE_URL=$(BASE_URL) MIN_BALANCE=$(MIN_BALANCE) MAX_BALANCE=$(MAX_BALANCE) \
		./scripts/run_load_test.sh $(VUS) $(STEADY_DURATION) $(COUNT)

## Alias for load - reseed + run the full load test.
test: load

## Stop the stack and wipe the db volume (fully clean slate).
clean:
	docker compose down -v
