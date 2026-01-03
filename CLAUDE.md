# CLAUDE.md - Rekko FRaaS

Este arquivo fornece orientações ao Claude Code para trabalhar com o repositório Rekko.

## 🎯 Project Overview

**Rekko** é uma plataforma de **Facial Recognition as a Service (FRaaS)** B2B para entrada em eventos. O sistema oferece reconhecimento facial de alta performance com suporte a múltiplos provedores e multi-tenancy completo.

### Stack Tecnológica

| Camada | Tecnologia | Justificativa |
|--------|------------|---------------|
| **API** | Go 1.22 + Fiber | Performance (P99 < 5ms target) |
| **Database** | PostgreSQL + pgvector | Embeddings + similarity search |
| **Cache/Queue** | PostgreSQL-native | Simplicidade operacional |
| **Providers** | AWS Rekognition, Azure Face | Abstração multi-provider |
| **Container** | Docker + distroless | Security + minimal footprint |
| **CI/CD** | GitHub Actions + Railway | Deploy automatizado |

### Targets de Performance

| Métrica | Target | Critical |
|---------|--------|----------|
| P50 Latency | < 2ms | < 5ms |
| P99 Latency | < 5ms | < 10ms |
| Throughput | > 10k req/s | > 5k req/s |
| Memory/request | < 1KB | < 5KB |
| Allocs/request | < 10 | < 50 |

---

## ⚠️ FLUXO OBRIGATÓRIO DE AGENTES

**REGRA CRÍTICA**: Claude é o **Tech Lead que delega**, NÃO o desenvolvedor que codifica.

### Hierarquia de Delegação

```
Claude (Tech Lead)
  → demand-classifier (valida requisitos, planeja)
    → execution-coordinator (orquestra fases)
      → Especialistas (implementam código)
        - go-fiber-specialist
        - go-testing-specialist
        - database-specialist
        - face-recognition-architect
        - etc.
```

### Quando Usar Cada Agente

| Tipo de Tarefa | Agente |
|----------------|--------|
| Implementar issue | `demand-classifier` (PRIMEIRA ação) |
| Handlers/Controllers | `go-fiber-specialist` |
| Unit/Integration Tests | `go-testing-specialist` |
| Goroutines/Channels | `go-concurrency-specialist` |
| Performance Profiling | `go-pprof-specialist` |
| Database/Migrations | `database-specialist` |
| Cache/Queue PostgreSQL | `queue-cache-specialist` |
| Docker/Compose | `docker-specialist` |
| CI/CD/Deploy | `deploy-specialist` |
| Face Recognition | `face-recognition-architect` |
| Provider Abstraction | `provider-abstraction-specialist` |
| LGPD/Security | `biometric-security-specialist` |
| Multi-tenancy | `multi-tenancy-architect` |
| Code Review | `code-reviewer` |

### Pipeline de Implementação

```
/implement #10
  → GATE 1: Estratégia de testes?
  → demand-classifier valida issue
  → Plano gerado
  → GATE 2: Aprovar plano?
  → execution-coordinator orquestra
  → Especialistas implementam
  → Validações (lint, build, test)
  → PR criado
  → code-reviewer analisa
  → GATE 3: Aprovar merge?
  → Merge + cleanup
```

---

## 🚨 REGRAS CRÍTICAS

### 1. Multi-tenancy em TODA Query

```go
// ❌ NUNCA: Query sem tenant
db.Query("SELECT * FROM faces WHERE id = $1", id)

// ✅ SEMPRE: Tenant do contexto
tenantID, err := tenant.FromContext(ctx)
db.Query("SELECT * FROM faces WHERE tenant_id = $1 AND id = $2", tenantID, id)
```

### 2. Error Handling Correto

```go
// ❌ NUNCA: Ignorar erro
result, _ := doSomething()

// ❌ NUNCA: Erro sem contexto
if err != nil {
    return err
}

// ✅ SEMPRE: Erro com contexto
if err != nil {
    return fmt.Errorf("tenant %s: failed to search faces: %w", tenantID, err)
}
```

