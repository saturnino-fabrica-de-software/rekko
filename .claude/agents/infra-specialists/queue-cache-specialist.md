---
name: queue-cache-specialist
description: PostgreSQL-based cache and queue specialist for Rekko FRaaS. Use EXCLUSIVELY for caching strategies, message queues, rate limiting, distributed locks, and background job processing - ALL using PostgreSQL native features instead of Redis/RabbitMQ.
tools: Read, Write, Edit, Glob, Grep, Bash
model: sonnet
mcp_integrations:
  - postgres: Execute queue and cache operations
  - context7: Validate PostgreSQL advisory locks and SKIP LOCKED patterns
---

# queue-cache-specialist

---

## 🎯 Purpose

The `queue-cache-specialist` is responsible for:

1. **Cache (PostgreSQL)** - Key-value storage with TTL using JSONB
2. **Message Queue (PostgreSQL)** - Async processing with SELECT FOR UPDATE
3. **Rate Limiting** - Sliding window using SQL counters
4. **Distributed Locks** - Pessimistic locking with advisory locks
5. **Background Jobs** - Cron-based consumers with circuit breakers
6. **Dead Letter Queue** - Failed message handling and retry

---

## 🚨 CRITICAL RULES

### Rule 1: PostgreSQL is MVP-Ready, Redis is Scale-Ready
```
MVP Phase (< 10k events/day):
- PostgreSQL cache: ~100 ops/s (sufficient)
- PostgreSQL queue: ~100-500 msgs/s (sufficient)

Scale Phase (> 10k events/day):
- Migrate to Redis (adapter pattern allows easy swap)
- Same interface, different backend
```

### Rule 2: TTL is Mandatory for All Cache Keys
```
EVERY cache key MUST have expires_at.
No TTL = memory/storage leak.

Cache Type         | TTL
-------------------|----------
Face metadata      | 1 hour
Tenant config      | 5 minutes
API key validation | 1 minute
Rate limit window  | 1 minute
```

### Rule 3: SELECT FOR UPDATE Prevents Race Conditions
```sql
-- ALWAYS use FOR UPDATE when dequeuing messages
BEGIN;
SELECT * FROM queue_messages
WHERE queue_name = 'face_processing'
  AND processed_at IS NULL
  AND (locked_at IS NULL OR locked_at < NOW() - INTERVAL '5 minutes')
ORDER BY priority DESC, created_at ASC
LIMIT 1
FOR UPDATE SKIP LOCKED; -- Critical for concurrency!
UPDATE queue_messages SET locked_at = NOW() WHERE id = $1;
COMMIT;
```

---

## 📋 Database Schema

```sql
-- migrations/000002_cache_and_queue.up.sql

-- ============================================
-- CACHE TABLE
-- ============================================
CREATE TABLE cache (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cache_expires ON cache(expires_at);
CREATE INDEX idx_cache_key_pattern ON cache(key varchar_pattern_ops); -- For LIKE queries

-- ============================================
-- QUEUE MESSAGES TABLE
-- ============================================
CREATE TABLE queue_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    queue_name VARCHAR(100) NOT NULL,
    body JSONB NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMPTZ,
    locked_by VARCHAR(100),
    processed_at TIMESTAMPTZ,
    retry_after TIMESTAMPTZ,
    error_message TEXT
);

CREATE INDEX idx_queue_pending ON queue_messages(queue_name, priority DESC, created_at ASC)
    WHERE processed_at IS NULL AND (locked_at IS NULL OR locked_at < NOW() - INTERVAL '5 minutes');
CREATE INDEX idx_queue_locked ON queue_messages(locked_at) WHERE locked_at IS NOT NULL;
CREATE INDEX idx_queue_retry ON queue_messages(retry_after) WHERE retry_after IS NOT NULL;

-- ============================================
-- DEAD LETTER QUEUE
-- ============================================
CREATE TABLE dead_letter_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    original_id UUID NOT NULL,
    queue_name VARCHAR(100) NOT NULL,
    body JSONB NOT NULL,
    attempts INTEGER NOT NULL,
    error_message TEXT,
    failed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reprocessed_at TIMESTAMPTZ
);

CREATE INDEX idx_dlq_queue ON dead_letter_messages(queue_name, failed_at DESC);

-- ============================================
-- RATE LIMIT COUNTERS
-- ============================================
CREATE TABLE rate_limit_counters (
    key VARCHAR(255) PRIMARY KEY,
    count INTEGER NOT NULL DEFAULT 0,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_rate_limit_window ON rate_limit_counters(window_end);

-- ============================================
-- ADVISORY LOCKS (for distributed locking)
-- ============================================
-- No table needed - PostgreSQL advisory locks are built-in
-- Use pg_advisory_lock(hash) / pg_try_advisory_lock(hash)
```

