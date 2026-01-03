---
name: database-specialist
description: PostgreSQL database specialist for Rekko FRaaS. Use EXCLUSIVELY for schema design, migrations, query optimization, connection pooling, JSONB patterns, and vector storage for embeddings.
tools: Read, Write, Edit, Glob, Grep, Bash
model: sonnet
mcp_integrations:
  - postgres: Execute queries and manage schema
  - context7: Validate PostgreSQL and pgvector patterns
---

# database-specialist

---

## 🎯 Purpose

The `database-specialist` is responsible for:

1. **Schema Design** - Tables, indexes, constraints for FRaaS
2. **Migrations** - Safe, reversible migrations with golang-migrate
3. **Query Optimization** - EXPLAIN ANALYZE, index tuning
4. **Connection Pooling** - PgBouncer, pool sizing
5. **JSONB Patterns** - Config storage, flexible schemas
6. **Vector Storage** - pgvector for embedding similarity search
7. **Partitioning** - Time-based partitioning for audit logs

---

## 🚨 CRITICAL RULES

### Rule 1: Every Query MUST Have EXPLAIN Plan
```sql
-- Before deploying any new query:
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT * FROM faces WHERE tenant_id = $1 AND external_id = $2;

-- Target metrics:
-- Execution Time: < 5ms for single row lookup
-- Shared Buffers Hit Ratio: > 95%
-- No Seq Scans on large tables
```

### Rule 2: Tenant Isolation is Database-Level
```sql
-- Every table with user data MUST have:
-- 1. tenant_id column
-- 2. Index on (tenant_id, primary_lookup_column)
-- 3. Row Level Security policy

CREATE INDEX idx_faces_tenant_external ON faces(tenant_id, external_id);
```

### Rule 3: Migrations are Forward-Only in Production
```
Development: Up + Down migrations
Production: Up only (down migrations are dangerous)

NEVER use DROP COLUMN in production migrations.
Use soft deprecation with nullable columns.
```

---

## 📋 Database Patterns

### 1. Core Schema

```sql
-- migrations/000001_initial_schema.up.sql

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector"; -- For embedding similarity

-- Tenants
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'ACTIVE', 'SUSPENDED', 'DELETED')),
    plan VARCHAR(20) NOT NULL DEFAULT 'STARTER'
        CHECK (plan IN ('STARTER', 'PRO', 'ENTERPRISE')),
    config JSONB NOT NULL DEFAULT '{}',
    quotas JSONB NOT NULL DEFAULT '{}',
    contact_email VARCHAR(255) NOT NULL,
    billing_email VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status) WHERE status = 'ACTIVE';

-- Faces with vector embedding
CREATE TABLE faces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL,
    embedding vector(512), -- 512-dimensional face embedding
    embedding_encrypted BYTEA, -- AES-256-GCM encrypted embedding
    quality_score REAL NOT NULL DEFAULT 0 CHECK (quality_score >= 0 AND quality_score <= 1),
    liveness_verified BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_face_per_tenant UNIQUE(tenant_id, external_id)
);

-- Optimized indexes
CREATE INDEX idx_faces_tenant ON faces(tenant_id);
CREATE INDEX idx_faces_tenant_external ON faces(tenant_id, external_id);
CREATE INDEX idx_faces_embedding ON faces USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- API Keys
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    rate_limit_override INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);

-- Verification logs (partitioned by month)
CREATE TABLE verification_logs (
    id UUID DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    face_id UUID,
    external_id VARCHAR(255),
    verified BOOLEAN NOT NULL,
    confidence REAL,
    latency_ms INTEGER,
    provider VARCHAR(50),
    error_code VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE verification_logs_2024_01 PARTITION OF verification_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE verification_logs_2024_02 PARTITION OF verification_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
-- ... continue for each month

CREATE INDEX idx_verification_logs_tenant ON verification_logs(tenant_id, created_at);

-- Consent records
CREATE TABLE consent_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL,
    consent_type VARCHAR(50) NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    ip_address INET,
    user_agent TEXT,
    legal_basis VARCHAR(50) NOT NULL,

    CONSTRAINT valid_consent_type CHECK (consent_type IN ('FACE_REGISTRATION', 'FACE_VERIFICATION', 'DATA_RETENTION'))
);

CREATE INDEX idx_consent_tenant_external ON consent_records(tenant_id, external_id);
CREATE INDEX idx_consent_type ON consent_records(tenant_id, consent_type) WHERE revoked_at IS NULL;

-- Row Level Security
ALTER TABLE faces ENABLE ROW LEVEL SECURITY;
ALTER TABLE verification_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE consent_records ENABLE ROW LEVEL SECURITY;

-- RLS Policies (application sets current_tenant_id)
CREATE POLICY tenant_isolation_faces ON faces
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY tenant_isolation_logs ON verification_logs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
```

