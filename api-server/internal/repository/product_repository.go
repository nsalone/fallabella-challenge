package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"api-server/internal/model"
)

type ProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

func (r *ProductRepository) ListStock(ctx context.Context) ([]model.ProductStock, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			p.sku,
			p.name,
			COALESCE(cs.quantity, 0) AS quantity
		FROM products p
		LEFT JOIN current_stock cs ON cs.sku = p.sku
		ORDER BY p.sku
	`)
	if err != nil {
		return nil, fmt.Errorf("query product stock: %w", err)
	}
	defer rows.Close()

	products := make([]model.ProductStock, 0)
	for rows.Next() {
		var product model.ProductStock
		if err := rows.Scan(&product.SKU, &product.Name, &product.Quantity); err != nil {
			return nil, fmt.Errorf("scan product stock: %w", err)
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product stock: %w", err)
	}
	return products, nil
}

func (r *ProductRepository) ProductExists(ctx context.Context, sku string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM products WHERE sku = $1)
	`, sku).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check product exists: %w", err)
	}
	return exists, nil
}

func (r *ProductRepository) ListMovements(ctx context.Context, sku string, limit, offset int) ([]model.Movement, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			event_id,
			sku,
			type,
			quantity,
			occurred_at
		FROM inventory_movements
		WHERE sku = $1
		ORDER BY occurred_at DESC
		LIMIT $2 OFFSET $3
	`, sku, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query movements: %w", err)
	}
	defer rows.Close()

	movements := make([]model.Movement, 0)
	for rows.Next() {
		var (
			movement   model.Movement
			occurredAt time.Time
		)
		if err := rows.Scan(
			&movement.EventID,
			&movement.SKU,
			&movement.Type,
			&movement.Quantity,
			&occurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		movement.OccurredAt = occurredAt.UTC().Format(time.RFC3339)
		movements = append(movements, movement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movements: %w", err)
	}
	return movements, nil
}
