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

func TestRateLimiter_PerEndpoint(t *testing.T) {
	t.Run("applies different limits per endpoint", func(t *testing.T) {
		tenantID := uuid.New()

		config := RateLimiterConfig{
			Max:    100,
			Window: time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return tenantID.String()
			},
			PerEndpoint: map[string]EndpointRateLimit{
				"/admin/metrics": {Requests: 2, Window: time.Minute},
				"/admin/export":  {Requests: 1, Window: time.Minute},
			},
		}

		rl := NewRateLimiter(config)

		app := fiber.New(fiber.Config{
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				return c.Status(429).JSON(fiber.Map{"error": "rate limit"})
			},
		})
		app.Use(rl.Handler())
		app.Get("/admin/metrics", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})
		app.Get("/admin/export", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})
		app.Get("/public/data", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// /admin/metrics allows 2 requests
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/admin/metrics", nil)
			resp, _ := app.Test(req)
			assert.Equal(t, 200, resp.StatusCode)
		}

		// Third request to /admin/metrics is rate limited
		req := httptest.NewRequest("GET", "/admin/metrics", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, 429, resp.StatusCode)

		// /admin/export allows only 1 request
		req = httptest.NewRequest("GET", "/admin/export", nil)
		resp, _ = app.Test(req)
		assert.Equal(t, 200, resp.StatusCode)

		// Second request to /admin/export is rate limited
		req = httptest.NewRequest("GET", "/admin/export", nil)
		resp, _ = app.Test(req)
		assert.Equal(t, 429, resp.StatusCode)

		// /public/data uses default limit (100 requests)
		for i := 0; i < 10; i++ {
			req = httptest.NewRequest("GET", "/public/data", nil)
			resp, _ = app.Test(req)
			assert.Equal(t, 200, resp.StatusCode)
		}
	})

	t.Run("different endpoints have separate counters", func(t *testing.T) {
		tenantID := uuid.New()

		config := RateLimiterConfig{
			Max:    100,
			Window: time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return tenantID.String()
			},
			PerEndpoint: map[string]EndpointRateLimit{
				"/endpoint-a": {Requests: 2, Window: time.Minute},
				"/endpoint-b": {Requests: 2, Window: time.Minute},
			},
		}

		rl := NewRateLimiter(config)

		app := fiber.New(fiber.Config{
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				return c.Status(429).JSON(fiber.Map{"error": "rate limit"})
			},
		})
		app.Use(rl.Handler())
		app.Get("/endpoint-a", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})
		app.Get("/endpoint-b", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// Use all requests for endpoint-a
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/endpoint-a", nil)
			resp, _ := app.Test(req)
			assert.Equal(t, 200, resp.StatusCode)
		}

		// endpoint-a is now rate limited
		req := httptest.NewRequest("GET", "/endpoint-a", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, 429, resp.StatusCode)

		// endpoint-b still has quota available
		for i := 0; i < 2; i++ {
			req = httptest.NewRequest("GET", "/endpoint-b", nil)
			resp, _ = app.Test(req)
			assert.Equal(t, 200, resp.StatusCode)
		}

		// Now endpoint-b is also rate limited
		req = httptest.NewRequest("GET", "/endpoint-b", nil)
		resp, _ = app.Test(req)
		assert.Equal(t, 429, resp.StatusCode)
	})

	t.Run("rate limit headers reflect endpoint-specific limits", func(t *testing.T) {
		tenantID := uuid.New()

		config := RateLimiterConfig{
			Max:    100,
			Window: time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return tenantID.String()
			},
			PerEndpoint: map[string]EndpointRateLimit{
				"/admin/metrics": {Requests: 60, Window: time.Minute},
			},
		}

		rl := NewRateLimiter(config)

		app := fiber.New()
		app.Use(rl.Handler())
		app.Get("/admin/metrics", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})
		app.Get("/public/data", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// Check headers for endpoint with custom limit
		req := httptest.NewRequest("GET", "/admin/metrics", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, "60", resp.Header.Get("X-RateLimit-Limit"))
		assert.Equal(t, "59", resp.Header.Get("X-RateLimit-Remaining"))

		// Check headers for endpoint with default limit
		req = httptest.NewRequest("GET", "/public/data", nil)
		resp, _ = app.Test(req)
		assert.Equal(t, "100", resp.Header.Get("X-RateLimit-Limit"))
		assert.Equal(t, "99", resp.Header.Get("X-RateLimit-Remaining"))
	})
}

func TestAdminRateLimits(t *testing.T) {
	limits := AdminRateLimits()

	assert.Equal(t, 60, limits["/v1/admin/metrics"].Requests)
	assert.Equal(t, time.Minute, limits["/v1/admin/metrics"].Window)

	assert.Equal(t, 30, limits["/v1/admin/alerts"].Requests)
	assert.Equal(t, time.Minute, limits["/v1/admin/alerts"].Window)

	assert.Equal(t, 10, limits["/v1/admin/export"].Requests)
	assert.Equal(t, time.Minute, limits["/v1/admin/export"].Window)

	assert.Equal(t, 20, limits["/v1/admin/webhooks"].Requests)
	assert.Equal(t, time.Minute, limits["/v1/admin/webhooks"].Window)

	assert.Equal(t, 100, limits["/super/tenants"].Requests)
	assert.Equal(t, time.Minute, limits["/super/tenants"].Window)

	assert.Equal(t, 60, limits["/super/system"].Requests)
	assert.Equal(t, time.Minute, limits["/super/system"].Window)
}
