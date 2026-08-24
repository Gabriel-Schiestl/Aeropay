# Evolution Log

This directory tracks the scaling journey of `payment-service` as a deliberate exercise
in identifying bottlenecks and proposing fixes.

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

- [v1](v1/README.md) — synchronous handler, single DB, single replica. Currently being
  scaled through configuration tuning.
