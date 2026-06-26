package ingest

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	EventTypeIn  = "IN"
	EventTypeOut = "OUT"

	insufficientStockError = "insufficient stock for OUT movement"
)

type rawEvent struct {
	EventID    string `json:"event_id"`
	SKU        string `json:"sku"`
	Type       string `json:"type"`
	Quantity   int64  `json:"quantity"`
	OccurredAt string `json:"occurred_at"`
}

type Event struct {
	EventID    string
	SKU        string
	Type       string
	Quantity   int64
	OccurredAt time.Time
}

type persistJob struct {
	Event
	FileName   string
	LineNumber int64
	RawLine    string
}

type PersistResult int

const (
	PersistInserted PersistResult = iota
	PersistDuplicate
	PersistRejectedInsufficientStock
)

func parseEvent(line string, knownSKUs map[string]struct{}) (Event, error) {
	var raw rawEvent
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return Event{}, fmt.Errorf("invalid JSON: %w", err)
	}

	raw.EventID = strings.TrimSpace(raw.EventID)
	raw.SKU = strings.TrimSpace(raw.SKU)
	raw.Type = strings.TrimSpace(raw.Type)

	if raw.EventID == "" {
		return Event{}, fmt.Errorf("event_id is required")
	}
	if raw.SKU == "" {
		return Event{}, fmt.Errorf("sku is required")
	}
	if _, ok := knownSKUs[raw.SKU]; !ok {
		return Event{}, fmt.Errorf("unknown sku %q", raw.SKU)
	}
	if raw.Type != EventTypeIn && raw.Type != EventTypeOut {
		return Event{}, fmt.Errorf("type must be IN or OUT, got %q", raw.Type)
	}
	if raw.Quantity <= 0 {
		return Event{}, fmt.Errorf("quantity must be greater than 0")
	}

	occurredAt, err := time.Parse(time.RFC3339, raw.OccurredAt)
	if err != nil {
		return Event{}, fmt.Errorf("occurred_at must be RFC3339: %w", err)
	}

	return Event{
		EventID:    raw.EventID,
		SKU:        raw.SKU,
		Type:       raw.Type,
		Quantity:   raw.Quantity,
		OccurredAt: occurredAt,
	}, nil
}
