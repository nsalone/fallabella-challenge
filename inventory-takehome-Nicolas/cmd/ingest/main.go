package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"takehome/internal/db"
	"takehome/internal/ingest"
)

const defaultDatabaseURL = "postgres://takehome:takehome@localhost:5438/inventory?sslmode=disable"

func main() {
	log.SetFlags(log.LstdFlags)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = defaultDatabaseURL
	}

	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("resolve working directory: %v", err)
	}

	cfg := ingest.Config{
		DatabaseURL:  databaseURL,
		ProductsPath: filepath.Join(root, "data", "products.csv"),
		EventsDir:    filepath.Join(root, "data", "events"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutdown requested")
		cancel()
	}()

	pool, err := newPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	if _, err := ingest.Run(ctx, pool, cfg); err != nil && ctx.Err() == nil {
		log.Fatalf("ingestion failed: %v", err)
	}

	if ctx.Err() != nil {
		os.Exit(1)
	}
}

func newPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.MaxConns = db.MaxConns

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
