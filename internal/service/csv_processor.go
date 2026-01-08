package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/grachmannico95/flip-test-be/internal/domain"
	"github.com/grachmannico95/flip-test-be/internal/eventbus"
	"github.com/grachmannico95/flip-test-be/pkg/logger"
)

type CSVProcessorInterface interface {
	ProcessStream(ctx context.Context, uploadID string, reader io.Reader) error
}

type CSVProcessor struct {
	eventBus eventbus.EventBus
	repo     domain.Repository
	logger   *logger.Logger
}

func NewCSVProcessor(eventBus eventbus.EventBus, repo domain.Repository, log *logger.Logger) *CSVProcessor {
	return &CSVProcessor{
		eventBus: eventBus,
		repo:     repo,
		logger:   log,
	}
}

func (p *CSVProcessor) ProcessStream(ctx context.Context, uploadID string, reader io.Reader) error {
	ctx = logger.WithUploadID(ctx, uploadID)

	p.logger.Info(ctx, "Starting CSV processing")

	csvReader := csv.NewReader(reader)
	csvReader.ReuseRecord = true // Optimize memory usage
	csvReader.TrimLeadingSpace = true

	lineNumber := 0
	successCount := 0
	errorCount := 0

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			p.logger.Error(ctx, "Failed to read CSV line",
				"line", lineNumber,
				"error", err,
			)
			errorCount++
			continue
		}

		lineNumber++

		tx, err := p.parseTransaction(record, lineNumber)
		if err != nil {
			p.logger.Warn(ctx, "Failed to parse transaction",
				"line", lineNumber,
				"error", err,
			)
			errorCount++
			continue
		}

		event := eventbus.Event{
			ID:   fmt.Sprintf("%s-%d", uploadID, lineNumber),
			Type: eventbus.EventTypeReconciliation,
			Payload: eventbus.ReconciliationEvent{
				UploadID:    uploadID,
				Transaction: tx,
				LineNumber:  lineNumber,
			},
			Timestamp: time.Now(),
		}

		err = p.eventBus.Publish(ctx, event)
		if err != nil {
			p.logger.Error(ctx, "Failed to publish event",
				"event_id", event.ID,
				"line", lineNumber,
				"error", err,
			)
			errorCount++
			continue
		}

		successCount++
	}

	if errorCount > 0 && successCount == 0 {
		err := p.repo.UpdateUploadStatus(ctx, uploadID, domain.UploadStatusFailed)
		if err != nil {
			p.logger.Error(ctx, "Failed to update upload status to failed",
				"error", err,
			)
		}
	} else {
		err := p.repo.UpdateUploadStatus(ctx, uploadID, domain.UploadStatusCompleted)
		if err != nil {
			p.logger.Error(ctx, "Failed to update upload status to completed",
				"error", err,
			)
		}
	}

	p.logger.Info(ctx, "CSV processing completed",
		"total_lines", lineNumber,
		"success_count", successCount,
		"error_count", errorCount,
	)

	return nil
}

func (p *CSVProcessor) parseTransaction(record []string, lineNumber int) (domain.Transaction, error) {
	if len(record) != 6 {
		return domain.Transaction{}, fmt.Errorf("invalid record format: expected 6 fields, got %d", len(record))
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(record[0]), 10, 64)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	amount, err := strconv.ParseInt(strings.TrimSpace(record[3]), 10, 64)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("invalid amount: %w", err)
	}

	txType := strings.TrimSpace(strings.ToUpper(record[2]))
	if txType != string(domain.TransactionTypeCredit) && txType != string(domain.TransactionTypeDebit) {
		return domain.Transaction{}, fmt.Errorf("invalid transaction type: %s", txType)
	}

	status := strings.TrimSpace(strings.ToUpper(record[4]))
	if status != string(domain.TransactionStatusSuccess) &&
		status != string(domain.TransactionStatusFailed) &&
		status != string(domain.TransactionStatusPending) {
		return domain.Transaction{}, fmt.Errorf("invalid status: %s", status)
	}

	return domain.Transaction{
		Timestamp:    timestamp,
		Counterparty: strings.TrimSpace(record[1]),
		Type:         domain.TransactionType(txType),
		Amount:       amount,
		Status:       domain.TransactionStatus(status),
		Description:  strings.TrimSpace(record[5]),
	}, nil
}
