package cache

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrCacheMiss is returned when a key is not found in cache
	ErrCacheMiss = errors.New("cache miss")
	// ErrCacheExpired is returned when a cached value has expired
	ErrCacheExpired = errors.New("cache expired")
)

// DB interface for database operations (compatible with pgxpool.Pool and pgxmock)
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PGCache implements a PostgreSQL-based cache with TTL support
type PGCache struct {
	db DB
}

// NewPGCache creates a new PostgreSQL cache
func NewPGCache(db *pgxpool.Pool) *PGCache {
	return &PGCache{db: db}
}

// NewPGCacheWithDB creates a new PostgreSQL cache with custom DB interface
func NewPGCacheWithDB(db DB) *PGCache {
	return &PGCache{db: db}
}

// Get retrieves a value from cache by key
func (c *PGCache) Get(ctx context.Context, key string) ([]byte, error) {
	query := `
		SELECT value, expires_at
		FROM cache_entries
		WHERE key = $1
	`

	var value []byte
	var expiresAt time.Time

	err := c.db.QueryRow(ctx, query, key).Scan(&value, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		// Delete expired entry
		_ = c.Delete(ctx, key)
		return nil, ErrCacheExpired
	}

	return value, nil
}

// Set stores a value in cache with TTL
func (c *PGCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	query := `
		INSERT INTO cache_entries (key, value, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value,
		    expires_at = EXCLUDED.expires_at,
		    created_at = NOW()
	`

	expiresAt := time.Now().Add(ttl)
	_, err := c.db.Exec(ctx, query, key, value, expiresAt)
	return err
}

// Delete removes a key from cache
func (c *PGCache) Delete(ctx context.Context, key string) error {
	query := `DELETE FROM cache_entries WHERE key = $1`
	_, err := c.db.Exec(ctx, query, key)
	return err
}

// DeletePattern removes all keys matching a pattern
func (c *PGCache) DeletePattern(ctx context.Context, pattern string) (int64, error) {
	query := `DELETE FROM cache_entries WHERE key LIKE $1`
	result, err := c.db.Exec(ctx, query, pattern)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// Clear removes all entries from cache
func (c *PGCache) Clear(ctx context.Context) (int64, error) {
	query := `DELETE FROM cache_entries`
	result, err := c.db.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// CleanupExpired removes all expired entries
func (c *PGCache) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM cache_entries WHERE expires_at < NOW()`
	result, err := c.db.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// Exists checks if a key exists and is not expired
func (c *PGCache) Exists(ctx context.Context, key string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM cache_entries
			WHERE key = $1 AND expires_at > NOW()
		)
	`

	var exists bool
	err := c.db.QueryRow(ctx, query, key).Scan(&exists)
	return exists, err
}

// GetMultiple retrieves multiple values by keys
func (c *PGCache) GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	query := `
		SELECT key, value
		FROM cache_entries
		WHERE key = ANY($1) AND expires_at > NOW()
	`

	rows, err := c.db.Query(ctx, query, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]byte)
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}

	return result, rows.Err()
}

// SetMultiple stores multiple key-value pairs with the same TTL
func (c *PGCache) SetMultiple(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := c.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO cache_entries (key, value, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value,
		    expires_at = EXCLUDED.expires_at,
		    created_at = NOW()
	`

	expiresAt := time.Now().Add(ttl)
	for key, value := range items {
		_, err := tx.Exec(ctx, query, key, value, expiresAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
