# Version 1 — synchronous monolith

**Status: closed.** v1 hit a throughput ceiling that no further tuning can close
without breaking the SLO's own realism constraints — see
[throughput_ceiling](throughput_ceiling/README.md). v2 is the next step.

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

![v1 architecture](image.png)

Observability (Prometheus + Grafana + postgres-exporter, `/metrics` and `/debug/pprof`
on the Go service) was added specifically to support the investigations below — it's
tooling, not part of the versioned architecture.

## Current configuration

| Setting | Value | Last changed by |
|---|---|---|
| `DB_MAX_OPEN_CONNS` | 30 | SLO baseline — capped at a size realistic without a pooler (PgBouncer) in front of Postgres, not raised further just to chase away pool wait. See [SLO](../README.md#slo). |
| `DB_MAX_IDLE_CONNS` | 10 | same as above |
| `DB_CONN_MAX_LIFETIME` | 30min | (initial) |
| `GOMAXPROCS` | 1 | [gomaxprocs_mismatch](gomaxprocs_mismatch/README.md) |
| Postgres `max_connections` | 100 | (initial, untouched) |
| `web` container limits | `cpus: 1`, `memory: 512M` | (initial, untouched) |

## Bottleneck log

| # | Load test | Bottleneck | Verdict | Fix |
|---|---|---|---|---|
| 1 | 50 VUs, 15s steady | [Connection pool exhaustion](pool_bottleneck/README.md) | CONFIRMED | `DB_MAX_OPEN_CONNS` 10→40, `DB_MAX_IDLE_CONNS` 5→20 |
| 2 | 100 VUs, 30s steady | [GOMAXPROCS/cgroup CPU quota mismatch](gomaxprocs_mismatch/README.md) | CONFIRMED (partial) | `GOMAXPROCS=1` |
| 3 | 100 VUs, 30s steady | [No prepared-statement reuse across requests](prepared_statements/README.md) | CONFIRMED | Prepare the 5 queries once in `NewPaymentRepository`, reuse via `tx.StmtContext` |
| 4 | 200 VUs, 30s steady, pool=30 (realistic) | [Throughput ceiling — pool saturated, no tuning fix left](throughput_ceiling/README.md) | CONFIRMED — architectural | none available within v1; **triggers v2** |

**Final v1 numbers:** 810 req/s sustained, p95=298.52ms, p99=689.25ms, 0% errors, at
pool=30 / GOMAXPROCS=1 / prepared statements in place. Meets the SLO's latency and error
targets, misses the 1000 req/s throughput target — see
[throughput_ceiling](throughput_ceiling/README.md) for why no further tuning closes
that gap.

Pool sizes above (40/20) are what investigations #1-#3 actually ran against.
After #3, the pool was deliberately lowered to 30/10 — not from a new bottleneck, but
to adopt a size realistic for production without PgBouncer, per the [SLO](../README.md#slo).
That's the value in "Current configuration" above, and what any test run after this
point in the log is measured against.