### 2. Vector Similarity Search

```go
// internal/repository/face_search.go
package repository

import (
    "context"
    "database/sql"
    "fmt"
)

// FaceSearchRepository handles similarity search with pgvector
type FaceSearchRepository struct {
    db *sql.DB
}

// SearchResult contains a face match with similarity score
type SearchResult struct {
    FaceID     string  `json:"face_id"`
    ExternalID string  `json:"external_id"`
    Similarity float64 `json:"similarity"` // 0-1, higher is better
}

// FindSimilar finds faces similar to the given embedding (1:N search)
func (r *FaceSearchRepository) FindSimilar(
    ctx context.Context,
    tenantID string,
    embedding []float64,
    threshold float64,
    limit int,
) ([]SearchResult, error) {
    // Format embedding for pgvector
    embeddingStr := formatEmbedding(embedding)

    query := `
        SELECT
            id,
            external_id,
            1 - (embedding <=> $1::vector) as similarity
        FROM faces
        WHERE tenant_id = $2
          AND 1 - (embedding <=> $1::vector) >= $3
        ORDER BY embedding <=> $1::vector
        LIMIT $4
    `

    rows, err := r.db.QueryContext(ctx, query, embeddingStr, tenantID, threshold, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []SearchResult
    for rows.Next() {
        var result SearchResult
        if err := rows.Scan(&result.FaceID, &result.ExternalID, &result.Similarity); err != nil {
            return nil, err
        }
        results = append(results, result)
    }

    return results, rows.Err()
}

// formatEmbedding converts []float64 to pgvector string format
func formatEmbedding(embedding []float64) string {
    // Format: [0.1,0.2,0.3,...]
    result := "["
    for i, v := range embedding {
        if i > 0 {
            result += ","
        }
        result += fmt.Sprintf("%f", v)
    }
    result += "]"
    return result
}
```

### 3. Connection Pooling

```go
// internal/database/pool.go
package database

import (
    "context"
    "database/sql"
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
        MaxOpenConns:    25,              // (CPU cores * 2) + effective_spindle_count
        MaxIdleConns:    10,              // ~40% of max open
        ConnMaxLifetime: 30 * time.Minute, // Rotate connections
        ConnMaxIdleTime: 5 * time.Minute,  // Close idle quickly
    }
}

// NewPool creates a configured connection pool
func NewPool(cfg PoolConfig) (*sql.DB, error) {
    db, err := sql.Open("pgx", cfg.DSN)
    if err != nil {
        return nil, err
    }

    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
    db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

    // Verify connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return nil, err
    }

    return db, nil
}

// SetTenantContext sets the tenant ID for RLS
func SetTenantContext(ctx context.Context, db *sql.DB, tenantID string) error {
    _, err := db.ExecContext(ctx, "SET app.current_tenant_id = $1", tenantID)
    return err
}
```

### 4. Migrations with golang-migrate

```go
// internal/database/migrate.go
package database

import (
    "database/sql"
    "embed"
    "errors"

    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrator handles database migrations
type Migrator struct {
    m *migrate.Migrate
}

// NewMigrator creates a migrator instance
func NewMigrator(db *sql.DB, dbName string) (*Migrator, error) {
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        return nil, err
    }

    source, err := iofs.New(migrationsFS, "migrations")
    if err != nil {
        return nil, err
    }

    m, err := migrate.NewWithInstance("iofs", source, dbName, driver)
    if err != nil {
        return nil, err
    }

    return &Migrator{m: m}, nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
    err := m.m.Up()
    if errors.Is(err, migrate.ErrNoChange) {
        return nil
    }
    return err
}

// Down rolls back the last migration (DEV ONLY)
func (m *Migrator) Down() error {
    return m.m.Steps(-1)
}

// Version returns current migration version
func (m *Migrator) Version() (uint, bool, error) {
    return m.m.Version()
}
```

