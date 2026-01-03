package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlogLogger_Log(t *testing.T) {
	tests := []struct {
		name          string
		event         Event
		wantEventType string
		wantProvider  string
		wantSuccess   bool
		wantHasError  bool
		wantHasFaceID bool
	}{
		{
			name: "face detected event",
			event: Event{
				TenantID:  uuid.New(),
				EventType: EventFaceDetected,
				Provider:  "rekognition",
				Success:   true,
				Metadata: map[string]string{
					"faces_count": "1",
				},
			},
			wantEventType: string(EventFaceDetected),
			wantProvider:  "rekognition",
			wantSuccess:   true,
			wantHasError:  false,
			wantHasFaceID: false,
		},
		{
			name: "face registered event with face ID",
			event: Event{
				TenantID:  uuid.New(),
				EventType: EventFaceRegistered,
				FaceID:    "face-123",
				Provider:  "rekognition",
				Success:   true,
			},
			wantEventType: string(EventFaceRegistered),
			wantProvider:  "rekognition",
			wantSuccess:   true,
			wantHasError:  false,
			wantHasFaceID: true,
		},
		{
			name: "failed face search event",
			event: Event{
				TenantID:  uuid.New(),
				EventType: EventFaceSearched,
				Provider:  "rekognition",
				Success:   false,
				Error:     "collection not found",
			},
			wantEventType: string(EventFaceSearched),
			wantProvider:  "rekognition",
			wantSuccess:   false,
			wantHasError:  true,
			wantHasFaceID: false,
		},
		{
			name: "face deleted event",
			event: Event{
				TenantID:  uuid.New(),
				EventType: EventFaceDeleted,
				FaceID:    "face-456",
				Provider:  "rekognition",
				Success:   true,
			},
			wantEventType: string(EventFaceDeleted),
			wantProvider:  "rekognition",
			wantSuccess:   true,
			wantHasError:  false,
			wantHasFaceID: true,
		},
		{
			name: "face compared event",
			event: Event{
				TenantID:  uuid.New(),
				EventType: EventFaceCompared,
				Provider:  "rekognition",
				Success:   true,
				Metadata: map[string]string{
					"similarity": "0.95",
				},
			},
			wantEventType: string(EventFaceCompared),
			wantProvider:  "rekognition",
			wantSuccess:   true,
			wantHasError:  false,
			wantHasFaceID: false,
		},
		{
			name: "event with IP and user agent",
			event: Event{
				TenantID:  uuid.New(),
				EventType: EventFaceVerified,
				Provider:  "rekognition",
				Success:   true,
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			wantEventType: string(EventFaceVerified),
			wantProvider:  "rekognition",
			wantSuccess:   true,
			wantHasError:  false,
			wantHasFaceID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewJSONHandler(&buf, nil)
			logger := slog.New(handler)

			auditLogger := NewSlogLogger(logger)
			err := auditLogger.Log(context.Background(), tt.event)

			require.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, tt.wantEventType)
			assert.Contains(t, output, tt.wantProvider)
			assert.Contains(t, output, "audit_event")
			assert.Contains(t, output, "audit")

			if tt.wantHasError {
				assert.Contains(t, output, tt.event.Error)
			}

			if tt.wantHasFaceID {
				assert.Contains(t, output, tt.event.FaceID)
			}
		})
	}
}

func TestSlogLogger_Log_GeneratesIDAndTimestamp(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	auditLogger := NewSlogLogger(logger)
	event := Event{
		TenantID:  uuid.New(),
		EventType: EventFaceDetected,
		Provider:  "rekognition",
		Success:   true,
	}

	err := auditLogger.Log(context.Background(), event)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "event_id")

	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.NotEmpty(t, lines)

	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)

	eventID, ok := logEntry["event_id"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, eventID)

	_, err = uuid.Parse(eventID)
	assert.NoError(t, err)
}

func TestSlogLogger_Log_UsesProvidedIDAndTimestamp(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	auditLogger := NewSlogLogger(logger)
	expectedID := uuid.New()
	expectedTimestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	event := Event{
		ID:        expectedID,
		Timestamp: expectedTimestamp,
		TenantID:  uuid.New(),
		EventType: EventFaceRegistered,
		Provider:  "rekognition",
		Success:   true,
	}

	err := auditLogger.Log(context.Background(), event)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, expectedID.String())
}

