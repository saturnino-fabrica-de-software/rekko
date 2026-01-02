# Development Guide - Rekko

This guide covers the development workflow for the Rekko FRaaS platform.

## Prerequisites

Install the required development tools:

```bash
make install-tools
```

This installs:
- `migrate` - Database migrations
- `golangci-lint` - Code linting
- `air` - Hot reload for development
- `goimports` - Import organization

## Quick Start

1. **Setup environment**
   ```bash
   make setup
   ```
   This will:
   - Start Docker containers (PostgreSQL)
   - Run database migrations
   - Prepare development environment

2. **Start development server with hot reload**
   ```bash
   make dev-server
   ```
   The server will automatically reload when you change Go files.

3. **Run the application manually**
   ```bash
   make run
   ```

## Development Workflow

### Hot Reload (Recommended)

Use Air for automatic recompilation during development:

```bash
make dev-server
```

Air configuration is in `.air.toml`:
- Watches: `cmd/`, `internal/`, `pkg/`
- Excludes: `*_test.go`, `vendor/`, `tmp/`
- Rebuild delay: 1000ms

### Code Quality

**Linting**
```bash
make lint
```

Configuration in `.golangci.yml`:
- Enabled linters: errcheck, gosimple, govet, ineffassign, staticcheck, gosec, etc.
- Cyclomatic complexity: max 15
- Security scanning enabled

**Formatting**
```bash
make fmt
```

Runs both `go fmt` and `goimports`.

### Testing

**Run all tests**
```bash
make test
```

**Run short tests (skip integration)**
```bash
make test-short
```

**Run with coverage**
```bash
make test-coverage
```

Generates `coverage.html` for visualization.

**Run benchmarks**
```bash
make bench
```

### Building

**Build binary**
```bash
make build
```

Output: `bin/rekko`

**Build Docker image**
```bash
make docker-build
```

Image: `rekko-api:latest`

The Dockerfile uses multi-stage build:
- **Builder stage**: golang:1.22-alpine
- **Final stage**: distroless (minimal attack surface)
- Image size target: < 20MB

### Database Operations

**Run migrations**
```bash
make db-migrate-up
```

**Rollback last migration**
```bash
make db-migrate-down
```

**Reset database (destructive!)**
```bash
make db-migrate-reset
```

**Check migration version**
```bash
make db-migrate-version
```

**Connect to database**
```bash
make db-psql
```

**Database status**
```bash
make db-status
```

### Docker Operations

**Start all services**
```bash
make docker-up
```

**Stop all services**
```bash
make docker-down
```

**View logs**
```bash
make docker-logs
```

**Clean containers and volumes**
```bash
make docker-clean
```

### Cleanup

**Clean build artifacts**
```bash
make clean
```

Removes:
- `bin/`
- `tmp/`
- `coverage.out`
- `coverage.html`
- `build-errors.log`

## Project Structure

```
rekko/
├── cmd/api/              # Application entry point
├── internal/             # Private application code
│   ├── api/             # HTTP handlers (Fiber)
│   ├── service/         # Business logic
│   ├── repository/      # Data access layer
│   ├── provider/        # Face recognition providers
│   ├── tenant/          # Multi-tenancy
│   ├── consent/         # LGPD consent
│   ├── crypto/          # Encryption
│   └── database/        # Database utilities
├── pkg/                 # Public libraries
├── migrations/          # SQL migrations
├── config/              # Configuration files
├── scripts/             # Utility scripts
└── docs/                # Documentation
```

## Environment Variables

Copy `.env.example` to `.env`:

```bash
cp .env.example .env
```

Key variables:
- `ENV` - Environment (development, production)
- `PORT` - Server port (default: 3000)
- `DATABASE_URL` - PostgreSQL connection string
- `LOG_LEVEL` - Logging level (debug, info, warn, error)
- `FACE_PROVIDER` - Provider to use (aws, azure, mock)

## Common Commands

| Command | Description |
|---------|-------------|
| `make help` | Show all available commands |
| `make setup` | Full setup (docker + migrations) |
| `make dev-server` | Start with hot reload |
| `make test` | Run all tests |
| `make lint` | Run linter |
| `make build` | Build binary |
| `make docker-build` | Build Docker image |
| `make clean` | Clean artifacts |

## Coding Standards

### Error Handling

Always provide context:

```go
// ❌ Bad
if err != nil {
    return err
}

// ✅ Good
if err != nil {
    return fmt.Errorf("tenant %s: failed to search faces: %w", tenantID, err)
}
```

### Multi-tenancy

Always include tenant_id:

```go
// ❌ Bad
db.Query("SELECT * FROM faces WHERE id = $1", id)

// ✅ Good
tenantID, err := tenant.FromContext(ctx)
db.Query("SELECT * FROM faces WHERE tenant_id = $1 AND id = $2", tenantID, id)
```

### Performance

Pre-allocate slices when size is known:

```go
// ❌ Avoid
var results []Result
for _, item := range items {
    results = append(results, process(item))
}

// ✅ Prefer
results := make([]Result, 0, len(items))
for _, item := range items {
    results = append(results, process(item))
}
```

## Troubleshooting

**Air not reloading**
- Check `.air.toml` configuration
- Ensure you're in the project root
- Check `build-errors.log`

**Linter errors**
- Run `make fmt` to auto-fix formatting issues
- Check `.golangci.yml` for disabled rules

**Database connection issues**
- Verify Docker is running: `docker ps`
- Check DATABASE_URL in `.env`
- Ensure migrations are up to date: `make db-migrate-version`

**Build failures**
- Clean and rebuild: `make clean && make build`
- Check Go version: `go version` (requires 1.22+)

## Performance Targets

| Metric | Target | Critical |
|--------|--------|----------|
| P50 Latency | < 2ms | < 5ms |
| P99 Latency | < 5ms | < 10ms |
| Memory/request | < 1KB | < 5KB |
| Allocs/request | < 10 | < 50 |

Monitor with:
```bash
make bench
```

## Security

- **Never** commit `.env` files
- **Always** use parameterized queries
- **Always** verify consent before processing biometric data
- **Always** encrypt embeddings at rest

## Additional Resources

- [Fiber Documentation](https://docs.gofiber.io/)
- [PostgreSQL + pgvector](https://github.com/pgvector/pgvector)
- [golangci-lint](https://golangci-lint.run/)
- [Air](https://github.com/air-verse/air)
