package ingest

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func loadProducts(ctx context.Context, pool *pgxpool.Pool, path string) (map[string]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open products file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read products header: %w", err)
	}
	if len(header) < 2 || header[0] != "sku" || header[1] != "name" {
		return nil, fmt.Errorf("unexpected products header: %v", header)
	}

	knownSKUs := make(map[string]struct{})

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin products transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read products row: %w", err)
		}
		if len(record) < 2 {
			return nil, fmt.Errorf("invalid products row: %v", record)
		}

		sku, name := record[0], record[1]
		if sku == "" {
			return nil, fmt.Errorf("empty sku in products file")
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO products (sku, name) VALUES ($1, $2)
			ON CONFLICT (sku) DO UPDATE SET name = EXCLUDED.name
		`, sku, name); err != nil {
			return nil, fmt.Errorf("upsert product %q: %w", sku, err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO current_stock (sku, quantity) VALUES ($1, 0)
			ON CONFLICT (sku) DO NOTHING
		`, sku); err != nil {
			return nil, fmt.Errorf("ensure stock row for %q: %w", sku, err)
		}

		knownSKUs[sku] = struct{}{}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit products: %w", err)
	}

	return knownSKUs, nil
}
