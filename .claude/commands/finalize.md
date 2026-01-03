# /finalize - Finalizar Branch e Mergear com Validação

Encerra a issue atual, garante commits semânticos, abre PR, valida com code-review, acompanha GitHub Actions e mergeia apenas quando tudo passar.

**Filosofia**: "Só merge com Actions passando - zero exceptions!"
**Output**: PR mergeado na main com todas as validações
**UX**: Guiado por etapas - Claude acompanha até o final
**Stack**: Go + Fiber - validações específicas para Go

---

## 🎯 O Que Este Comando Faz

```
/finalize
    ↓
1. Detecta branch atual e issue relacionada
    ↓
2. Verifica git status (pendências?)
    ↓
3. Commits semânticos (se necessário)
    ↓
4. Push para remoto
    ↓
5. Abre PR com descrição detalhada
    ↓
6. code-reviewer valida o PR
    ↓
7. Acompanha GitHub Actions
    ↓
8. Se falhar → Ajusta → Repete 7
    ↓
9. Se passar → Mergeia na main
    ↓
10. Cleanup (delete branch remota)
```

---

## 🚨 REGRAS CRÍTICAS

### Regra 1: NUNCA Mergear com Actions Falhando
```
❌ PROIBIDO: gh pr merge mesmo com CI vermelho
✅ OBRIGATÓRIO: Aguardar TODAS as checks passarem
```

### Regra 2: Commits Semânticos ANTES de PR
```
❌ PROIBIDO: git add . && git commit -m "wip"
✅ OBRIGATÓRIO: Commits atômicos com Conventional Commits
```

### Regra 3: Code Review ANTES de Merge
```
❌ PROIBIDO: Pular code-reviewer
✅ OBRIGATÓRIO: code-reviewer aprovar ou apontar fixes
```

### Regra 4: Validações Go Obrigatórias
```
❌ PROIBIDO: Mergear com golangci-lint warnings
✅ OBRIGATÓRIO: golangci-lint run sem erros
✅ OBRIGATÓRIO: go build ./... sem erros
✅ OBRIGATÓRIO: go test -race ./... passando
```

---

## 📋 Fluxo Detalhado

### ETAPA 1: Detectar Contexto

```bash
# 1.1 Identificar branch atual
BRANCH=$(git branch --show-current)
echo "Branch atual: $BRANCH"

# 1.2 Extrair número da issue do nome da branch
# Padrões suportados: feat/123-descricao, fix/456-bug, feat/issue-789
ISSUE_NUMBER=$(echo "$BRANCH" | grep -oE '[0-9]+' | head -1)

# 1.3 Verificar se issue existe
gh issue view $ISSUE_NUMBER --json title,state
```

**Se branch não seguir padrão ou issue não existir**: Perguntar ao usuário qual issue relacionar.

### ETAPA 2: Verificar Git Status

```bash
# 2.1 Verificar arquivos não commitados
git status --porcelain

# 2.2 Verificar se há commits locais não pushados
git log origin/$BRANCH..$BRANCH --oneline 2>/dev/null || echo "Branch não existe no remoto ainda"
```

**Possíveis estados**:
- **Limpo**: Tudo commitado → Pular para ETAPA 4
- **Staged**: Arquivos em stage → Commitar na ETAPA 3
- **Unstaged**: Arquivos modificados → Adicionar e commitar na ETAPA 3
- **Untracked**: Arquivos novos → Perguntar se incluir

### ETAPA 3: Commits Semânticos (SE NECESSÁRIO)

**CRÍTICO**: Não fazer um commit gigante! Separar por contexto Go.

```bash
# 3.1 Analisar arquivos modificados por package
git diff --name-only
git diff --cached --name-only

# 3.2 Agrupar por package Go
# - internal/service/* → feat/fix no service
# - internal/api/* → feat/fix na API
# - internal/repository/* → feat/fix no repository
# - *_test.go → test: ...
# - cmd/* → chore: ...
# - configs/* → chore: ...
```

**Padrão de Commits**:
```bash
# Exemplo de commits atômicos para Go
git add internal/service/face.go
git commit -m "feat(face): add liveness detection validation"

git add internal/api/handler/face.go
git commit -m "feat(api): expose liveness endpoint"

git add internal/service/face_test.go
git commit -m "test(face): add unit tests for liveness detection"
```

