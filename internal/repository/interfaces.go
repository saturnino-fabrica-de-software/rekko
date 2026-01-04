package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

// TenantRepositoryInterface defines operations for tenant data access
type TenantRepositoryInterface interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	GetByAPIKeyHash(ctx context.Context, hash string) (*domain.Tenant, error)
	Create(ctx context.Context, tenant *domain.Tenant) error
	Update(ctx context.Context, tenant *domain.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// APIKeyRepositoryInterface defines operations for API key data access
type APIKeyRepositoryInterface interface {
	Create(ctx context.Context, key *domain.APIKey) error
	GetByHash(ctx context.Context, hash string) (*domain.APIKey, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.APIKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// FaceRepositoryInterface defines operations for face data access
type FaceRepositoryInterface interface {
	Create(ctx context.Context, face *domain.Face) error
	GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Face, error)
	Delete(ctx context.Context, tenantID uuid.UUID, externalID string) error
	SearchByEmbedding(ctx context.Context, tenantID uuid.UUID, embedding []float64, threshold float64, limit int) ([]domain.SearchMatch, error)
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error)
}

// SearchAuditRepositoryInterface defines operations for search audit logging
type SearchAuditRepositoryInterface interface {
	Create(ctx context.Context, audit *domain.SearchAudit) error
}
