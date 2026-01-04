package metrics

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AggregationType defines how metrics are aggregated
type AggregationType string

const (
	AggregationSum   AggregationType = "sum"
	AggregationAvg   AggregationType = "avg"
	AggregationCount AggregationType = "count"
	AggregationP99   AggregationType = "p99"
	AggregationMin   AggregationType = "min"
	AggregationMax   AggregationType = "max"
)

// AggregatedMetric represents a pre-computed metric
type AggregatedMetric struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	MetricName      string
	MetricValue     float64
	AggregationType AggregationType
	PeriodStart     time.Time
	PeriodEnd       time.Time
	Metadata        map[string]interface{}
	CreatedAt       time.Time
}

// Repository handles database operations for metrics
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new metrics repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// SaveMetric stores an aggregated metric
func (r *Repository) SaveMetric(ctx context.Context, metric *AggregatedMetric) error {
	query := `
		INSERT INTO metrics_aggregated (
			tenant_id, metric_name, metric_value, aggregation_type,
			period_start, period_end, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, metric_name, aggregation_type, period_start)
		DO UPDATE SET
			metric_value = EXCLUDED.metric_value,
			period_end = EXCLUDED.period_end,
			metadata = EXCLUDED.metadata
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		metric.TenantID,
		metric.MetricName,
		metric.MetricValue,
		metric.AggregationType,
		metric.PeriodStart,
		metric.PeriodEnd,
		metric.Metadata,
	).Scan(&metric.ID, &metric.CreatedAt)

	return err
}

// GetMetricsByTenant retrieves metrics for a tenant within a time range
func (r *Repository) GetMetricsByTenant(
	ctx context.Context,
	tenantID uuid.UUID,
	metricName string,
	aggregationType AggregationType,
	start, end time.Time,
) ([]*AggregatedMetric, error) {
	query := `
		SELECT id, tenant_id, metric_name, metric_value, aggregation_type,
		       period_start, period_end, metadata, created_at
		FROM metrics_aggregated
		WHERE tenant_id = $1
		  AND metric_name = $2
		  AND aggregation_type = $3
		  AND period_start >= $4
		  AND period_end <= $5
		ORDER BY period_start ASC
	`

	rows, err := r.db.Query(ctx, query, tenantID, metricName, aggregationType, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*AggregatedMetric
	for rows.Next() {
		metric := &AggregatedMetric{}
		err := rows.Scan(
			&metric.ID,
			&metric.TenantID,
			&metric.MetricName,
			&metric.MetricValue,
			&metric.AggregationType,
			&metric.PeriodStart,
			&metric.PeriodEnd,
			&metric.Metadata,
			&metric.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}

	return metrics, rows.Err()
}

// GetAllTenantsMetrics retrieves metrics across all tenants (super admin)
func (r *Repository) GetAllTenantsMetrics(
	ctx context.Context,
	metricName string,
	aggregationType AggregationType,
	start, end time.Time,
) ([]*AggregatedMetric, error) {
	query := `
		SELECT id, tenant_id, metric_name, metric_value, aggregation_type,
		       period_start, period_end, metadata, created_at
		FROM metrics_aggregated
		WHERE metric_name = $1
		  AND aggregation_type = $2
		  AND period_start >= $3
		  AND period_end <= $4
		ORDER BY tenant_id, period_start ASC
	`

	rows, err := r.db.Query(ctx, query, metricName, aggregationType, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*AggregatedMetric
	for rows.Next() {
		metric := &AggregatedMetric{}
		err := rows.Scan(
			&metric.ID,
			&metric.TenantID,
			&metric.MetricName,
			&metric.MetricValue,
			&metric.AggregationType,
			&metric.PeriodStart,
			&metric.PeriodEnd,
			&metric.Metadata,
			&metric.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}

	return metrics, rows.Err()
}

// DeleteOldMetrics removes metrics older than the specified duration
func (r *Repository) DeleteOldMetrics(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM metrics_aggregated
		WHERE period_end < $1
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := r.db.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}
