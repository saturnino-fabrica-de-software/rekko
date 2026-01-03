package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// EventType defines the type of auditable event
type EventType string

const (
	EventFaceDetected   EventType = "FACE_DETECTED"
	EventFaceRegistered EventType = "FACE_REGISTERED"
	EventFaceVerified   EventType = "FACE_VERIFIED"
	EventFaceSearched   EventType = "FACE_SEARCHED"
	EventFaceDeleted    EventType = "FACE_DELETED"
	EventFaceCompared   EventType = "FACE_COMPARED"
)

// Event represents an audit event for LGPD compliance
type Event struct {
	ID         uuid.UUID         `json:"id"`
	Timestamp  time.Time         `json:"timestamp"`
	TenantID   uuid.UUID         `json:"tenant_id"`
	EventType  EventType         `json:"event_type"`
	ExternalID string            `json:"external_id,omitempty"`
	FaceID     string            `json:"face_id,omitempty"`
	Provider   string            `json:"provider"`
	Success    bool              `json:"success"`
	Error      string            `json:"error,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	IPAddress  string            `json:"ip_address,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
}

// Logger defines the interface for audit logging
type Logger interface {
	Log(ctx context.Context, event Event) error
}

// SlogLogger implements Logger using slog
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a new audit logger using slog
func NewSlogLogger(logger *slog.Logger) *SlogLogger {
	return &SlogLogger{
		logger: logger.With("component", "audit"),
	}
}

// Log records an audit event
func (l *SlogLogger) Log(ctx context.Context, event Event) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		l.logger.ErrorContext(ctx, "failed to marshal audit event",
			slog.String("error", err.Error()),
			slog.String("event_type", string(event.EventType)),
		)
		return err
	}

	l.logger.InfoContext(ctx, "audit_event",
		slog.String("event_id", event.ID.String()),
		slog.String("event_type", string(event.EventType)),
		slog.String("tenant_id", event.TenantID.String()),
		slog.String("provider", event.Provider),
		slog.Bool("success", event.Success),
		slog.String("event_data", string(eventJSON)),
	)

	return nil
}

// NoOpLogger is a logger that does nothing (for testing or when audit is disabled)
type NoOpLogger struct{}

// Log does nothing and returns nil
func (l *NoOpLogger) Log(_ context.Context, _ Event) error {
	return nil
}
