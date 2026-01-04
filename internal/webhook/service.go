package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db     *pgxpool.Pool
	client *http.Client
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		db: db,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Service) Send(ctx context.Context, webhook *Webhook, event EventPayload) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	signature := Sign(webhook.Secret, payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Rekko-Signature", signature)
	req.Header.Set("X-Rekko-Event", event.Type)
	req.Header.Set("User-Agent", "Rekko-Webhook/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return s.enqueue(ctx, webhook.ID, event.Type, payload, err.Error())
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		return s.enqueue(ctx, webhook.ID, event.Type, payload, fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	return s.updateLastTriggered(ctx, webhook.ID)
}

func (s *Service) enqueue(ctx context.Context, webhookID uuid.UUID, eventType string, payload []byte, errorMsg string) error {
	query := `
		INSERT INTO webhook_queue (webhook_id, event_type, payload, next_retry_at, last_error)
		VALUES ($1, $2, $3, NOW() + INTERVAL '1 second', $4)
	`

	_, err := s.db.Exec(ctx, query, webhookID, eventType, payload, errorMsg)
	if err != nil {
		return fmt.Errorf("enqueue webhook: %w", err)
	}

	return nil
}

func (s *Service) updateLastTriggered(ctx context.Context, webhookID uuid.UUID) error {
	query := `UPDATE webhooks SET last_triggered_at = NOW() WHERE id = $1`
	_, err := s.db.Exec(ctx, query, webhookID)
	return err
}

func (s *Service) GetWebhooksByTenant(ctx context.Context, tenantID uuid.UUID) ([]*Webhook, error) {
	query := `
		SELECT id, tenant_id, name, url, secret, events, enabled, last_triggered_at, created_at, updated_at
		FROM webhooks
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*Webhook
	for rows.Next() {
		var w Webhook
		var eventsJSON []byte

		err := rows.Scan(
			&w.ID, &w.TenantID, &w.Name, &w.URL, &w.Secret,
			&eventsJSON, &w.Enabled, &w.LastTriggeredAt,
			&w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}

		if err := json.Unmarshal(eventsJSON, &w.Events); err != nil {
			return nil, fmt.Errorf("unmarshal events: %w", err)
		}

		webhooks = append(webhooks, &w)
	}

	return webhooks, nil
}

func (s *Service) GetWebhooksByEvent(ctx context.Context, tenantID uuid.UUID, eventType string) ([]*Webhook, error) {
	query := `
		SELECT id, tenant_id, name, url, secret, events, enabled, last_triggered_at, created_at, updated_at
		FROM webhooks
		WHERE tenant_id = $1 AND enabled = true AND events @> $2::jsonb
	`

	eventsJSON, _ := json.Marshal([]string{eventType})

	rows, err := s.db.Query(ctx, query, tenantID, eventsJSON)
	if err != nil {
		return nil, fmt.Errorf("query webhooks by event: %w", err)
	}
	defer rows.Close()

	var webhooks []*Webhook
	for rows.Next() {
		var w Webhook
		var eventsJSON []byte

		err := rows.Scan(
			&w.ID, &w.TenantID, &w.Name, &w.URL, &w.Secret,
			&eventsJSON, &w.Enabled, &w.LastTriggeredAt,
			&w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}

		if err := json.Unmarshal(eventsJSON, &w.Events); err != nil {
			return nil, fmt.Errorf("unmarshal events: %w", err)
		}

		webhooks = append(webhooks, &w)
	}

	return webhooks, nil
}

func (s *Service) CreateWebhook(ctx context.Context, webhook *Webhook) error {
	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}

	query := `
		INSERT INTO webhooks (tenant_id, name, url, secret, events, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRow(ctx, query,
		webhook.TenantID, webhook.Name, webhook.URL,
		webhook.Secret, eventsJSON, webhook.Enabled,
	).Scan(&webhook.ID, &webhook.CreatedAt, &webhook.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create webhook: %w", err)
	}

	return nil
}

func (s *Service) DeleteWebhook(ctx context.Context, tenantID, webhookID uuid.UUID) error {
	query := `DELETE FROM webhooks WHERE id = $1 AND tenant_id = $2`

	result, err := s.db.Exec(ctx, query, webhookID, tenantID)
	if err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("webhook not found")
	}

	return nil
}
