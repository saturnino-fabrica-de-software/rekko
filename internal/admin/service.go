package admin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saturnino-fabrica-de-software/rekko/internal/metrics"
)

// Service handles admin business logic
type Service struct {
	metricsRepo *metrics.Repository
	db          *pgxpool.Pool
	logger      *slog.Logger
}

// NewService creates a new admin service
func NewService(metricsRepo *metrics.Repository, db *pgxpool.Pool, logger *slog.Logger) *Service {
	return &Service{
		metricsRepo: metricsRepo,
		db:          db,
		logger:      logger,
	}
}

// GetFacesMetrics retrieves metrics about faces
func (s *Service) GetFacesMetrics(ctx context.Context, tenantID uuid.UUID, params MetricsParams) (*FacesMetrics, error) {
	// Total registered faces (all time)
	var totalRegistered int64
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM faces 
		WHERE tenant_id = $1
	`, tenantID).Scan(&totalRegistered)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to count total faces: %w", tenantID, err)
	}

	// Active = same as total for now (no soft delete)
	active := totalRegistered

	// Timeline of registrations
	rows, err := s.db.Query(ctx, `
		SELECT 
			date_trunc($1, created_at) as period,
			COUNT(*) as registered
		FROM faces
		WHERE tenant_id = $2 
		  AND created_at BETWEEN $3 AND $4
		GROUP BY period
		ORDER BY period ASC
		LIMIT $5 OFFSET $6
	`, params.Interval, tenantID, params.StartDate, params.EndDate, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to query faces timeline: %w", tenantID, err)
	}
	defer rows.Close()

	timeline := make([]FacesTimeline, 0)
	for rows.Next() {
		var entry FacesTimeline
		var period interface{}
		err := rows.Scan(&period, &entry.Registered)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: failed to scan faces timeline: %w", tenantID, err)
		}
		entry.Period = fmt.Sprint(period)
		timeline = append(timeline, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("tenant %s: faces timeline iteration error: %w", tenantID, err)
	}

	return &FacesMetrics{
		TotalRegistered: totalRegistered,
		Active:          active,
		Timeline:        timeline,
	}, nil
}

// GetOperationsMetrics retrieves metrics about operations (verifications)
func (s *Service) GetOperationsMetrics(ctx context.Context, tenantID uuid.UUID, params MetricsParams) (*OperationsMetrics, error) {
	// Total operations (all time)
	var totalOperations int64
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM verifications 
		WHERE tenant_id = $1
	`, tenantID).Scan(&totalOperations)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to count total operations: %w", tenantID, err)
	}

	// By type (only verification for now)
	byType := map[string]int64{
		"verification": totalOperations,
	}

	// Timeline of operations
	rows, err := s.db.Query(ctx, `
		SELECT 
			date_trunc($1, created_at) as period,
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE verified = true) as success,
			COUNT(*) FILTER (WHERE verified = false) as failure
		FROM verifications
		WHERE tenant_id = $2 
		  AND created_at BETWEEN $3 AND $4
		GROUP BY period
		ORDER BY period ASC
		LIMIT $5 OFFSET $6
	`, params.Interval, tenantID, params.StartDate, params.EndDate, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to query operations timeline: %w", tenantID, err)
	}
	defer rows.Close()

	timeline := make([]OperationsTimeline, 0)
	for rows.Next() {
		var entry OperationsTimeline
		var period interface{}
		err := rows.Scan(&period, &entry.Total, &entry.Success, &entry.Failure)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: failed to scan operations timeline: %w", tenantID, err)
		}
		entry.Period = fmt.Sprint(period)
		timeline = append(timeline, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("tenant %s: operations timeline iteration error: %w", tenantID, err)
	}

	return &OperationsMetrics{
		TotalOperations: totalOperations,
		ByType:          byType,
		Timeline:        timeline,
	}, nil
}

