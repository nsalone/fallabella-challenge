package ingest

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"takehome/internal/db"
)

const (
	fileWorkers  = 4
	persistQueue = 256
)

type Config struct {
	DatabaseURL  string
	ProductsPath string
	EventsDir    string
}

type Stats struct {
	InvalidLines    atomic.Int64
	InsertedEvents  atomic.Int64
	DuplicateEvents atomic.Int64
}

func Run(ctx context.Context, pool *pgxpool.Pool, cfg Config) (Stats, error) {
	var stats Stats
	start := time.Now()

	log.Println("ingestion started")

	knownSKUs, err := loadProducts(ctx, pool, cfg.ProductsPath)
	if err != nil {
		return stats, fmt.Errorf("load products: %w", err)
	}
	log.Printf("products loaded: %d from %s", len(knownSKUs), cfg.ProductsPath)

	files, err := filepath.Glob(filepath.Join(cfg.EventsDir, "*.ndjson"))
	if err != nil {
		return stats, fmt.Errorf("list event files: %w", err)
	}
	log.Printf("files discovered: %d in %s", len(files), cfg.EventsDir)

	if len(files) == 0 {
		logSummary(start, &stats)
		return stats, nil
	}

	persistCh := make(chan Event, persistQueue)
	fileJobs := make(chan string, len(files))

	var fileWG sync.WaitGroup
	workers := fileWorkers
	if len(files) < workers {
		workers = len(files)
	}

	for i := 0; i < workers; i++ {
		fileWG.Add(1)
		go func() {
			defer fileWG.Done()
			processFiles(ctx, pool, fileJobs, persistCh, knownSKUs, &stats)
		}()
	}

	var dbWG sync.WaitGroup
	for i := 0; i < db.MaxConns; i++ {
		dbWG.Add(1)
		go func() {
			defer dbWG.Done()
			persistEvents(ctx, pool, persistCh, &stats)
		}()
	}

enqueue:
	for _, file := range files {
		select {
		case <-ctx.Done():
			break enqueue
		case fileJobs <- file:
		}
	}
	close(fileJobs)
	fileWG.Wait()
	close(persistCh)
	dbWG.Wait()

	logSummary(start, &stats)

	if err := ctx.Err(); err != nil {
		return stats, err
	}
	return stats, nil
}

func logSummary(start time.Time, stats *Stats) {
	log.Printf("Ingestion completed in %.2fs", time.Since(start).Seconds())
	log.Printf("Inserted: %d", stats.InsertedEvents.Load())
	log.Printf("Duplicates: %d", stats.DuplicateEvents.Load())
	log.Printf("Invalid: %d", stats.InvalidLines.Load())
}

func processFiles(
	ctx context.Context,
	pool *pgxpool.Pool,
	fileJobs <-chan string,
	persistCh chan<- Event,
	knownSKUs map[string]struct{},
	stats *Stats,
) {
	for filePath := range fileJobs {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := processFile(ctx, pool, filePath, persistCh, knownSKUs, stats); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("file failed: %s: %v", filePath, err)
		}
	}
}

func processFile(
	ctx context.Context,
	pool *pgxpool.Pool,
	filePath string,
	persistCh chan<- Event,
	knownSKUs map[string]struct{},
	stats *Stats,
) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	fileName := filepath.Base(filePath)
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNumber := int64(0)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}

		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		event, err := parseEvent(line, knownSKUs)
		if err != nil {
			stats.InvalidLines.Add(1)
			if recordErr := recordIngestionError(ctx, pool, fileName, lineNumber, line, err.Error()); recordErr != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return fmt.Errorf("record ingestion error at line %d: %w", lineNumber, recordErr)
			}
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case persistCh <- event:
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan file: %w", err)
	}

	log.Printf("file completed: %s", fileName)
	return nil
}

func persistEvents(ctx context.Context, pool *pgxpool.Pool, persistCh <-chan Event, stats *Stats) {
	for event := range persistCh {
		if err := ctx.Err(); err != nil {
			return
		}

		inserted, err := persistEvent(ctx, pool, event)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("persist event %s failed: %v", event.EventID, err)
			continue
		}
		if inserted {
			stats.InsertedEvents.Add(1)
		} else {
			stats.DuplicateEvents.Add(1)
		}
	}
}

func persistEvent(ctx context.Context, pool *pgxpool.Pool, event Event) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	var insertedEventID string
	err = tx.QueryRow(ctx, `
		INSERT INTO inventory_movements (event_id, sku, type, quantity, occurred_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id) DO NOTHING
		RETURNING event_id
	`, event.EventID, event.SKU, event.Type, event.Quantity, event.OccurredAt).Scan(&insertedEventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("insert movement: %w", err)
	}

	delta := event.Quantity
	if event.Type == EventTypeOut {
		delta = -event.Quantity
	}

	if _, err := tx.Exec(ctx, `
		UPDATE current_stock
		SET quantity = quantity + $1
		WHERE sku = $2
	`, delta, event.SKU); err != nil {
		return false, fmt.Errorf("update current stock: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit transaction: %w", err)
	}
	return true, nil
}

func recordIngestionError(ctx context.Context, pool *pgxpool.Pool, fileName string, lineNumber int64, rawLine, message string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO ingestion_errors (file_name, line_number, raw_line, error_message)
		VALUES ($1, $2, $3, $4)
	`, fileName, lineNumber, rawLine, message)
	if err != nil {
		return fmt.Errorf("insert ingestion error: %w", err)
	}
	return nil
}
