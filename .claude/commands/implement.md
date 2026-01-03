# /implement - Orquestrador Inteligente de Issues

Implementa issues do GitHub de forma semi-autônoma, seguindo o pipeline de agentes com qualidade garantida.

**Filosofia**: "Claude delega, especialistas implementam, quality gates validam"
**Input**: Número da issue (ex: #10, 10, ou URL)
**Output**: PR pronto para merge com testes passando
**UX**: Semi-autônomo com 3 gates humanos (Testes, Plano, Merge)

---

## ⛔ RESTRIÇÃO FUNDAMENTAL: DELEGAR, NÃO IMPLEMENTAR

**CRÍTICO**: O `/implement` é um ORQUESTRADOR. Claude NÃO escreve código diretamente.

### 🎯 Hierarquia de Delegação:
```
Claude (Tech Lead)
  → demand-classifier (valida e planeja)
    → execution-coordinator (orquestra fases)
      → Especialistas (implementam código)
        - go-fiber-specialist
        - go-testing-specialist
        - database-specialist
        - etc.
```

### 🚫 Claude NUNCA faz:
- Escrever código Go diretamente
- Usar Edit/Write para código
- Pular validações de qualidade
- Marcar checkboxes manualmente

### ✅ Claude SEMPRE faz:
- Delegar para `demand-classifier` via `Task()`
- Monitorar progresso
- Aprovar/rejeitar nos gates humanos

---

## 📋 Pipeline Completo

### GATE 1: Estratégia de Testes
```
Claude pergunta:
"Qual estratégia de testes para esta issue?"

Opções:
1. Unit tests (go test ./...)
2. Integration tests (testcontainers)
3. Ambos (unit + integration)
4. Nenhum (apenas lint/build)
```

### Fase 1: Validação da Issue
```
demand-classifier:
1. Ler issue via gh issue view
2. Validar se issue tem critérios de aceite
3. Verificar se há conflitos com código existente
4. Identificar agentes necessários

Output: Lista de tarefas + agentes mapeados
```

### Fase 2: Planejamento
```
demand-classifier:
1. Gerar lista TODO completa
2. Identificar dependências entre tarefas
3. Estimar complexidade

Output: Plano de execução
```

### GATE 2: Aprovar Plano
```
Claude apresenta:
"Plano de implementação:
1. [X] Criar interface FaceProvider
2. [X] Implementar AWSRekognitionProvider
3. [X] Adicionar testes unitários
4. [X] Atualizar migrations
5. [X] Adicionar endpoint em /api/v1/faces

Aprovar plano? (sim/cancelar)"
```

### Fase 3: Implementação
```
execution-coordinator:
Para cada tarefa do plano:
1. Identificar especialista correto
2. Delegar via Task(subagent_type="<especialista>")
3. Validar output do especialista
4. Rodar lint + build + tests
5. Marcar checkbox na issue
6. Fazer commit atômico
```

### Fase 4: Validação de Qualidade
```
Validações automáticas (em sequência):
1. golangci-lint run ./...
2. go build ./...
3. go test -race ./...
4. gosec ./...

Se QUALQUER falhar → corrigir ANTES de continuar
```

### Fase 5: Pull Request
```
github-workflow-specialist:
1. Criar PR com template
2. Adicionar reviewers
3. Vincular issue

Output: URL do PR
```

### Fase 6: Code Review Automático
```
code-reviewer:
1. Validar cobertura de testes
2. Verificar error handling
3. Checar concurrency patterns
4. Validar multi-tenancy isolation
5. Score final (0-10)

Se score < 8 → BLOQUEAR merge
```

### GATE 3: Aprovar Merge
```
Claude apresenta:
"PR #15 pronto:
- ✅ Testes passando (45/45)
- ✅ Code review: 9.2/10
- ✅ Lint: 0 errors
- ✅ Build: success

Aprovar merge? (sim/cancelar)"
```

---

## 🔧 Mapeamento de Agentes por Tarefa

| Tipo de Tarefa | Agente Especialista |
|----------------|---------------------|
| Handlers/Controllers | `go-fiber-specialist` |
| Business Logic | `go-fiber-specialist` |
| Database/Repository | `database-specialist` |
| Face Recognition | `face-recognition-architect` |
| Provider Integration | `provider-abstraction-specialist` |
| Unit Tests | `go-testing-specialist` |
| Integration Tests | `go-testing-specialist` |
| Performance | `go-pprof-specialist` |
| Concurrency | `go-concurrency-specialist` |
| Docker | `docker-specialist` |
| Migrations | `database-specialist` |
| CI/CD | `deploy-specialist` |
| Security/LGPD | `biometric-security-specialist` |
| Multi-tenancy | `multi-tenancy-architect` |
| Cache/Queue | `queue-cache-specialist` |

---

## 📝 Template de Commit

```
<type>(<scope>): <description>

- <bullet point 1>
- <bullet point 2>

Refs: #<issue_number>
```

Tipos válidos:
- `feat`: Nova funcionalidade
- `fix`: Correção de bug
- `refactor`: Refatoração sem mudança de comportamento
- `test`: Adição/modificação de testes
- `docs`: Documentação
- `perf`: Melhoria de performance
- `chore`: Manutenção

---

## 📝 Template de PR

```markdown
## 🎯 Summary

<Breve descrição do que foi implementado>

## 🔗 Related Issue

Closes #<issue_number>

## 📋 Changes

- [ ] <Mudança 1>
- [ ] <Mudança 2>
- [ ] <Mudança 3>

## 🧪 Testing

- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing completed

## 📊 Performance Impact

- P99 latency: <before> → <after>
- Memory: <before> → <after>

## 🔒 Security Checklist

- [ ] No secrets in code
- [ ] Input validation added
- [ ] LGPD compliance verified
- [ ] Multi-tenant isolation verified

## 📸 Screenshots (if applicable)

<screenshots>

---

**Code Review Score**: X/10
**Reviewed by**: code-reviewer agent
```

---

## 🚨 Fail-Fast Rules

### Bloqueio Automático:
| Condição | Ação |
|----------|------|
| `golangci-lint` falha | BLOQUEIA até corrigir |
| `go build` falha | BLOQUEIA até corrigir |
| `go test` falha | BLOQUEIA até corrigir |
| Code review < 8/10 | BLOQUEIA até corrigir |
| Race condition detectada | BLOQUEIA até corrigir |
| gosec HIGH/CRITICAL | BLOQUEIA até corrigir |

### Nunca Ignorar:
- ❌ `// nolint` sem justificativa
- ❌ `_ = err` (error ignorado)
- ❌ `panic()` em production code
- ❌ Hardcoded credentials
- ❌ Missing tenant isolation

---

## 🔄 Rollback

Se implementação falhar após merge:

```bash
# 1. Reverter commit
git revert <commit_sha>

# 2. Atualizar issue
gh issue comment <NUMBER> --body "⚠️ Implementação revertida: <motivo>"

# 3. Reabrir issue
gh issue reopen <NUMBER>
```

---

## 📊 Métricas de Sucesso

Uma implementação é considerada **completa** quando:

- ✅ Todos os critérios de aceite da issue atendidos
- ✅ Testes passando (100% dos novos)
- ✅ Lint sem erros
- ✅ Build sem erros
- ✅ Code review ≥ 8/10
- ✅ PR aprovado e mergeado
- ✅ Issue fechada automaticamente

---

## 💡 Exemplo de Uso

```bash
# Modo interativo (recomendado)
/implement
# → Claude pergunta: Qual issue implementar?
# → Usuário: #10
# → GATE 1: Estratégia de testes?
# → GATE 2: Aprovar plano?
# → Execução semi-autônoma...
# → GATE 3: Aprovar merge?

# Modo direto
/implement #10
/implement 10
/implement https://github.com/owner/repo/issues/10
```

---

## ✅ Checklist Interno

Antes de finalizar `/implement`:

- [ ] Issue validada e critérios de aceite claros
- [ ] Plano aprovado pelo usuário (GATE 2)
- [ ] Todos os especialistas delegados corretamente
- [ ] Cada tarefa validada (lint + build + test)
- [ ] Commits atômicos e bem descritos
- [ ] PR criado com template completo
- [ ] Code review ≥ 8/10
- [ ] Merge aprovado pelo usuário (GATE 3)
- [ ] Issue fechada via PR
