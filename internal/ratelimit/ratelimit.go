package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB interface for database operations
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

// RateLimiter provides PostgreSQL-based rate limiting with sliding window
type RateLimiter struct {
	db     DB
	window time.Duration
}

// NewRateLimiter creates a new rate limiter with sliding window
func NewRateLimiter(db *pgxpool.Pool, window time.Duration) *RateLimiter {
	return &RateLimiter{
		db:     db,
		window: window,
	}
}

// NewRateLimiterWithDB creates a rate limiter with custom DB interface
func NewRateLimiterWithDB(db DB, window time.Duration) *RateLimiter {
	return &RateLimiter{
		db:     db,
		window: window,
	}
}

// CheckSearchLimit checks if tenant has exceeded search rate limit
// Returns error if limit exceeded, nil otherwise
func (r *RateLimiter) CheckSearchLimit(ctx context.Context, tenantID uuid.UUID, limit int) error {
	if limit <= 0 {
		return nil // No limit configured
	}

	now := time.Now()
	windowStart := now.Add(-r.window)
	key := fmt.Sprintf("search_rate:%s", tenantID)

	// Use ON CONFLICT to atomically increment or insert counter
	query := `
		WITH current_count AS (
			INSERT INTO rate_limit_counters (key, count, window_start, window_end, tenant_id)
			VALUES ($1, 1, $2, $3, $4)
			ON CONFLICT (key)
			DO UPDATE SET
				count = CASE
					WHEN rate_limit_counters.window_end < $2 THEN 1
					ELSE rate_limit_counters.count + 1
				END,
				window_start = CASE
					WHEN rate_limit_counters.window_end < $2 THEN $2
					ELSE rate_limit_counters.window_start
				END,
				window_end = $3
			RETURNING count, window_start
		)
		SELECT count FROM current_count
	`

	var count int
	err := r.db.QueryRow(ctx, query, key, windowStart, now, tenantID).Scan(&count)
	if err != nil {
		return fmt.Errorf("check rate limit: %w", err)
	}

	if count > limit {
		return fmt.Errorf("rate limit exceeded: %d/%d requests in window", count, limit)
	}

	return nil
}

// CleanupExpired removes expired rate limit counters (run via cron)
func (r *RateLimiter) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM rate_limit_counters WHERE window_end < NOW() - INTERVAL '1 hour'`
	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// GetCurrentCount returns the current count for a tenant (for testing/monitoring)
func (r *RateLimiter) GetCurrentCount(ctx context.Context, tenantID uuid.UUID) (int, error) {
	key := fmt.Sprintf("search_rate:%s", tenantID)
	windowStart := time.Now().Add(-r.window)

	query := `
		SELECT count
		FROM rate_limit_counters
		WHERE key = $1 AND window_end > $2
	`

	var count int
	err := r.db.QueryRow(ctx, query, key, windowStart).Scan(&count)
	if err != nil {
		return 0, nil // No records = 0 count
	}

	return count, nil
}

// ResetLimit resets the rate limit for a tenant (admin operation)
func (r *RateLimiter) ResetLimit(ctx context.Context, tenantID uuid.UUID) error {
	key := fmt.Sprintf("search_rate:%s", tenantID)
	query := `DELETE FROM rate_limit_counters WHERE key = $1`
	_, err := r.db.Exec(ctx, query, key)
	return err
}
