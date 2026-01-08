package eventbus

import (
	"context"
	"fmt"

	"github.com/grachmannico95/flip-test-be/internal/domain"
	"github.com/grachmannico95/flip-test-be/pkg/logger"
)

type ReconciliationConsumer struct {
	repo        domain.Repository
	logger      *logger.Logger
	workerCount int
}

func NewReconciliationConsumer(repo domain.Repository, log *logger.Logger, workerCount int) *ReconciliationConsumer {
	return &ReconciliationConsumer{
		repo:        repo,
		logger:      log,
		workerCount: workerCount,
	}
}

func (rc *ReconciliationConsumer) Consume(ctx context.Context, event Event) error {
	// Check idempotency
	processed, err := rc.repo.IsEventProcessed(ctx, event.ID)
	if err != nil {
		rc.logger.Error(ctx, "Failed to check event processed status",
			"event_id", event.ID,
			"error", err,
		)
		return err
	}

	if processed {
		rc.logger.Debug(ctx, "Event already processed, skipping",
			"event_id", event.ID,
		)
		return nil
	}

	payload, ok := event.Payload.(ReconciliationEvent)
	if !ok {
		rc.logger.Error(ctx, "Invalid payload type for reconciliation event",
			"event_id", event.ID,
		)
		return fmt.Errorf("invalid payload type")
	}

	ctx = logger.WithUploadID(ctx, payload.UploadID)

	rc.logger.Debug(ctx, "Processing transaction",
		"event_id", event.ID,
		"line_number", payload.LineNumber,
		"status", payload.Transaction.Status,
		"type", payload.Transaction.Type,
		"amount", payload.Transaction.Amount,
	)

	err = rc.repo.AddTransaction(ctx, payload.UploadID, payload.Transaction, payload.LineNumber)
	if err != nil {
		rc.logger.Error(ctx, "Failed to add transaction",
			"event_id", event.ID,
			"line_number", payload.LineNumber,
			"error", err,
		)
		return err
	}

	err = rc.repo.MarkEventProcessed(ctx, event.ID)
	if err != nil {
		rc.logger.Error(ctx, "Failed to mark event as processed",
			"event_id", event.ID,
			"error", err,
		)
		return err
	}

	err = rc.repo.IncrementProcessedRows(ctx, payload.UploadID)
	if err != nil {
		rc.logger.Error(ctx, "Failed to increment processed rows",
			"event_id", event.ID,
			"error", err,
		)
	}

	rc.logger.Debug(ctx, "Transaction processed successfully",
		"event_id", event.ID,
		"line_number", payload.LineNumber,
	)

	return nil
}

func (rc *ReconciliationConsumer) GetWorkerCount() int {
	return rc.workerCount
}