// GetRequestsMetrics retrieves metrics about HTTP requests (approximated from faces + verifications)
func (s *Service) GetRequestsMetrics(ctx context.Context, tenantID uuid.UUID, params MetricsParams) (*RequestsMetrics, error) {
	// Timeline combining faces and verifications
	rows, err := s.db.Query(ctx, `
		WITH face_timeline AS (
			SELECT 
				date_trunc($1, created_at) as period,
				COUNT(*) as count
			FROM faces
			WHERE tenant_id = $2 
			  AND created_at BETWEEN $3 AND $4
			GROUP BY period
		),
		verification_timeline AS (
			SELECT 
				date_trunc($1, created_at) as period,
				COUNT(*) as count
			FROM verifications
			WHERE tenant_id = $2 
			  AND created_at BETWEEN $3 AND $4
			GROUP BY period
		)
		SELECT 
			COALESCE(f.period, v.period) as period,
			COALESCE(f.count, 0) as faces_register,
			COALESCE(v.count, 0) as faces_verify,
			COALESCE(f.count, 0) + COALESCE(v.count, 0) as total
		FROM face_timeline f
		FULL OUTER JOIN verification_timeline v ON f.period = v.period
		ORDER BY period ASC
		LIMIT $5 OFFSET $6
	`, params.Interval, tenantID, params.StartDate, params.EndDate, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to query requests timeline: %w", tenantID, err)
	}
	defer rows.Close()

	timeline := make([]RequestsTimeline, 0)
	var totalFacesRegister, totalFacesVerify int64

	for rows.Next() {
		var entry RequestsTimeline
		var period interface{}
		err := rows.Scan(&period, &entry.FacesRegister, &entry.FacesVerify, &entry.Total)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: failed to scan requests timeline: %w", tenantID, err)
		}
		entry.Period = fmt.Sprint(period)
		timeline = append(timeline, entry)

		totalFacesRegister += entry.FacesRegister
		totalFacesVerify += entry.FacesVerify
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("tenant %s: requests timeline iteration error: %w", tenantID, err)
	}

	totalRequests := totalFacesRegister + totalFacesVerify
	byEndpoint := map[string]int64{
		"/v1/faces":        totalFacesRegister,
		"/v1/faces/verify": totalFacesVerify,
	}

	return &RequestsMetrics{
		TotalRequests: totalRequests,
		ByEndpoint:    byEndpoint,
		Timeline:      timeline,
	}, nil
}

// GetLatencyMetrics retrieves latency performance metrics
func (s *Service) GetLatencyMetrics(ctx context.Context, tenantID uuid.UUID, params MetricsParams) (*LatencyMetrics, error) {
	// Get overall latency percentiles from verifications
	var avgMs, p50Ms, p95Ms, p99Ms float64
	err := s.db.QueryRow(ctx, `
		SELECT 
			AVG(latency_ms) as avg_ms,
			PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY latency_ms) as p50_ms,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms) as p95_ms,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms) as p99_ms
		FROM verifications
		WHERE tenant_id = $1
		  AND created_at BETWEEN $2 AND $3
	`, tenantID, params.StartDate, params.EndDate).Scan(&avgMs, &p50Ms, &p95Ms, &p99Ms)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to calculate latency percentiles: %w", tenantID, err)
	}

	// Timeline of latency metrics
	rows, err := s.db.Query(ctx, `
		SELECT 
			date_trunc($1, created_at) as period,
			AVG(latency_ms) as avg_ms,
			PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY latency_ms) as p50_ms,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms) as p95_ms,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms) as p99_ms
		FROM verifications
		WHERE tenant_id = $2 
		  AND created_at BETWEEN $3 AND $4
		GROUP BY period
		ORDER BY period ASC
		LIMIT $5 OFFSET $6
	`, params.Interval, tenantID, params.StartDate, params.EndDate, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to query latency timeline: %w", tenantID, err)
	}
	defer rows.Close()

	timeline := make([]LatencyTimeline, 0)
	for rows.Next() {
		var entry LatencyTimeline
		var period interface{}
		err := rows.Scan(&period, &entry.AverageMs, &entry.P50Ms, &entry.P95Ms, &entry.P99Ms)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: failed to scan latency timeline: %w", tenantID, err)
		}
		entry.Period = fmt.Sprint(period)
		timeline = append(timeline, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("tenant %s: latency timeline iteration error: %w", tenantID, err)
	}

	return &LatencyMetrics{
		AverageMs: avgMs,
		P50Ms:     p50Ms,
		P95Ms:     p95Ms,
		P99Ms:     p99Ms,
		Timeline:  timeline,
	}, nil
}