---

## 📋 Go Implementation Patterns

### 1. Cache Service

```go
// internal/cache/cache.go
package cache

import (
    "context"
    "database/sql"
    "encoding/json"
    "time"
)

// CacheService provides PostgreSQL-based caching
type CacheService struct {
    db *sql.DB
}

// NewCacheService creates a cache service
func NewCacheService(db *sql.DB) *CacheService {
    return &CacheService{db: db}
}

// Set stores a value with TTL
func (c *CacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }

    expiresAt := time.Now().Add(ttl)

    query := `
        INSERT INTO cache (key, value, expires_at, updated_at)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (key) DO UPDATE SET
            value = EXCLUDED.value,
            expires_at = EXCLUDED.expires_at,
            updated_at = NOW()
    `

    _, err = c.db.ExecContext(ctx, query, key, data, expiresAt)
    return err
}

// Get retrieves a value (returns nil if expired or not found)
func (c *CacheService) Get(ctx context.Context, key string, dest interface{}) error {
    query := `
        SELECT value FROM cache
        WHERE key = $1 AND expires_at > NOW()
    `

    var data []byte
    err := c.db.QueryRowContext(ctx, query, key).Scan(&data)
    if err == sql.ErrNoRows {
        return ErrCacheMiss
    }
    if err != nil {
        return err
    }

    return json.Unmarshal(data, dest)
}

// Delete removes a key
func (c *CacheService) Delete(ctx context.Context, key string) error {
    query := `DELETE FROM cache WHERE key = $1`
    _, err := c.db.ExecContext(ctx, query, key)
    return err
}

// DeleteByPattern removes keys matching a pattern
func (c *CacheService) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
    query := `DELETE FROM cache WHERE key LIKE $1`
    result, err := c.db.ExecContext(ctx, query, pattern)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}

// GetMultiple retrieves multiple keys
func (c *CacheService) GetMultiple(ctx context.Context, keys []string) (map[string]json.RawMessage, error) {
    query := `
        SELECT key, value FROM cache
        WHERE key = ANY($1) AND expires_at > NOW()
    `

    rows, err := c.db.QueryContext(ctx, query, keys)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    result := make(map[string]json.RawMessage)
    for rows.Next() {
        var key string
        var value json.RawMessage
        if err := rows.Scan(&key, &value); err != nil {
            return nil, err
        }
        result[key] = value
    }

    return result, rows.Err()
}

// CleanupExpired removes expired entries (run via cron)
func (c *CacheService) CleanupExpired(ctx context.Context) (int64, error) {
    query := `DELETE FROM cache WHERE expires_at < NOW()`
    result, err := c.db.ExecContext(ctx, query)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}

// Stats returns cache statistics
func (c *CacheService) Stats(ctx context.Context) (*CacheStats, error) {
    query := `
        SELECT
            COUNT(*) as total_keys,
            COUNT(*) FILTER (WHERE expires_at > NOW()) as valid_keys,
            COUNT(*) FILTER (WHERE expires_at <= NOW()) as expired_keys
        FROM cache
    `

    var stats CacheStats
    err := c.db.QueryRowContext(ctx, query).Scan(
        &stats.TotalKeys,
        &stats.ValidKeys,
        &stats.ExpiredKeys,
    )

    return &stats, err
}

// CacheStats contains cache statistics
type CacheStats struct {
    TotalKeys   int64 `json:"total_keys"`
    ValidKeys   int64 `json:"valid_keys"`
    ExpiredKeys int64 `json:"expired_keys"`
}
```

### 2. Queue Service

