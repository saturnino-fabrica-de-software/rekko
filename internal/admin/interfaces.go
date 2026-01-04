package admin

import (
	"context"

	"github.com/google/uuid"
)

// SuperAdminService defines the interface for super admin operations
type SuperAdminService interface {
	// Tenant operations
	ListAllTenants(ctx context.Context, limit, offset int) ([]TenantWithMetrics, error)
	GetTenantDetailedMetrics(ctx context.Context, tenantID uuid.UUID) (*TenantMetricsSummary, error)
	UpdateTenantQuota(ctx context.Context, tenantID uuid.UUID, req UpdateQuotaRequest) error

	// System operations
	GetSystemHealth(ctx context.Context) (*SystemHealth, error)
	GetSystemMetrics(ctx context.Context) (*SystemMetrics, error)

	// Provider operations
	GetProvidersStatus(ctx context.Context) ([]ProviderHealth, error)
}