// GetThroughputMetrics retrieves throughput performance metrics
func (s *Service) GetThroughputMetrics(ctx context.Context, tenantID uuid.UUID, params MetricsParams) (*ThroughputMetrics, error) {
	// Total requests in the period
	var totalRequests int64
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM (
			SELECT created_at FROM faces WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
			UNION ALL
			SELECT created_at FROM verifications WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
		) combined
	`, tenantID, params.StartDate, params.EndDate).Scan(&totalRequests)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to count total requests: %w", tenantID, err)
	}

	// Calculate requests per hour
	hours := params.EndDate.Sub(params.StartDate).Hours()
	if hours == 0 {
		hours = 1
	}
	requestsPerHour := float64(totalRequests) / hours

	// Timeline with hourly rates
	rows, err := s.db.Query(ctx, `
		WITH combined_requests AS (
			SELECT date_trunc($1, created_at) as period FROM faces 
			WHERE tenant_id = $2 AND created_at BETWEEN $3 AND $4
			UNION ALL
			SELECT date_trunc($1, created_at) as period FROM verifications 
			WHERE tenant_id = $2 AND created_at BETWEEN $3 AND $4
		)
		SELECT 
			period,
			COUNT(*) as requests,
			COUNT(*)::float / EXTRACT(EPOCH FROM ($1::interval)) * 3600 as requests_per_hour
		FROM combined_requests
		GROUP BY period
		ORDER BY period ASC
		LIMIT $5 OFFSET $6
	`, params.Interval, tenantID, params.StartDate, params.EndDate, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to query throughput timeline: %w", tenantID, err)
	}
	defer rows.Close()

	timeline := make([]ThroughputTimeline, 0)
	peakRequestsPerHour := 0.0
	for rows.Next() {
		var entry ThroughputTimeline
		var period interface{}
		err := rows.Scan(&period, &entry.Requests, &entry.RequestsPerHour)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: failed to scan throughput timeline: %w", tenantID, err)
		}
		entry.Period = fmt.Sprint(period)
		timeline = append(timeline, entry)

		if entry.RequestsPerHour > peakRequestsPerHour {
			peakRequestsPerHour = entry.RequestsPerHour
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("tenant %s: throughput timeline iteration error: %w", tenantID, err)
	}

	return &ThroughputMetrics{
		TotalRequests:       totalRequests,
		RequestsPerHour:     requestsPerHour,
		PeakRequestsPerHour: peakRequestsPerHour,
		Timeline:            timeline,
	}, nil
}

// GetErrorMetrics retrieves error rate metrics
func (s *Service) GetErrorMetrics(ctx context.Context, tenantID uuid.UUID, params MetricsParams) (*ErrorMetrics, error) {
	// Total errors (verification failures) in the period
	var totalErrors, totalOps int64
	err := s.db.QueryRow(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE verified = false) as total_errors,
			COUNT(*) as total_ops
		FROM verifications
		WHERE tenant_id = $1
		  AND created_at BETWEEN $2 AND $3
	`, tenantID, params.StartDate, params.EndDate).Scan(&totalErrors, &totalOps)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to count errors: %w", tenantID, err)
	}

	errorRate := 0.0
	if totalOps > 0 {
		errorRate = float64(totalErrors) / float64(totalOps) * 100
	}

	// By type (for now just verification failures)
	byType := map[string]int64{
		"verification_failed": totalErrors,
	}

	// Timeline of errors
	rows, err := s.db.Query(ctx, `
		SELECT 
			date_trunc($1, created_at) as period,
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE verified = false) as errors,
			(COUNT(*) FILTER (WHERE verified = false))::float / NULLIF(COUNT(*), 0) * 100 as error_rate
		FROM verifications
		WHERE tenant_id = $2 
		  AND created_at BETWEEN $3 AND $4
		GROUP BY period
		ORDER BY period ASC
		LIMIT $5 OFFSET $6
	`, params.Interval, tenantID, params.StartDate, params.EndDate, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to query error timeline: %w", tenantID, err)
	}
	defer rows.Close()

	timeline := make([]ErrorTimeline, 0)
	for rows.Next() {
		var entry ErrorTimeline
		var period interface{}
		err := rows.Scan(&period, &entry.Total, &entry.Errors, &entry.ErrorRate)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: failed to scan error timeline: %w", tenantID, err)
		}
		entry.Period = fmt.Sprint(period)
		timeline = append(timeline, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("tenant %s: error timeline iteration error: %w", tenantID, err)
	}

	return &ErrorMetrics{
		TotalErrors: totalErrors,
		ErrorRate:   errorRate,
		ByType:      byType,
		Timeline:    timeline,
	}, nil
}

