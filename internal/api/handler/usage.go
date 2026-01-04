package handler

import (
	"context"
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
}

func NewUsageHandler(service UsageService) *UsageHandler {
	return &UsageHandler{service: service}
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
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(summary)
}
