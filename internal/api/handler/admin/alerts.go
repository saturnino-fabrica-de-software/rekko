package admin

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/saturnino-fabrica-de-software/rekko/internal/alert"
)

type AlertsHandler struct {
	repo   *alert.Repository
	logger *slog.Logger
}

func NewAlertsHandler(repo *alert.Repository, logger *slog.Logger) *AlertsHandler {
	return &AlertsHandler{
		repo:   repo,
		logger: logger,
	}
}

type CreateAlertRequest struct {
	Name            string            `json:"name" validate:"required,min=3,max=255"`
	Conditions      []alert.Condition `json:"conditions" validate:"required,min=1"`
	ConditionLogic  string            `json:"condition_logic" validate:"required,oneof=AND OR"`
	WindowSeconds   int               `json:"window_seconds" validate:"required,min=60,max=86400"`
	CooldownSeconds int               `json:"cooldown_seconds" validate:"required,min=60,max=86400"`
	Severity        alert.Severity    `json:"severity" validate:"required,oneof=info warning critical"`
	Channels        []alert.Channel   `json:"channels" validate:"required,min=1"`
	Enabled         bool              `json:"enabled"`
}

type UpdateAlertRequest struct {
	Name            *string           `json:"name,omitempty" validate:"omitempty,min=3,max=255"`
	Conditions      []alert.Condition `json:"conditions,omitempty" validate:"omitempty,min=1"`
	ConditionLogic  *string           `json:"condition_logic,omitempty" validate:"omitempty,oneof=AND OR"`
	WindowSeconds   *int              `json:"window_seconds,omitempty" validate:"omitempty,min=60,max=86400"`
	CooldownSeconds *int              `json:"cooldown_seconds,omitempty" validate:"omitempty,min=60,max=86400"`
	Severity        *alert.Severity   `json:"severity,omitempty" validate:"omitempty,oneof=info warning critical"`
	Channels        []alert.Channel   `json:"channels,omitempty" validate:"omitempty,min=1"`
	Enabled         *bool             `json:"enabled,omitempty"`
}

type AlertResponse struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Conditions      []alert.Condition `json:"conditions"`
	ConditionLogic  string            `json:"condition_logic"`
	WindowSeconds   int               `json:"window_seconds"`
	CooldownSeconds int               `json:"cooldown_seconds"`
	Severity        alert.Severity    `json:"severity"`
	Channels        []alert.Channel   `json:"channels"`
	Enabled         bool              `json:"enabled"`
	LastTriggeredAt *string           `json:"last_triggered_at,omitempty"`
	CreatedAt       string            `json:"created_at"`
	UpdatedAt       string            `json:"updated_at"`
}

