package ws

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventFaceRegistered EventType = "face.registered"
	EventFaceDeleted    EventType = "face.deleted"
	EventVerification   EventType = "verification.completed"
	EventAlert          EventType = "alert.triggered"
	EventMetricUpdate   EventType = "metric.updated"
)

type Event struct {
	TenantID  uuid.UUID   `json:"-"`
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}
