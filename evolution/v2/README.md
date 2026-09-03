# Version 2 — asynchronous processing via Kafka + transactional outbox

**Status: built and validated end-to-end for the first time.** Two blocking bugs (see
"Known gaps" below) meant the async pipeline had never actually settled a payment
before this round of validation. It now does, but the first real settlement numbers
show a large new bottleneck (likely the Kafka consumer group setup) still to
investigate. This is the architectural response to v1's
[throughput ceiling](../v1/throughput_ceiling/README.md): a pool of DB connections
capped at a realistic size (30) could sustain ~810 req/s of full synchronous
processing, short of the SLO's 1000 req/s target, with no tuning fix available. v2
decouples request acceptance from payment processing so the two can scale
independently.

## Architecture

Three separate processes now exist (`docker-compose.yaml`), each independently
resource-limited:

- **`web`** (`cmd/main.go`) — HTTP API. `POST /payments` no longer touches
  `accounts`/`payments`/`ledger` at all. It only: checks the idempotency key, and if
  new, persists the request and enqueues it — both in **one Postgres transaction**
  (`SaveIdempotencyKey`, `payment.repository.go`). Responds without waiting for the
  payment to actually be processed.
- **`outbox_worker`** (`cmd/worker/outbox/main.go`) — polls `payments_outbox` for
  `status='pending'` rows (`SELECT ... FOR UPDATE SKIP LOCKED`, batched via
  `OUTBOX_BATCH_SIZE`, on a `OUTBOX_POLL_INTERVAL`-second ticker) and publishes each to
  Kafka, marking it `processed` in the same transaction as the publish attempt.
- **`workers`** (`cmd/worker/main.go`) — Kafka consumer group, decodes each
  message and calls the same debit/credit/ledger transaction v1 always had
  (`repository.Save`) — the actual money movement logic didn't change, only what
  triggers it.

### Why transactional outbox instead of publishing directly from `web`

Publishing to Kafka directly from the HTTP handler would create a dual-write problem:
the Postgres write and the Kafka publish are two separate systems with no shared
transaction, so a crash between them either loses the message (DB committed, publish
never happened) or double-processes it (publish happened, DB write rolled back, client
retries). Writing the intent to `payments_outbox` in the *same* transaction as the
idempotency check makes the whole "accept" step atomic — a message is published if and
only if the accept was durably committed. The `outbox_worker` is what actually talks to
Kafka, on a separate poll loop, decoupled from request latency.

### Idempotency

Two layers, per the earlier design discussion:

- **Acceptance-layer** (`idempotency_keys` table, `SaveIdempotencyKey`): a client-supplied
  `idempotency_key` plus a hash of the request body. Same key + same payload → replay
  the previous outcome. Same key + different payload → `409` (misuse). Key already
  `processing` → `409`. Previous attempt `error` → `422`.
- **Processing-layer**: relies on `payments.id` being a `PRIMARY KEY` generated at
  accept time — a duplicate delivery attempting to re-insert the same payment id would
  hit a unique-violation rather than double-processing. (Not yet exercised by an actual
  retry path — see gaps below.)
- **Write-back**: `CreatePaymentUseCase.Execute` now passes the original
  `IdempotencyKey` through to `Save`, which marks the key `completed` (with
  `payment_id`) **inside the same transaction** as the debit/credit — the key can never
  end up `completed` without the payment having actually committed, or vice versa. On
  any failure the transaction rolls back and the key is marked `error` in a separate,
  best-effort statement (`markIdempotencyKeyError`), since the main transaction is
  already gone by then. That statement itself is not retried — see the residual risk
  noted under "No retry or DLQ" below.

### Partitioning

The outbox worker publishes keyed by `from` account (`publisher.Publish(payload,
body.From)`). As discussed before implementing this: a single message can only carry
one partition key, so no choice of key can guarantee ordering across *both* sides of a
transfer — correctness (no double-spend, no lost update) is guaranteed independently by
Postgres row locks in `Save`, not by message ordering. Partitioning by `from` was chosen
to preserve the ordering of each account's own outgoing payments and to distribute load
across partitions, accepting that a debit can occasionally race a very recent credit to
the same account from a different counterparty.

## Current configuration

| Setting | Value |
|---|---|
| `QUEUE_TOPIC` | `finance.payments.payment.create.v1` |
| `QUEUE_TOPIC_MAX_PARTITIONS` | 3 |
| `QUEUE_MAX_CONSUMERS` | 5 (parallel fetch loops per `workers` process, same consumer group) |
| `OUTBOX_POLL_INTERVAL` | 3s |
| `OUTBOX_BATCH_SIZE` | 100 |
| `web` limits | `cpus: 1`, `memory: 512M` |
| `workers` limits | `cpus: 1`, `memory: 512M` |
| `outbox_worker` limits | `cpus: 0.5`, `memory: 96M` |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` | 30 / 10 (unchanged from v1's SLO baseline; each process has its own pool against the same Postgres instance) |

Migrations now run inside a transaction holding `pg_advisory_xact_lock(8675309)`
(`persistence/db.go`), since `web`, `workers` and `outbox_worker` can all start
concurrently and would otherwise race on schema creation.

## Known gaps (before drawing any throughput conclusion)

- **No retry or DLQ.** `consumer.processRecord` logs and drops the message on any
  handler error — a transient DB error or an insufficient-funds rejection are currently
  indistinguishable and both just vanish after being logged (now leaving the idempotency
  key correctly marked `error`, at least, per the fix above — but nothing acts on that).
  The retry/backoff + DLQ design discussed earlier (business rejection vs. transient
  error vs. exhausted-retries) isn't implemented yet.

## SLO, reframed for v2

The [SLO](../README.md#slo) (p95<300ms, p99<750ms, error rate<0.1%, 1000 req/s) was
defined against a synchronous request/response cycle. It now needs to split in two,
per the earlier design discussion:

- **Acceptance** (what `POST /payments` measures): should be very fast and largely
  decoupled from load, since it's just an idempotency check + one insert. This is what
  `scripts/k6/load_payments.js` measures directly.
- **Settlement** (real business outcome — debited, credited, or rejected): not
  observable from the HTTP response anymore. Needs the processing-side metrics listed
  above before it can be measured at all, let alone compared against a target.
