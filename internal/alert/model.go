package alert

import (
	"time"

	"github.com/google/uuid"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type Alert struct {
	ID              uuid.UUID   `json:"id"`
	TenantID        uuid.UUID   `json:"tenant_id"`
	Name            string      `json:"name"`
	Conditions      []Condition `json:"conditions"`
	ConditionLogic  string      `json:"condition_logic"`
	WindowSeconds   int         `json:"window_seconds"`
	CooldownSeconds int         `json:"cooldown_seconds"`
	Severity        Severity    `json:"severity"`
	Channels        []Channel   `json:"channels"`
	Enabled         bool        `json:"enabled"`
	LastTriggeredAt *time.Time  `json:"last_triggered_at,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type Condition struct {
	MetricName  string  `json:"metric_name"`
	Aggregation string  `json:"aggregation"`
	Operator    string  `json:"operator"`
	Threshold   float64 `json:"threshold"`
}

type Channel struct {
	Type      string    `json:"type"`
	WebhookID uuid.UUID `json:"webhook_id,omitempty"`
}

type AlertHistory struct {
	ID          uuid.UUID              `json:"id"`
	AlertID     uuid.UUID              `json:"alert_id"`
	TenantID    uuid.UUID              `json:"tenant_id"`
	TriggeredAt time.Time              `json:"triggered_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}
