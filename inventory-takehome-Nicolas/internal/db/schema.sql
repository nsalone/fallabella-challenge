CREATE TABLE IF NOT EXISTS products (
    sku  TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS inventory_movements (
    event_id    TEXT PRIMARY KEY,
    sku         TEXT NOT NULL REFERENCES products (sku),
    type        TEXT NOT NULL CHECK (type IN ('IN', 'OUT')),
    quantity    BIGINT NOT NULL CHECK (quantity > 0),
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_inventory_movements_sku_occurred_at
    ON inventory_movements (sku, occurred_at DESC);

CREATE TABLE IF NOT EXISTS current_stock (
    sku      TEXT PRIMARY KEY REFERENCES products (sku),
    quantity BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS ingestion_errors (
    id            BIGSERIAL PRIMARY KEY,
    file_name     TEXT NOT NULL,
    line_number   BIGINT NOT NULL,
    raw_line      TEXT NOT NULL,
    error_message TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
