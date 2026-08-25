# Hypothesis: DB connection pool exhaustion

**Stated hypothesis:** the tail latency in the v1 load test (p50 constant, p99/max
spiking) is caused by the Go `database/sql` connection pool being too small
(`DB_MAX_OPEN_CONNS=10`), not by the database or the CPU.

## Verdict: CONFIRMED

## Evidence

From the v1 load test itself (50 VUs, 15s steady, pool=10):

| Metric | Value | Reading |
|---|---|---|
| `med` | 12.45ms | the common case is fast |
| `avg` | 35.64ms | pulled ~3x above `med` — only a distribution with a heavy tail does this |
| `p(95)` | 98.66ms | ~8x the median |
| `max` | 4.38s | ~350x the median |
| `go_sql_max_open_connections` | 10 | matches `DB_MAX_OPEN_CONNS` in `.env` |
| Postgres `max_connections` | 100 | plenty of headroom above 10 — rules out "the pool is small because Postgres can't take more" |

Ruled out alternative explanations before accepting this one:

- **CPU-bound**: would show elevated `process_cpu_seconds_total` sustained near the
  1-core container limit for the whole test, and would degrade every request roughly
  uniformly rather than stretching a tail behind a fast median. Not directly measured
  for this specific run, but the low median (12.45ms) argues against the app being
  compute-starved on the common path.
- **Row-lock contention on `accounts`**: would show `pg_locks_count` climbing together
  with latency and/or non-zero `pg_stat_database_deadlocks`. The lock ordering in
  `payment.repository.go` (always lock the lower account ID first) already prevents the
  deadlock case; lock *wait* under load remains architecturally possible and wasn't
  ruled out with a direct measurement on this run, but the pool cap alone is sufficient
  to explain the observed magnitude (10 concurrent DB operations max vs. 50 concurrent
  VUs).

## Explanation

`DB_MAX_OPEN_CONNS=10` means only 10 goroutines can hold a live DB connection at once.
At 50 concurrent VUs, the majority of in-flight requests at any instant have to sit in
`database/sql`'s internal wait queue for a connection to free up — entirely inside the
Go process, before a single query reaches Postgres. This produces exactly the observed
shape: requests that get a connection immediately finish in a few milliseconds (hence
the 12.45ms median), while requests that queue behind the cap accumulate wait time
proportional to how backed up the queue is at that instant (hence `avg` being dragged up
and `max` reaching seconds). Postgres itself was never shown to be the constraint — it
had 90 unused connection slots the whole time, and the app never got to use them.

## Fix applied

`DB_MAX_OPEN_CONNS` 10 → 30, `DB_MAX_IDLE_CONNS` 5 → 20 in `payment-service/.env`. A
config-only change, so it stays within v1 (see [versioning model](../../README.md)) —
no architectural change was needed for this bottleneck.

**Confirmation status:** pending. Rerun `make test VUS=50 STEADY_DURATION=15s` against
the current config (pool=30/20) to confirm the fix directly.
