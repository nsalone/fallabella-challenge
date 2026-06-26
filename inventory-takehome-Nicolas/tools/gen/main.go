// Command gen produces synthetic stock-movement event data for the take-home.
//
// It writes:
//   - <out>/products.csv          : the product catalog (sku,name)
//   - <out>/events/part-NNN.ndjson: movement events, one JSON object per line
//
// Two properties are intentional and part of the exercise:
//   - A fraction of events are re-delivered (same event_id appears more than
//     once), simulating an at-least-once event stream.
//   - A small fraction of lines are malformed or hold invalid data, so the
//     ingest has to validate and skip them without aborting the whole run.
//
// Usage:
//
//	go run ./tools/gen                 # small sample (default)
//	go run ./tools/gen -n 2000000 -files 20   # large dataset for perf testing
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type event struct {
	EventID    string `json:"event_id"`
	SKU        string `json:"sku"`
	Type       string `json:"type"` // IN or OUT
	Quantity   int    `json:"quantity"`
	OccurredAt string `json:"occurred_at"`
}

func main() {
	out := flag.String("out", "data", "output directory")
	total := flag.Int("n", 2000, "approximate number of valid events to generate")
	files := flag.Int("files", 4, "number of event files to split across")
	products := flag.Int("products", 8, "number of products in the catalog")
	dupRate := flag.Float64("dup", 0.08, "fraction of events that are re-delivered (at-least-once)")
	badRate := flag.Float64("bad", 0.03, "fraction of lines that are malformed/invalid")
	seed := flag.Int64("seed", 42, "random seed (for reproducible data)")
	flag.Parse()

	rng := rand.New(rand.NewSource(*seed))

	if err := os.MkdirAll(filepath.Join(*out, "events"), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(1)
	}

	// --- product catalog ---------------------------------------------------
	skus := make([]string, *products)
	pf, err := os.Create(filepath.Join(*out, "products.csv"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "create products.csv:", err)
		os.Exit(1)
	}
	pw := bufio.NewWriter(pf)
	fmt.Fprintln(pw, "sku,name")
	names := []string{"Caja chica", "Caja grande", "Pallet estandar", "Film stretch",
		"Etiqueta termica", "Cinta de embalaje", "Separador carton", "Esquinero plastico",
		"Bolsa kraft", "Precinto seguridad"}
	for i := 0; i < *products; i++ {
		sku := fmt.Sprintf("SKU-%04d", i+1)
		skus[i] = sku
		name := names[i%len(names)]
		fmt.Fprintf(pw, "%s,%s\n", sku, name)
	}
	pw.Flush()
	pf.Close()

	// --- event writers (round-robin across files) --------------------------
	writers := make([]*bufio.Writer, *files)
	handles := make([]*os.File, *files)
	for i := 0; i < *files; i++ {
		f, err := os.Create(filepath.Join(*out, "events", fmt.Sprintf("part-%03d.ndjson", i)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "create event file:", err)
			os.Exit(1)
		}
		handles[i] = f
		writers[i] = bufio.NewWriter(f)
	}

	now := time.Now().UTC()
	emit := func(idx int, line string) { fmt.Fprintln(writers[idx], line) }

	writeEvent := func(idx int, e event) {
		b, _ := json.Marshal(e)
		emit(idx, string(b))
	}

	// Keep a pool of previously emitted events so we can re-deliver some.
	var emitted []event
	lineIdx := 0
	nextFile := func() int { f := lineIdx % *files; lineIdx++; return f }

	makeEvent := func(i int) event {
		typ := "IN"
		if rng.Float64() < 0.5 {
			typ = "OUT"
		}
		return event{
			EventID:    fmt.Sprintf("evt-%08d", i),
			SKU:        skus[rng.Intn(len(skus))],
			Type:       typ,
			Quantity:   1 + rng.Intn(50),
			OccurredAt: now.Add(-time.Duration(rng.Intn(30*24*60)) * time.Minute).Format(time.RFC3339),
		}
	}

	badLine := func(i int) string {
		switch rng.Intn(4) {
		case 0: // truncated / invalid JSON
			return fmt.Sprintf(`{"event_id":"evt-bad-%08d","sku":"%s","type":"IN","quantity":`, i, skus[rng.Intn(len(skus))])
		case 1: // negative quantity
			e := makeEvent(i)
			e.EventID = fmt.Sprintf("evt-bad-%08d", i)
			e.Quantity = -1 * (1 + rng.Intn(10))
			b, _ := json.Marshal(e)
			return string(b)
		case 2: // unknown movement type
			e := makeEvent(i)
			e.EventID = fmt.Sprintf("evt-bad-%08d", i)
			e.Type = "UNKNOWN"
			b, _ := json.Marshal(e)
			return string(b)
		default: // unknown sku
			e := makeEvent(i)
			e.EventID = fmt.Sprintf("evt-bad-%08d", i)
			e.SKU = "SKU-9999"
			b, _ := json.Marshal(e)
			return string(b)
		}
	}

	valid, dups, bad := 0, 0, 0
	for i := 0; i < *total; i++ {
		// occasionally inject a malformed line
		if rng.Float64() < *badRate {
			emit(nextFile(), badLine(i))
			bad++
		}
		e := makeEvent(i)
		writeEvent(nextFile(), e)
		emitted = append(emitted, e)
		valid++

		// occasionally re-deliver a previously emitted event (same event_id)
		if len(emitted) > 0 && rng.Float64() < *dupRate {
			dup := emitted[rng.Intn(len(emitted))]
			writeEvent(nextFile(), dup)
			dups++
		}
	}

	for i := range writers {
		writers[i].Flush()
		handles[i].Close()
	}

	fmt.Printf("products: %d\n", *products)
	fmt.Printf("valid events: %d\n", valid)
	fmt.Printf("re-delivered (duplicate event_id): %d\n", dups)
	fmt.Printf("malformed/invalid lines: %d\n", bad)
	fmt.Printf("files: %d  ->  %s\n", *files, filepath.Join(*out, "events"))
}
