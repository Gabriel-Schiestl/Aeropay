# Evolution Log

This directory tracks the scaling journey of `payment-service` as a deliberate exercise
in identifying bottlenecks and proposing fixes.

## SLO

Without a fixed target, tuning has no objective stopping point — there's always some
config change that pushes the ceiling a bit further. The SLO is what turns "how far can
we tune this?" into a yes/no question, and gives a principled trigger for moving to the
next version instead of tuning indefinitely.

| | Target |
|---|---|
| p95 latency | < 300ms |
| p99 latency | < 750ms |
| Error rate | < 0.1% |
| Sustained throughput | 1000 req/s |

Config is also constrained to what would be defensible in production, not whatever makes
a given test pass — e.g. the DB connection pool is capped at a size realistic for a
single Postgres instance *without* a pooler like PgBouncer in front of it (each Postgres
backend is a real, relatively expensive OS process), not pushed arbitrarily high just to
chase away wait metrics.

If v1 can't hit this SLO within realistic config, that's the trigger for v2 (see
versioning model below) — not "we could still tune it more."

This SLO is a starting point and expected to be revised as the exercise progresses.

## Versioning model

A version (`vN`) is one **architecture**, not one config. While a given architecture can
absorb more load purely through parameter tuning (pool sizes, resource limits, indexes,
timeouts, ...), every bottleneck found and fixed that way stays logged inside the same
`vN`. A new version only starts when a bottleneck is found that the current architecture
cannot absorb through tuning alone, and requires an actual design change (e.g. moving
from synchronous request handling to async processing).

Workflow for each bottleneck found inside a version:

1. Run `make load` (or `make test`) against the current config — **never run k6
   directly against a `web` container that already served previous traffic.** `make
   load` depends on `make reset-metrics`, which restarts `web` (clearing its in-process
   Prometheus counters: `go_sql_*`, `http_request_duration_seconds`, goroutines, ...)
   and resets Postgres's stats collector (`pg_stat_reset()`,
   `pg_stat_statements_reset()`) first. Skipping this mixes counters from unrelated
   runs together and produces evidence that can't be attributed to the test that just
   ran (this happened once already — see the "Confirmation status" note in
   [v1/pool_bottleneck](v1/pool_bottleneck/README.md)).
2. Propose a hypothesis for what limited the system.
3. The hypothesis is investigated against metrics/code and confirmed or refuted with
   evidence, written up in `vN/<bottleneck-name>/README.md`.
4. If the fix is a config/parameter change, apply it and keep logging under the same
   `vN`. If no tuning fix exists, the next architecture starts as `v(N+1)`.

## Versions

- [v1](v1/README.md) — **closed.** Synchronous handler, single DB, single replica.
  Final numbers: 810 req/s, p95=298.52ms, p99=689.25ms, 0% errors — met latency/error
  SLO, missed throughput. See [throughput_ceiling](v1/throughput_ceiling/README.md).
- [v2](v2/README.md) — **built, not yet load-tested.** Async processing: `web` accepts
  and enqueues via a transactional outbox, `workers` consume from Kafka and do the
  actual debit/credit. Has known gaps (idempotency write-back, retry/DLQ, processing
  observability) that need closing before its throughput can be meaningfully measured.
