# Database Migrations

Este diretório contém o código de gerenciamento de migrations do Rekko usando [golang-migrate](https://github.com/golang-migrate/migrate).

## Estrutura

```
internal/database/
├── migrations/
│   ├── 000001_init.up.sql      # Migration inicial (schema completo)
│   └── 000001_init.down.sql    # Rollback da migration inicial
├── migrate.go                   # Código de migração
├── pool.go                      # Connection pool
└── README.md
```

## Schema

### Tabelas

#### `tenants`
Multi-tenant organizations usando Rekko FRaaS.

| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key (auto-generated) |
| name | VARCHAR(255) | Nome do tenant |
| slug | VARCHAR(100) | Identificador único (URL-safe) |
| is_active | BOOLEAN | Status ativo/inativo |
| plan | VARCHAR(50) | Plano (starter, pro, enterprise) |
| settings | JSONB | Configurações flexíveis |
| created_at | TIMESTAMPTZ | Data de criação |
| updated_at | TIMESTAMPTZ | Data de atualização (auto-update via trigger) |

**Indexes:**
- `idx_tenants_slug` - Lookup por slug
- `idx_tenants_active` - Partial index para tenants ativos

**Constraints:**
- UNIQUE slug
- CHECK plan IN ('starter', 'pro', 'enterprise')

#### `api_keys`
API keys para autenticação de tenants (formato Stripe-like).

| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key (auto-generated) |
| tenant_id | UUID | FK para tenants (CASCADE DELETE) |
| name | VARCHAR(255) | Nome descritivo da key |
| key_hash | VARCHAR(64) | SHA256 hash da key completa |
| key_prefix | VARCHAR(16) | Primeiros 16 chars (ex: rekko_test_abc1) |
| environment | VARCHAR(10) | 'test' ou 'live' |
| is_active | BOOLEAN | Status ativo/revogado |
| last_used_at | TIMESTAMPTZ | Última utilização (nullable) |
| created_at | TIMESTAMPTZ | Data de criação |

**Indexes:**
- `idx_api_keys_hash` - Partial index para keys ativas (lookup rápido)
- `idx_api_keys_tenant` - Lookup por tenant
- `idx_api_keys_environment` - Composite index (tenant_id, environment) para keys ativas

**Constraints:**
- UNIQUE key_hash
- CHECK environment IN ('test', 'live')
- FK tenant_id REFERENCES tenants(id) ON DELETE CASCADE

### Formato de API Key

```
rekko_{environment}_{random}
```

**Exemplos:**
- `rekko_test_1a2b3c4d5e6f7g8h` - Desenvolvimento/Staging
- `rekko_live_9z8y7x6w5v4u3t2s` - Produção

## Executando Migrations

### Via Comando CLI

```bash
# Aplicar todas as migrations pendentes
go run ./cmd/migrate -action=up

# Reverter última migration (DEV ONLY)
go run ./cmd/migrate -action=down

# Ver versão atual
go run ./cmd/migrate -action=version

# Forçar versão específica (DANGEROUS)
go run ./cmd/migrate -action=force -steps=1
```

### Via Makefile

```bash
# Aplicar migrations
make migrate-up

# Reverter última migration
make migrate-down

# Ver status
make migrate-status

# Criar nova migration
make migrate-create NAME=add_users_table
```

### Programaticamente

```go
package main

import (
    "database/sql"
    "log"

    "github.com/saturnino-fabrica-de-software/rekko/internal/database"
)

func main() {
    db, err := sql.Open("pgx", "postgres://user:pass@localhost:5432/rekko_dev")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    migrator, err := database.NewMigrator(db, "rekko_dev")
    if err != nil {
        log.Fatal(err)
    }
    defer migrator.Close()

    if err := migrator.Up(); err != nil {
        log.Fatal(err)
    }

    log.Println("Migrations completed")
}
```

## Criando Novas Migrations

### Convenção de Nomenclatura

```
{version}_{description}.{up|down}.sql
```

**Exemplos:**
- `000001_init.up.sql`
- `000001_init.down.sql`
- `000002_add_faces_table.up.sql`
- `000002_add_faces_table.down.sql`

### Template

```sql
-- 000002_add_faces_table.up.sql
CREATE TABLE faces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL,
    embedding BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_face_per_tenant UNIQUE(tenant_id, external_id)
);

CREATE INDEX idx_faces_tenant ON faces(tenant_id);
```

```sql
-- 000002_add_faces_table.down.sql
DROP INDEX IF EXISTS idx_faces_tenant;
DROP TABLE IF EXISTS faces;
```

## Regras de Migrations

### ✅ DO

1. **Sempre criar UP e DOWN** - Migrations devem ser reversíveis
2. **Usar transactions implícitas** - PostgreSQL usa DDL transacional
3. **Testar rollback** - Validar que DOWN funciona corretamente
4. **Indexes otimizados** - Partial indexes para queries comuns
5. **CHECK constraints** - Validação no nível do banco
6. **Foreign keys com CASCADE** - Manter integridade referencial
7. **Comentários em tabelas** - Documentar propósito das tabelas

### ❌ DON'T

1. **NUNCA DROP COLUMN em produção** - Use soft deprecation
2. **NUNCA modificar migrations aplicadas** - Crie novas migrations
3. **NUNCA usar FORCE em produção** - Apenas em desenvolvimento
4. **NUNCA migrations sem testes** - Valide antes de aplicar
5. **NUNCA migrations destrutivas sem backup** - Backup primeiro

## Testando Migrations

### Script de Teste Automatizado

```bash
./scripts/test-migrations.sh
```

Este script:
1. Verifica conexão com PostgreSQL
2. Valida extensões (pgcrypto)
3. Verifica estrutura de tabelas
4. Valida indexes e constraints
5. Testa inserção de dados
6. Cleanup automático

### Testes Manuais

```bash
# Conectar ao banco
psql -h localhost -U rekko -d rekko_dev

# Verificar tabelas
\dt

# Verificar schema da tabela
\d tenants
\d api_keys

# Verificar indexes
\di

# Verificar constraints
\d+ tenants

# Inserir tenant de teste
INSERT INTO tenants (name, slug, plan, settings)
VALUES ('Test Org', 'test-org', 'pro', '{"max_faces": 1000}')
RETURNING id;

# Inserir API key de teste
INSERT INTO api_keys (tenant_id, name, key_hash, key_prefix, environment)
VALUES ('UUID_DO_TENANT', 'Test Key', 'hash123', 'rekko_test_abc1', 'test')
RETURNING id;

# Verificar foreign key cascade
DELETE FROM tenants WHERE slug = 'test-org';
-- Deve deletar api_keys automaticamente
```

## Troubleshooting

### Migration Dirty

Se uma migration falhar parcialmente:

```bash
# Verificar status
go run ./cmd/migrate -action=version
# Output: Current version: 1 (DIRTY)

# Corrigir manualmente no banco e forçar versão
go run ./cmd/migrate -action=force -steps=1
```

### Rollback Manual

```sql
-- Ver histórico de migrations
SELECT * FROM schema_migrations;

-- Reverter manualmente se necessário
-- Execute o conteúdo do arquivo .down.sql correspondente
```

### Recrear Schema (DEV ONLY)

```bash
# Dropar e recriar banco (PERDE TODOS OS DADOS)
psql -h localhost -U postgres -c "DROP DATABASE rekko_dev;"
psql -h localhost -U postgres -c "CREATE DATABASE rekko_dev OWNER rekko;"

# Aplicar migrations do zero
go run ./cmd/migrate -action=up
```

## Performance

### Connection Pool

O pool é configurado em `pool.go`:

```go
MaxOpenConns: 25      // (CPU cores * 2) + effective_spindle_count
MaxIdleConns: 10      // ~40% de max open
ConnMaxLifetime: 30m  // Rotacionar conexões
ConnMaxIdleTime: 5m   // Fechar idle rapidamente
```

### Índices

Todos os índices foram otimizados para queries comuns:

1. **Partial indexes** - Só indexa rows relevantes (ex: `WHERE is_active = true`)
2. **Composite indexes** - Ordem correta para queries (tenant_id, environment)
3. **UNIQUE indexes** - Garante unicidade (slug, key_hash)

## Referências

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [PostgreSQL DDL Transactions](https://www.postgresql.org/docs/current/ddl.html)
- [pgx Driver](https://github.com/jackc/pgx)
