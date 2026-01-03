# Quickstart - Seed de Desenvolvimento

Guia rápido para configurar o ambiente de desenvolvimento do Rekko.

## Setup Rápido (3 comandos)

```bash
# 1. Iniciar banco de dados
docker compose up -d postgres

# 2. Rodar migrations
./scripts/db.sh up

# 3. Criar tenant de desenvolvimento
./scripts/db.sh seed
```

## API Key de Desenvolvimento

Após rodar o seed, use esta API Key fixa para testes:

```
rekko_test_devdevdevdevdevdevdevdevdevdev00
```

## Teste Rápido

```bash
# Iniciar API
go run ./cmd/api

# Em outro terminal, testar endpoint
curl http://localhost:3000/health
```

## Próximos Passos

1. Ver exemplos de API em `scripts/api_examples.http`
2. Rodar testes: `./scripts/test_endpoints.sh all`
3. Explorar mais comandos: `./scripts/db.sh --help`

## Resumo de Comandos

| Comando | Descrição |
|---------|-----------|
| `./scripts/db.sh seed` | Criar tenant de dev |
| `./scripts/db.sh psql` | Conectar ao banco |
| `./scripts/db.sh status` | Ver status |
| `./scripts/db.sh reset` | Resetar tudo |

## Detalhes do Tenant Criado

- **Tenant ID**: `00000000-0000-0000-0000-000000000001`
- **Nome**: `Rekko Development`
- **Slug**: `rekko-dev`
- **Plan**: `enterprise`
- **Rate Limit**: 1000 req/s
- **Features**: Todas habilitadas

## API Key de Desenvolvimento

- **Key ID**: `00000000-0000-0000-0000-000000000002`
- **Prefix**: `rekko_test_devd`
- **Environment**: `test`
- **Full Key**: `rekko_test_devdevdevdevdevdevdevdevdevdev00`

## Troubleshooting

### Banco não conecta

```bash
# Verificar status
docker ps | grep postgres

# Reiniciar
docker compose restart postgres

# Ver logs
docker compose logs -f postgres
```

### Seed falhou

```bash
# Resetar e tentar novamente
./scripts/db.sh reset
./scripts/db.sh seed
```

### API Key não funciona

```bash
# Verificar se tenant existe
./scripts/db.sh psql
SELECT id, name, api_key_hash FROM tenants WHERE name = 'Rekko Development';

# Hash esperado:
# adf716ab3ebb2a1138973de4a44fe454c05c0d070e897fc55220af74807b25ae
```
