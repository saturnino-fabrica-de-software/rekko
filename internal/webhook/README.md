# Webhook System

Sistema de webhooks com retry exponencial e fila PostgreSQL-native.

## Arquitetura

### Service
Serviço responsável por enviar webhooks:

- **HMAC-SHA256**: Assinatura de payload para validação
- **Headers customizados**: `X-Rekko-Signature`, `X-Rekko-Event`
- **Enqueue automático**: Falhas são enviadas para fila de retry
- **Timeout**: 10s por requisição

### Worker
Worker que processa fila de webhooks com retry:

- **Polling**: A cada 5 segundos
- **FOR UPDATE SKIP LOCKED**: Evita race conditions
- **Exponential Backoff**: 1s, 2s, 4s, 8s, 16s
- **Max Attempts**: 5 tentativas
- **Batch Processing**: 10 jobs por vez

### Retry Strategy

```
Attempt 1: Immediately
Attempt 2: +1s  (2^0 = 1)
Attempt 3: +2s  (2^1 = 2)
Attempt 4: +4s  (2^2 = 4)
Attempt 5: +8s  (2^3 = 8)
Attempt 6: +16s (2^4 = 16) -> FAILED
```

## API Endpoints

### Listar Webhooks

```bash
GET /v1/admin/webhooks
X-API-Key: seu-api-key

Response:
{
  "webhooks": [
    {
      "id": "uuid",
      "name": "Production Alert",
      "url": "https://example.com/webhook",
      "events": ["face.registered", "alert.triggered"],
      "enabled": true,
      "last_triggered_at": "2026-01-04T12:34:56Z",
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

### Criar Webhook

```bash
POST /v1/admin/webhooks
X-API-Key: seu-api-key

{
  "name": "Production Alert",
  "url": "https://example.com/webhook",
  "events": ["face.registered", "alert.triggered"],
  "enabled": true
}

Response:
{
  "webhook": { ... },
  "secret": "generated-secret-key"
}
```

**IMPORTANTE**: O `secret` só é retornado na criação. Guarde-o para validar assinaturas.

### Deletar Webhook

```bash
DELETE /v1/admin/webhooks/:id
X-API-Key: seu-api-key

Response: 204 No Content
```

## Validar Webhook

No endpoint que recebe o webhook:

```go
import "github.com/saturnino-fabrica-de-software/rekko/internal/webhook"

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    signature := r.Header.Get("X-Rekko-Signature")
    
    body, _ := io.ReadAll(r.Body)
    
    if !webhook.Verify("your-secret", body, signature) {
        http.Error(w, "Invalid signature", 401)
        return
    }
    
    // Process webhook...
}
```

## Payload Format

```json
{
  "type": "face.registered",
  "data": {
    "face_id": "uuid",
    "external_id": "user-123"
  },
  "tenant_id": "uuid",
  "timestamp": "2026-01-04T12:34:56Z"
}
```

## Headers Enviados

```
Content-Type: application/json
X-Rekko-Signature: sha256=abc123...
X-Rekko-Event: face.registered
User-Agent: Rekko-Webhook/1.0
```

## Database Schema

```sql
CREATE TABLE webhooks (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(2048) NOT NULL,
    secret VARCHAR(255) NOT NULL,
    events JSONB NOT NULL DEFAULT '[]',
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE webhook_queue (
    id UUID PRIMARY KEY,
    webhook_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    next_retry_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Performance

- **Async Processing**: Não bloqueia requisição principal
- **Batch Processing**: 10 jobs por vez
- **SKIP LOCKED**: Permite múltiplos workers
- **TTL**: Jobs failed são mantidos para auditoria

## Monitoramento

Verificar jobs na fila:

```sql
-- Jobs pendentes
SELECT * FROM webhook_queue WHERE status = 'pending';

-- Jobs falhados
SELECT * FROM webhook_queue WHERE status = 'failed';

-- Taxa de sucesso
SELECT 
    status,
    COUNT(*) as count
FROM webhook_queue
GROUP BY status;
```
