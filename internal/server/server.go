package server

import (
	"context"
	"fmt"

	"github.com/grachmannico95/flip-test-be/internal/config"
	"github.com/grachmannico95/flip-test-be/internal/handler"
	"github.com/grachmannico95/flip-test-be/internal/middleware"
	"github.com/grachmannico95/flip-test-be/pkg/logger"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

type Server struct {
	echo             *echo.Echo
	cfg              *config.Config
	logger           *logger.Logger
	statementHandler *handler.StatementHandler
	healthHandler    *handler.HealthHandler
}

func New(
	cfg *config.Config,
	log *logger.Logger,
	statementHandler *handler.StatementHandler,
	healthHandler *handler.HealthHandler,
) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	return &Server{
		echo:             e,
		cfg:              cfg,
		logger:           log,
		statementHandler: statementHandler,
		healthHandler:    healthHandler,
	}
}

func (s *Server) Start() error {
	s.setupMiddleware()
	s.setupRoutes()

	addr := fmt.Sprintf("%s:%s", s.cfg.Server.Host, s.cfg.Server.Port)
	s.logger.Info(context.Background(), "Starting HTTP server",
		"address", addr,
	)

	return s.echo.Start(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info(ctx, "Shutting down HTTP server")
	return s.echo.Shutdown(ctx)
}

func (s *Server) setupMiddleware() {
	s.echo.Use(echoMiddleware.Recover())
	s.echo.Use(echoMiddleware.CORS())
	s.echo.Use(middleware.RequestID())
	s.echo.Use(middleware.Logging(s.logger))
}

func (s *Server) setupRoutes() {
	s.echo.GET("/health", s.healthHandler.Check)

	s.echo.POST("/statements", s.statementHandler.Upload)
	s.echo.GET("/balance", s.statementHandler.GetBalance)
	s.echo.GET("/transactions/issues", s.statementHandler.GetIssues)
}

func (s *Server) Handler() *echo.Echo {
	s.setupMiddleware()
	s.setupRoutes()
	return s.echo
}
