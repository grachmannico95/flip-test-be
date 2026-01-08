package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/grachmannico95/flip-test-be/internal/config"
	"github.com/grachmannico95/flip-test-be/internal/eventbus"
	"github.com/grachmannico95/flip-test-be/internal/handler"
	"github.com/grachmannico95/flip-test-be/internal/server"
	"github.com/grachmannico95/flip-test-be/internal/service"
	"github.com/grachmannico95/flip-test-be/internal/storage"
	"github.com/grachmannico95/flip-test-be/pkg/logger"
)

func main() {
	cfg := config.Load()

	log := logger.New(cfg.Logging.Level)
	defer log.Sync()

	ctx := context.Background()
	log.Info(ctx, "Starting application")

	repo := storage.NewMemoryStore()
	log.Info(ctx, "Repository initialized")

	eventBusCfg := &eventbus.Config{
		ChannelBuffer: cfg.EventBus.ChannelBufferSize,
		MaxRetries:    cfg.Worker.MaxRetries,
	}
	bus := eventbus.New(log, eventBusCfg)
	log.Info(ctx, "Event bus initialized")

	reconciliationConsumer := eventbus.NewReconciliationConsumer(
		repo,
		log,
		cfg.Worker.PoolSize,
	)
	log.Info(ctx, "Reconciliation consumer initialized",
		"worker_count", cfg.Worker.PoolSize,
	)

	err := bus.Subscribe(eventbus.EventTypeReconciliation, reconciliationConsumer)
	if err != nil {
		log.Fatal(ctx, "Failed to subscribe consumer",
			"error", err,
		)
	}

	err = bus.Start(ctx)
	if err != nil {
		log.Fatal(ctx, "Failed to start event bus",
			"error", err,
		)
	}

	csvProcessor := service.NewCSVProcessor(bus, repo, log)
	statementService := service.NewStatementService(repo, csvProcessor, log)
	log.Info(ctx, "Services initialized")

	statementHandler := handler.NewStatementHandler(statementService, log)
	healthHandler := handler.NewHealthHandler()
	log.Info(ctx, "Handlers initialized")

	srv := server.New(cfg, log, statementHandler, healthHandler)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatal(ctx, "Failed to start HTTP server",
				"error", err,
			)
		}
	}()

	log.Info(ctx, "Application started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info(ctx, "Received shutdown signal")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
	defer cancel()

	// Graceful shutdown in order:
	// 1. Stop accepting new HTTP requests
	log.Info(shutdownCtx, "Shutting down HTTP server")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error(shutdownCtx, "HTTP server shutdown error",
			"error", err,
		)
	}

	// 2. Stop event bus and wait for workers to finish
	if err := bus.Shutdown(shutdownCtx); err != nil {
		log.Error(shutdownCtx, "Event bus shutdown error",
			"error", err,
		)
	}

	log.Info(ctx, "Application stopped gracefully")
}
