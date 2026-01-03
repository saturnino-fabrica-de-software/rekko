package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

type APIKeyRepository struct {
	pool PgxPool
}

func NewAPIKeyRepository(pool PgxPool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

func (r *APIKeyRepository) Create(ctx context.Context, key *domain.APIKey) error {
	query := `
		INSERT INTO api_keys (id, tenant_id, name, key_hash, key_prefix, environment, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING created_at
	`

	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		key.ID,
		key.TenantID,
		key.Name,
		key.KeyHash,
		key.KeyPrefix,
		key.Environment,
		key.IsActive,
	).Scan(&key.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return &domain.AppError{
				Code:       "API_KEY_ALREADY_EXISTS",
				Message:    "API key with this hash already exists",
				StatusCode: 409,
			}
		}
		return fmt.Errorf("create api key: %w", err)
	}

	return nil
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, environment, is_active, last_used_at, created_at
		FROM api_keys
		WHERE key_hash = $1
	`

	var key domain.APIKey
	err := r.pool.QueryRow(ctx, query, hash).Scan(
		&key.ID,
		&key.TenantID,
		&key.Name,
		&key.KeyHash,
		&key.KeyPrefix,
		&key.Environment,
		&key.IsActive,
		&key.LastUsedAt,
		&key.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}

	return &key, nil
}

func (r *APIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, environment, is_active, last_used_at, created_at
		FROM api_keys
		WHERE id = $1
	`

	var key domain.APIKey
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&key.ID,
		&key.TenantID,
		&key.Name,
		&key.KeyHash,
		&key.KeyPrefix,
		&key.Environment,
		&key.IsActive,
		&key.LastUsedAt,
		&key.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get api key by id: %w", err)
	}

	return &key, nil
}

func (r *APIKeyRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, environment, is_active, last_used_at, created_at
		FROM api_keys
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list api keys by tenant: %w", err)
	}
	defer rows.Close()

	var keys []domain.APIKey
	for rows.Next() {
		var key domain.APIKey
		err := rows.Scan(
			&key.ID,
			&key.TenantID,
			&key.Name,
			&key.KeyHash,
			&key.KeyPrefix,
			&key.Environment,
			&key.IsActive,
			&key.LastUsedAt,
			&key.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return keys, nil
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE api_keys
		SET last_used_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("update last used: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrAPIKeyNotFound
	}

	return nil
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE api_keys
		SET is_active = false
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrAPIKeyNotFound
	}

	return nil
}

func (r *APIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM api_keys
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrAPIKeyNotFound
	}

	return nil
}
