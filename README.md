# Rekko

> Facial Recognition as a Service (FRaaS) para entrada em eventos

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.0-blue.svg)](https://www.typescriptlang.org/)
[![NestJS](https://img.shields.io/badge/NestJS-10.0-red.svg)](https://nestjs.com/)

## O que é o Rekko?

**Rekko** é uma API SaaS B2B de reconhecimento facial otimizada para o mercado de eventos. Permite que plataformas de venda de ingressos (como Ingresse, Sympla, Eventbrite) ofereçam entrada por reconhecimento facial em seus eventos.

## Principais Features

- **Cadastro de Face**: Registre faces de usuários durante o processo de compra
- **Verificação 1:1**: Compare a face na entrada com a face cadastrada
- **Liveness Detection**: Proteção contra fraudes com fotos/vídeos
- **Multi-tenancy**: Isolamento total de dados entre clientes
- **LGPD Compliant**: Termos de consentimento e exclusão de dados
- **Alta Performance**: Latência P99 < 500ms

## Quick Start

```bash
# Clone o repositório
git clone https://github.com/saturnino-fabrica-de-software/rekko.git
cd rekko

# Instale as dependências
npm install

# Configure o ambiente
cp .env.example .env

# Inicie em desenvolvimento
npm run dev
```

## API Overview

```http
# Cadastrar face
POST /v1/faces
Content-Type: multipart/form-data
Authorization: Bearer {api_key}

# Verificar face
POST /v1/faces/verify
Content-Type: multipart/form-data
Authorization: Bearer {api_key}

# Deletar face (LGPD)
DELETE /v1/faces/{external_id}
Authorization: Bearer {api_key}
```

## Stack

| Tecnologia | Uso |
|------------|-----|
| NestJS | Backend API |
| PostgreSQL | Database |
| Redis | Cache & Rate Limiting |
| DeepFace | Face Recognition (dev) |
| AWS Rekognition | Face Recognition (prod) |
| Terraform | Infrastructure as Code |

## Roadmap

- [x] Definição de arquitetura
- [ ] Setup do projeto
- [ ] API básica (register, verify, delete)
- [ ] Liveness detection
- [ ] Dashboard admin
- [ ] SDK JavaScript

## Documentação

- [Issue #1 - Especificação Completa](https://github.com/saturnino-fabrica-de-software/rekko/issues/1)

## Licença

MIT License - veja [LICENSE](LICENSE) para detalhes.

---

**Rekko** - Reconhecimento facial simples, rápido e confiável para eventos.
