package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saturnino-fabrica-de-software/rekko/internal/admin"
	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

func newTestApp(logger *slog.Logger) *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler(logger),
	})
}

func TestAdminAuth_SuperAdmin_Success(t *testing.T) {
	jwtService := admin.NewJWTService("test-secret", "rekko-test", 1*time.Hour)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	userID := uuid.New()
	token, err := jwtService.GenerateToken(userID, "admin@rekko.com", "super_admin")
	require.NoError(t, err)

	app := newTestApp(logger)
	app.Use(AdminAuth(AdminLevelSuper, AdminAuthDependencies{
		JWTService: jwtService,
		Logger:     logger,
	}))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAdminAuth_SuperAdmin_MissingToken(t *testing.T) {
	jwtService := admin.NewJWTService("test-secret", "rekko-test", 1*time.Hour)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	app := newTestApp(logger)
	app.Use(AdminAuth(AdminLevelSuper, AdminAuthDependencies{
		JWTService: jwtService,
		Logger:     logger,
	}))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdminAuth_SuperAdmin_InvalidToken(t *testing.T) {
	jwtService := admin.NewJWTService("test-secret", "rekko-test", 1*time.Hour)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	app := newTestApp(logger)
	app.Use(AdminAuth(AdminLevelSuper, AdminAuthDependencies{
		JWTService: jwtService,
		Logger:     logger,
	}))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdminAuth_SuperAdmin_InsufficientPrivileges(t *testing.T) {
	jwtService := admin.NewJWTService("test-secret", "rekko-test", 1*time.Hour)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	userID := uuid.New()
	// Generate token with tenant_admin role
	token, err := jwtService.GenerateToken(userID, "admin@rekko.com", "tenant_admin")
	require.NoError(t, err)

	app := newTestApp(logger)
	app.Use(AdminAuth(AdminLevelSuper, AdminAuthDependencies{
		JWTService: jwtService,
		Logger:     logger,
	}))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestAdminAuth_TenantAdmin_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tenantID := uuid.New()
	apiKey := &domain.APIKey{
		ID:        uuid.New(),
		TenantID:  tenantID,
		KeyPrefix: "rekko_test_abc1",
		IsActive:  true,
	}

	app := newTestApp(logger)
	// Simulate Auth middleware setting context
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(LocalTenantID, tenantID)
		c.Locals(LocalAPIKey, apiKey)
		return c.Next()
	})
	app.Use(AdminAuth(AdminLevelTenant, AdminAuthDependencies{
		Logger: logger,
	}))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAdminAuth_TenantAdmin_MissingContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	app := newTestApp(logger)
	app.Use(AdminAuth(AdminLevelTenant, AdminAuthDependencies{
		Logger: logger,
	}))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGetAdminUserID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	app := newTestApp(logger)
	userID := uuid.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals(LocalAdminUser, userID)
		id, err := GetAdminUserID(c)
		require.NoError(t, err)
		assert.Equal(t, userID, id)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	_, err := app.Test(req)
	require.NoError(t, err)
}

func TestGetAdminRole(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	app := newTestApp(logger)

	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals(LocalAdminRole, "super_admin")
		role, err := GetAdminRole(c)
		require.NoError(t, err)
		assert.Equal(t, "super_admin", role)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	_, err := app.Test(req)
	require.NoError(t, err)
}

func TestIsSuperAdmin(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{"super admin", "super_admin", true},
		{"tenant admin", "tenant_admin", false},
		{"no role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			app := newTestApp(logger)
			app.Get("/test", func(c *fiber.Ctx) error {
				if tt.role != "" {
					c.Locals(LocalAdminRole, tt.role)
				}
				result := IsSuperAdmin(c)
				assert.Equal(t, tt.expected, result)
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			_, err := app.Test(req)
			require.NoError(t, err)
		})
	}
}

func TestIsTenantAdmin(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{"tenant admin", "tenant_admin", true},
		{"super admin", "super_admin", false},
		{"no role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			app := newTestApp(logger)
			app.Get("/test", func(c *fiber.Ctx) error {
				if tt.role != "" {
					c.Locals(LocalAdminRole, tt.role)
				}
				result := IsTenantAdmin(c)
				assert.Equal(t, tt.expected, result)
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			_, err := app.Test(req)
			require.NoError(t, err)
		})
	}
}
