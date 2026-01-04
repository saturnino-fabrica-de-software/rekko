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

func (r *Repository) IncrementDaily(ctx context.Context, tenantID uuid.UUID, date time.Time, field string, amount int) error {
	if field != "registrations" && field != "verifications" && field != "liveness_checks" {
		return fmt.Errorf("invalid field: %s", field)
	}

	query := fmt.Sprintf(`
		INSERT INTO usage_daily (tenant_id, date, %s)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, date)
		DO UPDATE SET %s = usage_daily.%s + EXCLUDED.%s, updated_at = NOW()
	`, field, field, field, field)

	_, err := r.pool.Exec(ctx, query, tenantID, date, amount)
	if err != nil {
		return fmt.Errorf("tenant %s: increment daily %s: %w", tenantID, field, err)
	}

	return nil
}