**Regras de Commit**:
- Prefixo obrigatório: `feat|fix|docs|style|refactor|perf|test|chore|ci|build`
- Escopo entre parênteses: `(face)`, `(tenant)`, `(auth)`, `(api)`
- Descrição em inglês, imperativo, lowercase
- Sem co-autor Claude (conforme CLAUDE.md)

### ETAPA 4: Push para Remoto

```bash
# 4.1 Push com upstream se branch nova
git push -u origin $BRANCH

# 4.2 Verificar se push foi bem sucedido
git log origin/$BRANCH --oneline -3
```

### ETAPA 5: Abrir PR

```bash
# 5.1 Verificar se já existe PR aberto
EXISTING_PR=$(gh pr list --head $BRANCH --json number --jq '.[0].number')

if [ -n "$EXISTING_PR" ]; then
  echo "PR #$EXISTING_PR já existe"
  # Atualizar descrição se necessário
else
  # 5.2 Buscar todos os commits da branch
  COMMITS=$(git log main..$BRANCH --pretty=format:"- %s" | head -20)

  # 5.3 Criar PR com descrição detalhada
  gh pr create \
    --title "<Título baseado na issue>" \
    --body "## Summary

Closes #$ISSUE_NUMBER

## Changes

$COMMITS

## Test Plan

- [ ] Unit tests passing (\`go test ./...\`)
- [ ] Race detector passing (\`go test -race ./...\`)
- [ ] Lint passing (\`golangci-lint run\`)
- [ ] Build passing (\`go build ./...\`)
- [ ] Manual testing done

## Performance

- [ ] P99 < 5ms maintained
- [ ] No new allocations in hot path
- [ ] Benchmark compared (if applicable)

## Checklist

- [ ] Code follows Go best practices
- [ ] Error handling complete
- [ ] Context propagation correct
- [ ] Multi-tenancy isolation preserved
- [ ] LGPD compliance maintained
" \
    --base main \
    --head $BRANCH
fi
```

### ETAPA 6: Code Review (Agente)

**Delegar para code-reviewer**:

```
Chamar Task com subagent_type="code-reviewer":

"Revise o PR #<NUMBER> do branch $BRANCH.
Verifique:
1. Código segue padrões Go (effective go, uber style guide)
2. Error handling correto (errors.Is, errors.As, wrapping)
3. Context propagation (timeout, cancellation)
4. Concurrency safety (mutex, atomic, channels)
5. Performance (allocations, P99)
6. Multi-tenancy isolation
7. LGPD compliance

Se encontrar problemas:
- Liste cada problema com arquivo:linha
- Sugira correção específica
- Classifique: BLOCKER / WARNING / INFO

Se aprovado:
- Confirme aprovação
- Prossiga para monitoramento de Actions"
```

### ETAPA 7: Acompanhar GitHub Actions

```bash
# 7.1 Aguardar checks iniciarem
sleep 10

# 7.2 Listar checks do PR
PR_NUMBER=$(gh pr list --head $BRANCH --json number --jq '.[0].number')
gh pr checks $PR_NUMBER --watch

# 7.3 Verificar status final
gh pr checks $PR_NUMBER --json name,state --jq '.[] | "\(.name): \(.state)"'
```

**Estados possíveis**:
- **PENDING**: Aguardar (loop com sleep 30)
- **SUCCESS**: Prosseguir para ETAPA 9
- **FAILURE**: Ir para ETAPA 8

### ETAPA 8: Corrigir Falhas (SE NECESSÁRIO)

```bash
# 8.1 Identificar qual check falhou
gh run list --branch $BRANCH --limit 5
FAILED_RUN=$(gh run list --branch $BRANCH --status failure --json databaseId --jq '.[0].databaseId')

# 8.2 Ver logs do run que falhou
gh run view $FAILED_RUN --log-failed

# 8.3 Identificar erro específico para Go
# - golangci-lint error → corrigir e commitar
# - go build error → corrigir e commitar
# - go test failure → corrigir teste ou código
# - race detector → adicionar mutex/atomic
# - gosec → corrigir vulnerabilidade
```

