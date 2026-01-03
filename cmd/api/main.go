package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saturnino-fabrica-de-software/rekko/internal/api"
	"github.com/saturnino-fabrica-de-software/rekko/internal/api/middleware"
	"github.com/saturnino-fabrica-de-software/rekko/internal/config"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider/mock"
	"github.com/saturnino-fabrica-de-software/rekko/internal/repository"
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

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	// Verify database connection
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	logger.Info("connected to database")

	// Create repositories
	tenantRepo := repository.NewTenantRepository(pool)
	apiKeyRepo := repository.NewAPIKeyRepository(pool)
	faceRepo := repository.NewFaceRepository(pool)
	verificationRepo := repository.NewVerificationRepository(pool)

	// Create provider (mock for development)
	faceProvider := mock.New()
	logger.Info("using mock face provider")

	// Create last used worker for async API key updates
	lastUsedWorker := middleware.NewLastUsedWorker(
		apiKeyRepo,
		logger,
		middleware.DefaultLastUsedWorkerConfig(),
	)
	lastUsedWorker.Start()
	defer lastUsedWorker.Stop()

	// Setup dependencies
	deps := &api.Dependencies{
		TenantRepo:       tenantRepo,
		APIKeyRepo:       apiKeyRepo,
		FaceRepo:         faceRepo,
		VerificationRepo: verificationRepo,
		FaceProvider:     faceProvider,
		LastUsedWorker:   lastUsedWorker,
	}

	// Setup router with dependencies
	router := api.NewRouter(logger, deps)
	router.Setup()

	// Graceful shutdown
	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
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
	case <-shutdownCtx.Done():
		logger.Info("shutdown signal received")
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}

	// Graceful shutdown with timeout
	gracefulCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("shutting down server...")
	if err := router.Shutdown(); err != nil {
		logger.Error("shutdown error", slog.Any("error", err))
	}

	<-gracefulCtx.Done()
	logger.Info("server stopped")

	return nil
}
