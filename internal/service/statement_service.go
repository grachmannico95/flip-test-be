package service

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/grachmannico95/flip-test-be/internal/domain"
	"github.com/grachmannico95/flip-test-be/pkg/logger"
)

type StatementService interface {
	UploadStatement(ctx context.Context, reader io.Reader) (string, error)
	GetBalance(ctx context.Context, uploadID string) (int64, error)
	GetIssues(ctx context.Context, uploadID string, page, perPage int, status *domain.TransactionStatus) ([]domain.IssueTransaction, int, error)
	GetUploadStatus(ctx context.Context, uploadID string) (*domain.Upload, error)
}

type statementService struct {
	repo         domain.Repository
	csvProcessor CSVProcessorInterface
	logger       *logger.Logger
}

func NewStatementService(repo domain.Repository, csvProcessor CSVProcessorInterface, log *logger.Logger) StatementService {
	return &statementService{
		repo:         repo,
		csvProcessor: csvProcessor,
		logger:       log,
	}
}

func (s *statementService) UploadStatement(ctx context.Context, reader io.Reader) (string, error) {
	uploadID := uuid.New().String()

	ctx = logger.WithUploadID(ctx, uploadID)

	s.logger.Info(ctx, "Creating upload record")

	err := s.repo.CreateUpload(ctx, uploadID)
	if err != nil {
		s.logger.Error(ctx, "Failed to create upload",
			"error", err,
		)
		return "", err
	}

	go func() {
		processCtx := context.Background()
		processCtx = logger.WithUploadID(processCtx, uploadID)

		s.logger.Info(processCtx, "Starting async CSV processing")

		err := s.csvProcessor.ProcessStream(processCtx, uploadID, reader)
		if err != nil {
			s.logger.Error(processCtx, "CSV processing failed",
				"error", err,
			)
		} else {
			s.logger.Info(processCtx, "CSV processing completed successfully")
		}
	}()

	s.logger.Info(ctx, "Upload created, processing started")

	return uploadID, nil
}

func (s *statementService) GetBalance(ctx context.Context, uploadID string) (int64, error) {
	ctx = logger.WithUploadID(ctx, uploadID)

	s.logger.Debug(ctx, "Getting balance")

	balance, err := s.repo.GetBalance(ctx, uploadID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get balance",
			"error", err,
		)
		return 0, err
	}

	s.logger.Debug(ctx, "Balance retrieved",
		"balance", balance,
	)

	return balance, nil
}

func (s *statementService) GetIssues(ctx context.Context, uploadID string, page, perPage int, status *domain.TransactionStatus) ([]domain.IssueTransaction, int, error) {
	ctx = logger.WithUploadID(ctx, uploadID)

	s.logger.Debug(ctx, "Getting issues",
		"page", page,
		"per_page", perPage,
		"status", status,
	)

	issues, total, err := s.repo.GetIssues(ctx, uploadID, page, perPage, status)
	if err != nil {
		s.logger.Error(ctx, "Failed to get issues",
			"error", err,
		)
		return nil, 0, err
	}

	s.logger.Debug(ctx, "Issues retrieved",
		"total", total,
		"returned", len(issues),
	)

	return issues, total, nil
}

func (s *statementService) GetUploadStatus(ctx context.Context, uploadID string) (*domain.Upload, error) {
	ctx = logger.WithUploadID(ctx, uploadID)

	s.logger.Debug(ctx, "Getting upload status")

	upload, err := s.repo.GetUpload(ctx, uploadID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get upload",
			"error", err,
		)
		return nil, err
	}

	return upload, nil
}
