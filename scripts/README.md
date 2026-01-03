# Scripts

Utility scripts para desenvolvimento e manutenção do Rekko.

## seed_test_tenant.sql

Cria um tenant de teste para desenvolvimento local.

### Uso

#### Com Docker Compose (recomendado)

```bash
# Se o banco estiver rodando no Docker Compose
docker compose exec postgres psql -U postgres -d rekko -f /scripts/seed_test_tenant.sql
```

#### Com psql local

```bash
# Usando variável de ambiente
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/rekko?sslmode=disable"
psql $DATABASE_URL -f scripts/seed_test_tenant.sql

# Ou direto
psql "postgresql://postgres:postgres@localhost:5432/rekko?sslmode=disable" -f scripts/seed_test_tenant.sql
```

### Tenant Criado

- **ID**: `00000000-0000-0000-0000-000000000001`
- **Nome**: `Test Tenant`
- **API Key**: `test-api-key-rekko-dev`
- **Hash**: `4a526689e4037a92c11b6228a6aea1a26247875054585e177228365b9720e770`

### Configurações

```json
{
  "verification_threshold": 0.8,
  "max_faces_per_user": 5,
  "liveness_required": false,
  "retention_days": 90
}
```

### Idempotência

O script é idempotente - pode ser executado múltiplas vezes sem efeitos colaterais. Ele:

1. Deleta dados relacionados ao "Test Tenant" (se existir)
2. Deleta o tenant "Test Tenant" (se existir)
3. Recria o tenant com configurações padrão

### Testando Endpoints

Após rodar o seed, você pode testar os endpoints manualmente:

```bash
# Health check
curl http://localhost:8080/health

# Register face
curl -X POST http://localhost:8080/api/v1/faces \
  -H "X-API-Key: test-api-key-rekko-dev" \
  -H "Content-Type: application/json" \
  -d '{
    "external_id": "user-123",
    "image_base64": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
    "metadata": {"name": "John Doe", "event": "Test Event"}
  }'

# Verify face
curl -X POST http://localhost:8080/api/v1/verify \
  -H "X-API-Key: test-api-key-rekko-dev" \
  -H "Content-Type: application/json" \
  -d '{
    "external_id": "user-123",
    "image_base64": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
  }'

# Search similar faces
curl -X POST http://localhost:8080/api/v1/search \
  -H "X-API-Key: test-api-key-rekko-dev" \
  -H "Content-Type: application/json" \
  -d '{
    "image_base64": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
    "threshold": 0.8,
    "limit": 10
  }'

# Get face details
curl http://localhost:8080/api/v1/faces/user-123 \
  -H "X-API-Key: test-api-key-rekko-dev"

# Delete face
curl -X DELETE http://localhost:8080/api/v1/faces/user-123 \
  -H "X-API-Key: test-api-key-rekko-dev"

# List verifications (audit)
curl http://localhost:8080/api/v1/verifications?limit=10 \
  -H "X-API-Key: test-api-key-rekko-dev"
```

### Nota sobre Base64

O exemplo acima usa uma imagem 1x1 pixel em base64. Para testes reais, você precisará:

1. Usar uma imagem real de face
2. Converter para base64:

```bash
# No Linux/macOS
base64 -i face.jpg | tr -d '\n'

# Ou com Python
python3 -c "import base64; print(base64.b64encode(open('face.jpg', 'rb').read()).decode())"
```

## test_endpoints.sh

Script interativo para testar todos os endpoints da API usando cURL.

### Uso

```bash
# Testes individuais
./scripts/test_endpoints.sh health
./scripts/test_endpoints.sh register
./scripts/test_endpoints.sh verify
./scripts/test_endpoints.sh search
./scripts/test_endpoints.sh get
./scripts/test_endpoints.sh delete
./scripts/test_endpoints.sh list

# Rodar todos os testes em sequência
./scripts/test_endpoints.sh all
```

### Variáveis de Ambiente

```bash
# Personalizar configuração
API_BASE_URL=http://localhost:3000 \
API_KEY=your-custom-key \
EXTERNAL_ID=custom-user-id \
./scripts/test_endpoints.sh all
```

### Requisitos

