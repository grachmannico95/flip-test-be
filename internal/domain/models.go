package domain

import "time"

type TransactionType string

const (
	TransactionTypeCredit TransactionType = "CREDIT"
	TransactionTypeDebit  TransactionType = "DEBIT"
)

type TransactionStatus string

const (
	TransactionStatusSuccess TransactionStatus = "SUCCESS"
	TransactionStatusFailed  TransactionStatus = "FAILED"
	TransactionStatusPending TransactionStatus = "PENDING"
)

type Transaction struct {
	Timestamp    int64             `json:"timestamp"`
	Counterparty string            `json:"counterparty"`
	Type         TransactionType   `json:"type"`
	Amount       int64             `json:"amount"`
	Status       TransactionStatus `json:"status"`
	Description  string            `json:"description"`
}

type UploadStatus string

const (
	UploadStatusProcessing UploadStatus = "processing"
	UploadStatusCompleted  UploadStatus = "completed"
	UploadStatusFailed     UploadStatus = "failed"
)

type Upload struct {
	ID            string       `json:"id"`
	Status        UploadStatus `json:"status"`
	ProcessedRows int          `json:"processed_rows"`
	TotalRows     int          `json:"total_rows"`
	CreatedAt     time.Time    `json:"created_at"`
	CompletedAt   *time.Time   `json:"completed_at,omitempty"`
}

type IssueTransaction struct {
	Transaction
	LineNumber int `json:"line_number"`
}
