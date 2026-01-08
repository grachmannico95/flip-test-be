package storage

import (
	"context"
	"testing"

	"github.com/grachmannico95/flip-test-be/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_CreateUpload(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	uploadID := "test-upload-1"
	err := store.CreateUpload(ctx, uploadID)
	require.NoError(t, err)

	upload, err := store.GetUpload(ctx, uploadID)
	require.NoError(t, err)
	assert.Equal(t, uploadID, upload.ID)
	assert.Equal(t, domain.UploadStatusProcessing, upload.Status)
	assert.Equal(t, 0, upload.ProcessedRows)
}

func TestMemoryStore_GetUpload_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	_, err := store.GetUpload(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrUploadNotFound)
}

func TestMemoryStore_UpdateUploadStatus(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	uploadID := "test-upload-1"
	err := store.CreateUpload(ctx, uploadID)
	require.NoError(t, err)

	err = store.UpdateUploadStatus(ctx, uploadID, domain.UploadStatusCompleted)
	require.NoError(t, err)

	upload, err := store.GetUpload(ctx, uploadID)
	require.NoError(t, err)
	assert.Equal(t, domain.UploadStatusCompleted, upload.Status)
	assert.NotNil(t, upload.CompletedAt)
}

func TestMemoryStore_IncrementProcessedRows(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	uploadID := "test-upload-1"
	err := store.CreateUpload(ctx, uploadID)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		err = store.IncrementProcessedRows(ctx, uploadID)
		require.NoError(t, err)
	}

	upload, err := store.GetUpload(ctx, uploadID)
	require.NoError(t, err)
	assert.Equal(t, 5, upload.ProcessedRows)
}

func TestMemoryStore_AddTransaction(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	uploadID := "test-upload-1"
	err := store.CreateUpload(ctx, uploadID)
	require.NoError(t, err)

	tx := domain.Transaction{
		Timestamp:    1674507883,
		Counterparty: "JOHN DOE",
		Type:         domain.TransactionTypeDebit,
		Amount:       250000,
		Status:       domain.TransactionStatusSuccess,
		Description:  "restaurant",
	}

	err = store.AddTransaction(ctx, uploadID, tx, 1)
	require.NoError(t, err)
}

func TestMemoryStore_GetBalance(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	uploadID := "test-upload-1"
	err := store.CreateUpload(ctx, uploadID)
	require.NoError(t, err)

	err = store.AddTransaction(ctx, uploadID, domain.Transaction{
		Type:   domain.TransactionTypeCredit,
		Amount: 500000,
		Status: domain.TransactionStatusSuccess,
	}, 1)
	require.NoError(t, err)

	err = store.AddTransaction(ctx, uploadID, domain.Transaction{
		Type:   domain.TransactionTypeDebit,
		Amount: 250000,
		Status: domain.TransactionStatusSuccess,
	}, 2)
	require.NoError(t, err)

	err = store.AddTransaction(ctx, uploadID, domain.Transaction{
		Type:   domain.TransactionTypeDebit,
		Amount: 100000,
		Status: domain.TransactionStatusFailed,
	}, 3)
	require.NoError(t, err)

	balance, err := store.GetBalance(ctx, uploadID)
	require.NoError(t, err)
	assert.Equal(t, int64(250000), balance)
}

func TestMemoryStore_GetBalance_OnlySuccessTransactions(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	uploadID := "test-upload-1"
	err := store.CreateUpload(ctx, uploadID)
	require.NoError(t, err)

	err = store.AddTransaction(ctx, uploadID, domain.Transaction{
		Type:   domain.TransactionTypeCredit,
		Amount: 1000,
		Status: domain.TransactionStatusSuccess,
	}, 1)
	require.NoError(t, err)

	err = store.AddTransaction(ctx, uploadID, domain.Transaction{
		Type:   domain.TransactionTypeCredit,
		Amount: 2000,
		Status: domain.TransactionStatusFailed,
	}, 2)
	require.NoError(t, err)

	err = store.AddTransaction(ctx, uploadID, domain.Transaction{
		Type:   domain.TransactionTypeCredit,
		Amount: 3000,
		Status: domain.TransactionStatusPending,
	}, 3)
	require.NoError(t, err)

	balance, err := store.GetBalance(ctx, uploadID)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), balance)
}

