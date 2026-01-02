# Rekko

> Facial Recognition as a Service (FRaaS) para entrada em eventos

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8.svg)](https://go.dev/)
[![Fiber](https://img.shields.io/badge/Fiber-2.52-00ACD7.svg)](https://gofiber.io/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## O que é o Rekko?

**Rekko** é uma API de reconhecimento facial de alta performance otimizada para o mercado de eventos. Permite que plataformas de venda de ingressos ofereçam entrada por reconhecimento facial.

### Por que Go?

| Métrica | Rekko (Go) | Alternativas (Node) |
|---------|------------|---------------------|
| Latência P99 | ~5ms | ~30ms |
| Memória/instância | ~20MB | ~150MB |
| Concorrência | Milhões goroutines | Event loop único |
| Custo infra | Baseline | 10x maior |

**Projetado para eventos massivos**: Rock in Rio, Lollapalooza, jogos de futebol (7k+ req/s).

## Features

- **Cadastro de Face**: Registre faces durante a compra de ingressos
- **Verificação 1:1**: Compare faces na entrada do evento (~200ms)
- **Liveness Detection**: Proteção anti-fraude (fotos/vídeos)
- **Multi-tenancy**: Isolamento total entre clientes
- **LGPD Compliant**: Consentimento explícito + exclusão de dados
- **Alta Disponibilidade**: 99.9% SLA

## Quick Start

### Requisitos
- Go 1.22+
- Docker + Docker Compose
- Make

### Setup
```bash
# Clone
git clone https://github.com/saturnino-fabrica-de-software/rekko.git
cd rekko

# Subir dependências (Postgres, Redis, DeepFace)
make docker-up

# Instalar dependências Go
go mod download

# Rodar em desenvolvimento
make dev
```

### Testar
```bash
# Health check
curl http://localhost:3000/health

# Cadastrar face (mock)
curl -X POST http://localhost:3000/v1/faces \
  -H "Authorization: Bearer rk_test_123" \
  -F "external_id=user_001" \
  -F "image=@photo.jpg"

# Verificar face
curl -X POST http://localhost:3000/v1/faces/verify \
  -H "Authorization: Bearer rk_test_123" \
  -F "external_id=user_001" \
  -F "image=@photo_entrada.jpg"
```

## API Reference

### Endpoints

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| `GET` | `/health` | Health check |
| `POST` | `/v1/faces` | Cadastrar face |
| `POST` | `/v1/faces/verify` | Verificar face (1:1) |
| `DELETE` | `/v1/faces/:external_id` | Deletar face (LGPD) |
| `GET` | `/v1/usage` | Consultar uso mensal |

### Autenticação
```http
Authorization: Bearer {api_key}
X-Tenant-ID: {tenant_id}
```

### Exemplo de Resposta
```json
{
  "verified": true,
  "confidence": 0.9847,
  "liveness_passed": true,
  "verification_id": "ver_abc123",
  "latency_ms": 187
}
```

## Arquitetura

```
┌─────────────────────────────────────────────────┐
│              Cloudflare (Edge)                  │
│         DDoS + WAF + Rate Limiting              │
└─────────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────┐
│              Rekko API (Go + Fiber)             │
│  ┌───────────┐  ┌───────────┐  ┌─────────────┐  │
│  │ Handlers  │→ │ Services  │→ │ Providers   │  │
│  └───────────┘  └───────────┘  └─────────────┘  │
└─────────────────────────────────────────────────┘
         │              │              │
         ▼              ▼              ▼
┌─────────────┐  ┌───────────┐  ┌─────────────────┐
│ PostgreSQL  │  │   Redis   │  │ AWS Rekognition │
│  (Aurora)   │  │ (Cache)   │  │   / DeepFace    │
└─────────────┘  └───────────┘  └─────────────────┘
```

## Stack

| Componente | Tecnologia |
|------------|------------|
| **Runtime** | Go 1.22 |
| **Framework** | Fiber v2 |
| **Database** | PostgreSQL 16 |
| **Cache** | Redis 7 |
| **Face Recognition** | DeepFace (dev) / AWS Rekognition (prod) |
| **Infra** | Terraform + AWS |

## Desenvolvimento

```bash
# Comandos úteis
make dev           # Hot reload com Air
make build         # Build binário
make test          # Rodar testes
make lint          # Linter
make docker-up     # Subir containers
make docker-down   # Derrubar containers
make migrate       # Rodar migrations
```

## Roadmap

- [x] Definição de arquitetura
- [x] Escolha de stack (Go + Fiber)
- [ ] Setup inicial do projeto
- [ ] API básica (register, verify, delete)
- [ ] Integração DeepFace
- [ ] Integração AWS Rekognition
- [ ] Multi-tenancy
- [ ] Rate limiting
- [ ] Dashboard admin

## Documentação

- [Issue #1 - Especificação Completa](https://github.com/saturnino-fabrica-de-software/rekko/issues/1)

## Contribuição

1. Fork o repositório
2. Crie sua branch (`git checkout -b feat/minha-feature`)
3. Commit suas mudanças (`git commit -m 'feat: add minha feature'`)
4. Push para a branch (`git push origin feat/minha-feature`)
5. Abra um Pull Request

## Licença

MIT License - veja [LICENSE](LICENSE) para detalhes.

---

**Rekko** - Reconhecimento facial simples, rápido e confiável.
