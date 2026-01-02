package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/saturnino-fabrica-de-software/rekko/internal/api"
	"github.com/saturnino-fabrica-de-software/rekko/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	logger := config.NewLogger(cfg.Environment)
	slog.SetDefault(logger)

	logger.Info("starting Rekko API",
		slog.String("environment", cfg.Environment),
		slog.Int("port", cfg.Port),
	)

	// Setup router
	router := api.NewRouter(logger)
	router.Setup()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		logger.Info("server listening", slog.String("addr", addr))
		if err := router.Listen(addr); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("shutting down server...")
	if err := router.Shutdown(); err != nil {
		logger.Error("shutdown error", slog.Any("error", err))
	}

	<-shutdownCtx.Done()
	logger.Info("server stopped")

	return nil
}