### 3. LGPD para Dados Biométricos

```go
// ✅ OBRIGATÓRIO: Verificar consentimento
consent, err := s.consentService.Verify(ctx, externalID, ConsentTypeFaceRegistration)
if err != nil || !consent.Granted {
    return ErrConsentRequired
}

// ✅ OBRIGATÓRIO: Criptografar embeddings at rest
encrypted, err := s.crypto.Encrypt(embedding)
if err != nil {
    return fmt.Errorf("encrypt embedding: %w", err)
}

// ✅ OBRIGATÓRIO: Audit log
s.auditLogger.Log(ctx, AuditEvent{
    Action:     "FACE_REGISTERED",
    TenantID:   tenantID,
    ExternalID: externalID,
})
```

### 4. Performance: Zero-Allocation Patterns

```go
// ❌ EVITAR: Allocations desnecessárias
var results []Result
for _, item := range items {
    results = append(results, process(item))
}

// ✅ PREFERIR: Pre-allocate
results := make([]Result, 0, len(items))
for _, item := range items {
    results = append(results, process(item))
}

// ✅ PREFERIR: sync.Pool para objetos frequentes
var embeddingPool = sync.Pool{
    New: func() interface{} {
        return make([]float64, 512)
    },
}
```

### 5. Concurrency Correta

```go
// ❌ NUNCA: Goroutine sem mecanismo de parada
go func() {
    for {
        // Loop infinito
    }
}()

// ✅ SEMPRE: Context para cancelamento
go func(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case work := <-workChan:
            process(work)
        }
    }
}(ctx)
```

---

## 📋 Comandos Disponíveis

### `/debate` - Mesa Técnica
```bash
/debate criar sistema de liveness detection
# → Debate bloqueante com perguntas socráticas
# → Gera issue BDD detalhada
```

### `/implement` - Implementar Issue
```bash
/implement #10
# → Pipeline semi-autônomo com 3 gates
# → Delegação para especialistas
```

### `/benchmark` - Performance Profiling
```bash
/benchmark scan
/benchmark profile internal/service/face.go
/benchmark load /api/v1/faces/search --vus 100
```

### `/review` - Code Review
```bash
/review #15
# → Análise com score 0-10
# → APPROVE se >= 8/10
```

---

## 📁 Estrutura do Projeto

```
rekko/
├── cmd/
│   └── api/              # Entry point
├── internal/
│   ├── api/              # HTTP handlers (Fiber)
│   ├── service/          # Business logic
│   ├── repository/       # Data access
│   ├── provider/         # Face recognition providers
│   │   ├── aws/          # AWS Rekognition
│   │   ├── azure/        # Azure Face
│   │   └── mock/         # Mock for testing
│   ├── tenant/           # Multi-tenancy
│   ├── consent/          # LGPD consent management
│   ├── crypto/           # Encryption (AES-256-GCM)
│   ├── cache/            # PostgreSQL cache
│   ├── queue/            # PostgreSQL queue
│   └── database/
│       └── migrations/   # golang-migrate
├── pkg/                  # Shared libraries
├── config/               # Configuration
├── scripts/              # Utility scripts
├── .claude/
│   ├── agents/           # Specialist agents
│   ├── commands/         # Slash commands
│   └── settings.json     # Hooks and permissions
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

---

## 🔧 Convenções de Código

### Naming

```go
// Packages: lowercase, single word
package service
package repository

// Interfaces: verb or -er suffix
type FaceProvider interface {}
type Encrypter interface {}

// Structs: noun
type FaceService struct {}
type TenantMiddleware struct {}

