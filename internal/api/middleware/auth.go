package middleware

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
	"github.com/saturnino-fabrica-de-software/rekko/internal/repository"
)

const (
	// LocalTenantID is the key to retrieve tenant_id from context
	LocalTenantID = "tenant_id"
	// LocalTenant is the key to retrieve the full tenant from context
	LocalTenant = "tenant"
	// LocalAPIKey is the key to retrieve the API key entity from context
	LocalAPIKey = "api_key"
)

// AuthDependencies contains dependencies for authentication middleware
type AuthDependencies struct {
	TenantRepo     repository.TenantRepositoryInterface
	APIKeyRepo     repository.APIKeyRepositoryInterface
	Logger         *slog.Logger
	LastUsedWorker *LastUsedWorker // Optional: if nil, last_used updates are skipped
}

// Auth creates an authentication middleware using API Key
func Auth(deps AuthDependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Extract Bearer token
		apiKey := extractBearerToken(c)
		if apiKey == "" {
			deps.Logger.Debug("missing authorization header")
			return domain.ErrUnauthorized
		}

		// 2. Validate format
		if !domain.IsValidFormat(apiKey) {
			deps.Logger.Warn("invalid api key format", "prefix", extractPrefix(apiKey))
			return domain.ErrInvalidAPIKeyFormat
		}

		// 3. Hash and lookup
		hash := domain.HashAPIKey(apiKey)

		// 4. Get API Key from repository
		apiKeyEntity, err := deps.APIKeyRepo.GetByHash(c.Context(), hash)
		if err != nil {
			deps.Logger.Warn("api key not found", "error", err)
			return domain.ErrUnauthorized
		}

		// 5. Check if API key is active
		if !apiKeyEntity.IsActive {
			deps.Logger.Warn("api key is inactive", "key_id", apiKeyEntity.ID, "key_prefix", apiKeyEntity.KeyPrefix)
			return domain.ErrAPIKeyRevoked
		}

		// 6. Get tenant
		tenant, err := deps.TenantRepo.GetByID(c.Context(), apiKeyEntity.TenantID)
		if err != nil {
			deps.Logger.Warn("tenant not found", "tenant_id", apiKeyEntity.TenantID, "error", err)
			return domain.ErrUnauthorized
		}

		// 7. Check tenant is active
		if !tenant.IsActive {
			deps.Logger.Warn("tenant is inactive", "tenant_id", tenant.ID, "tenant_slug", tenant.Slug)
			return domain.ErrTenantInactive
		}

		// 8. Store in context
		c.Locals(LocalTenantID, tenant.ID)
		c.Locals(LocalTenant, tenant)
		c.Locals(LocalAPIKey, apiKeyEntity)

		deps.Logger.Debug("authenticated",
			"tenant_id", tenant.ID,
			"tenant_slug", tenant.Slug,
			"api_key_prefix", apiKeyEntity.KeyPrefix,
			"environment", apiKeyEntity.Environment,
		)

		// 9. Update last used in background (non-blocking, after all validations passed)
		if deps.LastUsedWorker != nil {
			deps.LastUsedWorker.Enqueue(apiKeyEntity.ID)
		}

		return c.Next()
	}
}

// extractBearerToken extracts token from Authorization header
func extractBearerToken(c *fiber.Ctx) string {
	auth := c.Get("Authorization")
	if auth == "" {
		return ""
	}

	// Expected format: "Bearer <token>"
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

// extractPrefix safely extracts first 16 characters for logging
func extractPrefix(apiKey string) string {
	if len(apiKey) >= 16 {
		return apiKey[:16]
	}
	return apiKey
}

// GetTenantID retrieves tenant_id from Fiber context
func GetTenantID(c *fiber.Ctx) (uuid.UUID, error) {
	tenantID, ok := c.Locals(LocalTenantID).(uuid.UUID)
	if !ok {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return tenantID, nil
}

// GetTenant retrieves full tenant from Fiber context
func GetTenant(c *fiber.Ctx) (*domain.Tenant, error) {
	tenant, ok := c.Locals(LocalTenant).(*domain.Tenant)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	return tenant, nil
}

// GetAPIKey retrieves API key entity from Fiber context
func GetAPIKey(c *fiber.Ctx) (*domain.APIKey, error) {
	apiKey, ok := c.Locals(LocalAPIKey).(*domain.APIKey)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	return apiKey, nil
}
