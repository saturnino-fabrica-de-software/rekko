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

func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT id, name, slug, is_active, plan, settings, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	var tenant domain.Tenant
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.IsActive,
		&tenant.Plan,
		&tenant.Settings,
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

func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	query := `
		SELECT id, name, slug, is_active, plan, settings, created_at, updated_at
		FROM tenants
		WHERE slug = $1
	`

	var tenant domain.Tenant
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.IsActive,
		&tenant.Plan,
		&tenant.Settings,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by slug: %w", err)
	}

	return &tenant, nil
}

func (r *TenantRepository) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*domain.Tenant, error) {
	query := `
		SELECT t.id, t.name, t.slug, t.is_active, t.plan, t.settings, t.created_at, t.updated_at
		FROM tenants t
		INNER JOIN api_keys ak ON ak.tenant_id = t.id
		WHERE ak.key_hash = $1 AND ak.is_active = true AND t.is_active = true
	`

	var tenant domain.Tenant
	err := r.pool.QueryRow(ctx, query, apiKeyHash).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.IsActive,
		&tenant.Plan,
		&tenant.Settings,
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

func (r *TenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		INSERT INTO tenants (id, name, slug, is_active, plan, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	if tenant.ID == uuid.Nil {
		tenant.ID = uuid.New()
	}

	if tenant.Settings == nil {
		tenant.Settings = make(map[string]interface{})
	}

	err := r.pool.QueryRow(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		tenant.IsActive,
		tenant.Plan,
		tenant.Settings,
	).Scan(&tenant.CreatedAt, &tenant.UpdatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return &domain.AppError{
				Code:       "TENANT_ALREADY_EXISTS",
				Message:    "Tenant with this slug already exists",
				StatusCode: 409,
			}
		}
		return fmt.Errorf("create tenant: %w", err)
	}

	return nil
}

func (r *TenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $2, slug = $3, is_active = $4, plan = $5, settings = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	if tenant.Settings == nil {
		tenant.Settings = make(map[string]interface{})
	}

	err := r.pool.QueryRow(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		tenant.IsActive,
		tenant.Plan,
		tenant.Settings,
	).Scan(&tenant.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrTenantNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return &domain.AppError{
				Code:       "TENANT_SLUG_CONFLICT",
				Message:    "Tenant with this slug already exists",
				StatusCode: 409,
			}
		}
		return fmt.Errorf("update tenant: %w", err)
	}

	return nil
}

func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM tenants
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrTenantNotFound
	}

	return nil
}
