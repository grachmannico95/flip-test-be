package service

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grachmannico95/flip-test-be/internal/domain"
	"github.com/grachmannico95/flip-test-be/mocks"
	"github.com/grachmannico95/flip-test-be/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewStatementService(t *testing.T) {
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")

	svc := NewStatementService(repo, csvProcessor, log)

	assert.NotNil(t, svc)
	assert.Implements(t, (*StatementService)(nil), svc)
}

func TestUploadStatement_Success(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	ctx := context.Background()
	reader := bytes.NewReader([]byte("test csv content"))

	// Mock expectations
	repo.EXPECT().
		CreateUpload(mock.Anything, mock.AnythingOfType("string")).
		Return(nil).
		Once()

	csvProcessor.EXPECT().
		ProcessStream(mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		Return(nil).
		Maybe()

	// Execute
	uploadID, err := svc.UploadStatement(ctx, reader)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, uploadID)
	assert.Len(t, uploadID, 36)

	time.Sleep(10 * time.Millisecond)
}

func TestUploadStatement_CreateUploadError(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	ctx := context.Background()
	reader := bytes.NewReader([]byte("test csv content"))
	expectedError := errors.New("database error")

	// Mock expectations
	repo.EXPECT().
		CreateUpload(mock.Anything, mock.AnythingOfType("string")).
		Return(expectedError).
		Once()

	// Execute
	uploadID, err := svc.UploadStatement(ctx, reader)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Empty(t, uploadID)
}

func TestGetBalance_Success(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	ctx := context.Background()
	uploadID := "test-upload-123"
	expectedBalance := int64(1500000)

	// Mock expectations
	repo.EXPECT().
		GetBalance(mock.Anything, uploadID).
		Return(expectedBalance, nil).
		Once()

	// Execute
	balance, err := svc.GetBalance(ctx, uploadID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedBalance, balance)
}

func TestGetBalance_Error(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	ctx := context.Background()
	uploadID := "test-upload-123"
	expectedError := errors.New("upload not found")

	// Mock expectations
	repo.EXPECT().
		GetBalance(mock.Anything, uploadID).
		Return(int64(0), expectedError).
		Once()

	// Execute
	balance, err := svc.GetBalance(ctx, uploadID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Equal(t, int64(0), balance)
}

func TestGetIssues_Success(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	ctx := context.Background()
	uploadID := "test-upload-123"
	page := 1
	perPage := 10
	status := domain.TransactionStatusFailed

	expectedIssues := []domain.IssueTransaction{
		{
			Transaction: domain.Transaction{
				Timestamp:    1674507885,
				Counterparty: "BOB SMITH",
				Type:         domain.TransactionTypeDebit,
				Amount:       100000,
				Status:       domain.TransactionStatusFailed,
				Description:  "invalid transaction",
			},
			LineNumber: 5,
		},
	}
	expectedTotal := 1

	// Mock expectations
	repo.EXPECT().
		GetIssues(mock.Anything, uploadID, page, perPage, &status).
		Return(expectedIssues, expectedTotal, nil).
		Once()

	// Execute
	issues, total, err := svc.GetIssues(ctx, uploadID, page, perPage, &status)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedIssues, issues)
	assert.Equal(t, expectedTotal, total)
	assert.Len(t, issues, 1)
	assert.Equal(t, "BOB SMITH", issues[0].Counterparty)
	assert.Equal(t, domain.TransactionStatusFailed, issues[0].Status)
}

func TestGetIssues_WithNilStatus(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	ctx := context.Background()
	uploadID := "test-upload-123"
	page := 1
	perPage := 10

	expectedIssues := []domain.IssueTransaction{
		{
			Transaction: domain.Transaction{
				Timestamp:    1674507885,
				Counterparty: "BOB SMITH",
				Type:         domain.TransactionTypeDebit,
				Amount:       100000,
				Status:       domain.TransactionStatusFailed,
				Description:  "invalid",
			},
			LineNumber: 5,
		},
		{
			Transaction: domain.Transaction{
				Timestamp:    1674507886,
				Counterparty: "ALICE WONDER",
				Type:         domain.TransactionTypeCredit,
				Amount:       300000,
				Status:       domain.TransactionStatusPending,
				Description:  "pending payment",
			},
			LineNumber: 6,
		},
	}
	expectedTotal := 2

	// Mock expectations - nil status should return all issues (FAILED and PENDING)
	repo.EXPECT().
		GetIssues(mock.Anything, uploadID, page, perPage, (*domain.TransactionStatus)(nil)).
		Return(expectedIssues, expectedTotal, nil).
		Once()

	// Execute
	issues, total, err := svc.GetIssues(ctx, uploadID, page, perPage, nil)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedIssues, issues)
	assert.Equal(t, expectedTotal, total)
	assert.Len(t, issues, 2)
}

