# /review - Code Review Automatizado

Code review automatizado para garantir qualidade, segurança e aderência aos padrões do Rekko.

**Filosofia**: "Review rigoroso mas construtivo. Bloquear problemas, sugerir melhorias."
**Input**: PR number, branch, ou arquivos modificados
**Output**: Relatório de review com score 0-10 e ação (APPROVE/REQUEST_CHANGES)
**UX**: Executado automaticamente no pipeline ou sob demanda

---

## 🎯 Critérios de Avaliação (10 pontos)

| Critério | Peso | Descrição |
|----------|------|-----------|
| **Error Handling** | 2.0 | Tratamento adequado de erros |
| **Concurrency** | 1.5 | Uso correto de goroutines/channels |
| **Testing** | 1.5 | Cobertura e qualidade dos testes |
| **Security** | 1.5 | LGPD, secrets, input validation |
| **Multi-tenancy** | 1.0 | Isolamento de tenant |
| **Performance** | 1.0 | P99, allocations, GC |
| **Code Style** | 0.5 | golangci-lint, formatting |
| **Documentation** | 0.5 | Comentários, godoc |
| **Maintainability** | 0.5 | Clean code, DRY, SOLID |

**Score mínimo para APPROVE**: 8.0/10

---

## 📋 Checklist de Review

### 1. Error Handling (2.0 pontos)

```go
// ❌ BLOQUEANTE: Error ignorado
result, _ := doSomething()

// ❌ BLOQUEANTE: Error não propagado
if err != nil {
    log.Printf("error: %v", err)
    // Não retorna erro!
}

// ✅ CORRETO: Error tratado e propagado
if err != nil {
    return nil, fmt.Errorf("failed to do something: %w", err)
}

// ✅ CORRETO: Error com contexto
if err != nil {
    return nil, fmt.Errorf("tenant %s: %w", tenantID, err)
}
```

**Checklist**:
- [ ] Nenhum `_ = err` sem justificativa
- [ ] Todos os errors propagados com `%w`
- [ ] Contexto adicionado aos errors
- [ ] Sentinel errors usados quando apropriado
- [ ] errors.Is/As usados para comparação

### 2. Concurrency (1.5 pontos)

```go
// ❌ BLOQUEANTE: Race condition
var counter int
go func() { counter++ }()
go func() { counter++ }()

// ❌ BLOQUEANTE: Goroutine leak
go func() {
    for {
        // Loop infinito sem contexto
    }
}()

// ✅ CORRETO: Mutex para shared state
var mu sync.Mutex
mu.Lock()
counter++
mu.Unlock()

// ✅ CORRETO: Context para cancelamento
go func(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // work
        }
    }
}(ctx)
```

**Checklist**:
- [ ] Sem race conditions (-race passa)
- [ ] Goroutines têm mecanismo de parada
- [ ] sync.Pool para objetos frequentes
- [ ] Channels fechados quando apropriado
- [ ] WaitGroup para sincronização

### 3. Testing (1.5 pontos)

```go
// ❌ BLOQUEANTE: Sem teste para função pública
func (s *Service) Search(...) {} // Sem test file

// ❌ PROBLEMA: Teste sem assertions
func TestSearch(t *testing.T) {
    _ = service.Search(...)
    // Não verifica nada!
}

// ✅ CORRETO: Teste table-driven
func TestSearch(t *testing.T) {
    tests := []struct {
        name    string
        input   SearchInput
        want    []Result
        wantErr bool
    }{
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := service.Search(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Search() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Checklist**:
- [ ] Funções públicas têm testes
- [ ] Edge cases cobertos
- [ ] Error paths testados
- [ ] Mocks apropriados (interfaces)
- [ ] Table-driven tests quando aplicável
- [ ] Benchmarks para código crítico

### 4. Security (1.5 pontos)

```go
// ❌ BLOQUEANTE: SQL Injection
query := fmt.Sprintf("SELECT * FROM faces WHERE id = '%s'", userInput)

// ❌ BLOQUEANTE: Secret hardcoded
const apiKey = "sk-1234567890"

// ❌ BLOQUEANTE: Sem validação de input
func (h *Handler) Create(c *fiber.Ctx) error {
    var req CreateRequest
    c.BodyParser(&req) // Sem validação!
    return h.service.Create(req)
}

// ✅ CORRETO: Parameterized query
query := "SELECT * FROM faces WHERE id = $1"
db.Query(query, userInput)

// ✅ CORRETO: Validação de input
func (h *Handler) Create(c *fiber.Ctx) error {
    var req CreateRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.ErrBadRequest
    }
    if err := validate.Struct(req); err != nil {
        return fiber.NewError(400, err.Error())
    }
    return h.service.Create(req)
}
```

**Checklist**:
- [ ] Nenhum secret hardcoded
- [ ] SQL parameterizado
- [ ] Input validado (validator)
- [ ] LGPD: Consentimento verificado
- [ ] Biometric data encrypted
- [ ] Audit logging para operações sensíveis

### 5. Multi-tenancy (1.0 pontos)

```go
// ❌ BLOQUEANTE: Query sem tenant filter
func (r *Repo) FindAll() ([]Face, error) {
    return r.db.Query("SELECT * FROM faces") // TODOS os tenants!
}

// ❌ BLOQUEANTE: Tenant não propagado
func (s *Service) Search(embedding []float64) ([]Result, error) {
    // Onde está o tenantID?
}

