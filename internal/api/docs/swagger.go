package docs

import (
	"github.com/go-swagno/swagno"
	"github.com/go-swagno/swagno/components/endpoint"
	"github.com/go-swagno/swagno/components/http/response"
	"github.com/go-swagno/swagno/components/mime"
	"github.com/go-swagno/swagno/components/parameter"
)

// RegisterFaceResponse represents the response for a successful face registration
type RegisterFaceResponse struct {
	FaceID       string  `json:"face_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ExternalID   string  `json:"external_id" example:"user-123"`
	QualityScore float64 `json:"quality_score" example:"0.95"`
	CreatedAt    string  `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

// VerifyFaceResponse represents the response for face verification
type VerifyFaceResponse struct {
	Verified   bool    `json:"verified" example:"true"`
	Confidence float64 `json:"confidence" example:"0.92"`
	ExternalID string  `json:"external_id" example:"user-123"`
	LatencyMs  int64   `json:"latency_ms" example:"45"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Code    string `json:"code" example:"VALIDATION_FAILED"`
	Message string `json:"message" example:"Request validation failed"`
}

// EmptyResponse represents no content response (204)
type EmptyResponse struct{}

// Admin Usage Metrics Types

// FacesTimeline represents timeline for face registrations
type FacesTimeline struct {
	Period     string `json:"period" example:"2024-01-01"`
	Registered int64  `json:"registered" example:"45"`
}

// FacesMetricsData contains face registration metrics
type FacesMetricsData struct {
	TotalRegistered int64           `json:"total_registered" example:"1500"`
	Active          int64           `json:"active" example:"1200"`
	Timeline        []FacesTimeline `json:"timeline"`
}

// OperationsTimeline represents timeline for operations
type OperationsTimeline struct {
	Period  string `json:"period" example:"2024-01-01"`
	Total   int64  `json:"total" example:"200"`
	Success int64  `json:"success" example:"180"`
	Failure int64  `json:"failure" example:"20"`
}

// OperationsMetricsData contains operation metrics
type OperationsMetricsData struct {
	TotalOperations int64                `json:"total_operations" example:"5000"`
	ByType          map[string]int64     `json:"by_type"`
	Timeline        []OperationsTimeline `json:"timeline"`
}

// RequestsTimeline represents timeline for HTTP requests
type RequestsTimeline struct {
	Period        string `json:"period" example:"2024-01-01"`
	Total         int64  `json:"total" example:"250"`
	FacesRegister int64  `json:"faces_register" example:"45"`
	FacesVerify   int64  `json:"faces_verify" example:"205"`
}

// RequestsMetricsData contains HTTP request metrics
type RequestsMetricsData struct {
	TotalRequests int64              `json:"total_requests" example:"6500"`
	ByEndpoint    map[string]int64   `json:"by_endpoint"`
	Timeline      []RequestsTimeline `json:"timeline"`
}

// Admin Performance Metrics Types

// LatencyTimeline represents latency timeline entry
type LatencyTimeline struct {
	Period    string  `json:"period" example:"2024-01-01"`
	AverageMs float64 `json:"average_ms" example:"45.5"`
	P50Ms     float64 `json:"p50_ms" example:"42.0"`
	P95Ms     float64 `json:"p95_ms" example:"98.5"`
	P99Ms     float64 `json:"p99_ms" example:"150.2"`
}

// LatencyMetricsData contains latency performance metrics
type LatencyMetricsData struct {
	AverageMs float64           `json:"average_ms" example:"45.5"`
	P50Ms     float64           `json:"p50_ms" example:"42.0"`
	P95Ms     float64           `json:"p95_ms" example:"98.5"`
	P99Ms     float64           `json:"p99_ms" example:"150.2"`
	Timeline  []LatencyTimeline `json:"timeline"`
}

// ThroughputTimeline represents throughput timeline entry
type ThroughputTimeline struct {
	Period          string  `json:"period" example:"2024-01-01"`
	Requests        int64   `json:"requests" example:"450"`
	RequestsPerHour float64 `json:"requests_per_hour" example:"18.75"`
}

// ThroughputMetricsData contains throughput performance metrics
type ThroughputMetricsData struct {
	TotalRequests       int64                `json:"total_requests" example:"10500"`
	RequestsPerHour     float64              `json:"requests_per_hour" example:"437.5"`
	PeakRequestsPerHour float64              `json:"peak_requests_per_hour" example:"650.0"`
	Timeline            []ThroughputTimeline `json:"timeline"`
}

// ErrorTimeline represents error timeline entry
type ErrorTimeline struct {
	Period    string  `json:"period" example:"2024-01-01"`
	Total     int64   `json:"total" example:"500"`
	Errors    int64   `json:"errors" example:"25"`
	ErrorRate float64 `json:"error_rate" example:"5.0"`
}

// ErrorMetricsData contains error rate metrics
type ErrorMetricsData struct {
	TotalErrors int64            `json:"total_errors" example:"250"`
	ErrorRate   float64          `json:"error_rate" example:"2.38"`
	ByType      map[string]int64 `json:"by_type"`
	Timeline    []ErrorTimeline  `json:"timeline"`
}

// Admin Quality Metrics Types

// QualityTimeline represents quality timeline entry
type QualityTimeline struct {
	Period         string  `json:"period" example:"2024-01-01"`
	AverageQuality float64 `json:"average_quality" example:"0.92"`
	MinQuality     float64 `json:"min_quality" example:"0.75"`
	MaxQuality     float64 `json:"max_quality" example:"0.99"`
}

// QualityMetricsData contains quality score metrics
type QualityMetricsData struct {
	AverageQuality float64           `json:"average_quality" example:"0.92"`
	MinQuality     float64           `json:"min_quality" example:"0.75"`
	MaxQuality     float64           `json:"max_quality" example:"0.99"`
	Timeline       []QualityTimeline `json:"timeline"`
}

// ConfidenceTimeline represents confidence timeline entry
type ConfidenceTimeline struct {
	Period            string  `json:"period" example:"2024-01-01"`
	AverageConfidence float64 `json:"average_confidence" example:"0.88"`
	MinConfidence     float64 `json:"min_confidence" example:"0.60"`
	MaxConfidence     float64 `json:"max_confidence" example:"0.98"`
}

// ConfidenceMetricsData contains confidence score metrics
type ConfidenceMetricsData struct {
	AverageConfidence float64              `json:"average_confidence" example:"0.88"`
	MinConfidence     float64              `json:"min_confidence" example:"0.60"`
	MaxConfidence     float64              `json:"max_confidence" example:"0.98"`
	Timeline          []ConfidenceTimeline `json:"timeline"`
}

// MatchTimeline represents match timeline entry
type MatchTimeline struct {
	Period            string  `json:"period" example:"2024-01-01"`
	Matches           int64   `json:"matches" example:"180"`
	Total             int64   `json:"total" example:"200"`
	MatchRate         float64 `json:"match_rate" example:"90.0"`
	AverageMatchScore float64 `json:"average_match_score" example:"0.92"`
}

// MatchMetricsData contains matching statistics
type MatchMetricsData struct {
	TotalMatches      int64           `json:"total_matches" example:"4500"`
	MatchRate         float64         `json:"match_rate" example:"90.0"`
	AverageMatchScore float64         `json:"average_match_score" example:"0.92"`
	Timeline          []MatchTimeline `json:"timeline"`
}

// Common Admin Types

// PeriodInfo represents the time period for metrics
type PeriodInfo struct {
	Start string `json:"start" example:"2024-01-01"`
	End   string `json:"end" example:"2024-01-31"`
}

// PaginationMeta contains pagination information
type PaginationMeta struct {
	Total  int `json:"total" example:"30"`
	Limit  int `json:"limit" example:"100"`
	Offset int `json:"offset" example:"0"`
}

// AdminResponseMeta contains metadata about the response
type AdminResponseMeta struct {
	TenantID    string     `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Period      PeriodInfo `json:"period"`
	GeneratedAt string     `json:"generated_at" example:"2024-01-01T00:00:00Z"`
}

// Response wrappers

// FacesMetricsResponse wraps faces metrics
type FacesMetricsResponse struct {
	Data       FacesMetricsData  `json:"data"`
	Meta       AdminResponseMeta `json:"meta"`
	Pagination *PaginationMeta   `json:"pagination,omitempty"`
}

// OperationsMetricsResponse wraps operations metrics
type OperationsMetricsResponse struct {
	Data       OperationsMetricsData `json:"data"`
	Meta       AdminResponseMeta     `json:"meta"`
	Pagination *PaginationMeta       `json:"pagination,omitempty"`
}

// RequestsMetricsResponse wraps requests metrics
type RequestsMetricsResponse struct {
	Data       RequestsMetricsData `json:"data"`
	Meta       AdminResponseMeta   `json:"meta"`
	Pagination *PaginationMeta     `json:"pagination,omitempty"`
}

// LatencyMetricsResponse wraps latency metrics
type LatencyMetricsResponse struct {
	Data       LatencyMetricsData `json:"data"`
	Meta       AdminResponseMeta  `json:"meta"`
	Pagination *PaginationMeta    `json:"pagination,omitempty"`
}

// ThroughputMetricsResponse wraps throughput metrics
type ThroughputMetricsResponse struct {
	Data       ThroughputMetricsData `json:"data"`
	Meta       AdminResponseMeta     `json:"meta"`
	Pagination *PaginationMeta       `json:"pagination,omitempty"`
}

// ErrorMetricsResponse wraps error metrics
type ErrorMetricsResponse struct {
	Data       ErrorMetricsData  `json:"data"`
	Meta       AdminResponseMeta `json:"meta"`
	Pagination *PaginationMeta   `json:"pagination,omitempty"`
}

// QualityMetricsResponse wraps quality metrics
type QualityMetricsResponse struct {
	Data       QualityMetricsData `json:"data"`
	Meta       AdminResponseMeta  `json:"meta"`
	Pagination *PaginationMeta    `json:"pagination,omitempty"`
}

// ConfidenceMetricsResponse wraps confidence metrics
type ConfidenceMetricsResponse struct {
	Data       ConfidenceMetricsData `json:"data"`
	Meta       AdminResponseMeta     `json:"meta"`
	Pagination *PaginationMeta       `json:"pagination,omitempty"`
}

// MatchMetricsResponse wraps match metrics
type MatchMetricsResponse struct {
	Data       MatchMetricsData  `json:"data"`
	Meta       AdminResponseMeta `json:"meta"`
	Pagination *PaginationMeta   `json:"pagination,omitempty"`
}

// NewSwagger creates and configures the Swagger documentation
func NewSwagger() *swagno.Swagger {
	sw := swagno.New(swagno.Config{
		Title:       "Rekko Face Recognition API",
		Version:     "v1.0.0",
		Description: "FRaaS (Facial Recognition as a Service) API for event access control with multi-tenancy support",
		Host:        "localhost:3000",
		Path:        "/v1",
	})

	endpoints := []*endpoint.EndPoint{
		// Faces endpoints (existing)

		// GET /v1/admin/metrics/quality - Quality Metrics
		endpoint.New(
			endpoint.GET,
			"/admin/metrics/quality",
			endpoint.WithTags("Admin Metrics - Quality"),
			endpoint.WithSummary("Get quality score metrics"),
			endpoint.WithDescription("Returns quality score metrics for face registrations"),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithParams(
				parameter.StrParam("start_date", parameter.Query, parameter.WithDescription("Start date (YYYY-MM-DD)")),
				parameter.StrParam("end_date", parameter.Query, parameter.WithDescription("End date (YYYY-MM-DD)")),
				parameter.StrParam("interval", parameter.Query, parameter.WithDescription("Aggregation interval: hour, day, week, month (default: day)")),
				parameter.IntParam("limit", parameter.Query, parameter.WithDescription("Maximum number of timeline points (default: 100, max: 1000)")),
				parameter.IntParam("offset", parameter.Query, parameter.WithDescription("Offset for timeline pagination (default: 0)")),
			),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(QualityMetricsResponse{}, "200", "Metrics retrieved successfully"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "VALIDATION_FAILED", Message: "Invalid date format"}, "400", "Bad Request"),
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing API key"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"}, "500", "Internal Server Error"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"ApiKeyAuth": {}}}),
		),

		// GET /v1/admin/metrics/confidence - Confidence Metrics
		endpoint.New(
			endpoint.GET,
			"/admin/metrics/confidence",
			endpoint.WithTags("Admin Metrics - Quality"),
			endpoint.WithSummary("Get confidence score metrics"),
			endpoint.WithDescription("Returns confidence score metrics for verifications"),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithParams(
				parameter.StrParam("start_date", parameter.Query, parameter.WithDescription("Start date (YYYY-MM-DD)")),
				parameter.StrParam("end_date", parameter.Query, parameter.WithDescription("End date (YYYY-MM-DD)")),
				parameter.StrParam("interval", parameter.Query, parameter.WithDescription("Aggregation interval: hour, day, week, month (default: day)")),
				parameter.IntParam("limit", parameter.Query, parameter.WithDescription("Maximum number of timeline points (default: 100, max: 1000)")),
				parameter.IntParam("offset", parameter.Query, parameter.WithDescription("Offset for timeline pagination (default: 0)")),
			),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(ConfidenceMetricsResponse{}, "200", "Metrics retrieved successfully"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "VALIDATION_FAILED", Message: "Invalid date format"}, "400", "Bad Request"),
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing API key"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"}, "500", "Internal Server Error"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"ApiKeyAuth": {}}}),
		),

		// GET /v1/admin/metrics/matches - Match Metrics
		endpoint.New(
			endpoint.GET,
			"/admin/metrics/matches",
			endpoint.WithTags("Admin Metrics - Quality"),
			endpoint.WithSummary("Get matching statistics"),
			endpoint.WithDescription("Returns matching statistics for verifications"),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithParams(
				parameter.StrParam("start_date", parameter.Query, parameter.WithDescription("Start date (YYYY-MM-DD)")),
				parameter.StrParam("end_date", parameter.Query, parameter.WithDescription("End date (YYYY-MM-DD)")),
				parameter.StrParam("interval", parameter.Query, parameter.WithDescription("Aggregation interval: hour, day, week, month (default: day)")),
				parameter.IntParam("limit", parameter.Query, parameter.WithDescription("Maximum number of timeline points (default: 100, max: 1000)")),
				parameter.IntParam("offset", parameter.Query, parameter.WithDescription("Offset for timeline pagination (default: 0)")),
			),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(MatchMetricsResponse{}, "200", "Metrics retrieved successfully"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "VALIDATION_FAILED", Message: "Invalid date format"}, "400", "Bad Request"),
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing API key"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"}, "500", "Internal Server Error"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"ApiKeyAuth": {}}}),
		),
	}

	sw.AddEndpoints(endpoints)

	return sw
}
