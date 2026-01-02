package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

type TenantRepository struct {
	pool PgxPool
}

func NewTenantRepository(pool PgxPool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

func (r *TenantRepository) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*domain.Tenant, error) {
	query := `
		SELECT id, name, api_key_hash, settings, is_active, created_at, updated_at
		FROM tenants
		WHERE api_key_hash = $1 AND is_active = true
	`

	var tenant domain.Tenant
	err := r.pool.QueryRow(ctx, query, apiKeyHash).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.APIKeyHash,
		&tenant.Settings,
		&tenant.IsActive,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by api key: %w", err)
	}

	return &tenant, nil
}

func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT id, name, api_key_hash, settings, is_active, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	var tenant domain.Tenant
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.APIKeyHash,
		&tenant.Settings,
		&tenant.IsActive,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}

	return &tenant, nil
}
