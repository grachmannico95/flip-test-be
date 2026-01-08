package domain

import "context"

type Repository interface {
	// Upload management
	CreateUpload(ctx context.Context, uploadID string) error
	GetUpload(ctx context.Context, uploadID string) (*Upload, error)
	UpdateUploadStatus(ctx context.Context, uploadID string, status UploadStatus) error
	IncrementProcessedRows(ctx context.Context, uploadID string) error

	// Transaction operations
	AddTransaction(ctx context.Context, uploadID string, tx Transaction, lineNumber int) error
	GetBalance(ctx context.Context, uploadID string) (int64, error)
	GetIssues(ctx context.Context, uploadID string, page, perPage int, status *TransactionStatus) ([]IssueTransaction, int, error)

	// Idempotency tracking
	IsEventProcessed(ctx context.Context, eventID string) (bool, error)
	MarkEventProcessed(ctx context.Context, eventID string) error
}
