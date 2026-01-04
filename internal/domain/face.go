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

// LivenessResult represents the result of a liveness check
type LivenessResult struct {
	IsLive     bool           `json:"is_live"`
	Confidence float64        `json:"confidence"`
	Reasons    []string       `json:"reasons,omitempty"`
	Checks     LivenessChecks `json:"checks"`
}

// LivenessChecks contains individual liveness check results
type LivenessChecks struct {
	EyesOpen     bool `json:"eyes_open"`
	FacingCamera bool `json:"facing_camera"`
	QualityOK    bool `json:"quality_ok"`
	SingleFace   bool `json:"single_face"`
}
