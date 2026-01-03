package middleware

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/repository"
)

// LastUsedWorker handles async updates of API key last_used_at with debouncing
type LastUsedWorker struct {
	apiKeyRepo repository.APIKeyRepositoryInterface
	logger     *slog.Logger

	// Channel with buffer to prevent blocking
	updateCh chan uuid.UUID

	// Debounce: track recently updated keys
	recentlyUpdated map[uuid.UUID]time.Time
	mu              sync.RWMutex

	// Config
	debounceInterval time.Duration
	batchInterval    time.Duration
	maxBatchSize     int

	// Lifecycle
	done chan struct{}
	wg   sync.WaitGroup
}

// LastUsedWorkerConfig holds configuration for the worker
type LastUsedWorkerConfig struct {
	BufferSize       int           // Channel buffer size (default: 1000)
	DebounceInterval time.Duration // Min interval between updates for same key (default: 1 minute)
	BatchInterval    time.Duration // Interval to process batch (default: 5 seconds)
	MaxBatchSize     int           // Max keys per batch (default: 100)
}

// DefaultLastUsedWorkerConfig returns default configuration
func DefaultLastUsedWorkerConfig() LastUsedWorkerConfig {
	return LastUsedWorkerConfig{
		BufferSize:       1000,
		DebounceInterval: 1 * time.Minute,
		BatchInterval:    5 * time.Second,
		MaxBatchSize:     100,
	}
}

// NewLastUsedWorker creates a new worker
func NewLastUsedWorker(
	apiKeyRepo repository.APIKeyRepositoryInterface,
	logger *slog.Logger,
	config LastUsedWorkerConfig,
) *LastUsedWorker {
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}
	if config.DebounceInterval == 0 {
		config.DebounceInterval = 1 * time.Minute
	}
	if config.BatchInterval == 0 {
		config.BatchInterval = 5 * time.Second
	}
	if config.MaxBatchSize == 0 {
		config.MaxBatchSize = 100
	}

	return &LastUsedWorker{
		apiKeyRepo:       apiKeyRepo,
		logger:           logger,
		updateCh:         make(chan uuid.UUID, config.BufferSize),
		recentlyUpdated:  make(map[uuid.UUID]time.Time),
		debounceInterval: config.DebounceInterval,
		batchInterval:    config.BatchInterval,
		maxBatchSize:     config.MaxBatchSize,
		done:             make(chan struct{}),
	}
}

// Start begins the background worker
func (w *LastUsedWorker) Start() {
	w.wg.Add(1)
	go w.run()
	w.logger.Info("last used worker started",
		"buffer_size", cap(w.updateCh),
		"debounce_interval", w.debounceInterval,
		"batch_interval", w.batchInterval,
	)
}

// Stop gracefully shuts down the worker
func (w *LastUsedWorker) Stop() {
	close(w.done)
	w.wg.Wait()
	w.logger.Info("last used worker stopped")
}

// Enqueue adds an API key ID for async last_used update
// Non-blocking: if buffer is full, the update is dropped
func (w *LastUsedWorker) Enqueue(keyID uuid.UUID) {
	// Check debounce
	w.mu.RLock()
	lastUpdate, exists := w.recentlyUpdated[keyID]
	w.mu.RUnlock()

	if exists && time.Since(lastUpdate) < w.debounceInterval {
		return // Skip - recently updated
	}

	// Non-blocking send
	select {
	case w.updateCh <- keyID:
		// Enqueued
	default:
		// Buffer full - drop update (it's just last_used, not critical)
		w.logger.Debug("last used update dropped - buffer full", "key_id", keyID)
	}
}

func (w *LastUsedWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.batchInterval)
	defer ticker.Stop()

	// Cleanup ticker for debounce map
	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	var batch []uuid.UUID

	for {
		select {
		case <-w.done:
			// Process remaining batch before exiting
			if len(batch) > 0 {
				w.processBatch(batch)
			}
			return

		case keyID := <-w.updateCh:
			batch = append(batch, keyID)
			if len(batch) >= w.maxBatchSize {
				w.processBatch(batch)
				batch = nil
			}

		case <-ticker.C:
			if len(batch) > 0 {
				w.processBatch(batch)
				batch = nil
			}

		case <-cleanupTicker.C:
			w.cleanupDebounceMap()
		}
	}
}

func (w *LastUsedWorker) processBatch(keyIDs []uuid.UUID) {
	if len(keyIDs) == 0 {
		return
	}

	// Deduplicate
	seen := make(map[uuid.UUID]struct{})
	unique := make([]uuid.UUID, 0, len(keyIDs))
	for _, id := range keyIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			unique = append(unique, id)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var successCount int
	for _, keyID := range unique {
		if err := w.apiKeyRepo.UpdateLastUsed(ctx, keyID); err != nil {
			w.logger.Error("failed to update last used", "key_id", keyID, "error", err)
			continue
		}

		// Update debounce map
		w.mu.Lock()
		w.recentlyUpdated[keyID] = time.Now()
		w.mu.Unlock()

		successCount++
	}

	if successCount > 0 {
		w.logger.Debug("batch last used update", "count", successCount)
	}
}

func (w *LastUsedWorker) cleanupDebounceMap() {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	for keyID, lastUpdate := range w.recentlyUpdated {
		if now.Sub(lastUpdate) > 2*w.debounceInterval {
			delete(w.recentlyUpdated, keyID)
		}
	}
}