- `curl` instalado
- `jq` instalado (para formatação JSON)
- API rodando

```bash
# Instalar jq no macOS
brew install jq

# Instalar jq no Ubuntu
sudo apt-get install jq
```

---

## api_examples.http

Arquivo de exemplos de requisições HTTP para uso com VS Code REST Client ou similares.

### Uso com VS Code

1. Instalar extensão: [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client)
2. Abrir `scripts/api_examples.http`
3. Clicar em "Send Request" acima de cada requisição

### Uso com IntelliJ IDEA / WebStorm

1. Abrir `scripts/api_examples.http`
2. Clicar no ícone de "play" ao lado de cada requisição

### Categorias de Exemplos

- **Health Check**: Verificar status da API
- **Face Registration**: Registrar faces (1:1 e batch)
- **Face Verification**: Verificar faces (1:1)
- **Face Search**: Buscar faces similares (1:N)
- **Face Retrieval**: Obter detalhes de faces
- **Face Update**: Atualizar metadata e embeddings
- **Face Deletion**: Soft e hard delete (LGPD)
- **Audit & Analytics**: Logs e estatísticas
- **Error Cases**: Testes de validação e tratamento de erros
- **Rate Limiting**: Testes de limite de requisições

---

## run_seed.sh

Script auxiliar para rodar o seed de forma simplificada (wrapper para db.sh seed).

### Uso

```bash
./scripts/run_seed.sh
```

Equivalente a:

```bash
make db-seed
# ou
./scripts/db.sh seed
```

---

## db.sh

Script principal de gerenciamento do banco de dados.

### Comandos Disponíveis

```bash
# Migrations
./scripts/db.sh up           # Rodar migrations
./scripts/db.sh down         # Rollback última migration
./scripts/db.sh reset        # Drop e recriar banco
./scripts/db.sh version      # Versão atual
./scripts/db.sh force <ver>  # Forçar versão

# Operações
./scripts/db.sh status       # Status do banco
./scripts/db.sh psql         # Conectar ao banco
./scripts/db.sh seed         # Seed de teste
./scripts/db.sh dump         # Backup
./scripts/db.sh restore      # Restaurar backup
```

### Variáveis de Ambiente

O script lê do `.env` ou usa valores padrão:

```bash
POSTGRES_HOST=localhost
POSTGRES_PORT=5433
POSTGRES_USER=rekko
POSTGRES_PASSWORD=rekko
POSTGRES_DB=rekko_dev
```

---

## Workflow Completo de Desenvolvimento

```bash
# 1. Iniciar ambiente
docker compose up -d
make db-migrate-up

# 2. Seed do tenant de teste
make db-seed
# ou
./scripts/db.sh seed

# 3. Iniciar API
make dev-server
# ou
go run ./cmd/api

# 4. Testar endpoints
./scripts/test_endpoints.sh all

# 5. Ou testar manualmente
./scripts/test_endpoints.sh register
./scripts/test_endpoints.sh verify

# 6. Conectar ao banco para debug
./scripts/db.sh psql

# 7. Ver logs
docker compose logs -f postgres
```

---

## Troubleshooting

### Erro: "Cannot connect to database"

```bash
# Verificar status
./scripts/db.sh status

# Reiniciar containers
docker compose restart postgres

# Verificar logs
docker compose logs postgres
```

### Erro: "Seed file not found"

```bash
# Verificar se está no diretório raiz do projeto
pwd  # deve mostrar /path/to/rekko

# Rodar do diretório correto
cd /path/to/rekko
./scripts/db.sh seed
```

### Erro: "Migration version mismatch"

```bash
# Ver versão atual
./scripts/db.sh version

# Forçar versão correta
./scripts/db.sh force 1

# Ou resetar completamente
./scripts/db.sh reset
```

### API Key não funciona

```bash
# Verificar se tenant foi criado
./scripts/db.sh psql
SELECT id, name, api_key_hash FROM tenants WHERE name = 'Test Tenant';

# Verificar hash correto
# Hash de "test-api-key-rekko-dev" deve ser:
# 4a526689e4037a92c11b6228a6aea1a26247875054585e177228365b9720e770

# Re-seed se necessário
./scripts/db.sh seed
```
