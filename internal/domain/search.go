package domain

import (
	"time"

	"github.com/google/uuid"
)

// SearchMatch represents a face match result from similarity search
type SearchMatch struct {
	FaceID     uuid.UUID              `json:"face_id"`
	ExternalID string                 `json:"external_id"`
	Similarity float64                `json:"similarity"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SearchResult represents the complete search response
type SearchResult struct {
	Matches    []SearchMatch `json:"matches"`
	TotalFaces int           `json:"total_faces"`
	LatencyMs  int64         `json:"latency_ms"`
	SearchID   uuid.UUID     `json:"search_id"`
}

// SearchAudit represents an audit log entry for search operations
type SearchAudit struct {
	ID                 uuid.UUID `json:"id"`
	TenantID           uuid.UUID `json:"tenant_id"`
	ResultsCount       int       `json:"results_count"`
	TopMatchExternalID *string   `json:"top_match_external_id,omitempty"`
	TopMatchSimilarity *float64  `json:"top_match_similarity,omitempty"`
	Threshold          float64   `json:"threshold"`
	MaxResults         int       `json:"max_results"`
	LatencyMs          int64     `json:"latency_ms"`
	ClientIP           string    `json:"client_ip"`
	CreatedAt          time.Time `json:"created_at"`
}
