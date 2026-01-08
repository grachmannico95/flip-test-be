package storage

import (
	"context"
	"sync"
	"time"

	"github.com/grachmannico95/flip-test-be/internal/domain"
)

type TransactionWithLine struct {
	Transaction domain.Transaction
	LineNumber  int
}

type MemoryStore struct {
	uploads         map[string]*domain.Upload
	transactions    map[string][]TransactionWithLine
	processedEvents map[string]bool
	mu              sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		uploads:         make(map[string]*domain.Upload),
		transactions:    make(map[string][]TransactionWithLine),
		processedEvents: make(map[string]bool),
	}
}

func (s *MemoryStore) CreateUpload(ctx context.Context, uploadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.uploads[uploadID] = &domain.Upload{
		ID:            uploadID,
		Status:        domain.UploadStatusProcessing,
		ProcessedRows: 0,
		TotalRows:     0,
		CreatedAt:     time.Now(),
	}

	s.transactions[uploadID] = []TransactionWithLine{}

	return nil
}

func (s *MemoryStore) GetUpload(ctx context.Context, uploadID string) (*domain.Upload, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	upload, exists := s.uploads[uploadID]
	if !exists {
		return nil, domain.ErrUploadNotFound
	}

	return upload, nil
}

func (s *MemoryStore) UpdateUploadStatus(ctx context.Context, uploadID string, status domain.UploadStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, exists := s.uploads[uploadID]
	if !exists {
		return domain.ErrUploadNotFound
	}

	upload.Status = status
	if status == domain.UploadStatusCompleted || status == domain.UploadStatusFailed {
		now := time.Now()
		upload.CompletedAt = &now
	}

	return nil
}

func (s *MemoryStore) IncrementProcessedRows(ctx context.Context, uploadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, exists := s.uploads[uploadID]
	if !exists {
		return domain.ErrUploadNotFound
	}

	upload.ProcessedRows++

	return nil
}

func (s *MemoryStore) AddTransaction(ctx context.Context, uploadID string, tx domain.Transaction, lineNumber int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.uploads[uploadID]
	if !exists {
		return domain.ErrUploadNotFound
	}

	s.transactions[uploadID] = append(s.transactions[uploadID], TransactionWithLine{
		Transaction: tx,
		LineNumber:  lineNumber,
	})

	return nil
}

func (s *MemoryStore) GetBalance(ctx context.Context, uploadID string) (int64, error) {
	// Balance = sum of CREDIT (+) and DEBIT (-) from SUCCESS transactions only

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.uploads[uploadID]
	if !exists {
		return 0, domain.ErrUploadNotFound
	}

	transactions, exists := s.transactions[uploadID]
	if !exists {
		return 0, nil
	}

	var balance int64
	for _, txWithLine := range transactions {
		tx := txWithLine.Transaction
		if tx.Status == domain.TransactionStatusSuccess {
			if tx.Type == domain.TransactionTypeCredit {
				balance += tx.Amount
			} else if tx.Type == domain.TransactionTypeDebit {
				balance -= tx.Amount
			}
		}
	}

	return balance, nil
}

func (s *MemoryStore) GetIssues(ctx context.Context, uploadID string, page, perPage int, status *domain.TransactionStatus) ([]domain.IssueTransaction, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.uploads[uploadID]
	if !exists {
		return nil, 0, domain.ErrUploadNotFound
	}

	transactions, exists := s.transactions[uploadID]
	if !exists {
		return []domain.IssueTransaction{}, 0, nil
	}

	var filtered []domain.IssueTransaction
	for _, txWithLine := range transactions {
		tx := txWithLine.Transaction

		if status != nil && tx.Status != *status {
			continue
		}

		if tx.Status == domain.TransactionStatusFailed || tx.Status == domain.TransactionStatusPending {
			filtered = append(filtered, domain.IssueTransaction{
				Transaction: tx,
				LineNumber:  txWithLine.LineNumber,
			})
		}
	}

	total := len(filtered)

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}

	start := (page - 1) * perPage
	end := start + perPage

	if start >= total {
		return []domain.IssueTransaction{}, total, nil
	}
	if end > total {
		end = total
	}

	return filtered[start:end], total, nil
}

func (s *MemoryStore) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.processedEvents[eventID], nil
}

func (s *MemoryStore) MarkEventProcessed(ctx context.Context, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.processedEvents[eventID] = true

	return nil
}
