package middleware

import (
	"time"

	"github.com/grachmannico95/flip-test-be/pkg/logger"
	"github.com/labstack/echo/v4"
)

func Logging(log *logger.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			log.Info(c.Request().Context(), "HTTP request",
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", c.Response().Status,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", c.Request().RemoteAddr,
			)

			return err
		}
	}
}
