package alert

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, a *Alert) error {
	conditions, err := json.Marshal(a.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}

	channels, err := json.Marshal(a.Channels)
	if err != nil {
		return fmt.Errorf("marshal channels: %w", err)
	}

	query := `
		INSERT INTO alerts (
			tenant_id, name, conditions, condition_logic, window_seconds,
			cooldown_seconds, severity, channels, enabled
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	err = r.db.QueryRow(ctx, query,
		a.TenantID, a.Name, conditions, a.ConditionLogic, a.WindowSeconds,
		a.CooldownSeconds, a.Severity, channels, a.Enabled,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create alert: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, tenantID, alertID uuid.UUID) (*Alert, error) {
	query := `
		SELECT id, tenant_id, name, conditions, condition_logic, window_seconds,
		       cooldown_seconds, severity, channels, enabled, last_triggered_at,
		       created_at, updated_at
		FROM alerts
		WHERE id = $1 AND tenant_id = $2
	`

	var a Alert
	var conditions, channels []byte

	err := r.db.QueryRow(ctx, query, alertID, tenantID).Scan(
		&a.ID, &a.TenantID, &a.Name, &conditions, &a.ConditionLogic,
		&a.WindowSeconds, &a.CooldownSeconds, &a.Severity, &channels,
		&a.Enabled, &a.LastTriggeredAt, &a.CreatedAt, &a.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get alert: %w", err)
	}

	if err := json.Unmarshal(conditions, &a.Conditions); err != nil {
		return nil, fmt.Errorf("unmarshal conditions: %w", err)
	}

	if err := json.Unmarshal(channels, &a.Channels); err != nil {
		return nil, fmt.Errorf("unmarshal channels: %w", err)
	}

	return &a, nil
}

func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*Alert, error) {
	query := `
		SELECT id, tenant_id, name, conditions, condition_logic, window_seconds,
		       cooldown_seconds, severity, channels, enabled, last_triggered_at,
		       created_at, updated_at
		FROM alerts
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*Alert
	for rows.Next() {
		var a Alert
		var conditions, channels []byte

		err := rows.Scan(
			&a.ID, &a.TenantID, &a.Name, &conditions, &a.ConditionLogic,
			&a.WindowSeconds, &a.CooldownSeconds, &a.Severity, &channels,
			&a.Enabled, &a.LastTriggeredAt, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}

		if err := json.Unmarshal(conditions, &a.Conditions); err != nil {
			return nil, fmt.Errorf("unmarshal conditions: %w", err)
		}

		if err := json.Unmarshal(channels, &a.Channels); err != nil {
			return nil, fmt.Errorf("unmarshal channels: %w", err)
		}

		alerts = append(alerts, &a)
	}

	return alerts, rows.Err()
}

func (r *Repository) Update(ctx context.Context, a *Alert) error {
	conditions, err := json.Marshal(a.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}

	channels, err := json.Marshal(a.Channels)
	if err != nil {
		return fmt.Errorf("marshal channels: %w", err)
	}

	query := `
		UPDATE alerts
		SET name = $3, conditions = $4, condition_logic = $5, window_seconds = $6,
		    cooldown_seconds = $7, severity = $8, channels = $9, enabled = $10,
		    updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
		RETURNING updated_at
	`

	err = r.db.QueryRow(ctx, query,
		a.ID, a.TenantID, a.Name, conditions, a.ConditionLogic,
		a.WindowSeconds, a.CooldownSeconds, a.Severity, channels, a.Enabled,
	).Scan(&a.UpdatedAt)

	if err != nil {
		return fmt.Errorf("update alert: %w", err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, tenantID, alertID uuid.UUID) error {
	query := `DELETE FROM alerts WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.Exec(ctx, query, alertID, tenantID)
	if err != nil {
		return fmt.Errorf("delete alert: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("alert not found")
	}

	return nil
}

func (r *Repository) ListEnabled(ctx context.Context) ([]*Alert, error) {
	query := `
		SELECT id, tenant_id, name, conditions, condition_logic, window_seconds,
		       cooldown_seconds, severity, channels, last_triggered_at
		FROM alerts
		WHERE enabled = true
		ORDER BY tenant_id, name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list enabled alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*Alert
	for rows.Next() {
		var a Alert
		var conditions, channels []byte

		err := rows.Scan(
			&a.ID, &a.TenantID, &a.Name, &conditions, &a.ConditionLogic,
			&a.WindowSeconds, &a.CooldownSeconds, &a.Severity, &channels,
			&a.LastTriggeredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}

		if err := json.Unmarshal(conditions, &a.Conditions); err != nil {
			return nil, fmt.Errorf("unmarshal conditions: %w", err)
		}

		if err := json.Unmarshal(channels, &a.Channels); err != nil {
			return nil, fmt.Errorf("unmarshal channels: %w", err)
		}

		alerts = append(alerts, &a)
	}

	return alerts, rows.Err()
}

func (r *Repository) UpdateLastTriggered(ctx context.Context, alertID uuid.UUID) error {
	query := `UPDATE alerts SET last_triggered_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, alertID)
	return err
}

func (r *Repository) SaveHistory(ctx context.Context, h *AlertHistory) error {
	metadata, err := json.Marshal(h.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO alert_history (
			alert_id, tenant_id, triggered_at, status, metadata
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err = r.db.QueryRow(ctx, query,
		h.AlertID, h.TenantID, h.TriggeredAt, h.Status, metadata,
	).Scan(&h.ID, &h.CreatedAt)

	if err != nil {
		return fmt.Errorf("save alert history: %w", err)
	}

	return nil
}

func (r *Repository) ListHistory(ctx context.Context, tenantID, alertID uuid.UUID, limit int) ([]*AlertHistory, error) {
	query := `
		SELECT id, alert_id, tenant_id, triggered_at, resolved_at, status, metadata, created_at
		FROM alert_history
		WHERE tenant_id = $1 AND alert_id = $2
		ORDER BY triggered_at DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, tenantID, alertID, limit)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()

	var history []*AlertHistory
	for rows.Next() {
		var h AlertHistory
		var metadata []byte

		err := rows.Scan(
			&h.ID, &h.AlertID, &h.TenantID, &h.TriggeredAt,
			&h.ResolvedAt, &h.Status, &metadata, &h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}

		if err := json.Unmarshal(metadata, &h.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}

		history = append(history, &h)
	}

	return history, rows.Err()
}
