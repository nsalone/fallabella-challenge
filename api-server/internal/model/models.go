package model

type ProductStock struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

type Movement struct {
	EventID    string `json:"eventId"`
	SKU        string `json:"sku"`
	Type       string `json:"type"`
	Quantity   int64  `json:"quantity"`
	OccurredAt string `json:"occurredAt"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type HealthResponse struct {
	Status string `json:"status"`
}
