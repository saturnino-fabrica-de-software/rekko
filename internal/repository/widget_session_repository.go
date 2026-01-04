package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

type WidgetSessionRepository struct {
	pool PgxPool
}

func NewWidgetSessionRepository(pool PgxPool) *WidgetSessionRepository {
	return &WidgetSessionRepository{pool: pool}
}

// Create creates a new widget session
func (r *WidgetSessionRepository) Create(ctx context.Context, session *domain.WidgetSession) error {
	query := `
		INSERT INTO widget_sessions (id, tenant_id, origin, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING created_at
	`

	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		session.ID,
		session.TenantID,
		session.Origin,
		session.ExpiresAt,
	).Scan(&session.CreatedAt)

	if err != nil {
		return fmt.Errorf("create widget session: %w", err)
	}

	return nil
}

// GetByID retrieves a widget session by ID
func (r *WidgetSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.WidgetSession, error) {
	query := `
		SELECT id, tenant_id, origin, expires_at, created_at
		FROM widget_sessions
		WHERE id = $1
	`

	var session domain.WidgetSession
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.TenantID,
		&session.Origin,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWidgetSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get widget session by id: %w", err)
	}

	return &session, nil
}

// DeleteExpired removes all expired sessions
// Returns the number of deleted sessions
func (r *WidgetSessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM widget_sessions
		WHERE expires_at < NOW()
	`

	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete expired widget sessions: %w", err)
	}

	return result.RowsAffected(), nil
}

// Delete removes a specific session by ID
func (r *WidgetSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM widget_sessions
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete widget session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrWidgetSessionNotFound
	}

	return nil
}
