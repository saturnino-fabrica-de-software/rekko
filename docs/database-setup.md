# Database Setup - Rekko FRaaS

Documentação completa da configuração de banco de dados do Rekko.

## Stack

- **PostgreSQL 16** - Database principal
- **pgvector** - Extensão para similarity search com embeddings
- **golang-migrate** - Gerenciamento de migrations

## Estrutura

```
rekko/
├── docker-compose.yml           # PostgreSQL container
├── migrations/
│   ├── 000001_init.up.sql      # Schema inicial
│   ├── 000001_init.down.sql    # Rollback
│   └── README.md               # Documentação detalhada
├── scripts/
│   └── db.sh                   # Utilitário de gerenciamento
└── Makefile                    # Comandos rápidos
```

## Schema Overview

### 1. tenants
Base do multi-tenancy. Cada tenant é isolado.

```sql
CREATE TABLE tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    api_key_hash VARCHAR(255) UNIQUE,
    settings JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
```

**Indexes:**
- `idx_tenants_api_key_hash` - Lookup rápido por API key
- `idx_tenants_is_active` - Filtro por status

### 2. faces
Storage de embeddings faciais (512 dimensões).

```sql
CREATE TABLE faces (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    external_id VARCHAR(255) NOT NULL,
    embedding vector(512),           -- pgvector
    metadata JSONB DEFAULT '{}',
    quality_score DECIMAL(5,4),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    UNIQUE(tenant_id, external_id)
);
```

**Indexes:**
- `idx_faces_tenant_id` - Tenant isolation
- `idx_faces_tenant_external` - Composite para lookup
- `idx_faces_embedding` - IVFFlat para similarity search (criar após seed)

**Embedding Index (criar manualmente após seed):**
```sql
CREATE INDEX idx_faces_embedding ON faces
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
```

### 3. verifications
Audit log de verificações (compliance LGPD).

```sql
CREATE TABLE verifications (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    face_id UUID REFERENCES faces(id),
    external_id VARCHAR(255) NOT NULL,
    verified BOOLEAN NOT NULL,
    confidence DECIMAL(5,4),
    liveness_passed BOOLEAN,
    latency_ms INTEGER,
    created_at TIMESTAMPTZ
);
```

**Indexes:**
- `idx_verifications_tenant_created` - Time-series queries

### 4. usage_records
Metering para billing.

```sql
CREATE TABLE usage_records (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    period VARCHAR(7) NOT NULL,      -- YYYY-MM
    registrations INTEGER DEFAULT 0,
    verifications INTEGER DEFAULT 0,
    deletions INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    UNIQUE(tenant_id, period)
);
```

## Quick Start

### 1. Iniciar PostgreSQL

```bash
# Usando docker-compose
docker compose up -d

# Ou usando Makefile
make docker-up

# Verificar logs
docker compose logs -f postgres
```

### 2. Verificar Status

```bash
# Usando script
./scripts/db.sh status

# Ou usando Makefile
make db-status
```

**Output esperado:**
```
ℹ Database status:
  Host: localhost:5433
  Database: rekko_dev
  User: rekko
✓ Container rekko-postgres is running
✓ Database is accessible
```

### 3. Rodar Migrations

```bash
# Instalar golang-migrate (se não tiver)
make install-tools

# Rodar migrations
./scripts/db.sh up
# ou
make db-migrate-up
```

### 4. Seed com Dados de Teste

```bash
./scripts/db.sh seed
# ou
make db-seed
```

### 5. Conectar via psql

```bash
./scripts/db.sh psql
# ou
make db-psql
```

## Comandos Disponíveis

### Via Makefile (Recomendado)

```bash
make help                  # Lista todos os comandos

# Docker
make docker-up            # Start PostgreSQL
make docker-down          # Stop PostgreSQL
make docker-logs          # View logs
make docker-clean         # Remove containers + volumes

# Database
make db-status            # Status do database
make db-psql              # Conectar via psql
make db-seed              # Seed com dados de teste
make db-dump              # Criar backup

# Migrations
make db-migrate-up        # Run migrations
make db-migrate-down      # Rollback last migration
make db-migrate-reset     # Drop + recreate
make db-migrate-version   # Show current version

# Workflow
make dev                  # Start tudo (docker + migrations)
make dev-clean            # Stop tudo
```

### Via Script (./scripts/db.sh)

```bash
./scripts/db.sh up         # Run migrations
./scripts/db.sh down       # Rollback last
./scripts/db.sh reset      # Drop + recreate
./scripts/db.sh version    # Show version
./scripts/db.sh force 1    # Force version
./scripts/db.sh status     # Database status
./scripts/db.sh psql       # Connect to db
./scripts/db.sh dump       # Create backup
./scripts/db.sh restore    # Restore backup
./scripts/db.sh seed       # Seed test data
```

## Connection Strings