// Functions: verb + object
func (s *FaceService) SearchFaces(ctx context.Context, ...) {}
func (m *TenantMiddleware) ExtractTenant(c *fiber.Ctx) {}
```

### Files

```
internal/service/face.go           # Main implementation
internal/service/face_test.go      # Unit tests
internal/service/face_bench_test.go # Benchmarks
internal/service/face_integration_test.go # Integration tests
```

### Commits (Conventional Commits - INGLÊS)

```bash
feat(face): add liveness detection endpoint
fix(tenant): prevent cross-tenant data access
perf(search): reduce allocations in similarity search
test(provider): add AWS Rekognition integration tests
docs(api): update OpenAPI specification
refactor(crypto): extract encryption to separate service
```

---

## 🧪 Testes

### Unit Tests (Table-Driven)

```go
func TestFaceService_Search(t *testing.T) {
    tests := []struct {
        name      string
        tenantID  string
        embedding []float64
        threshold float64
        want      []Result
        wantErr   bool
    }{
        {
            name:      "successful search",
            tenantID:  "tenant-1",
            embedding: testEmbedding,
            threshold: 0.8,
            want:      []Result{{FaceID: "face-1", Similarity: 0.95}},
            wantErr:   false,
        },
        {
            name:      "no matches above threshold",
            tenantID:  "tenant-1",
            embedding: testEmbedding,
            threshold: 0.99,
            want:      []Result{},
            wantErr:   false,
        },
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

### Integration Tests (Testcontainers)

```go
func TestFaceRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()
    postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "pgvector/pgvector:pg16",
            ExposedPorts: []string{"5432/tcp"},
            WaitingFor:   wait.ForListeningPort("5432/tcp"),
        },
        Started: true,
    })
    require.NoError(t, err)
    defer postgres.Terminate(ctx)

    // Run tests against container...
}
```

### Benchmarks

```go
func BenchmarkFaceSearch(b *testing.B) {
    svc := setupTestService(b)
    embedding := generateTestEmbedding()

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, _ = svc.Search(testCtx, "tenant-1", embedding, 0.8, 10)
    }
}
```

---

## 🔐 Segurança

### Biometric Data (LGPD)

1. **Consentimento**: Verificar ANTES de processar
2. **Criptografia**: AES-256-GCM para embeddings at rest
3. **Audit**: Log de TODAS as operações biométricas
4. **Retenção**: Respeitar período configurado por tenant
5. **Deletion**: Implementar right to deletion

### API Security

1. **API Key**: Header X-API-Key + tenant association
2. **Rate Limiting**: Por tenant, não global
3. **Input Validation**: Validator no handler
4. **SQL**: Queries parametrizadas SEMPRE

---

## 📊 Observabilidade

### Métricas (Prometheus)

```go
// HTTP
http_requests_total{method, path, status}
http_request_duration_seconds{method, path}

// Business
face_registrations_total{tenant}
face_searches_total{tenant, provider}
face_search_duration_seconds{tenant, provider}

// Infrastructure
db_query_duration_seconds{operation}
provider_request_duration_seconds{provider}
```

### Logs (Structured JSON)

```json
{
  "level": "info",
  "msg": "face search completed",
  "tenant_id": "tenant-1",
  "faces_found": 3,
  "threshold": 0.8,
  "duration_ms": 2.5,
  "provider": "aws"
}
```

---

## 🚀 Development

### Prerequisites

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/air-verse/air@latest
go install github.com/go-delve/delve/cmd/dlv@latest
```

### Start Development

```bash
# Start services
docker compose up -d

# Run with hot reload
air

# Or manually
go run ./cmd/api serve
```

### Common Commands

```bash
# Build
go build -o bin/rekko ./cmd/api

# Test
go test -v -race ./...

# Lint
golangci-lint run ./...

# Benchmark
go test -bench=. -benchmem ./...

# Profile
go test -cpuprofile=cpu.prof -bench=BenchmarkFaceSearch ./internal/service/
go tool pprof -http=:8080 cpu.prof
```

---

## 📚 Documentação

- **Agentes**: `.claude/agents/`
- **Comandos**: `.claude/commands/`
- **API**: `docs/openapi.yaml` (quando implementado)

---

**Mantido por**: Claude Code + Equipe Rekko
**Stack**: Go 1.22 + Fiber + PostgreSQL + pgvector
**Target**: P99 < 5ms | Multi-tenant | LGPD Compliant
