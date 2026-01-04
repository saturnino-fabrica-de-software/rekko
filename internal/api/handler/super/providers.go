package super

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/saturnino-fabrica-de-software/rekko/internal/admin"
)

type ProvidersHandler struct {
	adminService admin.SuperAdminService
	logger       *slog.Logger
}

func NewProvidersHandler(adminService admin.SuperAdminService, logger *slog.Logger) *ProvidersHandler {
	return &ProvidersHandler{
		adminService: adminService,
		logger:       logger,
	}
}

// GetProvidersStatus handles GET /super/providers
func (h *ProvidersHandler) GetProvidersStatus(c *fiber.Ctx) error {
	providers, err := h.adminService.GetProvidersStatus(c.Context())
	if err != nil {
		h.logger.Error("failed to get providers status", "error", err)
		return fiber.ErrInternalServerError
	}

	return c.JSON(fiber.Map{
		"data": providers,
	})
}
