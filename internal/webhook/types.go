package webhook

import (
	"time"

	"github.com/google/uuid"
)

type Webhook struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	Name            string     `json:"name"`
	URL             string     `json:"url"`
	Secret          string     `json:"-"`
	Events          []string   `json:"events"`
	Enabled         bool       `json:"enabled"`
	LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type WebhookJob struct {
	ID          uuid.UUID  `json:"id"`
	WebhookID   uuid.UUID  `json:"webhook_id"`
	EventType   string     `json:"event_type"`
	Payload     []byte     `json:"payload"`
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	NextRetryAt *time.Time `json:"next_retry_at,omitempty"`
	Status      string     `json:"status"`
	LastError   string     `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type EventPayload struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	TenantID  uuid.UUID   `json:"tenant_id"`
	Timestamp time.Time   `json:"timestamp"`
}
