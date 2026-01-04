package metrics

import (
	"context"
	"log/slog"
	"time"
)

// Aggregator performs periodic metrics aggregation
type Aggregator struct {
	repo     *Repository
	logger   *slog.Logger
	interval time.Duration
	done     chan struct{}
}

// NewAggregator creates a new metrics aggregator worker
func NewAggregator(repo *Repository, logger *slog.Logger, interval time.Duration) *Aggregator {
	if interval == 0 {
		interval = 1 * time.Minute
	}

	return &Aggregator{
		repo:     repo,
		logger:   logger,
		interval: interval,
		done:     make(chan struct{}),
	}
}

// Start begins the aggregation worker
func (a *Aggregator) Start(ctx context.Context) {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	a.logger.Info("metrics aggregator started", "interval", a.interval)

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("metrics aggregator stopped")
			return
		case <-a.done:
			a.logger.Info("metrics aggregator stopped")
			return
		case <-ticker.C:
			a.aggregate(ctx)
		}
	}
}

// Stop gracefully shuts down the aggregator
func (a *Aggregator) Stop() {
	close(a.done)
}

// aggregate performs the actual aggregation work
func (a *Aggregator) aggregate(ctx context.Context) {
	a.logger.Debug("running metrics aggregation")

	// Clean up old metrics (older than 90 days)
	deleted, err := a.repo.DeleteOldMetrics(ctx, 90*24*time.Hour)
	if err != nil {
		a.logger.Error("failed to delete old metrics", "error", err)
	} else if deleted > 0 {
		a.logger.Info("deleted old metrics", "count", deleted)
	}

	// TODO: Implement actual metric aggregation logic
	// This would involve:
	// 1. Query raw metric data (from application logs, database, etc.)
	// 2. Compute aggregations (sum, avg, p99, etc.)
	// 3. Store results via repo.SaveMetric()
	//
	// Example:
	// - API request count per tenant
	// - Average response time
	// - Error rate
	// - Face recognition success rate
	// - Storage usage
}
