package admin

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/admin"
	"github.com/saturnino-fabrica-de-software/rekko/internal/api/middleware"
)

// MetricsUsageHandler handles metrics usage endpoints
type MetricsUsageHandler struct {
	adminService *admin.Service
	logger       *slog.Logger
}

// NewMetricsUsageHandler creates a new metrics usage handler
func NewMetricsUsageHandler(adminService *admin.Service, logger *slog.Logger) *MetricsUsageHandler {
	return &MetricsUsageHandler{
		adminService: adminService,
		logger:       logger,
	}
}

// GetFacesMetrics handles GET /v1/admin/metrics/faces
func (h *MetricsUsageHandler) GetFacesMetrics(c *fiber.Ctx) error {
	tenantID, ok := c.Locals(middleware.LocalTenantID).(uuid.UUID)
	if !ok {
		h.logger.Warn("tenant ID not found in context")
		return fiber.ErrUnauthorized
	}

	params, err := h.parseMetricsParams(c)
	if err != nil {
		h.logger.Debug("invalid metrics params", "error", err, "tenant_id", tenantID)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	metrics, err := h.adminService.GetFacesMetrics(c.Context(), tenantID, params)
	if err != nil {
		h.logger.Error("failed to get faces metrics", "error", err, "tenant_id", tenantID)
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

// GetOperationsMetrics handles GET /v1/admin/metrics/operations
func (h *MetricsUsageHandler) GetOperationsMetrics(c *fiber.Ctx) error {
	tenantID, ok := c.Locals(middleware.LocalTenantID).(uuid.UUID)
	if !ok {
		h.logger.Warn("tenant ID not found in context")
		return fiber.ErrUnauthorized
	}

	params, err := h.parseMetricsParams(c)
	if err != nil {
		h.logger.Debug("invalid metrics params", "error", err, "tenant_id", tenantID)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	metrics, err := h.adminService.GetOperationsMetrics(c.Context(), tenantID, params)
	if err != nil {
		h.logger.Error("failed to get operations metrics", "error", err, "tenant_id", tenantID)
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

// GetRequestsMetrics handles GET /v1/admin/metrics/requests
func (h *MetricsUsageHandler) GetRequestsMetrics(c *fiber.Ctx) error {
	tenantID, ok := c.Locals(middleware.LocalTenantID).(uuid.UUID)
	if !ok {
		h.logger.Warn("tenant ID not found in context")
		return fiber.ErrUnauthorized
	}

	params, err := h.parseMetricsParams(c)
	if err != nil {
		h.logger.Debug("invalid metrics params", "error", err, "tenant_id", tenantID)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	metrics, err := h.adminService.GetRequestsMetrics(c.Context(), tenantID, params)
	if err != nil {
		h.logger.Error("failed to get requests metrics", "error", err, "tenant_id", tenantID)
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

// parseMetricsParams parses and validates query parameters
func (h *MetricsUsageHandler) parseMetricsParams(c *fiber.Ctx) (admin.MetricsParams, error) {
	startDate := c.Query("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDate := c.Query("end_date", time.Now().Format("2006-01-02"))
	interval := c.Query("interval", "day")
	limit := c.QueryInt("limit", 100)
	offset := c.QueryInt("offset", 0)

	// Validate interval
	if interval != "hour" && interval != "day" && interval != "week" && interval != "month" {
		interval = "day"
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
