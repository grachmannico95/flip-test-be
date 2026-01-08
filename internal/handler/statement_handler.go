package handler

import (
	"net/http"
	"strconv"

	"github.com/grachmannico95/flip-test-be/internal/domain"
	"github.com/grachmannico95/flip-test-be/internal/service"
	"github.com/grachmannico95/flip-test-be/pkg/logger"
	"github.com/labstack/echo/v4"
)

type StatementHandler struct {
	service service.StatementService
	logger  *logger.Logger
}

func NewStatementHandler(service service.StatementService, log *logger.Logger) *StatementHandler {
	return &StatementHandler{
		service: service,
		logger:  log,
	}
}

func (h *StatementHandler) Upload(c echo.Context) error {
	ctx := c.Request().Context()

	h.logger.Info(ctx, "Handling upload request")

	file, err := c.FormFile("file")
	if err != nil {
		h.logger.Error(ctx, "Failed to get file from request",
			"error", err,
		)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "file is required",
		})
	}

	src, err := file.Open()
	if err != nil {
		h.logger.Error(ctx, "Failed to open file",
			"error", err,
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to open file",
		})
	}
	defer src.Close()

	uploadID, err := h.service.UploadStatement(ctx, src)
	if err != nil {
		h.logger.Error(ctx, "Failed to upload statement",
			"error", err,
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to upload statement",
		})
	}

	h.logger.Info(ctx, "Upload successful",
		"upload_id", uploadID,
	)

	return c.JSON(http.StatusAccepted, map[string]string{
		"upload_id": uploadID,
		"status":    "processing",
	})
}

func (h *StatementHandler) GetBalance(c echo.Context) error {
	ctx := c.Request().Context()

	uploadID := c.QueryParam("upload_id")
	if uploadID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "upload_id is required",
		})
	}

	h.logger.Debug(ctx, "Getting balance",
		"upload_id", uploadID,
	)

	balance, err := h.service.GetBalance(ctx, uploadID)
	if err != nil {
		if err == domain.ErrUploadNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "upload not found",
			})
		}

		h.logger.Error(ctx, "Failed to get balance",
			"upload_id", uploadID,
			"error", err,
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to get balance",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"upload_id": uploadID,
		"balance":   balance,
	})
}

func (h *StatementHandler) GetIssues(c echo.Context) error {
	ctx := c.Request().Context()

	uploadID := c.QueryParam("upload_id")
	if uploadID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "upload_id is required",
		})
	}

	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(c.QueryParam("per_page"))
	if err != nil || perPage < 1 {
		perPage = 10
	}

	var statusFilter *domain.TransactionStatus
	statusParam := c.QueryParam("status")
	if statusParam != "" {
		status := domain.TransactionStatus(statusParam)
		if status == domain.TransactionStatusFailed || status == domain.TransactionStatusPending {
			statusFilter = &status
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "status must be FAILED or PENDING",
			})
		}
	}

	h.logger.Debug(ctx, "Getting issues",
		"upload_id", uploadID,
		"page", page,
		"per_page", perPage,
		"status", statusFilter,
	)

	issues, total, err := h.service.GetIssues(ctx, uploadID, page, perPage, statusFilter)
	if err != nil {
		if err == domain.ErrUploadNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "upload not found",
			})
		}

		h.logger.Error(ctx, "Failed to get issues",
			"upload_id", uploadID,
			"error", err,
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to get issues",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"upload_id": uploadID,
		"items":     issues,
		"page":      page,
		"per_page":  perPage,
		"total":     total,
	})
}
