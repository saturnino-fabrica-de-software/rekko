package super

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/admin"
)

type TenantsHandler struct {
	adminService admin.SuperAdminService
	logger       *slog.Logger
}

func NewTenantsHandler(adminService admin.SuperAdminService, logger *slog.Logger) *TenantsHandler {
	return &TenantsHandler{
		adminService: adminService,
		logger:       logger,
	}
}

// ListTenants handles GET /super/tenants
func (h *TenantsHandler) ListTenants(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	if limit > 100 {
		limit = 100
	}
	if limit < 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	tenants, err := h.adminService.ListAllTenants(c.Context(), limit, offset)
	if err != nil {
		h.logger.Error("failed to list tenants", "error", err)
		return fiber.ErrInternalServerError
	}

	return c.JSON(fiber.Map{
		"data": tenants,
		"meta": fiber.Map{
			"total":  len(tenants),
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetTenantMetrics handles GET /super/tenants/:id/metrics
func (h *TenantsHandler) GetTenantMetrics(c *fiber.Ctx) error {
	tenantIDStr := c.Params("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.logger.Debug("invalid tenant id", "id", tenantIDStr)
		return fiber.NewError(fiber.StatusBadRequest, "invalid tenant ID format")
	}

	metrics, err := h.adminService.GetTenantDetailedMetrics(c.Context(), tenantID)
	if err != nil {
		h.logger.Error("failed to get tenant metrics", "error", err, "tenant_id", tenantID)
		return fiber.ErrInternalServerError
	}

	return c.JSON(fiber.Map{
		"data": metrics,
		"meta": fiber.Map{
			"tenant_id": tenantID.String(),
		},
	})
}

// UpdateTenantQuota handles POST /super/tenants/:id/quota
func (h *TenantsHandler) UpdateTenantQuota(c *fiber.Ctx) error {
	tenantIDStr := c.Params("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.logger.Debug("invalid tenant id", "id", tenantIDStr)
		return fiber.NewError(fiber.StatusBadRequest, "invalid tenant ID format")
	}

	var req admin.UpdateQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Debug("invalid request body", "error", err)
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.adminService.UpdateTenantQuota(c.Context(), tenantID, req); err != nil {
		h.logger.Error("failed to update tenant quota", "error", err, "tenant_id", tenantID)
		return fiber.ErrInternalServerError
	}

	return c.JSON(fiber.Map{
		"message":   "quota updated successfully",
		"tenant_id": tenantID.String(),
	})
}
