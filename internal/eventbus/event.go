package eventbus

import (
	"time"

	"github.com/grachmannico95/flip-test-be/internal/domain"
)

type EventType string

const (
	EventTypeReconciliation EventType = "reconciliation"
)

type Event struct {
	ID        string      `json:"id"`
	Type      EventType   `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	Retries   int         `json:"retries"`
}

type ReconciliationEvent struct {
	UploadID    string             `json:"upload_id"`
	Transaction domain.Transaction `json:"transaction"`
	LineNumber  int                `json:"line_number"`
}
