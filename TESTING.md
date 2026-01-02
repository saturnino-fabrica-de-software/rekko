# Testing Guide - Rekko

Este guia explica como executar os diferentes tipos de testes no projeto Rekko.

## Tipos de Testes

### 1. Unit Tests
Testes de unidade rápidos que não dependem de recursos externos.

```bash
# Executar apenas testes de unidade
make test-unit

# Ou diretamente com Go
go test -v -short ./...
```

### 2. Integration Tests
Testes de integração que usam testcontainers para criar um PostgreSQL real.

```bash
# Executar testes de integração
make test-integration

# Ou diretamente com Go
go test -v -tags=integration ./...
```

**Requisitos:**
- Docker deve estar rodando
- Testcontainers irá baixar a imagem `pgvector/pgvector:pg16` automaticamente

### 3. All Tests
Executar todos os testes com race detector.

```bash
# Executar todos os testes
make test

# Ou diretamente com Go
go test -v -race ./...
```

### 4. Coverage
Executar testes e gerar relatório de cobertura.

```bash
# Gerar relatório de cobertura
make test-coverage

# Abre o arquivo coverage.html no navegador
open coverage.html
```

### 5. Benchmarks
Executar benchmarks de performance.

```bash
# Executar benchmarks
make bench

# Ou diretamente com Go
go test -bench=. -benchmem ./...
```

## Estrutura de Testes

```
rekko/
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go          # Unit tests
│   ├── domain/
│   │   ├── errors.go
│   │   └── errors_test.go          # Unit tests
│   └── api/
│       ├── handler/
│       │   ├── health.go
│       │   └── health_test.go      # Unit tests
│       └── integration_test.go     # Integration tests (build tag)
```

## Convenções

### Build Tags
- Integration tests usam o build tag `//go:build integration`
- Isso permite executá-los separadamente dos unit tests

### Table-Driven Tests
Todos os testes seguem o padrão table-driven:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {
            name:    "caso de sucesso",
            input:   validInput,
            want:    expectedOutput,
            wantErr: false,
        },
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Parallel Tests
Testes que podem rodar em paralelo devem usar:

```go
t.Run(tt.name, func(t *testing.T) {
    t.Parallel()
    // Test implementation
})
```

## CI/CD

Os testes são executados automaticamente no GitHub Actions:

```yaml
# .github/workflows/test.yml
- name: Run Unit Tests
  run: make test-unit

- name: Run Integration Tests
  run: make test-integration
```

## Targets de Cobertura

| Package | Target Coverage |
|---------|----------------|
| `internal/config` | >= 80% |
| `internal/domain` | >= 90% |
| `internal/api/handler` | >= 80% |
| `internal/service` | >= 90% |

## Troubleshooting

### Integration Tests Falhando

**Problema:** `Failed to start container`

**Solução:**
1. Verificar se Docker está rodando: `docker ps`
2. Verificar espaço em disco: `docker system df`
3. Limpar containers antigos: `docker system prune`

### Race Detector Lento

**Problema:** Testes com `-race` muito lentos

**Solução:**
- Usar `make test-unit` para desenvolvimento rápido
- Race detector apenas antes de commit/PR

### Cobertura Baixa

**Problema:** Coverage < 80%

**Solução:**
1. Identificar pacotes com baixa cobertura:
   ```bash
   go test -coverprofile=coverage.out ./...
   go tool cover -func=coverage.out | grep -E "^total:|low_package"
   ```
2. Adicionar testes para casos não cobertos

## Exemplos

### Executar Testes de um Pacote Específico

```bash
# Apenas config
go test -v ./internal/config/

# Apenas domain
go test -v ./internal/domain/

# Apenas handlers
go test -v ./internal/api/handler/
```

### Executar um Teste Específico

```bash
# Por nome
go test -v -run TestLoad ./internal/config/

# Com padrão
go test -v -run TestAppError ./internal/domain/
```

### Debug de Testes

```bash
# Com logs detalhados
go test -v -race ./... 2>&1 | tee test.log

# Com delve (debugger)
dlv test ./internal/config/
```

## Recursos

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testcontainers Go](https://golang.testcontainers.org/)
- [Table Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [Go Test Comments](https://pkg.go.dev/cmd/go#hdr-Test_packages)
