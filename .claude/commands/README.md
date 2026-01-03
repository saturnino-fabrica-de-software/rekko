# Slash Commands - Quick Reference

**Purpose**: Comandos padronizados para workflow de desenvolvimento, qualidade e performance do Rekko.

**Filosofia**:
- **Conversational UX**: Zero flags para decorar - Claude pergunta o que você quer fazer! 💬
- **Go-First**: Todos os comandos otimizados para Go e FRaaS
- **Performance**: Target P99 < 5ms sempre em mente

---

## 🎯 Comandos Disponíveis

### 1. `/debate` - Parceiro Técnico para Design de Features

**Purpose**: Debate colaborativo técnico onde Claude questiona decisões e gera issue BDD detalhada.

**Quick Start**:
```bash
# Modo Interativo (Recomendado) 💬
/debate
# → Claude pergunta: Qual tipo de debate? (design/refactor/performance/security/integration)
# → Claude pergunta: Descrição da feature
# → Claude cria issue e inicia debate bloqueante

# Modo Direto
/debate criar sistema de liveness detection
/debate otimizar embedding search para 1M faces
```

**Características**:
- 🤝 Parceiro sênior (questiona, não aceita passivamente)
- 💭 Método socrático (perguntas que fazem pensar)
- 🎯 Foco em P99, multi-tenancy, LGPD
- 📋 Gera issue BDD completa com código Go
- 🔍 **context7**: Busca docs oficiais Go/Fiber automaticamente

**📚 Full Documentation**: `.claude/commands/debate.md`

---

### 2. `/implement` - Orquestrador Inteligente de Issues

**Purpose**: Implementa issues do GitHub de forma semi-autônoma com qualidade garantida.

**Quick Start**:
```bash
# Modo Interativo (Recomendado) 💬
/implement
# → Claude pergunta: Qual issue implementar? (aceita #10, 10, ou URL)
# → GATE 1: Estratégia de testes?
# → GATE 2: Aprovar plano?
# → Execução semi-autônoma...
# → GATE 3: Aprovar merge?

# Modo Direto
/implement #10
/implement 15
```

**Características**:
- 🤖 Semi-autônomo (3 gates: Testes, Plano, Merge)
- 🎯 Claude delega, especialistas implementam
- ✅ Validações automáticas (golangci-lint, go test, gosec)
- 📊 Quality gates obrigatórios (score ≥ 8/10)

**📚 Full Documentation**: `.claude/commands/implement.md`

---

### 3. `/benchmark` - Performance Profiling & Benchmarking

**Purpose**: Profiling e benchmarking para garantir P99 < 5ms.

**Quick Start**:
```bash
# Modo Interativo (Recomendado) 💬
/benchmark
# → Claude pergunta: Qual operação? (scan/profile/bench/load/compare)
# → Claude pergunta: Qual escopo? (endpoint/função/package)

# Modo Direto
/benchmark scan
/benchmark profile internal/service/face.go
/benchmark load /api/v1/faces/search --vus 100
```

**Características**:
- 📊 CPU/Memory profiling com pprof
- 🔥 Hot path identification
- 📈 Load testing com k6/vegeta
- 🔄 Comparison A/B entre commits

**📚 Full Documentation**: `.claude/commands/benchmark.md`

---

### 4. `/finalize` - Finalizar Branch e Mergear ⭐ **NOVO**

**Purpose**: Encerra issue, commits semânticos, PR, code-review, acompanha Actions, mergeia apenas quando passar.

**Quick Start**:
```bash
# Modo Interativo (Recomendado) 💬
/finalize
# → Detecta branch atual e issue relacionada
# → Verifica mudanças não commitadas
# → Faz commits semânticos Go (por package)
# → Abre PR com descrição detalhada
# → code-reviewer valida automaticamente
# → Acompanha GitHub Actions (golangci-lint, go test, etc.)
# → Ajusta se falhar (até 3x)
# → Mergeia na main + cleanup

# Após trabalho em feat/5-face-registration
/finalize
# → Completa todo o fluxo até o merge
```

**Garantias**:
- 🛡️ NUNCA mergeia com Actions falhando
- 📝 Commits sempre semânticos (feat/fix/test por package)
- 🔍 Code review automático (Go patterns, concurrency, P99)
- 🔄 Loop de correção (golangci-lint --fix, race detector)
- 🧹 Cleanup automático (branch + issue fechadas)

**📚 Full Documentation**: `.claude/commands/finalize.md`

---

### 5. `/partner` - Parceiro Estratégico de Arquitetura e Design ⭐ **NOVO**

**Purpose**: Ativa modo de parceiro participativo com papel dual de Arquiteto Go + Product Designer DX

