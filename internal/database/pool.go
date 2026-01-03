package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PoolConfig defines connection pool settings
type PoolConfig struct {
	DSN             string
	MaxOpenConns    int           // Max open connections
	MaxIdleConns    int           // Max idle connections
	ConnMaxLifetime time.Duration // Max connection lifetime
	ConnMaxIdleTime time.Duration // Max idle time before close
}

// DefaultPoolConfig returns optimized pool settings for Rekko
func DefaultPoolConfig(dsn string) PoolConfig {
	return PoolConfig{
		DSN:             dsn,
		MaxOpenConns:    25,               // (CPU cores * 2) + effective_spindle_count
		MaxIdleConns:    10,               // ~40% of max open
		ConnMaxLifetime: 30 * time.Minute, // Rotate connections
		ConnMaxIdleTime: 5 * time.Minute,  // Close idle quickly
	}
}

// NewPool creates a configured connection pool
func NewPool(cfg PoolConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

// HealthCheck verifies database connectivity
func HealthCheck(ctx context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database unhealthy: %w", err)
	}

	return nil
}

// Stats returns connection pool statistics
func Stats(db *sql.DB) sql.DBStats {
	return db.Stats()
}