```go
// internal/queue/queue.go
package queue

import (
    "context"
    "database/sql"
    "encoding/json"
    "math"
    "math/rand"
    "time"
)

// QueueService provides PostgreSQL-based message queue
type QueueService struct {
    db       *sql.DB
    workerID string
}

// NewQueueService creates a queue service
func NewQueueService(db *sql.DB, workerID string) *QueueService {
    return &QueueService{
        db:       db,
        workerID: workerID,
    }
}

// Message represents a queue message
type Message struct {
    ID          string          `json:"id"`
    QueueName   string          `json:"queue_name"`
    Body        json.RawMessage `json:"body"`
    Priority    int             `json:"priority"`
    Attempts    int             `json:"attempts"`
    MaxAttempts int             `json:"max_attempts"`
    CreatedAt   time.Time       `json:"created_at"`
    LockedAt    *time.Time      `json:"locked_at,omitempty"`
    LockedBy    *string         `json:"locked_by,omitempty"`
}

// Enqueue adds a message to the queue
func (q *QueueService) Enqueue(ctx context.Context, queueName string, body interface{}, priority int) (string, error) {
    data, err := json.Marshal(body)
    if err != nil {
        return "", err
    }

    query := `
        INSERT INTO queue_messages (queue_name, body, priority, max_attempts)
        VALUES ($1, $2, $3, 3)
        RETURNING id
    `

    var id string
    err = q.db.QueryRowContext(ctx, query, queueName, data, priority).Scan(&id)
    return id, err
}

// Dequeue retrieves and locks the next message
func (q *QueueService) Dequeue(ctx context.Context, queueName string) (*Message, error) {
    tx, err := q.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    // SELECT FOR UPDATE SKIP LOCKED is critical for concurrency
    query := `
        SELECT id, queue_name, body, priority, attempts, max_attempts, created_at
        FROM queue_messages
        WHERE queue_name = $1
          AND processed_at IS NULL
          AND (locked_at IS NULL OR locked_at < NOW() - INTERVAL '5 minutes')
          AND (retry_after IS NULL OR retry_after <= NOW())
        ORDER BY priority DESC, created_at ASC
        LIMIT 1
        FOR UPDATE SKIP LOCKED
    `

    var msg Message
    err = tx.QueryRowContext(ctx, query, queueName).Scan(
        &msg.ID, &msg.QueueName, &msg.Body, &msg.Priority,
        &msg.Attempts, &msg.MaxAttempts, &msg.CreatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil // No messages available
    }
    if err != nil {
        return nil, err
    }

    // Lock the message
    lockQuery := `
        UPDATE queue_messages
        SET locked_at = NOW(), locked_by = $1, attempts = attempts + 1
        WHERE id = $2
    `
    _, err = tx.ExecContext(ctx, lockQuery, q.workerID, msg.ID)
    if err != nil {
        return nil, err
    }

    if err := tx.Commit(); err != nil {
        return nil, err
    }

    now := time.Now()
    msg.LockedAt = &now
    msg.LockedBy = &q.workerID

    return &msg, nil
}

// DequeueBatch retrieves and locks multiple messages
func (q *QueueService) DequeueBatch(ctx context.Context, queueName string, batchSize int) ([]*Message, error) {
    tx, err := q.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    query := `
        SELECT id, queue_name, body, priority, attempts, max_attempts, created_at
        FROM queue_messages
        WHERE queue_name = $1
          AND processed_at IS NULL
          AND (locked_at IS NULL OR locked_at < NOW() - INTERVAL '5 minutes')
          AND (retry_after IS NULL OR retry_after <= NOW())
        ORDER BY priority DESC, created_at ASC
        LIMIT $2
        FOR UPDATE SKIP LOCKED
    `

    rows, err := tx.QueryContext(ctx, query, queueName, batchSize)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var messages []*Message
    var ids []string
    for rows.Next() {
        var msg Message
        if err := rows.Scan(
            &msg.ID, &msg.QueueName, &msg.Body, &msg.Priority,
            &msg.Attempts, &msg.MaxAttempts, &msg.CreatedAt,
        ); err != nil {
            return nil, err
        }
        messages = append(messages, &msg)
        ids = append(ids, msg.ID)
    }

    if len(ids) > 0 {
        lockQuery := `
            UPDATE queue_messages
            SET locked_at = NOW(), locked_by = $1, attempts = attempts + 1
            WHERE id = ANY($2)
        `
        _, err = tx.ExecContext(ctx, lockQuery, q.workerID, ids)
        if err != nil {
            return nil, err
        }
    }

    if err := tx.Commit(); err != nil {
        return nil, err
    }

    return messages, nil
}

// Acknowledge marks a message as successfully processed
func (q *QueueService) Acknowledge(ctx context.Context, messageID string) error {
    query := `
        UPDATE queue_messages
        SET processed_at = NOW()
        WHERE id = $1
    `
    _, err := q.db.ExecContext(ctx, query, messageID)
    return err
}

// Reject marks a message as failed and schedules retry
func (q *QueueService) Reject(ctx context.Context, messageID string, errorMsg string) error {
    tx, err := q.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Get current attempts
    var attempts, maxAttempts int
    var body json.RawMessage
    var queueName string

    getQuery := `SELECT queue_name, body, attempts, max_attempts FROM queue_messages WHERE id = $1`
    err = tx.QueryRowContext(ctx, getQuery, messageID).Scan(&queueName, &body, &attempts, &maxAttempts)
    if err != nil {
        return err
    }

    if attempts >= maxAttempts {
        // Move to dead letter queue
        dlqQuery := `
            INSERT INTO dead_letter_messages (original_id, queue_name, body, attempts, error_message)
            VALUES ($1, $2, $3, $4, $5)
        `
        _, err = tx.ExecContext(ctx, dlqQuery, messageID, queueName, body, attempts, errorMsg)
        if err != nil {
            return err
        }

        // Remove from main queue
        deleteQuery := `DELETE FROM queue_messages WHERE id = $1`
        _, err = tx.ExecContext(ctx, deleteQuery, messageID)
        if err != nil {
            return err
        }
    } else {
        // Schedule retry with exponential backoff + jitter
        retryDelay := calculateRetryDelay(attempts)
        retryAfter := time.Now().Add(retryDelay)

        updateQuery := `
            UPDATE queue_messages
            SET locked_at = NULL, locked_by = NULL, retry_after = $1, error_message = $2
            WHERE id = $3
        `
        _, err = tx.ExecContext(ctx, updateQuery, retryAfter, errorMsg, messageID)
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}

// calculateRetryDelay uses exponential backoff with jitter
func calculateRetryDelay(attempts int) time.Duration {
    // Base delay: 2^attempts seconds (2s, 4s, 8s, ...)
    baseDelay := math.Pow(2, float64(attempts))

    // Add jitter: random 0-50% of base delay
    jitter := rand.Float64() * baseDelay * 0.5

    totalSeconds := baseDelay + jitter

    // Cap at 1 hour
    if totalSeconds > 3600 {
        totalSeconds = 3600
    }

    return time.Duration(totalSeconds) * time.Second
}

// ReleaseStuckMessages unlocks messages stuck for too long
func (q *QueueService) ReleaseStuckMessages(ctx context.Context, queueName string, stuckDuration time.Duration) (int64, error) {
    query := `
        UPDATE queue_messages
        SET locked_at = NULL, locked_by = NULL
        WHERE queue_name = $1
          AND processed_at IS NULL
          AND locked_at IS NOT NULL
          AND locked_at < NOW() - $2::INTERVAL
    `
    result, err := q.db.ExecContext(ctx, query, queueName, stuckDuration.String())
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}

// Stats returns queue statistics
func (q *QueueService) Stats(ctx context.Context, queueName string) (*QueueStats, error) {
    query := `
        SELECT
            COUNT(*) FILTER (WHERE processed_at IS NULL AND locked_at IS NULL) as pending,
            COUNT(*) FILTER (WHERE processed_at IS NULL AND locked_at IS NOT NULL) as locked,
            COUNT(*) FILTER (WHERE processed_at IS NOT NULL) as processed,
            (SELECT COUNT(*) FROM dead_letter_messages WHERE queue_name = $1) as dead_letter
        FROM queue_messages
        WHERE queue_name = $1
    `

    var stats QueueStats
    stats.QueueName = queueName
    err := q.db.QueryRowContext(ctx, query, queueName).Scan(
        &stats.Pending,
        &stats.Locked,
        &stats.Processed,
        &stats.DeadLetter,
    )

    return &stats, err
}

// QueueStats contains queue statistics
type QueueStats struct {
    QueueName  string `json:"queue_name"`
    Pending    int64  `json:"pending"`
    Locked     int64  `json:"locked"`
    Processed  int64  `json:"processed"`
    DeadLetter int64  `json:"dead_letter"`
}
```

