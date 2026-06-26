# Design decisions

This document summarizes the main technical choices for the inventory take-home solution.

## 1. Main design decisions

I implemented **two backend services**:

1. A **batch ingestion service** (`inventory-takehome-Nicolas/cmd/ingest`) that processes NDJSON files and writes to PostgreSQL.
2. A **read-only API service** (`api-server/cmd/api`) that exposes stock and movement history.

Both services belong to the same inventory domain, so I kept **one PostgreSQL database**. The batch writes; the API reads. There is no direct HTTP or message-bus coupling between them.

I did **not implement SAGA**. There is no distributed transaction across services—only a local transaction per event inside the batch. At-least-once delivery is handled with idempotent inserts keyed by `event_id`, not with compensating actions across participants.

The batch and the API communicate **only through PostgreSQL**.

## 2. Ingestion design

- **Streaming reads** — NDJSON files are read line by line with `bufio.Scanner` so large files are never loaded fully into memory.
- **Concurrent file processing** — Multiple files are processed in parallel via a bounded file-worker pool.
- **Bounded persistence** — Valid events are written by a worker pool sized to match the database connection limit.
- **Connection pool** — `pgxpool` is configured with `MaxConns = 10` before the pool is created.
- **Idempotency** — `event_id` is the primary key on `inventory_movements`. Inserts use `ON CONFLICT (event_id) DO NOTHING`.
- **Stock updates** — `current_stock` is updated only when a movement row is actually inserted.
- **Invalid events** — Malformed or invalid lines are inserted into `ingestion_errors` and skipped; the batch continues.
- **Graceful shutdown** — `SIGINT` / `SIGTERM` cancels context; workers drain and connections close cleanly.

## 3. Negative stock decision

I allowed **negative stock** in `current_stock`.

The specification defines stock as the result of applying all valid movements (`IN` adds, `OUT` subtracts). The explicitly invalid cases are:

- malformed JSON
- negative `quantity`
- unknown `type`
- unknown `sku`

The spec does **not** state that an `OUT` movement causing negative stock is invalid. Preventing negative stock would be a business rule on top of the given requirements.

If that rule were required, I would enforce it atomically in the stock update (for example, `UPDATE ... WHERE quantity >= requested_quantity` and treat zero rows updated as a rejected movement).

## Alternative implementation: reject negative stock

I also implemented and validated an **alternative business rule** in a separate branch.

- The original implementation allows negative stock because the specification defines stock as the result of applying all valid inventory movements and does not explicitly state that an `OUT` movement should be rejected when stock is insufficient.
- In many real inventory systems, preventing negative stock is a common business rule.
- To validate this alternative interpretation, I implemented a version where an `OUT` movement is rejected if it would make the stock negative.
- The validation is performed atomically inside the same database transaction using an `UPDATE ... WHERE quantity >= requested_quantity`.
- If there is not enough stock:
  - the movement is not persisted,
  - the transaction is rolled back,
  - the event is recorded in `ingestion_errors`,
  - processing continues normally.

Results obtained with the sample dataset:

| Metric | Value |
|---|---|
| Inserted | 1,926 |
| Duplicates | 175 |
| Invalid | 81 |
| Rejected (insufficient stock) | 87 |

> This implementation is maintained as an alternative branch because the specification does not explicitly require non-negative stock. Both implementations are valid depending on whether inventory movements are interpreted as immutable events or as business commands that must be validated before being accepted.

## 4. API design

- The API is **read-only** — no ingestion or mutation endpoints.
- **Current stock** — `GET /products/stock` joins `products` and `current_stock`.
- **Movement history** — `GET /products/{sku}/movements` with `limit` / `offset` pagination (default limit 100, max 500).
- **Errors** — JSON error bodies; `404` when the SKU does not exist; `500` for unexpected database errors.
- **CORS** — Enabled for browser access from the React frontend.

For very large tables and deep pagination, **cursor / keyset pagination** (e.g. by `occurred_at, event_id`) would be a better future improvement than large `OFFSET` values.

## 5. Frontend design

- Minimal **React + TypeScript** UI.
- Lists products with **current stock**.
- Selecting a product loads its **movement history** from the API.
- **Visual polish was intentionally not prioritized** — focus is on correct API consumption and idiomatic TypeScript.

The frontend reads from the API only. It reflects whatever stock is currently in PostgreSQL, including partial results while ingestion is still running.

## 6. Trade-offs and limitations

| Area | Choice | Limitation |
|---|---|---|
| Throughput | One transaction per event | Correct and simple, but not maximum ingest throughput |
| File reading | `bufio.Scanner` with 1 MB max line | Fine for this dataset; `bufio.Reader` is safer for unbounded line lengths |
| Pagination | `LIMIT` / `OFFSET` | Simple; slow for very deep pages |
| Testing | Manual + stress run | More automated integration tests would improve confidence |
| Observability | Standard logs | Structured logs and metrics would help in production |
| Checkpoints | None | Re-run re-processes all files (safe due to idempotency) |

**Possible future improvements:**

- Batch inserts or PostgreSQL `COPY` for higher ingest throughput.
- Dedicated error-writer pool if invalid-line volume grows.
- Versioned migrations (e.g. golang-migrate) instead of idempotent `CREATE IF NOT EXISTS`.
- Resume / watermark after interrupted ingestion runs.

## 7. Stress test result

Dataset generated with:

```bash
go run ./tools/gen -n 2000000 -files 20
```

Ingestion results:

| Metric | Value |
|---|---|
| Inserted (unique events) | 2,000,000 |
| Duplicates | 159,972 |
| Invalid lines | 59,763 |
| Total ingestion time | 358.86s |

**Re-run behavior:**

- Inserted: **0**
- All valid events treated as **duplicates**
- Stock **unchanged**

This confirms idempotency under at-least-once delivery at scale.

## 8. AI usage

AI was used to speed up **scaffolding, documentation, Docker setup, and review** of implementation choices.

I reviewed and adjusted the generated output, especially around:

- **Idempotency** — stock must change only on first insert of an `event_id`.
- **Database concurrency** — pool size and worker counts must stay aligned.
- **Docker Compose dependencies** — API and frontend available while ingestion is still running.
- **Negative stock** — aligned with the spec’s definition of stock vs. invalid events.

All code, identifiers, and comments remain in English as required.
