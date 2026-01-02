package domain

import (
	"time"

	"github.com/google/uuid"
)

// Face representa uma face cadastrada no sistema
type Face struct {
	ID           uuid.UUID              `json:"id"`
	TenantID     uuid.UUID              `json:"-"`
	ExternalID   string                 `json:"external_id"`
	Embedding    []float64              `json:"-"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	QualityScore float64                `json:"quality_score"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// Verification representa um registro de verificação (audit)
type Verification struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"-"`
	FaceID         *uuid.UUID `json:"face_id,omitempty"`
	ExternalID     string     `json:"external_id"`
	Verified       bool       `json:"verified"`
	Confidence     float64    `json:"confidence"`
	LivenessPassed *bool      `json:"liveness_passed,omitempty"`
	LatencyMs      int64      `json:"latency_ms"`
	CreatedAt      time.Time  `json:"created_at"`
}