func (h *AlertsHandler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(uuid.UUID)

	alerts, err := h.repo.ListByTenant(c.Context(), tenantID)
	if err != nil {
		h.logger.Error("failed to list alerts", "tenant_id", tenantID, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list alerts",
		})
	}

	if alerts == nil {
		alerts = []*alert.Alert{}
	}

	response := make([]AlertResponse, 0, len(alerts))
	for _, a := range alerts {
		var lastTriggered *string
		if a.LastTriggeredAt != nil {
			t := a.LastTriggeredAt.Format("2006-01-02T15:04:05Z07:00")
			lastTriggered = &t
		}

		response = append(response, AlertResponse{
			ID:              a.ID,
			Name:            a.Name,
			Conditions:      a.Conditions,
			ConditionLogic:  a.ConditionLogic,
			WindowSeconds:   a.WindowSeconds,
			CooldownSeconds: a.CooldownSeconds,
			Severity:        a.Severity,
			Channels:        a.Channels,
			Enabled:         a.Enabled,
			LastTriggeredAt: lastTriggered,
			CreatedAt:       a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:       a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return c.JSON(fiber.Map{
		"alerts": response,
	})
}

func (h *AlertsHandler) Get(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(uuid.UUID)

	alertIDStr := c.Params("id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid alert ID",
		})
	}

	a, err := h.repo.GetByID(c.Context(), tenantID, alertID)
	if err != nil {
		h.logger.Error("failed to get alert",
			"alert_id", alertID,
			"tenant_id", tenantID,
			"error", err,
		)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Alert not found",
		})
	}

	var lastTriggered *string
	if a.LastTriggeredAt != nil {
		t := a.LastTriggeredAt.Format("2006-01-02T15:04:05Z07:00")
		lastTriggered = &t
	}

	return c.JSON(fiber.Map{
		"alert": AlertResponse{
			ID:              a.ID,
			Name:            a.Name,
			Conditions:      a.Conditions,
			ConditionLogic:  a.ConditionLogic,
			WindowSeconds:   a.WindowSeconds,
			CooldownSeconds: a.CooldownSeconds,
			Severity:        a.Severity,
			Channels:        a.Channels,
			Enabled:         a.Enabled,
			LastTriggeredAt: lastTriggered,
			CreatedAt:       a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:       a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
}

func (h *AlertsHandler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(uuid.UUID)

	var req CreateAlertRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	a := &alert.Alert{
		TenantID:        tenantID,
		Name:            req.Name,
		Conditions:      req.Conditions,
		ConditionLogic:  req.ConditionLogic,
		WindowSeconds:   req.WindowSeconds,
		CooldownSeconds: req.CooldownSeconds,
		Severity:        req.Severity,
		Channels:        req.Channels,
		Enabled:         req.Enabled,
	}

	if err := h.repo.Create(c.Context(), a); err != nil {
		h.logger.Error("failed to create alert", "tenant_id", tenantID, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create alert",
		})
	}

	h.logger.Info("alert created",
		"alert_id", a.ID,
		"tenant_id", tenantID,
		"name", a.Name,
		"severity", a.Severity,
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"alert": AlertResponse{
			ID:              a.ID,
			Name:            a.Name,
			Conditions:      a.Conditions,
			ConditionLogic:  a.ConditionLogic,
			WindowSeconds:   a.WindowSeconds,
			CooldownSeconds: a.CooldownSeconds,
			Severity:        a.Severity,
			Channels:        a.Channels,
			Enabled:         a.Enabled,
			CreatedAt:       a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:       a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
}

func (h *AlertsHandler) Update(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(uuid.UUID)

	alertIDStr := c.Params("id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid alert ID",
		})
	}

	var req UpdateAlertRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	a, err := h.repo.GetByID(c.Context(), tenantID, alertID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Alert not found",
		})
	}

	if req.Name != nil {
		a.Name = *req.Name
	}
	if req.Conditions != nil {
		a.Conditions = req.Conditions
	}
	if req.ConditionLogic != nil {
		a.ConditionLogic = *req.ConditionLogic
	}
	if req.WindowSeconds != nil {
		a.WindowSeconds = *req.WindowSeconds
	}
	if req.CooldownSeconds != nil {
		a.CooldownSeconds = *req.CooldownSeconds
	}
	if req.Severity != nil {
		a.Severity = *req.Severity
	}
	if req.Channels != nil {
		a.Channels = req.Channels
	}
	if req.Enabled != nil {
		a.Enabled = *req.Enabled
	}

	if err := h.repo.Update(c.Context(), a); err != nil {
		h.logger.Error("failed to update alert",
			"alert_id", alertID,
			"tenant_id", tenantID,
			"error", err,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update alert",
		})
	}

	h.logger.Info("alert updated",
		"alert_id", a.ID,
		"tenant_id", tenantID,
	)

	var lastTriggered *string
	if a.LastTriggeredAt != nil {
		t := a.LastTriggeredAt.Format("2006-01-02T15:04:05Z07:00")
		lastTriggered = &t
	}

	return c.JSON(fiber.Map{
		"alert": AlertResponse{
			ID:              a.ID,
			Name:            a.Name,
			Conditions:      a.Conditions,
			ConditionLogic:  a.ConditionLogic,
			WindowSeconds:   a.WindowSeconds,
			CooldownSeconds: a.CooldownSeconds,
			Severity:        a.Severity,
			Channels:        a.Channels,
			Enabled:         a.Enabled,
			LastTriggeredAt: lastTriggered,
			CreatedAt:       a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:       a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
}

func (h *AlertsHandler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(uuid.UUID)

	alertIDStr := c.Params("id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid alert ID",
		})
	}

	if err := h.repo.Delete(c.Context(), tenantID, alertID); err != nil {
		h.logger.Error("failed to delete alert",
			"alert_id", alertID,
			"tenant_id", tenantID,
			"error", err,
		)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Alert not found",
		})
	}

	h.logger.Info("alert deleted",
		"alert_id", alertID,
		"tenant_id", tenantID,
	)

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *AlertsHandler) ListHistory(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(uuid.UUID)

	alertIDStr := c.Params("id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid alert ID",
		})
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l := c.QueryInt("limit", 50); l > 0 && l <= 100 {
			limit = l
		}
	}

	history, err := h.repo.ListHistory(c.Context(), tenantID, alertID, limit)
	if err != nil {
		h.logger.Error("failed to list alert history",
			"alert_id", alertID,
			"tenant_id", tenantID,
			"error", err,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list alert history",
		})
	}

	type HistoryResponse struct {
		ID          uuid.UUID              `json:"id"`
		TriggeredAt string                 `json:"triggered_at"`
		ResolvedAt  *string                `json:"resolved_at,omitempty"`
		Status      string                 `json:"status"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if history == nil {
		history = []*alert.AlertHistory{}
	}

	response := make([]HistoryResponse, 0, len(history))
	for _, h := range history {
		var resolvedAt *string
		if h.ResolvedAt != nil {
			t := h.ResolvedAt.Format("2006-01-02T15:04:05Z07:00")
			resolvedAt = &t
		}

		response = append(response, HistoryResponse{
			ID:          h.ID,
			TriggeredAt: h.TriggeredAt.Format("2006-01-02T15:04:05Z07:00"),
			ResolvedAt:  resolvedAt,
			Status:      h.Status,
			Metadata:    h.Metadata,
		})
	}

	return c.JSON(fiber.Map{
		"history": response,
		"count":   len(response),
	})
}
