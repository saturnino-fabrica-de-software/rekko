# WebSocket Real-time Events

Sistema de eventos em tempo real usando WebSocket com isolamento por tenant.

## Arquitetura

### Hub Pattern
O `Hub` gerencia todas as conexões WebSocket ativas e broadcasts de eventos:

- **Registro de clientes**: Clientes se registram ao conectar
- **Isolamento por tenant**: Eventos são enviados apenas para clientes do mesmo tenant
- **Broadcast assíncrono**: Canal com buffer de 256 mensagens
- **Thread-safe**: Usa `sync.RWMutex` para acesso concorrente

### Client
Cada conexão WebSocket é representada por um `Client`:

- **ReadPump**: Goroutine que lê mensagens do cliente (mantém conexão viva)
- **WritePump**: Goroutine que envia mensagens para o cliente
- **Buffer**: Canal com 256 slots para mensagens pendentes

## Tipos de Eventos

```go
const (
    EventFaceRegistered  EventType = "face.registered"
    EventFaceDeleted     EventType = "face.deleted"
    EventVerification    EventType = "verification.completed"
    EventAlert           EventType = "alert.triggered"
    EventMetricUpdate    EventType = "metric.updated"
)
```

## Uso

### Conectar ao WebSocket

```bash
# Conectar com autenticação
wscat -c "ws://localhost:3000/v1/ws" \
  -H "X-API-Key: seu-api-key"
```

### Enviar Evento de Código

```go
// No handler ou service
hub.BroadcastToTenant(
    tenantID,
    ws.EventFaceRegistered,
    map[string]interface{}{
        "face_id": faceID,
        "external_id": externalID,
    },
)
```

### Payload Recebido

```json
{
  "type": "face.registered",
  "data": {
    "face_id": "123e4567-e89b-12d3-a456-426614174000",
    "external_id": "user-123"
  },
  "timestamp": "2026-01-04T12:34:56Z"
}
```

## Segurança

- **Autenticação**: Requer API Key válida (middleware Auth)
- **Isolamento**: Clientes só recebem eventos do seu tenant
- **Rate Limiting**: Aplica-se também ao WebSocket upgrade

## Performance

- **Buffer de mensagens**: 256 por cliente
- **Broadcast assíncrono**: Não bloqueia o sender
- **Cleanup automático**: Clientes desconectados são removidos