**Quick Start**:
```bash
# Modo Interativo (Recomendado) 💬
/partner
# → Claude assume papel dual de:
#   - Arquiteto Go Sênior (P99 < 5ms, goroutines, multi-tenancy, LGPD)
#   - Product Designer DX (Developer Experience, API design, SDK)
# → Questiona, propõe melhorias, identifica gaps
# → UMA pergunta por vez (método socrático)

# Modo Direto (Avançado) 🔧
/partner criar sistema de liveness detection para prevenir spoofing
/partner a busca de similaridade está lenta com muitas faces
/partner devo usar worker pool ou goroutine por request para processamento de faces?
```

**Características**:
- 🤝 Parceiro participativo (NUNCA passivo)
- 🏗️ Arquiteto Go: P99 < 5ms, concorrência, mutex vs channels, LGPD
- 🎨 Designer DX: API design, SDK experience, error messages, time-to-first-call
- 💭 Método socrático (uma pergunta por vez, aguarda resposta)
- 🚀 NUNCA MVP mindset (performance desde v1)
- 📈 Identifica gaps proativamente (multi-tenancy, compliance)
- 🎯 SEMPRE dá opinião fundamentada

**Perguntas que SEMPRE faz**:
- "Como isso se comporta com 100 tenants simultâneos?"
- "Qual o impacto na latência P99?"
- "Isso mantém isolamento de tenant?"
- "Precisamos de mutex ou atomic aqui?"
- "Isso está compliant com LGPD?"

**Anti-Patterns** (O que Claude NUNCA faz):
- ❌ "Ok, vou fazer como você pediu" (passividade)
- ❌ "Para o MVP podemos simplificar..." (MVP mindset)
- ❌ "Tenho 5 perguntas: 1) 2) 3)..." (bombardeio)
- ❌ "Você pode escolher entre mutex ou channel" (falta de opinião)

**📚 Full Documentation**: `.claude/commands/partner.md`

---

### 6. `/review` - Code Review Automatizado

**Purpose**: Code review automatizado com score 0-10 e ação APPROVE/REQUEST_CHANGES.

**Quick Start**:
```bash
# Modo Interativo (Recomendado) 💬
/review
# → Claude pergunta: Qual PR/branch/arquivos?
# → Claude analisa e gera relatório

# Modo Direto
/review #15
/review feat/face-search
```

**Características**:
- 📋 9 critérios de avaliação (error handling, concurrency, security, etc.)
- 🎯 Score mínimo 8/10 para APPROVE
- 🚨 Issues categorizadas (BLOCKING/IMPORTANT/SUGGESTION)
- 🔐 Foco em multi-tenancy e LGPD

**📚 Full Documentation**: `.claude/commands/review.md`

---

## 🔄 Workflow Recomendado

### Feature Development (Completo) ⭐ **RECOMENDADO**

```bash
# 1. Debater feature (design colaborativo)
/debate criar sistema de liveness detection
# → Gera issue #10 BDD detalhada

# 2. Implementar issue gerada
/implement #10
# → Executa semi-autônomo com validações

# 3. Finalizar e mergear
/finalize
# → Commits semânticos
# → PR com code-review
# → Acompanha Actions até passar
# → Merge automático + cleanup
```

**Fluxo completo**: `/debate` → `/implement` → `/finalize` → Merged! 🎉

### Performance Optimization

```bash
# 1. Identificar gargalos
/benchmark scan

# 2. Profile detalhado
/benchmark profile internal/service/face.go

# 3. Debate otimização
/debate otimizar embedding search

# 4. Implementar
/implement #12

# 5. Validar melhoria
/benchmark compare main..feat/optimization
```

### Pre-Commit Workflow

```bash
# 1. Review das mudanças
/review internal/service/

# 2. Benchmark (se código crítico)
/benchmark bench Search

# Se tudo ok → safe to commit
```

---

## 📊 Targets de Performance (Rekko)

| Métrica | Target | Critical |
|---------|--------|----------|
| P50 Latency | < 2ms | < 5ms |
| P99 Latency | < 5ms | < 10ms |
| Throughput | > 10k req/s | > 5k req/s |
| Allocs/request | < 10 | < 50 |
| Code Review Score | ≥ 8/10 | ≥ 7/10 |

---

## 🔒 Checklist de Segurança (Todos os Comandos)

- [ ] LGPD: Consentimento verificado para biometria
- [ ] Multi-tenancy: Isolamento de tenant em TODA query
- [ ] Secrets: Nenhum hardcoded, via environment
- [ ] Input: Validação em todas as entradas
- [ ] Encryption: Embeddings criptografados at rest

---

**Created**: 2024-01-15
**Status**: Active
**Stack**: Go 1.22 + Fiber + PostgreSQL + pgvector
