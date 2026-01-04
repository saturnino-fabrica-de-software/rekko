package alert

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type Worker struct {
	repo     *Repository
	engine   *Engine
	notifier *Notifier
	logger   *slog.Logger
	interval time.Duration
	done     chan struct{}
}

func NewWorker(repo *Repository, engine *Engine, notifier *Notifier, logger *slog.Logger, interval time.Duration) *Worker {
	if interval == 0 {
		interval = 30 * time.Second
	}

	return &Worker{
		repo:     repo,
		engine:   engine,
		notifier: notifier,
		logger:   logger,
		interval: interval,
		done:     make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	defer close(w.done)

	w.logger.Info("alert worker started", "interval", w.interval)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("alert worker stopped")
			return
		case <-w.done:
			w.logger.Info("alert worker stopped")
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *Worker) Stop() {
	close(w.done)
}

func (w *Worker) process(ctx context.Context) {
	w.logger.Debug("processing alerts")

	alerts, err := w.repo.ListEnabled(ctx)
	if err != nil {
		w.logger.Error("failed to list enabled alerts", "error", err)
		return
	}

	w.logger.Debug("found enabled alerts", "count", len(alerts))

	for _, alert := range alerts {
		if err := w.evaluateAlert(ctx, alert); err != nil {
			w.logger.Error("failed to evaluate alert",
				"alert_id", alert.ID,
				"alert_name", alert.Name,
				"tenant_id", alert.TenantID,
				"error", err,
			)
		}
	}
}

func (w *Worker) evaluateAlert(ctx context.Context, alert *Alert) error {
	now := time.Now()

	if !w.engine.ShouldTrigger(alert, now) {
		w.logger.Debug("alert in cooldown",
			"alert_id", alert.ID,
			"alert_name", alert.Name,
			"last_triggered", alert.LastTriggeredAt,
		)
		return nil
	}

	triggered, metadata, err := w.engine.Evaluate(ctx, alert)
	if err != nil {
		return err
	}

	if !triggered {
		w.logger.Debug("alert conditions not met",
			"alert_id", alert.ID,
			"alert_name", alert.Name,
		)
		return nil
	}

	w.logger.Info("alert triggered",
		"alert_id", alert.ID,
		"alert_name", alert.Name,
		"tenant_id", alert.TenantID,
		"severity", alert.Severity,
	)

	history := &AlertHistory{
		ID:          uuid.New(),
		AlertID:     alert.ID,
		TenantID:    alert.TenantID,
		TriggeredAt: now,
		Status:      "triggered",
		Metadata:    metadata,
	}

	if err := w.repo.SaveHistory(ctx, history); err != nil {
		w.logger.Error("failed to save alert history",
			"alert_id", alert.ID,
			"error", err,
		)
	}

	if err := w.repo.UpdateLastTriggered(ctx, alert.ID); err != nil {
		w.logger.Error("failed to update last triggered",
			"alert_id", alert.ID,
			"error", err,
		)
	}

	if err := w.notifier.Send(ctx, alert, history); err != nil {
		w.logger.Error("failed to send notification",
			"alert_id", alert.ID,
			"error", err,
		)
	}

	return nil
}
