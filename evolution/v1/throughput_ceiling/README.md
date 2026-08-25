# Hypothesis: v1 cannot hit the throughput SLO under realistic config — architectural ceiling

**Stated hypothesis:** doubling VUs stopped doubling throughput (100→512 req/s,
150→~700+ req/s, 200→860 req/s — an informal staircase, all three runs back-to-back
without a metrics reset in between, so directional only). That sub-linear curve looks
like v1 has hit a real ceiling, not something left to tune.

## Verdict: CONFIRMED — no tuning fix available within v1's realistic config

## Evidence

Clean, isolated run (`make test VUS=200 STEADY_DURATION=30s`, freshly reset via `make
reset-metrics`, pool at the SLO baseline of 30/10, GOMAXPROCS=1, prepared statements
already in place): 47322 requests matched exactly between k6 and
`http_requests_total`, confirming this reading is attributable to this run alone.

| Metric | Value |
|---|---|
| Throughput | 810.06 req/s |
| `p(95)` | 298.52ms |
| `p(99)` | 689.25ms |
| `avg` / `med` | 85.43ms / 46.34ms |
| `max` | 2.95s |
| `go_sql_wait_count_total` | 38710 / 47322 requests (~82%) waited for a connection |
| `go_sql_wait_duration_seconds_total` | 2399.4s cumulative over a ~58.4s test — an average of ~41 requests queued for a connection at any given instant, sustained through the run |
| CPU utilization | ~68.5% (40.03 core-s / ~58.4s) — elevated, but with headroom, not the primary constraint |
| `pg_locks_count` / deadlocks | negligible / 0 — no lock contention |

Against the [SLO](../../README.md#slo): latency passes (barely — p95 at 298.52ms
against a 300ms target), error rate passes (0%), **throughput fails** (810 vs. 1000
req/s target).

## Explanation

The connection pool (`DB_MAX_OPEN_CONNS=30`) is saturated and is the dominant,
sustained constraint — not a transient blip, but ~41 requests queued for a connection
throughout essentially the whole test. Unlike every prior bottleneck in this log, there
is no tuning knob left to turn here without breaking the premise the pool size was set
under: 30 is already the ceiling considered realistic for a single Postgres instance
without a connection pooler (PgBouncer) in front of it. Raising it further to chase this
number would mean testing a configuration nobody would actually run in production,
defeating the purpose of the SLO exercise (see [SLO](../../README.md#slo)).

CPU still has headroom (~68.5%), so it isn't the current cap — but it's also not far
enough below saturation to promise much more throughput even if the pool constraint were
lifted. Either way, closing this gap means requests need to hold a DB connection for a
shorter, or less frequently blocking, window than "the entire synchronous
lock-debit-credit-commit chain per HTTP request" — which is an architecture change
(e.g. decoupling request acceptance from DB processing via a queue), not a config change.

## Outcome

v1 is closed at this ceiling: **810 req/s sustained, p95=298.52ms, p99=689.25ms, 0%
errors, pool=30 (realistic), single Postgres instance, single `web` replica.** It meets
the SLO's latency and error-rate targets but not the throughput target, and no further
in-architecture tuning is available to close that gap — this is the trigger for v2.