func TestSlogLogger_Log_IncludesAllEventTypes(t *testing.T) {
	eventTypes := []EventType{
		EventFaceDetected,
		EventFaceRegistered,
		EventFaceVerified,
		EventFaceSearched,
		EventFaceDeleted,
		EventFaceCompared,
	}

	for _, eventType := range eventTypes {
		t.Run(string(eventType), func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewJSONHandler(&buf, nil)
			logger := slog.New(handler)

			auditLogger := NewSlogLogger(logger)
			event := Event{
				TenantID:  uuid.New(),
				EventType: eventType,
				Provider:  "rekognition",
				Success:   true,
			}

			err := auditLogger.Log(context.Background(), event)
			require.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, string(eventType))
		})
	}
}

func TestNoOpLogger_Log(t *testing.T) {
	logger := &NoOpLogger{}

	event := Event{
		ID:        uuid.New(),
		Timestamp: time.Now(),
		TenantID:  uuid.New(),
		EventType: EventFaceDetected,
		Provider:  "rekognition",
		Success:   true,
		Metadata: map[string]string{
			"test": "value",
		},
	}

	err := logger.Log(context.Background(), event)

	assert.NoError(t, err)
}

func TestNoOpLogger_Log_MultipleEvents(t *testing.T) {
	logger := &NoOpLogger{}

	for i := 0; i < 100; i++ {
		event := Event{
			TenantID:  uuid.New(),
			EventType: EventFaceSearched,
			Provider:  "rekognition",
			Success:   true,
		}

		err := logger.Log(context.Background(), event)
		assert.NoError(t, err)
	}
}

func TestSlogLogger_Log_WithMetadata(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	auditLogger := NewSlogLogger(logger)
	event := Event{
		TenantID:  uuid.New(),
		EventType: EventFaceSearched,
		Provider:  "rekognition",
		Success:   true,
		Metadata: map[string]string{
			"faces_count":    "5",
			"threshold":      "0.8",
			"execution_time": "150ms",
		},
	}

	err := auditLogger.Log(context.Background(), event)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "faces_count")
	assert.Contains(t, output, "threshold")
	assert.Contains(t, output, "execution_time")
}

func TestLoggerInterface_Compliance(t *testing.T) {
	var _ Logger = (*SlogLogger)(nil)
	var _ Logger = (*NoOpLogger)(nil)
}

func TestEventType_Constants(t *testing.T) {
	assert.Equal(t, EventType("FACE_DETECTED"), EventFaceDetected)
	assert.Equal(t, EventType("FACE_REGISTERED"), EventFaceRegistered)
	assert.Equal(t, EventType("FACE_VERIFIED"), EventFaceVerified)
	assert.Equal(t, EventType("FACE_SEARCHED"), EventFaceSearched)
	assert.Equal(t, EventType("FACE_DELETED"), EventFaceDeleted)
	assert.Equal(t, EventType("FACE_COMPARED"), EventFaceCompared)
}

func TestEvent_JSONSerialization(t *testing.T) {
	event := Event{
		ID:         uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		TenantID:   uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"),
		EventType:  EventFaceRegistered,
		ExternalID: "ext-123",
		FaceID:     "face-456",
		Provider:   "rekognition",
		Success:    true,
		Metadata: map[string]string{
			"key": "value",
		},
		IPAddress: "192.168.1.1",
		UserAgent: "TestAgent/1.0",
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.ID, decoded.ID)
	assert.Equal(t, event.TenantID, decoded.TenantID)
	assert.Equal(t, event.EventType, decoded.EventType)
	assert.Equal(t, event.ExternalID, decoded.ExternalID)
	assert.Equal(t, event.FaceID, decoded.FaceID)
	assert.Equal(t, event.Provider, decoded.Provider)
	assert.Equal(t, event.Success, decoded.Success)
	assert.Equal(t, event.Metadata, decoded.Metadata)
	assert.Equal(t, event.IPAddress, decoded.IPAddress)
	assert.Equal(t, event.UserAgent, decoded.UserAgent)
}

func TestEvent_JSONSerialization_OmitsEmptyFields(t *testing.T) {
	event := Event{
		TenantID:  uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"),
		EventType: EventFaceDetected,
		Provider:  "rekognition",
		Success:   true,
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "external_id")
	assert.NotContains(t, jsonStr, "face_id")
	assert.NotContains(t, jsonStr, "error")
	assert.NotContains(t, jsonStr, "ip_address")
	assert.NotContains(t, jsonStr, "user_agent")
}
