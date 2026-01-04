package admin

import "time"

// MetricsParams holds query parameters for metrics endpoints
type MetricsParams struct {
	StartDate time.Time
	EndDate   time.Time
	Interval  string // hour, day, week, month
	Limit     int
	Offset    int
}

// MetricsResponse is the standard response wrapper for metrics endpoints
type MetricsResponse struct {
	Data       interface{}     `json:"data"`
	Meta       ResponseMeta    `json:"meta"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

// ResponseMeta contains metadata about the metrics response
type ResponseMeta struct {
	TenantID    string    `json:"tenant_id"`
	Period      Period    `json:"period"`
	GeneratedAt time.Time `json:"generated_at"`
}

// Period represents a time period
type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// PaginationMeta contains pagination information
type PaginationMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// FacesMetrics contains metrics about faces
type FacesMetrics struct {
	TotalRegistered int64           `json:"total_registered"`
	Active          int64           `json:"active"`
	Timeline        []FacesTimeline `json:"timeline"`
}

// FacesTimeline represents a timeline entry for faces metrics
type FacesTimeline struct {
	Period     string `json:"period"`
	Registered int64  `json:"registered"`
}

// OperationsMetrics contains metrics about operations
type OperationsMetrics struct {
	TotalOperations int64                `json:"total_operations"`
	ByType          map[string]int64     `json:"by_type"`
	Timeline        []OperationsTimeline `json:"timeline"`
}

// OperationsTimeline represents a timeline entry for operations metrics
type OperationsTimeline struct {
	Period  string `json:"period"`
	Total   int64  `json:"total"`
	Success int64  `json:"success"`
	Failure int64  `json:"failure"`
}

// RequestsMetrics contains metrics about HTTP requests
type RequestsMetrics struct {
	TotalRequests int64              `json:"total_requests"`
	ByEndpoint    map[string]int64   `json:"by_endpoint"`
	Timeline      []RequestsTimeline `json:"timeline"`
}

// RequestsTimeline represents a timeline entry for requests metrics
type RequestsTimeline struct {
	Period        string `json:"period"`
	Total         int64  `json:"total"`
	FacesRegister int64  `json:"faces_register"`
	FacesVerify   int64  `json:"faces_verify"`
}

// LatencyMetrics contains latency performance metrics
type LatencyMetrics struct {
	AverageMs float64           `json:"average_ms"`
	P50Ms     float64           `json:"p50_ms"`
	P95Ms     float64           `json:"p95_ms"`
	P99Ms     float64           `json:"p99_ms"`
	Timeline  []LatencyTimeline `json:"timeline"`
}

// LatencyTimeline represents a timeline entry for latency metrics
type LatencyTimeline struct {
	Period    string  `json:"period"`
	AverageMs float64 `json:"average_ms"`
	P50Ms     float64 `json:"p50_ms"`
	P95Ms     float64 `json:"p95_ms"`
	P99Ms     float64 `json:"p99_ms"`
}

// ThroughputMetrics contains throughput performance metrics
type ThroughputMetrics struct {
	TotalRequests       int64                `json:"total_requests"`
	RequestsPerHour     float64              `json:"requests_per_hour"`
	PeakRequestsPerHour float64              `json:"peak_requests_per_hour"`
	Timeline            []ThroughputTimeline `json:"timeline"`
}

// ThroughputTimeline represents a timeline entry for throughput metrics
type ThroughputTimeline struct {
	Period          string  `json:"period"`
	Requests        int64   `json:"requests"`
	RequestsPerHour float64 `json:"requests_per_hour"`
}

// ErrorMetrics contains error rate metrics
type ErrorMetrics struct {
	TotalErrors int64            `json:"total_errors"`
	ErrorRate   float64          `json:"error_rate"`
	ByType      map[string]int64 `json:"by_type"`
	Timeline    []ErrorTimeline  `json:"timeline"`
}

// ErrorTimeline represents a timeline entry for error metrics
type ErrorTimeline struct {
	Period    string  `json:"period"`
	Total     int64   `json:"total"`
	Errors    int64   `json:"errors"`
	ErrorRate float64 `json:"error_rate"`
}

// QualityMetrics contains quality score metrics for faces
type QualityMetrics struct {
	AverageQuality float64           `json:"average_quality"`
	MinQuality     float64           `json:"min_quality"`
	MaxQuality     float64           `json:"max_quality"`
	Timeline       []QualityTimeline `json:"timeline"`
}

// QualityTimeline represents a timeline entry for quality metrics
type QualityTimeline struct {
	Period         string  `json:"period"`
	AverageQuality float64 `json:"average_quality"`
	MinQuality     float64 `json:"min_quality"`
	MaxQuality     float64 `json:"max_quality"`
}

// ConfidenceMetrics contains confidence score metrics for verifications
type ConfidenceMetrics struct {
	AverageConfidence float64              `json:"average_confidence"`
	MinConfidence     float64              `json:"min_confidence"`
	MaxConfidence     float64              `json:"max_confidence"`
	Timeline          []ConfidenceTimeline `json:"timeline"`
}

// ConfidenceTimeline represents a timeline entry for confidence metrics
type ConfidenceTimeline struct {
	Period            string  `json:"period"`
	AverageConfidence float64 `json:"average_confidence"`
	MinConfidence     float64 `json:"min_confidence"`
	MaxConfidence     float64 `json:"max_confidence"`
}

// MatchMetrics contains matching statistics
type MatchMetrics struct {
	TotalMatches      int64           `json:"total_matches"`
	MatchRate         float64         `json:"match_rate"`
	AverageMatchScore float64         `json:"average_match_score"`
	Timeline          []MatchTimeline `json:"timeline"`
}

// MatchTimeline represents a timeline entry for match metrics
type MatchTimeline struct {
	Period            string  `json:"period"`
	Matches           int64   `json:"matches"`
	Total             int64   `json:"total"`
	MatchRate         float64 `json:"match_rate"`
	AverageMatchScore float64 `json:"average_match_score"`
}

// Super Admin Types

// TenantWithMetrics represents a tenant with summary metrics
type TenantWithMetrics struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	PlanType  string               `json:"plan_type"`
	IsActive  bool                 `json:"is_active"`
	CreatedAt string               `json:"created_at"`
	Metrics   TenantMetricsSummary `json:"metrics"`
}

// TenantMetricsSummary contains aggregated metrics for a tenant
type TenantMetricsSummary struct {
	TotalFaces    int64   `json:"total_faces"`
	TotalRequests int64   `json:"total_requests"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	ErrorRate     float64 `json:"error_rate"`
}

