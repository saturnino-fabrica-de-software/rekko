package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

const (
	// LocalTenantID is the key to retrieve tenant_id from context
	LocalTenantID = "tenant_id"
	// LocalTenant is the key to retrieve the full tenant from context
	LocalTenant = "tenant"
)

// TenantRepository interface for tenant lookup
type TenantRepository interface {
	GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*domain.Tenant, error)
}

// Auth creates an authentication middleware using API Key
func Auth(tenantRepo TenantRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Extract Bearer token
		apiKey := extractBearerToken(c)
		if apiKey == "" {
			return domain.ErrUnauthorized
		}

		// 2. Generate API Key hash
		hash := hashAPIKey(apiKey)

		// 3. Lookup tenant by hash
		tenant, err := tenantRepo.GetByAPIKeyHash(c.Context(), hash)
		if err != nil {
			// Any error (not found or DB error) returns 401
			// Don't reveal whether API Key exists or not
			return domain.ErrUnauthorized
		}

		// 4. Verify tenant is active
		if !tenant.IsActive {
			return domain.ErrUnauthorized
		}

		// 5. Set tenant in context
		c.Locals(LocalTenantID, tenant.ID)
		c.Locals(LocalTenant, tenant)

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

// hashAPIKey generates SHA-256 hash of API Key
func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
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
