package alert

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/saturnino-fabrica-de-software/rekko/internal/webhook"
)

type Notifier struct {
	webhookService *webhook.Service
	logger         *slog.Logger
}

func NewNotifier(webhookService *webhook.Service, logger *slog.Logger) *Notifier {
	return &Notifier{
		webhookService: webhookService,
		logger:         logger,
	}
}

func (n *Notifier) Send(ctx context.Context, alert *Alert, history *AlertHistory) error {
	var errors []error

	for _, channel := range alert.Channels {
		if err := n.sendToChannel(ctx, channel, alert, history); err != nil {
			n.logger.Error("failed to send to channel",
				"channel_type", channel.Type,
				"alert_id", alert.ID,
				"error", err,
			)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send %d/%d notifications", len(errors), len(alert.Channels))
	}

	return nil
}

func (n *Notifier) sendToChannel(ctx context.Context, channel Channel, alert *Alert, history *AlertHistory) error {
	switch channel.Type {
	case "webhook":
		return n.sendWebhook(ctx, channel.WebhookID, alert, history)
	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

func (n *Notifier) sendWebhook(ctx context.Context, webhookID uuid.UUID, alert *Alert, history *AlertHistory) error {
	webhooks, err := n.webhookService.GetWebhooksByTenant(ctx, alert.TenantID)
	if err != nil {
		return fmt.Errorf("get webhooks: %w", err)
	}

	var targetWebhook *webhook.Webhook
	for _, w := range webhooks {
		if w.ID == webhookID && w.Enabled {
			targetWebhook = w
			break
		}
	}

	if targetWebhook == nil {
		return fmt.Errorf("webhook %s not found or disabled", webhookID)
	}

	payload := webhook.EventPayload{
		Type:     "alert.triggered",
		TenantID: alert.TenantID,
		Data: map[string]interface{}{
			"alert": map[string]interface{}{
				"id":       alert.ID,
				"name":     alert.Name,
				"severity": alert.Severity,
			},
			"history": map[string]interface{}{
				"id":           history.ID,
				"triggered_at": history.TriggeredAt,
				"metadata":     history.Metadata,
			},
		},
	}

	if err := n.webhookService.Send(ctx, targetWebhook, payload); err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}

	n.logger.Info("alert notification sent",
		"alert_id", alert.ID,
		"webhook_id", webhookID,
		"tenant_id", alert.TenantID,
	)

	return nil
}