// GetQualityMetrics retrieves quality score metrics for face registrations
func (s *Service) GetQualityMetrics(ctx context.Context, tenantID uuid.UUID, params MetricsParams) (*QualityMetrics, error) {
	// Get overall quality statistics
	var avgQuality, minQuality, maxQuality float64
	err := s.db.QueryRow(ctx, `
		SELECT 
			AVG(quality_score) as avg_quality,
			MIN(quality_score) as min_quality,
			MAX(quality_score) as max_quality
		FROM faces
		WHERE tenant_id = $1
		  AND created_at BETWEEN $2 AND $3
	`, tenantID, params.StartDate, params.EndDate).Scan(&avgQuality, &minQuality, &maxQuality)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to calculate quality statistics: %w", tenantID, err)
	}

	// Timeline of quality metrics
	rows, err := s.db.Query(ctx, `
		SELECT 
			date_trunc($1, created_at) as period,
			AVG(quality_score) as avg_quality,
			MIN(quality_score) as min_quality,
			MAX(quality_score) as max_quality
		FROM faces
		WHERE tenant_id = $2 
		  AND created_at BETWEEN $3 AND $4
		GROUP BY period
		ORDER BY period ASC
		LIMIT $5 OFFSET $6
	`, params.Interval, tenantID, params.StartDate, params.EndDate, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to query quality timeline: %w", tenantID, err)
	}
	defer rows.Close()

	timeline := make([]QualityTimeline, 0)
	for rows.Next() {
		var entry QualityTimeline
		var period interface{}
		err := rows.Scan(&period, &entry.AverageQuality, &entry.MinQuality, &entry.MaxQuality)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: failed to scan quality timeline: %w", tenantID, err)
		}
		entry.Period = fmt.Sprint(period)
		timeline = append(timeline, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("tenant %s: quality timeline iteration error: %w", tenantID, err)
	}

	return &QualityMetrics{
		AverageQuality: avgQuality,
		MinQuality:     minQuality,
		MaxQuality:     maxQuality,
		Timeline:       timeline,
	}, nil
}