### Development
```
postgres://rekko:rekko@localhost:5433/rekko_dev?sslmode=disable
```

### Production (Railway/Render)
```
postgres://<user>:<password>@<host>:<port>/<database>?sslmode=require
```

## Backup & Restore

### Manual Backup

```bash
# Via script
./scripts/db.sh dump ./backups/my_backup.sql

# Via pg_dump direto
pg_dump -h localhost -p 5433 -U rekko -d rekko_dev > backup.sql
```

### Restore

```bash
# Via script
./scripts/db.sh restore ./backups/my_backup.sql

# Via psql direto
psql -h localhost -p 5433 -U rekko -d rekko_dev < backup.sql
```

## Monitoring Queries

### Migration Status

```sql
SELECT * FROM schema_migrations;
```

### Table Sizes

```sql
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

### Index Usage

```sql
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

### Active Connections

```sql
SELECT
    datname,
    count(*) as connections
FROM pg_stat_activity
WHERE datname IS NOT NULL
GROUP BY datname;
```

### Slow Queries (requires pg_stat_statements)

```sql
SELECT
    query,
    calls,
    total_exec_time,
    mean_exec_time,
    max_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
```

## Performance Tuning

### Connection Pooling

```go
db.SetMaxOpenConns(25)              // Max open connections
db.SetMaxIdleConns(10)              // Max idle connections
db.SetConnMaxLifetime(30 * time.Minute)
db.SetConnMaxIdleTime(5 * time.Minute)
```

### Vector Index Optimization

Após inserir dados significativos (> 1000 faces):

```sql
-- Recalcular statistics
ANALYZE faces;

-- Criar índice IVFFlat
CREATE INDEX CONCURRENTLY idx_faces_embedding ON faces
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- Verificar uso
SELECT * FROM pg_stat_user_indexes WHERE indexrelname = 'idx_faces_embedding';
```

## Troubleshooting

### Container não sobe

```bash
# Verificar logs
docker compose logs postgres

# Verificar porta em uso
lsof -i :5433

# Limpar volumes
docker compose down -v
docker compose up -d
```

### Migration stuck

```bash
# Ver versão atual
./scripts/db.sh version

# Forçar versão (cuidado!)
./scripts/db.sh force 1
```

### Connection refused

```bash
# Verificar se container está rodando
docker compose ps

# Verificar health check
docker inspect rekko-postgres | grep Health

# Testar conexão
pg_isready -h localhost -p 5433 -U rekko
```

### pgvector extension missing

```bash
# Conectar ao database
make db-psql

# Criar extension manualmente
CREATE EXTENSION IF NOT EXISTS vector;

# Verificar versão
SELECT * FROM pg_available_extensions WHERE name = 'vector';
```

## Best Practices

### ✅ DO

1. **Sempre usar tenant_id nas queries**
   ```go
   db.Query("SELECT * FROM faces WHERE tenant_id = $1 AND external_id = $2", tenantID, externalID)
   ```

2. **Usar transações para operações múltiplas**
   ```go
   tx, _ := db.Begin()
   defer tx.Rollback()
   // ... operations
   tx.Commit()
   ```

3. **Criar índice IVFFlat APÓS seed de dados**

4. **Fazer backup antes de migrations em produção**

5. **Usar EXPLAIN ANALYZE para queries novas**
   ```sql
   EXPLAIN ANALYZE SELECT * FROM faces WHERE tenant_id = '...' AND external_id = '...';
   ```

### ❌ DON'T

1. **NUNCA fazer query sem tenant_id**
2. **NUNCA usar SELECT * em produção**
3. **NUNCA editar migrations após deploy**
4. **NUNCA usar DOWN migrations em produção**
5. **NUNCA armazenar embeddings sem criptografia (produção)**

## Security

### Multi-tenancy Isolation

Toda query DEVE incluir `tenant_id`:

```go
// ❌ ERRADO
db.Query("SELECT * FROM faces WHERE id = $1", id)

// ✅ CORRETO
tenantID := tenant.FromContext(ctx)
db.Query("SELECT * FROM faces WHERE tenant_id = $1 AND id = $2", tenantID, id)
```

### Row Level Security (Futuro)

```sql
ALTER TABLE faces ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON faces
USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

### Encryption at Rest (Produção)

```go
// Criptografar embedding antes de salvar
encrypted, err := crypto.Encrypt(embedding)
db.Exec("INSERT INTO faces (embedding) VALUES ($1)", encrypted)
```

## References

- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [IVFFlat Index](https://github.com/pgvector/pgvector#ivfflat)

## Changelog

### v0.0.1 - Initial Setup
- PostgreSQL 16 + pgvector
- Schema multi-tenant (tenants, faces, verifications, usage_records)
- Migrations com golang-migrate
- Scripts de gerenciamento (db.sh, Makefile)