// ✅ CORRETO: Tenant do contexto
func (r *Repo) FindAll(ctx context.Context) ([]Face, error) {
    tenantID, err := tenant.FromContext(ctx)
    if err != nil {
        return nil, err
    }
    return r.db.Query("SELECT * FROM faces WHERE tenant_id = $1", tenantID)
}
```

**Checklist**:
- [ ] TODA query inclui tenant_id
- [ ] Tenant propagado via context
- [ ] Nenhum dado cross-tenant exposto
- [ ] Rate limiting per-tenant
- [ ] API key associada a tenant

### 6. Performance (1.0 pontos)

```go
// ❌ PROBLEMA: Allocations desnecessárias
func processItems(items []Item) {
    var results []Result
    for _, item := range items {
        results = append(results, process(item)) // Growing slice
    }
}

// ❌ PROBLEMA: N+1 query
for _, user := range users {
    faces, _ := repo.FindByUser(user.ID) // Query por user!
}

// ✅ CORRETO: Pre-allocate
func processItems(items []Item) {
    results := make([]Result, 0, len(items))
    for _, item := range items {
        results = append(results, process(item))
    }
}

// ✅ CORRETO: Batch query
faces, _ := repo.FindByUsers(userIDs)
```

**Checklist**:
- [ ] Slices pré-alocados quando tamanho conhecido
- [ ] Nenhum N+1 query
- [ ] sync.Pool para objetos frequentes
- [ ] Context timeout em operações externas
- [ ] Benchmark para código hot path

### 7. Code Style (0.5 pontos)

**Checklist**:
- [ ] golangci-lint passa sem erros
- [ ] gofmt aplicado
- [ ] Naming conventions (CamelCase, etc.)
- [ ] Package organization correta
- [ ] Imports organizados

### 8. Documentation (0.5 pontos)

```go
// ❌ PROBLEMA: Função pública sem doc
func Search(ctx context.Context, query string) ([]Result, error) {

// ✅ CORRETO: Godoc completo
// Search performs a similarity search against the face database.
// It returns faces with similarity >= threshold, limited to maxResults.
//
// The search is scoped to the tenant from the context.
// Returns ErrNoTenant if tenant is not set in context.
func Search(ctx context.Context, embedding []float64, threshold float64, maxResults int) ([]Result, error) {
```

**Checklist**:
- [ ] Funções públicas documentadas
- [ ] Packages têm doc.go
- [ ] Exemplos em godoc quando útil
- [ ] README atualizado

### 9. Maintainability (0.5 pontos)

**Checklist**:
- [ ] Funções < 50 linhas
- [ ] Complexidade ciclomática < 10
- [ ] DRY (sem duplicação)
- [ ] SOLID principles aplicados
- [ ] Dependency injection via interfaces

---

## 📊 Formato de Relatório

```markdown
# 📋 Code Review Report

**PR**: #15
**Branch**: feat/face-search
**Author**: @developer
**Reviewer**: code-reviewer agent
**Date**: 2024-01-15

## 🎯 Score: 8.5/10 ✅ APPROVE

| Critério | Score | Max | Notes |
|----------|-------|-----|-------|
| Error Handling | 1.8 | 2.0 | Missing context in 1 error |
| Concurrency | 1.5 | 1.5 | ✅ |
| Testing | 1.2 | 1.5 | Missing edge case test |
| Security | 1.5 | 1.5 | ✅ |
| Multi-tenancy | 1.0 | 1.0 | ✅ |
| Performance | 0.8 | 1.0 | Could use sync.Pool |
| Code Style | 0.5 | 0.5 | ✅ |
| Documentation | 0.5 | 0.5 | ✅ |
| Maintainability | 0.5 | 0.5 | ✅ |

## 🚨 Issues Found

### 🔴 BLOCKING (must fix)
_None_

### 🟡 IMPORTANT (should fix)
1. **Missing error context** (line 45)
   ```go
   // Current
   return nil, err
   // Suggested
   return nil, fmt.Errorf("search faces: %w", err)
   ```

2. **Missing test for empty results** (face_test.go)
   - Add test case for when no faces match threshold

### 🟢 SUGGESTIONS (nice to have)
1. Consider using sync.Pool for embedding slices
2. Add benchmark for Search function

## ✅ Approved Changes
- Clean implementation of face search
- Good use of interfaces for provider abstraction
- Proper tenant isolation

## 📌 Action: APPROVE
```

---

## 🔄 Integração com Pipeline

```yaml
# Chamado automaticamente pelo execution-coordinator
code-reviewer:
  triggers:
    - PR created
    - PR updated
  actions:
    - Analyze changes
    - Generate report
    - Post comment on PR
    - Block merge if score < 8
```

---

## 💡 Exemplo de Uso

```bash
# Modo interativo
/review
# → Claude pergunta: Qual PR/branch/arquivos?
# → Claude analisa e gera relatório

# Modo direto
/review #15
/review feat/face-search
/review internal/service/face.go
```

---

## ✅ Checklist Interno

Antes de finalizar review:

- [ ] Todos os arquivos modificados analisados
- [ ] Checklist de cada critério verificado
- [ ] Score calculado corretamente
- [ ] Issues categorizadas (BLOCKING/IMPORTANT/SUGGESTION)
- [ ] Sugestões de código quando aplicável
- [ ] Relatório formatado e legível
- [ ] Ação correta (APPROVE se ≥8, REQUEST_CHANGES se <8)
