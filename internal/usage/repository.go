package usage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetPlanByID(ctx context.Context, planID string) (*Plan, error) {
	query := `
		SELECT id, name, monthly_price, quota_registrations, quota_verifications, overage_price, created_at, updated_at
		FROM plans
		WHERE id = $1
	`

	var plan Plan
	err := r.pool.QueryRow(ctx, query, planID).Scan(
		&plan.ID,
		&plan.Name,
		&plan.MonthlyPrice,
		&plan.QuotaRegistrations,
		&plan.QuotaVerifications,
		&plan.OveragePrice,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("plan not found: %s", planID)
	}
	if err != nil {
		return nil, fmt.Errorf("get plan by id: %w", err)
	}

	return &plan, nil
}

func (r *Repository) GetPlanWithOverrides(ctx context.Context, tenantID uuid.UUID, planID string) (*Plan, error) {
	plan, err := r.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT quota_registrations, quota_verifications, overage_price
		FROM tenant_plan_overrides
		WHERE tenant_id = $1
	`

	var overrideReg, overrideVer *int
	var overridePrice *float64

	err = r.pool.QueryRow(ctx, query, tenantID).Scan(&overrideReg, &overrideVer, &overridePrice)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("tenant %s: get plan overrides: %w", tenantID, err)
	}

	if overrideReg != nil {
		plan.QuotaRegistrations = *overrideReg
	}
	if overrideVer != nil {
		plan.QuotaVerifications = *overrideVer
	}
	if overridePrice != nil {
		plan.OveragePrice = *overridePrice
	}

	return plan, nil
}

func (r *Repository) GetDailyUsage(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) ([]UsageRecord, error) {
	query := `
		SELECT id, tenant_id, date, registrations, verifications, liveness_checks, created_at, updated_at
		FROM usage_daily
		WHERE tenant_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: get daily usage: %w", tenantID, err)
	}
	defer rows.Close()

	var records []UsageRecord
	for rows.Next() {
		var record UsageRecord
		err := rows.Scan(
			&record.ID,
			&record.TenantID,
			&record.Date,
			&record.Registrations,
			&record.Verifications,
			&record.LivenessChecks,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: scan usage record: %w", tenantID, err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tenant %s: iterate usage records: %w", tenantID, err)
	}

	return records, nil
}

func (r *Repository) AggregatePeriod(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*UsageRecord, error) {
	query := `
		SELECT 
			COALESCE(SUM(registrations), 0) as total_registrations,
			COALESCE(SUM(verifications), 0) as total_verifications,
			COALESCE(SUM(liveness_checks), 0) as total_liveness_checks
		FROM usage_daily
		WHERE tenant_id = $1 AND date >= $2 AND date <= $3
	`

	var record UsageRecord
	record.TenantID = tenantID
	record.Date = startDate

	err := r.pool.QueryRow(ctx, query, tenantID, startDate, endDate).Scan(
		&record.Registrations,
		&record.Verifications,
		&record.LivenessChecks,
	)

	if err != nil {
		return nil, fmt.Errorf("tenant %s: aggregate period: %w", tenantID, err)
	}

	return &record, nil
}

// Pre-built queries to avoid SQL injection via fmt.Sprintf
var incrementQueries = map[string]string{
	"registrations": `
		INSERT INTO usage_daily (tenant_id, date, registrations)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, date)
		DO UPDATE SET registrations = usage_daily.registrations + EXCLUDED.registrations, updated_at = NOW()
	`,
	"verifications": `
		INSERT INTO usage_daily (tenant_id, date, verifications)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, date)
		DO UPDATE SET verifications = usage_daily.verifications + EXCLUDED.verifications, updated_at = NOW()
	`,
	"liveness_checks": `
		INSERT INTO usage_daily (tenant_id, date, liveness_checks)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, date)
		DO UPDATE SET liveness_checks = usage_daily.liveness_checks + EXCLUDED.liveness_checks, updated_at = NOW()
	`,
}

func (r *Repository) IncrementDaily(ctx context.Context, tenantID uuid.UUID, date time.Time, field string, amount int) error {
	query, ok := incrementQueries[field]
	if !ok {
		return fmt.Errorf("invalid field: %s", field)
	}

	_, err := r.pool.Exec(ctx, query, tenantID, date, amount)
	if err != nil {
		return fmt.Errorf("tenant %s: increment daily %s: %w", tenantID, field, err)
	}

	return nil
}

// GetActiveTenantsWithPlan returns all active tenants with their plan IDs for quota checking
func (r *Repository) GetActiveTenantsWithPlan(ctx context.Context) ([]TenantPlan, error) {
	query := `
		SELECT t.id, COALESCE(t.plan_id, 'starter') as plan_id
		FROM tenants t
		WHERE t.is_active = true
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get active tenants: %w", err)
	}
	defer rows.Close()

	var tenants []TenantPlan
	for rows.Next() {
		var tp TenantPlan
		if err := rows.Scan(&tp.TenantID, &tp.PlanID); err != nil {
			return nil, fmt.Errorf("scan tenant plan: %w", err)
		}
		tenants = append(tenants, tp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenants: %w", err)
	}

	return tenants, nil
}
