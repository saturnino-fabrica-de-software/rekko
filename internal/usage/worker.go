package usage

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// TenantGetter interface to get active tenants with their plan
type TenantGetter interface {
	GetActiveTenantsWithPlan(ctx context.Context) ([]TenantPlan, error)
}

// TenantPlan represents a tenant with its plan ID
type TenantPlan struct {
	TenantID uuid.UUID
	PlanID   string
}

// Worker checks quotas periodically and sends alerts
type Worker struct {
	service      *Service
	tenantGetter TenantGetter
	logger       *slog.Logger
	interval     time.Duration
}

// NewWorker creates a new quota check worker
func NewWorker(service *Service, tenantGetter TenantGetter, logger *slog.Logger, interval time.Duration) *Worker {
	return &Worker{
		service:      service,
		tenantGetter: tenantGetter,
		logger:       logger,
		interval:     interval,
	}
}

// Run starts the worker loop
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Info("quota check worker started", "interval", w.interval)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("quota check worker stopped")
			return
		case <-ticker.C:
			w.checkAllTenants(ctx)
		}
	}
}

func (w *Worker) checkAllTenants(ctx context.Context) {
	tenants, err := w.tenantGetter.GetActiveTenantsWithPlan(ctx)
	if err != nil {
		w.logger.Error("failed to get active tenants", "error", err)
		return
	}

	for _, tenant := range tenants {
		if err := w.service.CheckQuota(ctx, tenant.TenantID, tenant.PlanID); err != nil {
			w.logger.Warn("failed to check quota for tenant",
				"error", err,
				"tenant_id", tenant.TenantID,
			)
		}
	}

	w.logger.Debug("quota check completed", "tenants_checked", len(tenants))
}
