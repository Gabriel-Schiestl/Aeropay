# Hypothesis: no prepared-statement reuse across requests

**Stated hypothesis:** after fixing [GOMAXPROCS](../gomaxprocs_mismatch/README.md), CPU
stayed near saturation and `avg`≈`med` persisted at 100 VUs — candidate cause: the
repository re-prepares every SQL statement from scratch on every request instead of
reusing prepared statements across the connection pool.

## Verdict: CONFIRMED

## Evidence

Code inspection of `payment.repository.go` (before this fix) showed every call to
`Save()` doing:

`BeginTx` → `PrepareContext`+query (lock account 1) → `PrepareContext`+query (lock
account 2) → `PrepareContext`+exec (insert payment) → `PrepareContext`+exec (debit) →
`PrepareContext`+exec (ledger row) → `PrepareContext`+exec (credit) →
`PrepareContext`+exec (ledger row) → `Commit`

Seven `tx.PrepareContext(ctx, sqlString)` calls per payment, each creating a brand-new,
uniquely-named server-side prepared statement and immediately deallocating it via
`defer stmt.Close()` — regardless of whether the connection serving that request had
already prepared the identical query moments before. ~16 sequential round trips to
Postgres per payment, with zero reuse.

Metrics right after the GOMAXPROCS fix (100 VUs / 30s, pool=40), before this fix:

| Metric | Value |
|---|---|
| `med` | 54.07ms |
| `avg` | 54.36ms (ratio ≈ 1.0 — still converged with `med`) |
| CPU utilization during test | ~78% (48.33 core-s / ~62s) — barely down from ~82% pre-GOMAXPROCS fix |

CPU staying almost as high as before, despite GOMAXPROCS already being fixed, meant the
pressure was mostly real repeated per-request work, not scheduling overhead.

## Explanation

Go's `database/sql` has a mechanism for exactly this: a `*sql.Stmt` obtained from
`db.PrepareContext` (on the `*sql.DB`, pool-aware) keeps an internal per-connection
cache (`Stmt.css`, verified directly in `$GOROOT/src/database/sql/sql.go`). Calling
`tx.StmtContext(ctx, dbStmt)` checks that cache first and only prepares a fresh
server-side statement on a connection the *first* time that connection is used with it —
capping total Prepare cost at (5 unique queries × pool size), never per-request.

The old code bypassed this entirely by calling `tx.PrepareContext(ctx, sqlString)`
directly with a raw string instead of deriving from a pre-existing `*sql.Stmt` — so
`database/sql` had no cache to consult and prepared+deallocated from scratch every time,
on every connection, forever.

## Fix applied

`NewPaymentRepository` now prepares the 5 unique statements once, via
`db.PrepareContext`, storing them as `*sql.Stmt` fields on the repository. `Save`,
`lockAccount`, `debitAccount` and `creditAccount` now get a transaction-scoped handle via
`tx.StmtContext(ctx, preparedStmt)` instead of re-preparing from a string. Manual
`.Close()` calls were removed — statements obtained via `StmtContext` close automatically
when the transaction commits or rolls back. Code-only change, stays within v1.

(A follow-up question about this fix — whether prepared statements aren't inherently
per-connection, and so can't just be "shared" outright — is answered by the explanation
above: they are per-connection at the Postgres protocol level, and the fix works by
amortizing that per-connection cost across the pool via `database/sql`'s own cache,
not by pretending connections don't matter.)

## Confirmation

Re-ran the same 100 VUs / 30s test (pool still 40 at this point):

| Metric | After GOMAXPROCS only | After this fix too |
|---|---|---|
| `med` | 54.07ms | 17.36ms (-68%) |
| `avg` | 54.36ms | 42.06ms |
| `avg`/`med` ratio | ≈ 1.0 | ≈ 2.4 |
| Throughput | 483.36 req/s | 515.31 req/s (+7%) |
| CPU utilization | ~78% | ~51% |

The `avg`/`med` ratio going back up is the interesting result: it means the "typical
fast + queueing tail" shape (the same signature as
[pool_bottleneck](../pool_bottleneck/README.md)) reappeared cleanly — CPU pressure and
per-request overhead are no longer masking it. At 100 VUs against a 40-connection pool,
the pool itself is now the sole clearly visible bottleneck, confirmed independently via
`go_sql_wait_count_total` = 6123 waits / 256.5s cumulative wait on that same run. (The
pool was later lowered to 30 for [SLO](../../README.md#slo) realism, after this
investigation.)
