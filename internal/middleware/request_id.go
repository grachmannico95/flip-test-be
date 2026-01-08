package middleware

import (
	"github.com/google/uuid"
	"github.com/grachmannico95/flip-test-be/pkg/logger"
	"github.com/labstack/echo/v4"
)

func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			traceID := c.Request().Header.Get("X-Trace-ID")
			if traceID == "" {
				traceID = uuid.New().String()
			}

			ctx := logger.WithTraceID(c.Request().Context(), traceID)
			c.SetRequest(c.Request().WithContext(ctx))

			c.Response().Header().Set("X-Trace-ID", traceID)

			return next(c)
		}
	}
}
