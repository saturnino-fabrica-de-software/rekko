package handler

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/api/middleware"
	"github.com/saturnino-fabrica-de-software/rekko/internal/usage"
)

type UsageService interface {
	GetCurrentUsage(ctx context.Context, tenantID uuid.UUID, planID string) (*usage.UsageSummary, error)
	GetUsageForPeriod(ctx context.Context, tenantID uuid.UUID, planID, period string) (*usage.UsageSummary, error)
}

type UsageHandler struct {
	service UsageService
	logger  *slog.Logger
}

func NewUsageHandler(service UsageService, logger *slog.Logger) *UsageHandler {
	return &UsageHandler{
		service: service,
		logger:  logger,
	}
}

func (h *UsageHandler) GetUsage(c *fiber.Ctx) error {
	tenant, err := middleware.GetTenant(c)
	if err != nil {
		return err
	}

	period := strings.TrimSpace(c.Query("period"))

	var summary *usage.UsageSummary
	if period == "" {
		summary, err = h.service.GetCurrentUsage(c.Context(), tenant.ID, tenant.Plan)
	} else {
		summary, err = h.service.GetUsageForPeriod(c.Context(), tenant.ID, tenant.Plan, period)
	}

	if err != nil {
		h.logger.Error("failed to get usage",
			"error", err,
			"tenant_id", tenant.ID,
			"period", period,
		)
		if strings.Contains(err.Error(), "invalid period format") {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid period format, use YYYY-MM")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve usage data")
	}

	return c.JSON(summary)
}