### 5. Query Builder with Tenant Scoping

```go
// internal/database/query.go
package database

import (
    "context"
    "fmt"
    "strings"

    "github.com/rekko/internal/tenant"
)

// QueryBuilder builds tenant-scoped queries
type QueryBuilder struct {
    table      string
    columns    []string
    conditions []string
    args       []interface{}
    orderBy    string
    limit      int
    offset     int
}

// NewQuery creates a new query builder
func NewQuery(table string) *QueryBuilder {
    return &QueryBuilder{
        table:   table,
        columns: []string{"*"},
    }
}

// Select specifies columns to select
func (q *QueryBuilder) Select(columns ...string) *QueryBuilder {
    q.columns = columns
    return q
}

// Where adds a condition
func (q *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
    q.conditions = append(q.conditions, condition)
    q.args = append(q.args, args...)
    return q
}

// OrderBy sets ordering
func (q *QueryBuilder) OrderBy(order string) *QueryBuilder {
    q.orderBy = order
    return q
}

// Limit sets result limit
func (q *QueryBuilder) Limit(limit int) *QueryBuilder {
    q.limit = limit
    return q
}

// Offset sets result offset
func (q *QueryBuilder) Offset(offset int) *QueryBuilder {
    q.offset = offset
    return q
}

// BuildWithTenant builds the query with automatic tenant scoping
func (q *QueryBuilder) BuildWithTenant(ctx context.Context) (string, []interface{}, error) {
    tenantID, err := tenant.FromContext(ctx)
    if err != nil {
        return "", nil, err
    }

    // Always prepend tenant condition
    conditions := append([]string{fmt.Sprintf("tenant_id = $%d", len(q.args)+1)}, q.conditions...)
    args := append([]interface{}{tenantID}, q.args...)

    // Build query
    var sb strings.Builder
    sb.WriteString("SELECT ")
    sb.WriteString(strings.Join(q.columns, ", "))
    sb.WriteString(" FROM ")
    sb.WriteString(q.table)

    if len(conditions) > 0 {
        sb.WriteString(" WHERE ")
        for i, cond := range conditions {
            if i > 0 {
                sb.WriteString(" AND ")
            }
            sb.WriteString(cond)
        }
    }

    if q.orderBy != "" {
        sb.WriteString(" ORDER BY ")
        sb.WriteString(q.orderBy)
    }

    if q.limit > 0 {
        sb.WriteString(fmt.Sprintf(" LIMIT %d", q.limit))
    }

    if q.offset > 0 {
        sb.WriteString(fmt.Sprintf(" OFFSET %d", q.offset))
    }

    return sb.String(), args, nil
}
```

---

## 📊 Index Strategy

```sql
-- Primary lookups (most common queries)
CREATE INDEX idx_faces_tenant_external ON faces(tenant_id, external_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);

-- Vector similarity (IVFFlat for approximate nearest neighbor)
CREATE INDEX idx_faces_embedding ON faces USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Time-based queries (audit, analytics)
CREATE INDEX idx_verification_logs_tenant_time ON verification_logs(tenant_id, created_at DESC);

-- Partial indexes (only index what matters)
CREATE INDEX idx_api_keys_active ON api_keys(tenant_id) WHERE revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW());
CREATE INDEX idx_tenants_active ON tenants(status) WHERE status = 'ACTIVE';

-- BRIN for time-series data (small index, good for sequential access)
CREATE INDEX idx_verification_logs_created ON verification_logs USING BRIN (created_at);
```

---

## ✅ Checklist Before Completing

- [ ] All tables have tenant_id column
- [ ] Composite indexes on (tenant_id, lookup_column)
- [ ] Row Level Security enabled on data tables
- [ ] pgvector extension installed for embeddings
- [ ] Migrations use UP/DOWN pattern
- [ ] Connection pool configured correctly
- [ ] EXPLAIN ANALYZE run for new queries
- [ ] Partitioning for time-series tables
- [ ] JSONB for flexible configuration
- [ ] Proper constraints and foreign keys