func TestGetIssues_Error(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	ctx := context.Background()
	uploadID := "test-upload-123"
	page := 1
	perPage := 10
	status := domain.TransactionStatusFailed
	expectedError := errors.New("upload not found")

	// Mock expectations
	repo.EXPECT().
		GetIssues(mock.Anything, uploadID, page, perPage, &status).
		Return(nil, 0, expectedError).
		Once()

	// Execute
	issues, total, err := svc.GetIssues(ctx, uploadID, page, perPage, &status)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, issues)
	assert.Equal(t, 0, total)
}

func TestGetIssues_Pagination(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	ctx := context.Background()
	uploadID := "test-upload-123"

	// Test page 2
	page := 2
	perPage := 5

	expectedIssues := []domain.IssueTransaction{
		{
			Transaction: domain.Transaction{
				Timestamp:    1674507890,
				Counterparty: "USER 6",
				Type:         domain.TransactionTypeDebit,
				Amount:       50000,
				Status:       domain.TransactionStatusFailed,
				Description:  "failed 6",
			},
			LineNumber: 11,
		},
	}
	expectedTotal := 11 // Total across all pages

	// Mock expectations
	repo.EXPECT().
		GetIssues(mock.Anything, uploadID, page, perPage, (*domain.TransactionStatus)(nil)).
		Return(expectedIssues, expectedTotal, nil).
		Once()

	// Execute
	issues, total, err := svc.GetIssues(ctx, uploadID, page, perPage, nil)

	// Assert
	require.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.Equal(t, expectedTotal, total)
}

func TestGetUploadStatus_Success(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	ctx := context.Background()
	uploadID := "test-upload-123"
	createdAt := time.Now()
	completedAt := time.Now().Add(1 * time.Minute)

	expectedUpload := &domain.Upload{
		ID:            uploadID,
		Status:        domain.UploadStatusCompleted,
		ProcessedRows: 100,
		TotalRows:     100,
		CreatedAt:     createdAt,
		CompletedAt:   &completedAt,
	}

	// Mock expectations
	repo.EXPECT().
		GetUpload(mock.Anything, uploadID).
		Return(expectedUpload, nil).
		Once()

	// Execute
	upload, err := svc.GetUploadStatus(ctx, uploadID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedUpload, upload)
	assert.Equal(t, uploadID, upload.ID)
	assert.Equal(t, domain.UploadStatusCompleted, upload.Status)
	assert.Equal(t, 100, upload.ProcessedRows)
	assert.NotNil(t, upload.CompletedAt)
}

func TestGetUploadStatus_Processing(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	ctx := context.Background()
	uploadID := "test-upload-123"
	createdAt := time.Now()

	expectedUpload := &domain.Upload{
		ID:            uploadID,
		Status:        domain.UploadStatusProcessing,
		ProcessedRows: 50,
		TotalRows:     100,
		CreatedAt:     createdAt,
		CompletedAt:   nil,
	}

	// Mock expectations
	repo.EXPECT().
		GetUpload(mock.Anything, uploadID).
		Return(expectedUpload, nil).
		Once()

	// Execute
	upload, err := svc.GetUploadStatus(ctx, uploadID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedUpload, upload)
	assert.Equal(t, domain.UploadStatusProcessing, upload.Status)
	assert.Nil(t, upload.CompletedAt)
}

func TestGetUploadStatus_Error(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	ctx := context.Background()
	uploadID := "test-upload-123"
	expectedError := domain.ErrUploadNotFound

	// Mock expectations
	repo.EXPECT().
		GetUpload(mock.Anything, uploadID).
		Return(nil, expectedError).
		Once()

	// Execute
	upload, err := svc.GetUploadStatus(ctx, uploadID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, upload)
}

func TestStatementService_ContextPropagation(t *testing.T) {
	// Setup
	repo := mocks.NewMockRepository(t)
	csvProcessor := mocks.NewMockCSVProcessorInterface(t)
	log := logger.New("info")
	svc := NewStatementService(repo, csvProcessor, log)

	uploadID := "test-upload-123"

	// Create context with trace ID
	ctx := logger.WithTraceID(context.Background(), "test-trace-123")

	// Mock expectations - verify context is passed with upload_id
	repo.EXPECT().
		GetBalance(mock.MatchedBy(func(ctx context.Context) bool {
			// Verify context has upload_id added by service
			return logger.GetUploadID(ctx) == uploadID
		}), uploadID).
		Return(int64(1000), nil).
		Once()

	// Execute
	_, err := svc.GetBalance(ctx, uploadID)

	// Assert
	require.NoError(t, err)
}