**Correções comuns em Go**:
```bash
# Lint error
golangci-lint run --fix ./...
git add -A && git commit -m "fix(lint): resolve golangci-lint warnings"

# Test failure
go test -v ./internal/service/... 2>&1 | tee test.log
# Analisar log, corrigir, commitar
git commit -m "fix(test): resolve failing test in face service"

# Race condition
go test -race ./... 2>&1 | grep -A 10 "DATA RACE"
# Adicionar mutex ou usar atomic
git commit -m "fix(race): add mutex to protect shared state"
```

**Após correção**:
```bash
# Commitar fix
git add <arquivos-corrigidos>
git commit -m "fix(ci): resolve <tipo-do-erro>"
git push

# Voltar para ETAPA 7 (acompanhar novamente)
```

**Loop máximo**: 3 tentativas. Se falhar 3x, parar e reportar ao usuário.

### ETAPA 9: Mergear na Main

```bash
# 9.1 Verificar TODAS as checks passaram
CHECKS_PASSED=$(gh pr checks $PR_NUMBER --json state --jq 'all(.state == "SUCCESS" or .state == "SKIPPED")')

if [ "$CHECKS_PASSED" != "true" ]; then
  echo "❌ BLOQUEADO: Ainda há checks não passando"
  exit 1
fi

# 9.2 Mergear com squash (commits limpos na main)
gh pr merge $PR_NUMBER --squash --delete-branch

# 9.3 Confirmar merge
gh pr view $PR_NUMBER --json state --jq '.state'
# Deve retornar: MERGED
```

### ETAPA 10: Cleanup e Sincronização (OBRIGATÓRIO)

**⚠️ CRÍTICO**: SEMPRE voltar para main e sincronizar após merge!

```bash
# 10.1 Voltar para main IMEDIATAMENTE após merge
git checkout main

# 10.2 Pull do que acabou de mergear (OBRIGATÓRIO!)
git pull origin main
# Isso garante que o desenvolvedor está com o código mais recente
# incluindo o squash commit que acabou de ser mergeado

# 10.3 Verificar que está sincronizado
git log --oneline -3
# Deve mostrar o commit do PR que acabou de mergear

# 10.4 Deletar branch local (se ainda existir)
git branch -d $BRANCH 2>/dev/null || true

# 10.5 Fechar issue se não fechou automaticamente
gh issue close $ISSUE_NUMBER --comment "Closed via PR #$PR_NUMBER merge"

# 10.6 Confirmar estado final
echo "✅ Issue #$ISSUE_NUMBER fechada"
echo "✅ PR #$PR_NUMBER mergeado"
echo "✅ Branch $BRANCH deletada"
echo "✅ Você está na main ATUALIZADA com o merge"
echo ""
echo "📍 Último commit:"
git log --oneline -1
```

**Por que isso é obrigatório?**
- Garante que você está com o código mais recente
- Evita conflitos na próxima branch
- Confirma visualmente que o merge foi aplicado

---

## 🔄 Fluxograma de Decisões

```
/finalize
    │
    ▼
┌─────────────────────┐
│ Detectar branch     │
│ e issue relacionada │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐     ┌──────────────┐
│ Há mudanças não     │ SIM │ Fazer commits│
│ commitadas?         ├────►│ semânticos   │
└─────────┬───────────┘     └──────┬───────┘
          │ NÃO                    │
          ▼◄───────────────────────┘
┌─────────────────────┐
│ Push para remoto    │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐     ┌──────────────┐
│ PR já existe?       │ NÃO │ Criar PR     │
└─────────┬───────────┼────►│ detalhado    │
          │ SIM             └──────┬───────┘
          ▼◄───────────────────────┘
┌─────────────────────┐
│ code-reviewer       │
│ valida PR           │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ Aguardar GitHub     │◄─────────────────┐
│ Actions             │                  │
└─────────┬───────────┘                  │
          │                              │
          ▼                              │
┌─────────────────────┐     ┌────────────┴─┐
│ Checks passaram?    │ NÃO │ Corrigir e   │
│                     ├────►│ commitar fix │
└─────────┬───────────┘     └──────────────┘
          │ SIM
          ▼
┌─────────────────────┐
│ Mergear na main     │
│ (squash + delete)   │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ ✅ Concluído!       │
│ Issue fechada       │
│ Branch deletada     │
└─────────────────────┘
```

---

