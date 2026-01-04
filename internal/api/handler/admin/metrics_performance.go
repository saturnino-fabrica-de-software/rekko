package admin

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/admin"
	"github.com/saturnino-fabrica-de-software/rekko/internal/api/middleware"
)

// MetricsPerformanceHandler handles performance metrics endpoints
type MetricsPerformanceHandler struct {
	adminService *admin.Service
	logger       *slog.Logger
}

// NewMetricsPerformanceHandler creates a new metrics performance handler
func NewMetricsPerformanceHandler(adminService *admin.Service, logger *slog.Logger) *MetricsPerformanceHandler {
	return &MetricsPerformanceHandler{
		adminService: adminService,
		logger:       logger,
	}
}

// GetLatencyMetrics handles GET /v1/admin/metrics/latency
func (h *MetricsPerformanceHandler) GetLatencyMetrics(c *fiber.Ctx) error {
	tenantID, ok := c.Locals(middleware.LocalTenantID).(uuid.UUID)
	if !ok {
		h.logger.Warn("tenant ID not found in context")
		return fiber.ErrUnauthorized
	}

	params, err := parseMetricsParams(c)
	if err != nil {
		h.logger.Debug("invalid metrics params", "error", err, "tenant_id", tenantID)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	metrics, err := h.adminService.GetLatencyMetrics(c.Context(), tenantID, params)
	if err != nil {
		h.logger.Error("failed to get latency metrics", "error", err, "tenant_id", tenantID)
		return fiber.ErrInternalServerError
	}

	return c.JSON(admin.MetricsResponse{
		Data: metrics,
		Meta: admin.ResponseMeta{
			TenantID:    tenantID.String(),
			Period:      admin.Period{Start: params.StartDate.Format("2006-01-02"), End: params.EndDate.Format("2006-01-02")},
			GeneratedAt: time.Now(),
		},
		Pagination: &admin.PaginationMeta{
			Total:  len(metrics.Timeline),
			Limit:  params.Limit,
			Offset: params.Offset,
		},
	})
}

// GetThroughputMetrics handles GET /v1/admin/metrics/throughput
func (h *MetricsPerformanceHandler) GetThroughputMetrics(c *fiber.Ctx) error {
	tenantID, ok := c.Locals(middleware.LocalTenantID).(uuid.UUID)
	if !ok {
		h.logger.Warn("tenant ID not found in context")
		return fiber.ErrUnauthorized
	}

	params, err := parseMetricsParams(c)
	if err != nil {
		h.logger.Debug("invalid metrics params", "error", err, "tenant_id", tenantID)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	metrics, err := h.adminService.GetThroughputMetrics(c.Context(), tenantID, params)
	if err != nil {
		h.logger.Error("failed to get throughput metrics", "error", err, "tenant_id", tenantID)
		return fiber.ErrInternalServerError
	}

	return c.JSON(admin.MetricsResponse{
		Data: metrics,
		Meta: admin.ResponseMeta{
			TenantID:    tenantID.String(),
			Period:      admin.Period{Start: params.StartDate.Format("2006-01-02"), End: params.EndDate.Format("2006-01-02")},
			GeneratedAt: time.Now(),
		},
		Pagination: &admin.PaginationMeta{
			Total:  len(metrics.Timeline),
			Limit:  params.Limit,
			Offset: params.Offset,
		},
	})
}

// GetErrorMetrics handles GET /v1/admin/metrics/errors
func (h *MetricsPerformanceHandler) GetErrorMetrics(c *fiber.Ctx) error {
	tenantID, ok := c.Locals(middleware.LocalTenantID).(uuid.UUID)
	if !ok {
		h.logger.Warn("tenant ID not found in context")
		return fiber.ErrUnauthorized
	}

	params, err := parseMetricsParams(c)
	if err != nil {
		h.logger.Debug("invalid metrics params", "error", err, "tenant_id", tenantID)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	metrics, err := h.adminService.GetErrorMetrics(c.Context(), tenantID, params)
	if err != nil {
		h.logger.Error("failed to get error metrics", "error", err, "tenant_id", tenantID)
		return fiber.ErrInternalServerError
	}

	return c.JSON(admin.MetricsResponse{
		Data: metrics,
		Meta: admin.ResponseMeta{
			TenantID:    tenantID.String(),
			Period:      admin.Period{Start: params.StartDate.Format("2006-01-02"), End: params.EndDate.Format("2006-01-02")},
			GeneratedAt: time.Now(),
		},
		Pagination: &admin.PaginationMeta{
			Total:  len(metrics.Timeline),
			Limit:  params.Limit,
			Offset: params.Offset,
		},
	})
}

// parseMetricsParams parses and validates query parameters (shared helper)
func parseMetricsParams(c *fiber.Ctx) (admin.MetricsParams, error) {
	startDate := c.Query("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDate := c.Query("end_date", time.Now().Format("2006-01-02"))
	const defaultInterval = "day"
	interval := c.Query("interval", defaultInterval)
	limit := c.QueryInt("limit", 100)
	offset := c.QueryInt("offset", 0)

	// Validate interval
	if interval != "hour" && interval != defaultInterval && interval != "week" && interval != "month" {
		interval = defaultInterval
	}

	// Cap limit
	if limit > 1000 {
		limit = 1000
	}
	if limit < 0 {
		limit = 100
	}

	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	// Parse dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return admin.MetricsParams{}, fiber.NewError(fiber.StatusBadRequest, "invalid start_date format, expected YYYY-MM-DD")
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return admin.MetricsParams{}, fiber.NewError(fiber.StatusBadRequest, "invalid end_date format, expected YYYY-MM-DD")
	}

	// Validate date range
	if start.After(end) {
		return admin.MetricsParams{}, fiber.NewError(fiber.StatusBadRequest, "start_date must be before or equal to end_date")
	}

	return admin.MetricsParams{
		StartDate: start,
		EndDate:   end,
		Interval:  interval,
		Limit:     limit,
		Offset:    offset,
	}, nil
}