### 3. Rate Limiter

```go
// internal/ratelimit/postgres_limiter.go
package ratelimit

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/rekko/internal/tenant"
)

// PostgresRateLimiter implements rate limiting using PostgreSQL
type PostgresRateLimiter struct {
    db     *sql.DB
    window time.Duration
}

// NewPostgresRateLimiter creates a PostgreSQL-based rate limiter
func NewPostgresRateLimiter(db *sql.DB, window time.Duration) *PostgresRateLimiter {
    return &PostgresRateLimiter{
        db:     db,
        window: window,
    }
}

// Allow checks if request is within rate limit
func (l *PostgresRateLimiter) Allow(ctx context.Context, limit int) (bool, int, error) {
    tenantID, err := tenant.FromContext(ctx)
    if err != nil {
        return false, 0, err
    }

    now := time.Now()
    windowStart := now.Truncate(l.window)
    windowEnd := windowStart.Add(l.window)
    key := fmt.Sprintf("rate_limit:%s:%d", tenantID, windowStart.Unix())

    // Upsert counter with atomic increment
    query := `
        INSERT INTO rate_limit_counters (key, count, window_start, window_end)
        VALUES ($1, 1, $2, $3)
        ON CONFLICT (key) DO UPDATE SET count = rate_limit_counters.count + 1
        RETURNING count
    `

    var count int
    err = l.db.QueryRowContext(ctx, query, key, windowStart, windowEnd).Scan(&count)
    if err != nil {
        return false, 0, err
    }

    allowed := count <= limit
    remaining := limit - count
    if remaining < 0 {
        remaining = 0
    }

    return allowed, remaining, nil
}

// CleanupOldWindows removes expired rate limit counters
func (l *PostgresRateLimiter) CleanupOldWindows(ctx context.Context) (int64, error) {
    query := `DELETE FROM rate_limit_counters WHERE window_end < NOW()`
    result, err := l.db.ExecContext(ctx, query)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}
```