// SystemHealth represents system-wide health status
type SystemHealth struct {
	Status    string           `json:"status"`
	Database  ServiceHealth    `json:"database"`
	Providers []ProviderHealth `json:"providers"`
	Uptime    string           `json:"uptime"`
	Version   string           `json:"version"`
}

// ServiceHealth represents health of a single service
type ServiceHealth struct {
	Status  string `json:"status"`
	Latency string `json:"latency"`
	Message string `json:"message,omitempty"`
}

// ProviderHealth represents health of a face recognition provider
type ProviderHealth struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Message string `json:"message,omitempty"`
}

// SystemMetrics contains system-wide metrics
type SystemMetrics struct {
	Memory            MemoryMetrics `json:"memory"`
	Goroutines        int           `json:"goroutines"`
	DBConnections     DBConnMetrics `json:"db_connections"`
	RequestsPerSecond float64       `json:"requests_per_second"`
}

// MemoryMetrics contains Go runtime memory metrics
type MemoryMetrics struct {
	Alloc      uint64 `json:"alloc_bytes"`
	TotalAlloc uint64 `json:"total_alloc_bytes"`
	Sys        uint64 `json:"sys_bytes"`
	NumGC      uint32 `json:"num_gc"`
}

// DBConnMetrics contains database connection pool metrics
type DBConnMetrics struct {
	TotalConns int32 `json:"total_conns"`
	IdleConns  int32 `json:"idle_conns"`
	MaxConns   int32 `json:"max_conns"`
}

// UpdateQuotaRequest represents a request to update tenant quotas
type UpdateQuotaRequest struct {
	MaxFaces         *int     `json:"max_faces,omitempty"`
	MaxRequestsHour  *int     `json:"max_requests_hour,omitempty"`
	MaxRequestsMonth *int     `json:"max_requests_month,omitempty"`
	ThresholdValue   *float64 `json:"threshold_value,omitempty"`
}
