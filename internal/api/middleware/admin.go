package middleware

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/admin"
	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

const (
	// LocalAdminUser is the key to retrieve admin user from context
	LocalAdminUser = "admin_user"
	// LocalAdminRole is the key to retrieve admin role from context
	LocalAdminRole = "admin_role"
)

// AdminLevel defines the level of admin access required
type AdminLevel string

const (
	// AdminLevelTenant requires tenant admin access (API key auth)
	AdminLevelTenant AdminLevel = "tenant_admin"
	// AdminLevelSuper requires super admin access (JWT auth)
	AdminLevelSuper AdminLevel = "super_admin"
)

// AdminAuthDependencies contains dependencies for admin authentication
type AdminAuthDependencies struct {
	JWTService *admin.JWTService
	Logger     *slog.Logger
}

// AdminAuth creates an authentication middleware for admin endpoints
// For tenant_admin: uses existing API key auth (must be chained after Auth middleware)
// For super_admin: uses JWT auth from Authorization header
func AdminAuth(requiredLevel AdminLevel, deps AdminAuthDependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch requiredLevel {
		case AdminLevelSuper:
			return validateSuperAdmin(c, deps)
		case AdminLevelTenant:
			return validateTenantAdmin(c, deps)
		default:
			deps.Logger.Error("invalid admin level", "level", requiredLevel)
			return domain.ErrUnauthorized
		}
	}
}

// validateSuperAdmin validates JWT token for super admin access
func validateSuperAdmin(c *fiber.Ctx, deps AdminAuthDependencies) error {
	// Extract JWT token from Authorization header
	token := extractBearerToken(c)
	if token == "" {
		deps.Logger.Debug("missing authorization header for super admin")
		return domain.ErrUnauthorized
	}

	// Validate JWT token
	claims, err := deps.JWTService.ValidateToken(token)
	if err != nil {
		deps.Logger.Warn("invalid JWT token", "error", err)
		return domain.ErrUnauthorized
	}

	// Verify role is super_admin
	if claims.Role != "super_admin" {
		deps.Logger.Warn("insufficient privileges", "role", claims.Role, "required", "super_admin")
		return domain.ErrForbidden
	}

	// Store admin info in context
	c.Locals(LocalAdminUser, claims.UserID)
	c.Locals(LocalAdminRole, claims.Role)

	deps.Logger.Debug("super admin authenticated",
		"user_id", claims.UserID,
		"email", claims.Email,
		"role", claims.Role,
	)

	return c.Next()
}

// validateTenantAdmin validates that the request is from a tenant admin
// Requires that Auth middleware has already been executed
func validateTenantAdmin(c *fiber.Ctx, deps AdminAuthDependencies) error {
	// Check if tenant context exists (set by Auth middleware)
	tenantID, ok := c.Locals(LocalTenantID).(uuid.UUID)
	if !ok || tenantID == uuid.Nil {
		deps.Logger.Debug("tenant admin auth requires prior API key authentication")
		return domain.ErrUnauthorized
	}

	// Get API key from context
	apiKey, ok := c.Locals(LocalAPIKey).(*domain.APIKey)
	if !ok {
		deps.Logger.Debug("API key not found in context")
		return domain.ErrUnauthorized
	}

	// Store admin info in context
	c.Locals(LocalAdminRole, "tenant_admin")

	deps.Logger.Debug("tenant admin authenticated",
		"tenant_id", tenantID,
		"api_key_prefix", apiKey.KeyPrefix,
	)

	return c.Next()
}

// GetAdminUserID retrieves admin user ID from context (for super admin)
func GetAdminUserID(c *fiber.Ctx) (uuid.UUID, error) {
	userID, ok := c.Locals(LocalAdminUser).(uuid.UUID)
	if !ok {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return userID, nil
}

// GetAdminRole retrieves admin role from context
func GetAdminRole(c *fiber.Ctx) (string, error) {
	role, ok := c.Locals(LocalAdminRole).(string)
	if !ok {
		return "", domain.ErrUnauthorized
	}
	return role, nil
}

// IsSuperAdmin checks if the current request is from a super admin
func IsSuperAdmin(c *fiber.Ctx) bool {
	role, ok := c.Locals(LocalAdminRole).(string)
	return ok && role == "super_admin"
}

// IsTenantAdmin checks if the current request is from a tenant admin
func IsTenantAdmin(c *fiber.Ctx) bool {
	role, ok := c.Locals(LocalAdminRole).(string)
	return ok && role == "tenant_admin"
}