// GetConfidenceMetrics retrieves confidence score metrics for verifications
func (s *Service) GetConfidenceMetrics(ctx context.Context, tenantID uuid.UUID, params MetricsParams) (*ConfidenceMetrics, error) {
	// Get overall confidence statistics
	var avgConfidence, minConfidence, maxConfidence float64
	err := s.db.QueryRow(ctx, `
		SELECT 
			AVG(confidence) as avg_confidence,
			MIN(confidence) as min_confidence,
			MAX(confidence) as max_confidence
		FROM verifications
		WHERE tenant_id = $1
		  AND created_at BETWEEN $2 AND $3
	`, tenantID, params.StartDate, params.EndDate).Scan(&avgConfidence, &minConfidence, &maxConfidence)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to calculate confidence statistics: %w", tenantID, err)
	}

	// Timeline of confidence metrics
	rows, err := s.db.Query(ctx, `
		SELECT 
			date_trunc($1, created_at) as period,
			AVG(confidence) as avg_confidence,
			MIN(confidence) as min_confidence,
			MAX(confidence) as max_confidence
		FROM verifications
		WHERE tenant_id = $2 
		  AND created_at BETWEEN $3 AND $4
		GROUP BY period
		ORDER BY period ASC
		LIMIT $5 OFFSET $6
	`, params.Interval, tenantID, params.StartDate, params.EndDate, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to query confidence timeline: %w", tenantID, err)
	}
	defer rows.Close()

	timeline := make([]ConfidenceTimeline, 0)
	for rows.Next() {
		var entry ConfidenceTimeline
		var period interface{}
		err := rows.Scan(&period, &entry.AverageConfidence, &entry.MinConfidence, &entry.MaxConfidence)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: failed to scan confidence timeline: %w", tenantID, err)
		}
		entry.Period = fmt.Sprint(period)
		timeline = append(timeline, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("tenant %s: confidence timeline iteration error: %w", tenantID, err)
	}

	return &ConfidenceMetrics{
		AverageConfidence: avgConfidence,
		MinConfidence:     minConfidence,
		MaxConfidence:     maxConfidence,
		Timeline:          timeline,
	}, nil
}

// GetMatchMetrics retrieves matching statistics
func (s *Service) GetMatchMetrics(ctx context.Context, tenantID uuid.UUID, params MetricsParams) (*MatchMetrics, error) {
	// Get overall match statistics
	var totalMatches, totalVerifications int64
	var avgMatchScore float64
	err := s.db.QueryRow(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE verified = true) as total_matches,
			COUNT(*) as total_verifications,
			AVG(confidence) FILTER (WHERE verified = true) as avg_match_score
		FROM verifications
		WHERE tenant_id = $1
		  AND created_at BETWEEN $2 AND $3
	`, tenantID, params.StartDate, params.EndDate).Scan(&totalMatches, &totalVerifications, &avgMatchScore)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to calculate match statistics: %w", tenantID, err)
	}

	matchRate := 0.0
	if totalVerifications > 0 {
		matchRate = float64(totalMatches) / float64(totalVerifications) * 100
	}

	// Timeline of match metrics
	rows, err := s.db.Query(ctx, `
		SELECT 
			date_trunc($1, created_at) as period,
			COUNT(*) FILTER (WHERE verified = true) as matches,
			COUNT(*) as total,
			(COUNT(*) FILTER (WHERE verified = true))::float / NULLIF(COUNT(*), 0) * 100 as match_rate,
			AVG(confidence) FILTER (WHERE verified = true) as avg_match_score
		FROM verifications
		WHERE tenant_id = $2 
		  AND created_at BETWEEN $3 AND $4
		GROUP BY period
		ORDER BY period ASC
		LIMIT $5 OFFSET $6
	`, params.Interval, tenantID, params.StartDate, params.EndDate, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to query match timeline: %w", tenantID, err)
	}
	defer rows.Close()

	timeline := make([]MatchTimeline, 0)
	for rows.Next() {
		var entry MatchTimeline
		var period interface{}
		err := rows.Scan(&period, &entry.Matches, &entry.Total, &entry.MatchRate, &entry.AverageMatchScore)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: failed to scan match timeline: %w", tenantID, err)
		}
		entry.Period = fmt.Sprint(period)
		timeline = append(timeline, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("tenant %s: match timeline iteration error: %w", tenantID, err)
	}

	return &MatchMetrics{
		TotalMatches:      totalMatches,
		MatchRate:         matchRate,
		AverageMatchScore: avgMatchScore,
		Timeline:          timeline,
	}, nil
}
