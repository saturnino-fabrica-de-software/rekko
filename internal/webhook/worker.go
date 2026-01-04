package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Worker struct {
	db      *pgxpool.Pool
	service *Service
	logger  *slog.Logger
	stopCh  chan struct{}
}

func NewWorker(db *pgxpool.Pool, service *Service, logger *slog.Logger) *Worker {
	return &Worker{
		db:      db,
		service: service,
		logger:  logger,
		stopCh:  make(chan struct{}),
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	w.logger.Info("webhook worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("webhook worker stopped")
			return
		case <-w.stopCh:
			w.logger.Info("webhook worker stopped")
			return
		case <-ticker.C:
			if err := w.processQueue(ctx); err != nil {
				w.logger.Error("failed to process webhook queue", "error", err)
			}
		}
	}
}

func (w *Worker) Stop() {
	close(w.stopCh)
}

func (w *Worker) processQueue(ctx context.Context) error {
	query := `
		SELECT id, webhook_id, event_type, payload, attempts, max_attempts
		FROM webhook_queue
		WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 10
	`

	rows, err := w.db.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("query webhook queue: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var job WebhookJob

		err := rows.Scan(
			&job.ID, &job.WebhookID, &job.EventType,
			&job.Payload, &job.Attempts, &job.MaxAttempts,
		)
		if err != nil {
			w.logger.Error("failed to scan webhook job", "error", err)
			continue
		}

		if err := w.processJob(ctx, &job); err != nil {
			w.logger.Error("failed to process webhook job",
				"job_id", job.ID,
				"webhook_id", job.WebhookID,
				"attempts", job.Attempts,
				"error", err,
			)
		}
	}

	return nil
}

func (w *Worker) processJob(ctx context.Context, job *WebhookJob) error {
	webhook, err := w.getWebhook(ctx, job.WebhookID)
	if err != nil {
		return w.markFailed(ctx, job.ID, fmt.Sprintf("webhook not found: %v", err))
	}

	if !webhook.Enabled {
		return w.markFailed(ctx, job.ID, "webhook disabled")
	}

	var event EventPayload
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return w.markFailed(ctx, job.ID, fmt.Sprintf("invalid payload: %v", err))
	}

	if err := w.service.Send(ctx, webhook, event); err != nil {
		return w.scheduleRetry(ctx, job, err.Error())
	}

	return w.markComplete(ctx, job.ID)
}

func (w *Worker) getWebhook(ctx context.Context, webhookID uuid.UUID) (*Webhook, error) {
	query := `
		SELECT id, tenant_id, name, url, secret, events, enabled, last_triggered_at, created_at, updated_at
		FROM webhooks
		WHERE id = $1
	`

	var webhook Webhook
	var eventsJSON []byte

	err := w.db.QueryRow(ctx, query, webhookID).Scan(
		&webhook.ID, &webhook.TenantID, &webhook.Name, &webhook.URL, &webhook.Secret,
		&eventsJSON, &webhook.Enabled, &webhook.LastTriggeredAt,
		&webhook.CreatedAt, &webhook.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(eventsJSON, &webhook.Events); err != nil {
		return nil, err
	}

	return &webhook, nil
}

func (w *Worker) scheduleRetry(ctx context.Context, job *WebhookJob, errorMsg string) error {
	if job.Attempts >= job.MaxAttempts {
		return w.markFailed(ctx, job.ID, errorMsg)
	}

	delay := time.Duration(1<<job.Attempts) * time.Second
	nextRetry := time.Now().Add(delay)

	query := `
		UPDATE webhook_queue
		SET attempts = attempts + 1,
		    next_retry_at = $1,
		    last_error = $2,
		    status = 'pending',
		    updated_at = NOW()
		WHERE id = $3
	`

	_, err := w.db.Exec(ctx, query, nextRetry, errorMsg, job.ID)
	if err != nil {
		return fmt.Errorf("schedule retry: %w", err)
	}

	w.logger.Info("webhook job scheduled for retry",
		"job_id", job.ID,
		"attempts", job.Attempts+1,
		"next_retry", nextRetry,
	)

	return nil
}

func (w *Worker) markComplete(ctx context.Context, jobID uuid.UUID) error {
	query := `
		UPDATE webhook_queue
		SET status = 'delivered',
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := w.db.Exec(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("mark complete: %w", err)
	}

	w.logger.Info("webhook job completed", "job_id", jobID)
	return nil
}

func (w *Worker) markFailed(ctx context.Context, jobID uuid.UUID, errorMsg string) error {
	query := `
		UPDATE webhook_queue
		SET status = 'failed',
		    last_error = $1,
		    updated_at = NOW()
		WHERE id = $2
	`

	_, err := w.db.Exec(ctx, query, errorMsg, jobID)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}

	w.logger.Warn("webhook job failed", "job_id", jobID, "error", errorMsg)
	return nil
}
