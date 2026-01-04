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


// Super Admin Types

// TenantMetricsSummary contains summary metrics for a tenant
type TenantMetricsSummary struct {
	TotalFaces    int64   `json:"total_faces" example:"250"`
	TotalRequests int64   `json:"total_requests" example:"5000"`
	AvgLatencyMs  float64 `json:"avg_latency_ms" example:"45.5"`
	ErrorRate     float64 `json:"error_rate" example:"2.5"`
}

// TenantWithMetrics represents a tenant with metrics
type TenantWithMetrics struct {
	ID        string               `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string               `json:"name" example:"Acme Corp"`
	PlanType  string               `json:"plan_type" example:"pro"`
	IsActive  bool                 `json:"is_active" example:"true"`
	CreatedAt string               `json:"created_at" example:"2024-01-01T00:00:00Z"`
	Metrics   TenantMetricsSummary `json:"metrics"`
}

// ListTenantsResponse wraps list of tenants with metrics
type ListTenantsResponse struct {
	Data []TenantWithMetrics `json:"data"`
	Meta map[string]int      `json:"meta"`
}

// TenantDetailedMetricsResponse wraps detailed tenant metrics
type TenantDetailedMetricsResponse struct {
	Data TenantMetricsSummary `json:"data"`
	Meta map[string]string    `json:"meta"`
}

// ServiceHealth represents health of a single service
type ServiceHealth struct {
	Status  string `json:"status" example:"healthy"`
	Latency string `json:"latency" example:"< 1ms"`
	Message string `json:"message,omitempty"`
}

// ProviderHealth represents health of a face recognition provider
type ProviderHealth struct {
	Name    string `json:"name" example:"rekognition"`
	Status  string `json:"status" example:"healthy"`
	Latency string `json:"latency,omitempty" example:"15ms"`
	Message string `json:"message,omitempty"`
}

// SystemHealthResponse represents system health check response
type SystemHealthResponse struct {
	Status    string           `json:"status" example:"healthy"`
	Database  ServiceHealth    `json:"database"`
	Providers []ProviderHealth `json:"providers"`
	Uptime    string           `json:"uptime" example:"24h30m"`
	Version   string           `json:"version" example:"1.0.0"`
}

// MemoryMetrics contains Go runtime memory metrics
type MemoryMetrics struct {
	Alloc      uint64 `json:"alloc_bytes" example:"5242880"`
	TotalAlloc uint64 `json:"total_alloc_bytes" example:"104857600"`
	Sys        uint64 `json:"sys_bytes" example:"20971520"`
	NumGC      uint32 `json:"num_gc" example:"42"`
}

// DBConnMetrics contains database connection pool metrics
type DBConnMetrics struct {
	TotalConns int32 `json:"total_conns" example:"10"`
	IdleConns  int32 `json:"idle_conns" example:"8"`
	MaxConns   int32 `json:"max_conns" example:"20"`
}

// SystemMetricsData contains system-wide metrics
type SystemMetricsData struct {
	Memory            MemoryMetrics `json:"memory"`
	Goroutines        int           `json:"goroutines" example:"50"`
	DBConnections     DBConnMetrics `json:"db_connections"`
	RequestsPerSecond float64       `json:"requests_per_second" example:"125.5"`
}

// SystemMetricsResponse wraps system metrics
type SystemMetricsResponse struct {
	Data SystemMetricsData `json:"data"`
}

// ProvidersStatusResponse wraps provider status list
type ProvidersStatusResponse struct {
	Data []ProviderHealth `json:"data"`
}

// UpdateQuotaRequest represents a request to update tenant quotas
type UpdateQuotaRequest struct {
	MaxFaces         *int     `json:"max_faces,omitempty" example:"5000"`
	MaxRequestsHour  *int     `json:"max_requests_hour,omitempty" example:"10000"`
	MaxRequestsMonth *int     `json:"max_requests_month,omitempty" example:"1000000"`
	ThresholdValue   *float64 `json:"threshold_value,omitempty" example:"0.85"`
}

// UpdateQuotaResponse represents response after quota update
type UpdateQuotaResponse struct {
	Message  string `json:"message" example:"quota updated successfully"`
	TenantID string `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
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

		// Super Admin Endpoints

		// GET /v1/super/tenants - List all tenants
		endpoint.New(
			endpoint.GET,
			"/super/tenants",
			endpoint.WithTags("Super Admin"),
			endpoint.WithSummary("List all tenants with metrics"),
			endpoint.WithDescription("Returns a list of all tenants with summary metrics (requires super admin JWT authentication)"),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithParams(
				parameter.IntParam("limit", parameter.Query, parameter.WithDescription("Maximum number of tenants (default: 50, max: 100)")),
				parameter.IntParam("offset", parameter.Query, parameter.WithDescription("Offset for pagination (default: 0)")),
			),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(ListTenantsResponse{}, "200", "Tenants retrieved successfully"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing JWT token"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "FORBIDDEN", Message: "Insufficient privileges"}, "403", "Forbidden"),
				response.New(ErrorResponse{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"}, "500", "Internal Server Error"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"BearerAuth": {}}}),
		),

		// GET /v1/super/tenants/:id/metrics - Get tenant detailed metrics
		endpoint.New(
			endpoint.GET,
			"/super/tenants/{id}/metrics",
			endpoint.WithTags("Super Admin"),
			endpoint.WithSummary("Get detailed metrics for a tenant"),
			endpoint.WithDescription("Returns detailed metrics for a specific tenant (requires super admin JWT authentication)"),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithParams(
				parameter.StrParam("id", parameter.Path, parameter.WithDescription("Tenant UUID")),
			),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(TenantDetailedMetricsResponse{}, "200", "Metrics retrieved successfully"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "VALIDATION_FAILED", Message: "Invalid tenant ID format"}, "400", "Bad Request"),
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing JWT token"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "FORBIDDEN", Message: "Insufficient privileges"}, "403", "Forbidden"),
				response.New(ErrorResponse{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"}, "500", "Internal Server Error"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"BearerAuth": {}}}),
		),

		// POST /v1/super/tenants/:id/quota - Update tenant quota
		endpoint.New(
			endpoint.POST,
			"/super/tenants/{id}/quota",
			endpoint.WithTags("Super Admin"),
			endpoint.WithSummary("Update tenant quota settings"),
			endpoint.WithDescription("Updates quota settings for a specific tenant (requires super admin JWT authentication)"),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithConsume([]mime.MIME{mime.JSON}),
			endpoint.WithParams(
				parameter.StrParam("id", parameter.Path, parameter.WithDescription("Tenant UUID")),
			),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(UpdateQuotaResponse{}, "200", "Quota updated successfully"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "VALIDATION_FAILED", Message: "Invalid request body"}, "400", "Bad Request"),
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing JWT token"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "FORBIDDEN", Message: "Insufficient privileges"}, "403", "Forbidden"),
				response.New(ErrorResponse{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"}, "500", "Internal Server Error"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"BearerAuth": {}}}),
		),

		// GET /v1/super/system/health - System health check
		endpoint.New(
			endpoint.GET,
			"/super/system/health",
			endpoint.WithTags("Super Admin"),
			endpoint.WithSummary("Get system health status"),
			endpoint.WithDescription("Returns health status of all system components (requires super admin JWT authentication)"),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(SystemHealthResponse{}, "200", "System is healthy"),
				response.New(SystemHealthResponse{Status: "degraded"}, "200", "System is degraded"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing JWT token"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "FORBIDDEN", Message: "Insufficient privileges"}, "403", "Forbidden"),
				response.New(SystemHealthResponse{Status: "unhealthy"}, "503", "System is unhealthy"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"BearerAuth": {}}}),
		),

		// GET /v1/super/system/metrics - System metrics
		endpoint.New(
			endpoint.GET,
			"/super/system/metrics",
			endpoint.WithTags("Super Admin"),
			endpoint.WithSummary("Get system-wide metrics"),
			endpoint.WithDescription("Returns system-wide metrics (memory, goroutines, DB connections, etc.) (requires super admin JWT authentication)"),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(SystemMetricsResponse{}, "200", "Metrics retrieved successfully"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing JWT token"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "FORBIDDEN", Message: "Insufficient privileges"}, "403", "Forbidden"),
				response.New(ErrorResponse{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"}, "500", "Internal Server Error"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"BearerAuth": {}}}),
		),

		// GET /v1/super/providers - Providers status
		endpoint.New(
			endpoint.GET,
			"/super/providers",
			endpoint.WithTags("Super Admin"),
			endpoint.WithSummary("Get face recognition providers status"),
			endpoint.WithDescription("Returns status of all face recognition providers (requires super admin JWT authentication)"),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(ProvidersStatusResponse{}, "200", "Providers status retrieved successfully"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing JWT token"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "FORBIDDEN", Message: "Insufficient privileges"}, "403", "Forbidden"),
				response.New(ErrorResponse{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"}, "500", "Internal Server Error"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"BearerAuth": {}}}),
		),
	}

	sw.AddEndpoints(endpoints)

	return sw
}
