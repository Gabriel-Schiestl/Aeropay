# Version 1 — synchronous monolith

This version's architecture is fixed; only configuration changes as bottlenecks are
found and resolved by tuning. It moves to v2 only when a bottleneck shows up that this
architecture cannot absorb through tuning alone.

## Architecture (fixed for this version)

- **Go (Gin) HTTP API**, single handler: `POST /payments`.
- **Fully synchronous processing** — the HTTP handler blocks on the whole database
  transaction before responding; no queue, no batching, no background workers.
- **One Postgres instance**, no read replica, no partitioning.
- **Transactional transfer logic** (`payment.repository.go`): a single transaction per
  payment that:
  1. `SELECT ... FOR UPDATE` on the `from` account
  2. `SELECT ... FOR UPDATE` on the `to` account (both locks always taken in ascending
     account-ID order, to prevent deadlocks between two payments moving money in
     opposite directions between the same pair of accounts)
  3. `INSERT` the payment row
  4. `UPDATE` (debit `from`, reject if insufficient funds) + `INSERT` ledger row
  5. `UPDATE` (credit `to`) + `INSERT` ledger row
  6. `COMMIT`
- **Single `web` replica**, container capped at `cpus: 1`, `memory: 512M`
  (`docker-compose.yaml`).

Observability (Prometheus + Grafana + postgres-exporter, `/metrics` and `/debug/pprof`
on the Go service) was added specifically to support the investigations below — it's
tooling, not part of the versioned architecture.

## Current configuration

| Setting | Value | Last changed by |
|---|---|---|
| `DB_MAX_OPEN_CONNS` | 40 | [pool_bottleneck](pool_bottleneck/README.md) |
| `DB_MAX_IDLE_CONNS` | 20 | [pool_bottleneck](pool_bottleneck/README.md) |
| `DB_CONN_MAX_LIFETIME` | 30min | (initial) |
| Postgres `max_connections` | 100 | (initial, untouched) |
| `web` container limits | `cpus: 1`, `memory: 512M` | (initial, untouched) |

## Bottleneck log

| # | Load test | Bottleneck | Verdict | Fix |
|---|---|---|---|---|
| 1 | 50 VUs, 15s steady | [Connection pool exhaustion](pool_bottleneck/README.md) | CONFIRMED | `DB_MAX_OPEN_CONNS` 10→40, `DB_MAX_IDLE_CONNS` 5→20 |