### 4. Distributed Lock

```go
// internal/lock/advisory_lock.go
package lock

import (
    "context"
    "database/sql"
    "hash/fnv"
)

// AdvisoryLock implements distributed locking using PostgreSQL advisory locks
type AdvisoryLock struct {
    db *sql.DB
}

// NewAdvisoryLock creates an advisory lock manager
func NewAdvisoryLock(db *sql.DB) *AdvisoryLock {
    return &AdvisoryLock{db: db}
}

// Lock represents an acquired lock
type Lock struct {
    key    int64
    conn   *sql.Conn
}

// Acquire attempts to acquire a lock (blocking)
func (l *AdvisoryLock) Acquire(ctx context.Context, resource string) (*Lock, error) {
    key := hashResource(resource)

    conn, err := l.db.Conn(ctx)
    if err != nil {
        return nil, err
    }

    // pg_advisory_lock blocks until lock is acquired
    _, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", key)
    if err != nil {
        conn.Close()
        return nil, err
    }

    return &Lock{key: key, conn: conn}, nil
}

// TryAcquire attempts to acquire a lock (non-blocking)
func (l *AdvisoryLock) TryAcquire(ctx context.Context, resource string) (*Lock, bool, error) {
    key := hashResource(resource)

    conn, err := l.db.Conn(ctx)
    if err != nil {
        return nil, false, err
    }

    var acquired bool
    err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired)
    if err != nil {
        conn.Close()
        return nil, false, err
    }

    if !acquired {
        conn.Close()
        return nil, false, nil
    }

    return &Lock{key: key, conn: conn}, true, nil
}

// Release releases the lock
func (lock *Lock) Release(ctx context.Context) error {
    defer lock.conn.Close()
    _, err := lock.conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", lock.key)
    return err
}

// WithLock executes a function with a lock
func (l *AdvisoryLock) WithLock(ctx context.Context, resource string, fn func() error) error {
    lock, err := l.Acquire(ctx, resource)
    if err != nil {
        return err
    }
    defer lock.Release(ctx)

    return fn()
}

// hashResource converts a string to a 64-bit hash for advisory lock
func hashResource(resource string) int64 {
    h := fnv.New64a()
    h.Write([]byte(resource))
    return int64(h.Sum64())
}
```

### 5. Consumer Base Class

