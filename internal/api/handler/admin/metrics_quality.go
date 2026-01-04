package admin

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/admin"
	"github.com/saturnino-fabrica-de-software/rekko/internal/api/middleware"
)

// MetricsQualityHandler handles quality metrics endpoints
type MetricsQualityHandler struct {
	adminService *admin.Service
	logger       *slog.Logger
}

// NewMetricsQualityHandler creates a new metrics quality handler
func NewMetricsQualityHandler(adminService *admin.Service, logger *slog.Logger) *MetricsQualityHandler {
	return &MetricsQualityHandler{
		adminService: adminService,
		logger:       logger,
	}
}

// GetQualityMetrics handles GET /v1/admin/metrics/quality
func (h *MetricsQualityHandler) GetQualityMetrics(c *fiber.Ctx) error {
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

	metrics, err := h.adminService.GetQualityMetrics(c.Context(), tenantID, params)
	if err != nil {
		h.logger.Error("failed to get quality metrics", "error", err, "tenant_id", tenantID)
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

// GetConfidenceMetrics handles GET /v1/admin/metrics/confidence
func (h *MetricsQualityHandler) GetConfidenceMetrics(c *fiber.Ctx) error {
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

	metrics, err := h.adminService.GetConfidenceMetrics(c.Context(), tenantID, params)
	if err != nil {
		h.logger.Error("failed to get confidence metrics", "error", err, "tenant_id", tenantID)
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

// GetMatchMetrics handles GET /v1/admin/metrics/matches
func (h *MetricsQualityHandler) GetMatchMetrics(c *fiber.Ctx) error {
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

	metrics, err := h.adminService.GetMatchMetrics(c.Context(), tenantID, params)
	if err != nil {
		h.logger.Error("failed to get match metrics", "error", err, "tenant_id", tenantID)
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
