# Database Migrations

Este diretório contém as migrations do banco de dados PostgreSQL do Rekko.

## Estrutura

- `000001_init.up.sql` - Migration inicial (criar schema)
- `000001_init.down.sql` - Rollback da migration inicial

## Schema Overview

### Tables

1. **tenants** - Multi-tenancy base
   - Stores tenant information, API key hash, and settings
   - Primary isolation layer for the system

2. **faces** - Facial embeddings storage
   - Stores face embeddings as 512-dimensional vectors (pgvector)
   - One face per external_id per tenant
   - Metadata stored as JSONB for flexibility

3. **verifications** - Audit log for verifications
   - Records all verification attempts
   - Tracks confidence, liveness, latency

4. **usage_records** - Billing data
   - Tracks registrations, verifications, deletions per tenant per month

## Extensions Required

- `uuid-ossp` - UUID generation
- `vector` - pgvector for similarity search

## Indexes

### Performance Indexes
- `idx_faces_tenant_external` - Fast lookup by tenant + external_id
- `idx_verifications_tenant_created` - Time-series queries for audit

### Future Index (after data load)
```sql
CREATE INDEX idx_faces_embedding ON faces
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
```

## Running Migrations

### Using golang-migrate (recommended)

```bash
# Install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Up
migrate -path ./migrations -database "postgres://rekko:rekko@localhost:5433/rekko_dev?sslmode=disable" up

# Down (rollback last)
migrate -path ./migrations -database "postgres://rekko:rekko@localhost:5433/rekko_dev?sslmode=disable" down 1

# Force version (if stuck)
migrate -path ./migrations -database "postgres://rekko:rekko@localhost:5433/rekko_dev?sslmode=disable" force 1
```

### Manual (psql)

```bash
# Up
psql -U rekko -d rekko_dev -f migrations/000001_init.up.sql

# Down
psql -U rekko -d rekko_dev -f migrations/000001_init.down.sql
```

## Connection String

```
postgres://rekko:rekko@localhost:5433/rekko_dev?sslmode=disable
```

## Docker Compose

Start PostgreSQL with pgvector:

```bash
docker compose up -d
docker compose logs -f postgres
```

Health check:
```bash
docker compose ps
```

## Best Practices

1. **Never edit migrations after they run in production**
2. **Always test down migrations in development**
3. **Use transactions** (migrations auto-wrap in golang-migrate)
4. **Index strategy**: Create after data load for better accuracy
5. **Constraints**: Use CHECK constraints for data integrity

## Monitoring

```sql
-- Check migration version
SELECT * FROM schema_migrations;

-- Table sizes
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Index usage
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```
