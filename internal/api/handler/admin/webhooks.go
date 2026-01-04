package admin

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/saturnino-fabrica-de-software/rekko/internal/webhook"
)

type WebhooksHandler struct {
	service *webhook.Service
	logger  *slog.Logger
}

func NewWebhooksHandler(service *webhook.Service, logger *slog.Logger) *WebhooksHandler {
	return &WebhooksHandler{
		service: service,
		logger:  logger,
	}
}

type CreateWebhookRequest struct {
	Name    string   `json:"name" validate:"required,min=3,max=255"`
	URL     string   `json:"url" validate:"required,url,max=2048"`
	Events  []string `json:"events" validate:"required,min=1"`
	Enabled bool     `json:"enabled"`
}

type WebhookResponse struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	Events          []string  `json:"events"`
	Enabled         bool      `json:"enabled"`
	LastTriggeredAt *string   `json:"last_triggered_at,omitempty"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
}

func (h *WebhooksHandler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(uuid.UUID)

	webhooks, err := h.service.GetWebhooksByTenant(c.Context(), tenantID)
	if err != nil {
		h.logger.Error("failed to list webhooks", "tenant_id", tenantID, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list webhooks",
		})
	}

	response := make([]WebhookResponse, 0, len(webhooks))
	for _, w := range webhooks {
		var lastTriggered *string
		if w.LastTriggeredAt != nil {
			t := w.LastTriggeredAt.Format("2006-01-02T15:04:05Z07:00")
			lastTriggered = &t
		}

		response = append(response, WebhookResponse{
			ID:              w.ID,
			Name:            w.Name,
			URL:             w.URL,
			Events:          w.Events,
			Enabled:         w.Enabled,
			LastTriggeredAt: lastTriggered,
			CreatedAt:       w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:       w.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return c.JSON(fiber.Map{
		"webhooks": response,
	})
}

func (h *WebhooksHandler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(uuid.UUID)

	var req CreateWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	secret, err := generateSecret(32)
	if err != nil {
		h.logger.Error("failed to generate secret", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate webhook secret",
		})
	}

	w := &webhook.Webhook{
		TenantID: tenantID,
		Name:     req.Name,
		URL:      req.URL,
		Secret:   secret,
		Events:   req.Events,
		Enabled:  req.Enabled,
	}

	if err := h.service.CreateWebhook(c.Context(), w); err != nil {
		h.logger.Error("failed to create webhook", "tenant_id", tenantID, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create webhook",
		})
	}

	h.logger.Info("webhook created",
		"webhook_id", w.ID,
		"tenant_id", tenantID,
		"name", w.Name,
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"webhook": WebhookResponse{
			ID:        w.ID,
			Name:      w.Name,
			URL:       w.URL,
			Events:    w.Events,
			Enabled:   w.Enabled,
			CreatedAt: w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: w.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		"secret": secret,
	})
}

func (h *WebhooksHandler) Delete(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(uuid.UUID)

	webhookIDStr := c.Params("id")
	webhookID, err := uuid.Parse(webhookIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	if err := h.service.DeleteWebhook(c.Context(), tenantID, webhookID); err != nil {
		h.logger.Error("failed to delete webhook",
			"webhook_id", webhookID,
			"tenant_id", tenantID,
			"error", err,
		)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Webhook not found",
		})
	}

	h.logger.Info("webhook deleted",
		"webhook_id", webhookID,
		"tenant_id", tenantID,
	)

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
