# Hypothesis: GOMAXPROCS/cgroup CPU quota mismatch

**Stated hypothesis:** at 100 VUs (pool=40), the typical request got slow — not just the
tail — which the connection pool alone doesn't explain. Candidate cause: the Go runtime
scheduling more OS threads than the container's CPU quota can actually run, causing
scheduling overhead under concurrent load.

## Verdict: CONFIRMED (partial contributing factor)

## Evidence

At 100 VUs / 30s steady, pool=40, before this fix:

| Metric | Value | Reading |
|---|---|---|
| `med` | 82.78ms | vs. 12.45ms at 50 VUs — the *typical* request got much slower, not just a minority |
| `avg` | 83.68ms | ratio `avg`/`med` ≈ 1.01 — nearly identical to median, unlike the pool-bottleneck signature (`avg`/`med` ≈ 2.9 at 50 VUs) where a fast majority coexists with a slow minority |
| `process_cpu_seconds_total` delta over the test | ~50.48 core-seconds over a ~61.7s wall-clock window | ≈ 82% of the container's 1-CPU quota, sustained for the whole test |

A converged `avg`≈`med` with the *whole* distribution shifted up points at something
that costs every request roughly the same, rather than a queue a minority gets stuck
behind — that shape doesn't match pool contention on its own.

Checked inside the running container:

```
nproc                              → 12   (the host's core count)
cpu.cfs_quota_us / cfs_period_us   → 100000 / 100000  (quota = 1.0 CPU)
GOMAXPROCS env var                 → not set
```

## Explanation

Docker's `cpus: "1"` limit is enforced via the CFS bandwidth controller (a quota per
scheduling period), not via `cpuset` (CPU affinity). That means `sched_getaffinity`
inside the container still reports all 12 host cores, and Go's runtime defaults
`GOMAXPROCS` from that affinity mask — so the Go scheduler believed it had 12 CPUs to
spread goroutines across, while the cgroup only ever grants the container the equivalent
of 1 CPU-second of execution per wall-clock second. Under concurrent load (100 VUs), the
runtime schedules work across up to 12 OS threads that all get throttled by the same
1-CPU quota, producing scheduling churn (context switches, CFS throttling) that taxes
every request roughly equally — explaining why the *median* moved, not just the tail.

## Fix applied

`GOMAXPROCS=1` set in `payment-service/.env`, matching the container's `cpus: "1"`
limit exactly. Config-only change, stays within v1.

## Confirmation

Re-ran the same 100 VUs / 30s test (pool still 40 at this point — it was later lowered
to 30 for [SLO](../../README.md#slo) realism, after this investigation):

| Metric | Before (no GOMAXPROCS) | After (`GOMAXPROCS=1`) |
|---|---|---|
| `med` | 82.78ms | 54.07ms (-35%) |
| `p(95)` | 121.34ms | 93.43ms (-23%) |
| Throughput | 409.73 req/s | 483.36 req/s (+18%) |

Verified `go_sched_gomaxprocs_threads` reads `1` after the restart. Real, measurable
improvement — but `avg`/`med` stayed close together (54.36ms/54.07ms) and CPU stayed
near saturation (~78%), meaning this wasn't the whole story. See
[prepared_statements](../prepared_statements/README.md) for the second factor found
right after this one.