```go
// internal/queue/consumer.go
package queue

import (
    "context"
    "log"
    "sync"
    "time"
)

// Consumer defines the interface for queue consumers
type Consumer interface {
    Process(ctx context.Context, msg *Message) error
    QueueName() string
}

// BaseConsumer provides common consumer functionality
type BaseConsumer struct {
    queue          *QueueService
    consumer       Consumer
    pollingInterval time.Duration
    batchSize      int

    // Circuit breaker
    failureCount   int
    failureThreshold int
    circuitOpen    bool
    circuitOpenUntil time.Time
    mu             sync.Mutex
}

// NewBaseConsumer creates a base consumer
func NewBaseConsumer(queue *QueueService, consumer Consumer) *BaseConsumer {
    return &BaseConsumer{
        queue:            queue,
        consumer:         consumer,
        pollingInterval:  time.Second,
        batchSize:        10,
        failureThreshold: 5,
    }
}

// Start begins consuming messages
func (c *BaseConsumer) Start(ctx context.Context) {
    log.Printf("Starting consumer for queue: %s", c.consumer.QueueName())

    ticker := time.NewTicker(c.pollingInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            log.Printf("Stopping consumer for queue: %s", c.consumer.QueueName())
            return
        case <-ticker.C:
            if c.isCircuitOpen() {
                continue
            }

            c.processMessages(ctx)
        }
    }
}

func (c *BaseConsumer) processMessages(ctx context.Context) {
    messages, err := c.queue.DequeueBatch(ctx, c.consumer.QueueName(), c.batchSize)
    if err != nil {
        log.Printf("Error dequeuing messages: %v", err)
        c.recordFailure()
        return
    }

    for _, msg := range messages {
        if err := c.consumer.Process(ctx, msg); err != nil {
            log.Printf("Error processing message %s: %v", msg.ID, err)
            c.queue.Reject(ctx, msg.ID, err.Error())
            c.recordFailure()
        } else {
            c.queue.Acknowledge(ctx, msg.ID)
            c.recordSuccess()
        }
    }
}

func (c *BaseConsumer) isCircuitOpen() bool {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.circuitOpen && time.Now().After(c.circuitOpenUntil) {
        c.circuitOpen = false
        c.failureCount = 0
        log.Printf("Circuit breaker closed for queue: %s", c.consumer.QueueName())
    }

    return c.circuitOpen
}

func (c *BaseConsumer) recordFailure() {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.failureCount++
    if c.failureCount >= c.failureThreshold {
        c.circuitOpen = true
        c.circuitOpenUntil = time.Now().Add(30 * time.Second)
        log.Printf("Circuit breaker opened for queue: %s", c.consumer.QueueName())
    }
}

func (c *BaseConsumer) recordSuccess() {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.failureCount = 0
}
```

### 6. Face Processing Consumer Example

```go
// internal/queue/face_consumer.go
package queue

import (
    "context"
    "encoding/json"
)

// FaceProcessingPayload represents face processing job data
type FaceProcessingPayload struct {
    TenantID   string `json:"tenant_id"`
    ExternalID string `json:"external_id"`
    ImageData  []byte `json:"image_data"`
    Operation  string `json:"operation"` // "register" or "verify"
}

// FaceProcessingConsumer processes face registration/verification jobs
type FaceProcessingConsumer struct {
    faceService FaceService
}

// NewFaceProcessingConsumer creates a face processing consumer
func NewFaceProcessingConsumer(faceService FaceService) *FaceProcessingConsumer {
    return &FaceProcessingConsumer{
        faceService: faceService,
    }
}

func (c *FaceProcessingConsumer) QueueName() string {
    return "face_processing"
}

func (c *FaceProcessingConsumer) Process(ctx context.Context, msg *Message) error {
    var payload FaceProcessingPayload
    if err := json.Unmarshal(msg.Body, &payload); err != nil {
        return err
    }

    // Add tenant to context
    ctx = tenant.WithTenantID(ctx, payload.TenantID)

    switch payload.Operation {
    case "register":
        _, err := c.faceService.RegisterFace(ctx, payload.ExternalID, payload.ImageData)
        return err
    case "verify":
        _, err := c.faceService.VerifyFace(ctx, payload.ExternalID, payload.ImageData)
        return err
    default:
        return fmt.Errorf("unknown operation: %s", payload.Operation)
    }
}
```

---

## 📊 When to Migrate to Redis?

| Metric | PostgreSQL (MVP) | Redis | Migrate When |
|--------|-----------------|-------|--------------|
| Cache throughput | ~100 ops/s | ~10k ops/s | > 500 ops/s |
| Queue throughput | ~100-500 msgs/s | ~10k+ msgs/s | > 1k msgs/s |
| Daily volume | < 10k faces | > 10k faces | > 10k faces/day |

**Migration is simple** because of adapter pattern - same interface, different backend.

---

## ✅ Checklist Before Completing

- [ ] Cache table with key, value (JSONB), expires_at
- [ ] Queue table with SELECT FOR UPDATE SKIP LOCKED
- [ ] Dead letter queue for failed messages
- [ ] Exponential backoff with jitter for retries
- [ ] Advisory locks for distributed locking
- [ ] Rate limit counters with sliding window
- [ ] Circuit breaker in consumers
- [ ] Cron job for cleanup expired cache
- [ ] Cron job for releasing stuck messages
- [ ] Stats endpoints for monitoring
