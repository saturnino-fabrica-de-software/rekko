package super

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/saturnino-fabrica-de-software/rekko/internal/admin"
)

type SystemHandler struct {
	adminService admin.SuperAdminService
	logger       *slog.Logger
}

func NewSystemHandler(adminService admin.SuperAdminService, logger *slog.Logger) *SystemHandler {
	return &SystemHandler{
		adminService: adminService,
		logger:       logger,
	}
}

// GetSystemHealth handles GET /super/system/health
func (h *SystemHandler) GetSystemHealth(c *fiber.Ctx) error {
	health, err := h.adminService.GetSystemHealth(c.Context())
	if err != nil {
		h.logger.Error("failed to get system health", "error", err)
		return fiber.ErrInternalServerError
	}

	statusCode := fiber.StatusOK
	switch health.Status {
	case "unhealthy":
		statusCode = fiber.StatusServiceUnavailable
	case "degraded", "healthy":
		statusCode = fiber.StatusOK
	}

	return c.Status(statusCode).JSON(health)
}

// GetSystemMetrics handles GET /super/system/metrics
func (h *SystemHandler) GetSystemMetrics(c *fiber.Ctx) error {
	metrics, err := h.adminService.GetSystemMetrics(c.Context())
	if err != nil {
		h.logger.Error("failed to get system metrics", "error", err)
		return fiber.ErrInternalServerError
	}

	return c.JSON(fiber.Map{
		"data": metrics,
	})
}
