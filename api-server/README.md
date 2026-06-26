# Inventory API

Read-only REST API for querying product stock and movement history from PostgreSQL.

## Requirements

- Go 1.22
- PostgreSQL (same database used by the ingestion service)

## Getting started

Start PostgreSQL (from the ingestion project):

```bash
cd ../inventory-takehome-Nicolas
docker compose up -d
go run ./cmd/ingest
```

Run the API:

```bash
go run ./cmd/api
```

The server listens on `http://localhost:8080` by default.

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://takehome:takehome@localhost:5438/inventory?sslmode=disable` | PostgreSQL connection string |
| `ADDR` | `:8080` | HTTP listen address |

Copy `.env.example` to `.env` if you want to override defaults locally.

## Endpoints

### Health

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "ok"
}
```

### Product stock

```bash
curl http://localhost:8080/products/stock
```

```json
[
  {
    "sku": "SKU-0001",
    "name": "Product name",
    "quantity": 123
  }
]
```

### Product movements

```bash
curl "http://localhost:8080/products/SKU-0001/movements?limit=100&offset=0"
```

Query parameters:

- `limit` — default `100`, max `500`
- `offset` — default `0`

```json
[
  {
    "eventId": "evt-000001",
    "sku": "SKU-0001",
    "type": "IN",
    "quantity": 10,
    "occurredAt": "2026-06-01T02:12:46Z"
  }
]
```

## Error responses

Errors are returned as JSON:

```json
{
  "error": "message"
}
```

- `400` — invalid pagination values
- `404` — product not found
- `500` — unexpected database errors

## Project layout

```
cmd/api/main.go
internal/api/router.go
internal/api/handlers.go
internal/db/postgres.go
internal/repository/product_repository.go
internal/model/models.go
```
