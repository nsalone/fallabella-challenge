# Inventory Front API

React + TypeScript frontend for the inventory API.

## Run

```bash
npm install
npm run dev
```

Default URL:

```text
http://localhost:5173
```

The frontend calls:

- `GET /health`
- `GET /products/stock`
- `GET /products/{sku}/movements?limit=100&offset=0`

By default it uses:

```text
http://localhost:8080
```