func TestMemoryStore_GetIssues(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	uploadID := "test-upload-1"
	err := store.CreateUpload(ctx, uploadID)
	require.NoError(t, err)

	err = store.AddTransaction(ctx, uploadID, domain.Transaction{
		Status: domain.TransactionStatusSuccess,
	}, 1)
	require.NoError(t, err)

	err = store.AddTransaction(ctx, uploadID, domain.Transaction{
		Status:       domain.TransactionStatusFailed,
		Counterparty: "FAILED USER",
	}, 2)
	require.NoError(t, err)

	err = store.AddTransaction(ctx, uploadID, domain.Transaction{
		Status:       domain.TransactionStatusPending,
		Counterparty: "PENDING USER",
	}, 3)
	require.NoError(t, err)

	issues, total, err := store.GetIssues(ctx, uploadID, 1, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, issues, 2)
}

func TestMemoryStore_GetIssues_WithStatusFilter(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	uploadID := "test-upload-1"
	err := store.CreateUpload(ctx, uploadID)
	require.NoError(t, err)

	err = store.AddTransaction(ctx, uploadID, domain.Transaction{
		Status: domain.TransactionStatusFailed,
	}, 1)
	require.NoError(t, err)

	err = store.AddTransaction(ctx, uploadID, domain.Transaction{
		Status: domain.TransactionStatusPending,
	}, 2)
	require.NoError(t, err)

	failedStatus := domain.TransactionStatusFailed
	issues, total, err := store.GetIssues(ctx, uploadID, 1, 10, &failedStatus)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, issues, 1)
	assert.Equal(t, domain.TransactionStatusFailed, issues[0].Status)
}

func TestMemoryStore_GetIssues_Pagination(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	uploadID := "test-upload-1"
	err := store.CreateUpload(ctx, uploadID)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		err = store.AddTransaction(ctx, uploadID, domain.Transaction{
			Status: domain.TransactionStatusFailed,
		}, i+1)
		require.NoError(t, err)
	}

	issues, total, err := store.GetIssues(ctx, uploadID, 1, 2, nil)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, issues, 2)

	issues, total, err = store.GetIssues(ctx, uploadID, 2, 2, nil)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, issues, 2)

	issues, total, err = store.GetIssues(ctx, uploadID, 3, 2, nil)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, issues, 1)
}

func TestMemoryStore_IsEventProcessed(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	eventID := "event-1"

	processed, err := store.IsEventProcessed(ctx, eventID)
	require.NoError(t, err)
	assert.False(t, processed)

	err = store.MarkEventProcessed(ctx, eventID)
	require.NoError(t, err)

	processed, err = store.IsEventProcessed(ctx, eventID)
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestMemoryStore_Concurrency(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	uploadID := "test-upload-1"
	err := store.CreateUpload(ctx, uploadID)
	require.NoError(t, err)

	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(id int) {
			_ = store.AddTransaction(ctx, uploadID, domain.Transaction{
				Type:   domain.TransactionTypeCredit,
				Amount: 1000,
				Status: domain.TransactionStatusSuccess,
			}, id)

			_ = store.IncrementProcessedRows(ctx, uploadID)

			_, _ = store.GetBalance(ctx, uploadID)

			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	upload, err := store.GetUpload(ctx, uploadID)
	require.NoError(t, err)
	assert.Equal(t, 100, upload.ProcessedRows)

	balance, err := store.GetBalance(ctx, uploadID)
	require.NoError(t, err)
	assert.Equal(t, int64(100000), balance)
}