## 🛡️ Validações Go Obrigatórias

### Antes de Mergear, SEMPRE Verificar:

```bash
# Lint completo
golangci-lint run ./...

# Build
go build ./...

# Testes com race detector
go test -race ./...

# Security scan
gosec ./...

# Verificar go.mod está limpo
go mod tidy
go mod verify
```

### Checklist Go:

- [ ] **golangci-lint**: Zero warnings
- [ ] **go build**: Compila sem erros
- [ ] **go test**: Todos passando
- [ ] **go test -race**: Sem race conditions
- [ ] **gosec**: Sem vulnerabilidades
- [ ] **go mod tidy**: go.mod limpo

### Se Qualquer Check Falhar:

1. **NÃO** fazer merge manual
2. Identificar causa raiz no log
3. Corrigir localmente
4. Commitar fix com mensagem descritiva
5. Push e aguardar checks novamente

---

## ⚠️ Tratamento de Erros Go

### Erro: "golangci-lint found issues"
```bash
# Ver detalhes
golangci-lint run ./... --out-format=colored-line-number

# Auto-fix quando possível
golangci-lint run --fix ./...

# Commit
git commit -m "fix(lint): resolve golangci-lint issues"
```

### Erro: "race condition detected"
```bash
# Identificar onde
go test -race ./... 2>&1 | grep -A 20 "DATA RACE"

# Solução típica: adicionar mutex
# Commit
git commit -m "fix(race): add mutex to protect concurrent access"
```

### Erro: "go build failed"
```bash
# Ver erro específico
go build -v ./... 2>&1

# Geralmente: import cycle, undefined symbol, type mismatch
# Commit
git commit -m "fix(build): resolve compilation error in <package>"
```

### Erro: "Conflitos de merge"
```bash
# Solução
git fetch origin main
git rebase origin/main
# Resolver conflitos
git rebase --continue
git push --force-with-lease
```

### Erro: "Check falhou 3+ vezes"
```
Solução:
1. Parar o loop automático
2. Reportar ao usuário os erros recorrentes
3. Aguardar intervenção manual
4. Não tentar merge automático
```

---

## ✅ Checklist Final

Antes de reportar sucesso, confirmar:

- [ ] Issue fechada no GitHub
- [ ] PR mostra status "Merged"
- [ ] Branch remota deletada
- [ ] Branch local deletada
- [ ] **git checkout main** executado
- [ ] **git pull origin main** executado
- [ ] `git log --oneline -1` mostra o commit do merge
- [ ] Nenhuma action ainda executando

---

## 📝 Exemplo de Uso

```bash
# Usuário está em feat/5-face-registration
/finalize

# Claude responde:
# 📍 Branch detectada: feat/5-face-registration
# 📍 Issue relacionada: #5 - Implement face registration API
#
# 🔍 Verificando git status...
# - 4 arquivos .go modificados não commitados
# - 1 arquivo _test.go em stage
#
# 📝 Criando commits semânticos:
# - feat(face): add face registration service ✅
# - feat(api): add POST /faces endpoint ✅
# - test(face): add unit tests for registration ✅
#
# 🚀 Push para origin/feat/5-face-registration... ✅
#
# 📋 Criando PR #12... ✅
#
# 🔍 Executando code-review...
# - Código Go: ✅ Aprovado
# - Error handling: ✅ Correto
# - Concurrency: ✅ Safe
# - Multi-tenancy: ✅ Isolado
#
# ⏳ Aguardando GitHub Actions...
# - golangci-lint: ✅ passed
# - go-build: ✅ passed
# - go-test: ✅ passed
# - go-test-race: ✅ passed
# - gosec: ✅ passed
#
# 🎉 Todas as checks passaram!
#
# 🔀 Mergeando PR #12 na main... ✅
# 🗑️ Deletando branch feat/5-face-registration... ✅
# 🔒 Fechando issue #5... ✅
#
# 🔄 Voltando para main...
# git checkout main ✅
# git pull origin main ✅
#
# 📍 Último commit na main:
# def5678 feat(face): add face registration service (#12)
#
# ✅ FINALIZADO COM SUCESSO!
# - Issue #5 fechada
# - PR #12 mergeado
# - Branch deletada (local + remota)
# - Você está na main SINCRONIZADA com o merge
```
