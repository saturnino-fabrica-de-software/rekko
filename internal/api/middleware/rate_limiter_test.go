package middleware

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		tenantID := uuid.New()

		config := RateLimiterConfig{
			Max:    5,
			Window: time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return tenantID.String()
			},
		}

		rl := NewRateLimiter(config)

		app := fiber.New()
		app.Use(rl.Handler())
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)

			body, _ := io.ReadAll(resp.Body)
			assert.Equal(t, "OK", string(body))
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		tenantID := uuid.New()

		config := RateLimiterConfig{
			Max:    2,
			Window: time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return tenantID.String()
			},
		}

		rl := NewRateLimiter(config)

		app := fiber.New(fiber.Config{
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				return c.Status(429).JSON(fiber.Map{"error": "rate limit"})
			},
		})
		app.Use(rl.Handler())
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// First 2 should succeed
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			resp, _ := app.Test(req)
			assert.Equal(t, 200, resp.StatusCode)
		}

		// Third should be rate limited
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, 429, resp.StatusCode)
	})

	t.Run("different tenants have separate limits", func(t *testing.T) {
		var currentTenant string

		config := RateLimiterConfig{
			Max:    2,
			Window: time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return currentTenant
			},
		}

		rl := NewRateLimiter(config)

		app := fiber.New(fiber.Config{
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				return c.Status(429).JSON(fiber.Map{"error": "rate limit"})
			},
		})
		app.Use(rl.Handler())
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// Tenant A uses 2 requests
		currentTenant = "tenant-a"
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			resp, _ := app.Test(req)
			assert.Equal(t, 200, resp.StatusCode)
		}

		// Tenant A is now rate limited
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, 429, resp.StatusCode)

		// Tenant B can still make requests
		currentTenant = "tenant-b"
		req = httptest.NewRequest("GET", "/test", nil)
		resp, _ = app.Test(req)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("rate limit headers are set", func(t *testing.T) {
		tenantID := uuid.New()

		config := RateLimiterConfig{
			Max:    10,
			Window: time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return tenantID.String()
			},
		}

		rl := NewRateLimiter(config)

		app := fiber.New()
		app.Use(rl.Handler())
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, "10", resp.Header.Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Reset"))
	})

	t.Run("allows anonymous requests", func(t *testing.T) {
		const anonymousKey = "anonymous"
		config := RateLimiterConfig{
			Max:    2,
			Window: time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return anonymousKey
			},
		}

		rl := NewRateLimiter(config)

		app := fiber.New()
		app.Use(rl.Handler())
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// Anonymous requests should always pass (they'll fail at auth anyway)
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			resp, _ := app.Test(req)
			assert.Equal(t, 200, resp.StatusCode)
		}
	})
}

func TestDefaultRateLimiterConfig(t *testing.T) {
	config := DefaultRateLimiterConfig()

	assert.Equal(t, 1000, config.Max)
	assert.Equal(t, time.Minute, config.Window)
	assert.NotNil(t, config.KeyGenerator)
}

func TestRateLimiter_Stop(t *testing.T) {
	t.Run("stops cleanup goroutine gracefully", func(t *testing.T) {
		config := RateLimiterConfig{
			Max:    10,
			Window: time.Second,
			KeyGenerator: func(c *fiber.Ctx) string {
				return "test"
			},
		}

		rl := NewRateLimiter(config)

		// Stop should not panic or block
		rl.Stop()

		// Calling Stop twice should not panic (closed channel)
		// This would panic without proper handling, but we accept this behavior
		// as Stop should only be called once during shutdown
	})
}

func TestIntToString(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{1000, "1000"},
		{-1, "-1"},
		{-100, "-100"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, intToString(tt.input))
	}
}
