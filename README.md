# Inventory take-home challenge

Synthetic inventory system: a batch job ingests stock-movement events into PostgreSQL, a read-only API exposes the data, and a React frontend consumes the API.

## Components

| Component | Location | Role |
|---|---|---|
| **Ingestion batch** | `inventory-takehome-Nicolas/` | Reads NDJSON events, validates them, persists movements, updates stock |
| **Read-only API** | `api-server/` | Serves current stock and movement history over HTTP |
| **Frontend** | `inventory-front/` | React + TypeScript UI for stock and movement history |
| **PostgreSQL** | Docker Compose | Shared database for ingestion writes and API reads |

## Architecture

```
Batch (ingest)  ──writes──▶  PostgreSQL  ◀──reads──  API  ◀──  Frontend
```

- The **batch** writes products, movements, and current stock.
- The **API** reads from the same database; it does not run ingestion.
- The **frontend** calls the API only.

Ingestion and the API are separate processes. They communicate only through PostgreSQL.

## How to run

From the repository root:

```bash
docker compose down -v
docker compose up --build
```

This starts PostgreSQL, the ingestion batch, the API, and the frontend.

- PostgreSQL must be healthy before ingestion and the API start.
- The API starts as soon as PostgreSQL is ready; it does **not** wait for ingestion to finish.
- The frontend starts after the API is available.

## URLs

| Service | URL |
|---|---|
| Frontend | http://localhost:5174 |
| API health | http://localhost:8080/health |
| Product stock | http://localhost:8080/products/stock |
| Product movements | http://localhost:8080/products/SKU-0001/movements?limit=100&offset=0 |

## Important behavior

- **Independent services** — Ingestion runs on its own schedule/lifecycle. The API is a long-running read service.
- **Reads during ingestion** — While the batch processes large NDJSON files, the API keeps serving queries from PostgreSQL.
- **Live stock updates** — The frontend may show partial or changing stock values as ingestion progresses and commits new movements.
- **Idempotent ingestion** — Re-running the batch is safe. `event_id` is the unique key; duplicates do not change stock twice.

## Optional stress dataset

Generate ~2M events across 20 files (from the repository root):

```bash
docker run --rm -v "$PWD/inventory-takehome-Nicolas:/app" -w /app golang:1.22 \
  go run ./tools/gen -n 2000000 -files 20
```

Then restart ingestion (or the full stack) to process the larger dataset.

## Useful curl commands

```bash
curl http://localhost:8080/health
curl http://localhost:8080/products/stock
curl "http://localhost:8080/products/SKU-0001/movements?limit=100&offset=0"
```

## Project layout

```
.
├── README.md
├── DECISIONS.md
├── docker-compose.yml
├── inventory-takehome-Nicolas/   # ingestion batch + schema + sample data
├── api-server/                     # read-only REST API
└── inventory-front/                # React + TypeScript UI
```

Design rationale and trade-offs are documented in [DECISIONS.md](./DECISIONS.md) (English).
