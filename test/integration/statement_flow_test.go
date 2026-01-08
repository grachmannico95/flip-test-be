package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/grachmannico95/flip-test-be/internal/config"
	"github.com/grachmannico95/flip-test-be/internal/domain"
	"github.com/grachmannico95/flip-test-be/internal/eventbus"
	"github.com/grachmannico95/flip-test-be/internal/handler"
	"github.com/grachmannico95/flip-test-be/internal/server"
	"github.com/grachmannico95/flip-test-be/internal/service"
	"github.com/grachmannico95/flip-test-be/internal/storage"
	"github.com/grachmannico95/flip-test-be/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*httptest.Server, eventbus.EventBus) {
	log := logger.NewNop()
	repo := storage.NewMemoryStore()

	eventBusCfg := &eventbus.Config{
		ChannelBuffer: 100,
		MaxRetries:    3,
	}
	bus := eventbus.New(log, eventBusCfg)

	reconciliationConsumer := eventbus.NewReconciliationConsumer(repo, log, 5)
	err := bus.Subscribe(eventbus.EventTypeReconciliation, reconciliationConsumer)
	require.NoError(t, err)

	err = bus.Start(context.Background())
	require.NoError(t, err)

	csvProcessor := service.NewCSVProcessor(bus, repo, log)
	statementService := service.NewStatementService(repo, csvProcessor, log)

	statementHandler := handler.NewStatementHandler(statementService, log)
	healthHandler := handler.NewHealthHandler()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: "8080",
			Host: "0.0.0.0",
		},
	}

	srv := server.New(cfg, log, statementHandler, healthHandler)

	testServer := httptest.NewServer(srv.Handler())

	return testServer, bus
}

func TestStatementUploadFlow(t *testing.T) {
	srv, bus := setupTestServer(t)
	defer srv.Close()
	defer bus.Shutdown(context.Background())

	csvContent := `1674507883,JOHN DOE,DEBIT,250000,SUCCESS,restaurant
1674507884,JANE DOE,CREDIT,500000,SUCCESS,salary
1674507885,BOB SMITH,DEBIT,100000,FAILED,invalid transaction
1674507886,ALICE WONDER,CREDIT,300000,PENDING,pending payment`

	uploadID := uploadCSV(t, srv.URL+"/statements", csvContent)
	assert.NotEmpty(t, uploadID)
	time.Sleep(2 * time.Second)

	// Get balance
	balance := getBalance(t, srv.URL+"/balance", uploadID)
	assert.Equal(t, int64(250000), balance)

	// Get all issues
	issues := getIssues(t, srv.URL+"/transactions/issues", uploadID, 1, 10, "")
	assert.Equal(t, 2, len(issues))

	// Get failed issues
	failedIssues := getIssues(t, srv.URL+"/transactions/issues", uploadID, 1, 10, "FAILED")
	assert.Equal(t, 1, len(failedIssues))
	assert.Equal(t, domain.TransactionStatusFailed, domain.TransactionStatus(failedIssues[0]["status"].(string)))

	// Get pending issues
	pendingIssues := getIssues(t, srv.URL+"/transactions/issues", uploadID, 1, 10, "PENDING")
	assert.Equal(t, 1, len(pendingIssues))
	assert.Equal(t, domain.TransactionStatusPending, domain.TransactionStatus(pendingIssues[0]["status"].(string)))
}

func TestHealthCheck(t *testing.T) {
	srv, bus := setupTestServer(t)
	defer srv.Close()
	defer bus.Shutdown(context.Background())

	resp, err := http.Get(srv.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "ok", result["status"])
	assert.NotEmpty(t, result["timestamp"])
}

func TestPagination(t *testing.T) {
	srv, bus := setupTestServer(t)
	defer srv.Close()
	defer bus.Shutdown(context.Background())

	csvContent := `1674507883,USER1,DEBIT,100000,FAILED,error1
1674507884,USER2,DEBIT,100000,FAILED,error2
1674507885,USER3,DEBIT,100000,FAILED,error3
1674507886,USER4,DEBIT,100000,FAILED,error4
1674507887,USER5,DEBIT,100000,FAILED,error5`

	uploadID := uploadCSV(t, srv.URL+"/statements", csvContent)
	time.Sleep(2 * time.Second)

	// Get page 1
	issues := getIssues(t, srv.URL+"/transactions/issues", uploadID, 1, 2, "")
	assert.Equal(t, 2, len(issues))

	// Get page 2
	issues = getIssues(t, srv.URL+"/transactions/issues", uploadID, 2, 2, "")
	assert.Equal(t, 2, len(issues))

	// Get page 3
	issues = getIssues(t, srv.URL+"/transactions/issues", uploadID, 3, 2, "")
	assert.Equal(t, 1, len(issues))
}

func TestUploadNotFound(t *testing.T) {
	srv, bus := setupTestServer(t)
	defer srv.Close()
	defer bus.Shutdown(context.Background())

	resp, err := http.Get(srv.URL + "/balance?upload_id=nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func uploadCSV(t *testing.T, url, csvContent string) string {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.csv")
	require.NoError(t, err)

	_, err = io.WriteString(part, csvContent)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req, err := http.NewRequest("POST", url, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	uploadID, ok := result["upload_id"].(string)
	require.True(t, ok)

	return uploadID
}

func getBalance(t *testing.T, url, uploadID string) int64 {
	resp, err := http.Get(url + "?upload_id=" + uploadID)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	balance, ok := result["balance"].(float64)
	require.True(t, ok)

	return int64(balance)
}

func getIssues(t *testing.T, url, uploadID string, page, perPage int, status string) []map[string]interface{} {
	reqURL := url + "?upload_id=" + uploadID
	if page > 0 {
		reqURL += "&page=" + string(rune(page+'0'))
	}
	if perPage > 0 {
		reqURL += "&per_page=" + string(rune(perPage+'0'))
	}
	if status != "" {
		reqURL += "&status=" + status
	}

	resp, err := http.Get(reqURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	items, ok := result["items"].([]interface{})
	require.True(t, ok)

	var issues []map[string]interface{}
	for _, item := range items {
		issue, ok := item.(map[string]interface{})
		require.True(t, ok)
		issues = append(issues, issue)
	}

	return issues
}

func TestMain(m *testing.M) {
	// Setup
	code := m.Run()
	// Teardown
	os.Exit(code)
}
