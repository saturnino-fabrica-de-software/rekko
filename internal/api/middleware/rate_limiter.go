package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

// EndpointRateLimit defines rate limit for a specific endpoint
type EndpointRateLimit struct {
	Requests int
	Window   time.Duration
}

// RateLimiterConfig holds configuration for rate limiting
type RateLimiterConfig struct {
	// Max requests per window (default for all endpoints)
	Max int
	// Window duration (default for all endpoints)
	Window time.Duration
	// Key generator function - returns tenant ID from context
	KeyGenerator func(c *fiber.Ctx) string
	// PerEndpoint contains custom rate limits for specific endpoints
	PerEndpoint map[string]EndpointRateLimit
}

// DefaultRateLimiterConfig returns default configuration
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Max:    1000,
		Window: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			tenantID, ok := c.Locals(LocalTenantID).(uuid.UUID)
			if !ok {
				return "anonymous"
			}
			return tenantID.String()
		},
		PerEndpoint: make(map[string]EndpointRateLimit),
	}
}

// AdminRateLimits returns rate limits for admin endpoints
func AdminRateLimits() map[string]EndpointRateLimit {
	return map[string]EndpointRateLimit{
		"/v1/admin/metrics":  {Requests: 60, Window: time.Minute},
		"/v1/admin/alerts":   {Requests: 30, Window: time.Minute},
		"/v1/admin/export":   {Requests: 10, Window: time.Minute},
		"/v1/admin/webhooks": {Requests: 20, Window: time.Minute},
		"/super/tenants":     {Requests: 100, Window: time.Minute},
		"/super/system":      {Requests: 60, Window: time.Minute},
	}
}

// tenantLimiter tracks rate limiting state for a tenant
type tenantLimiter struct {
	count      int
	windowEnd  time.Time
	lastAccess time.Time
}

// RateLimiter implements per-tenant rate limiting with per-endpoint customization
type RateLimiter struct {
	config   RateLimiterConfig
	limiters map[string]*tenantLimiter
	mu       sync.RWMutex
	done     chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if config.Max == 0 {
		config.Max = 1000
	}
	if config.Window == 0 {
		config.Window = time.Minute
	}
	if config.KeyGenerator == nil {
		config.KeyGenerator = DefaultRateLimiterConfig().KeyGenerator
	}
	if config.PerEndpoint == nil {
		config.PerEndpoint = make(map[string]EndpointRateLimit)
	}

	rl := &RateLimiter{
		config:   config,
		limiters: make(map[string]*tenantLimiter),
		done:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Stop gracefully shuts down the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.done)
}

// Handler returns the Fiber middleware handler
func (rl *RateLimiter) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := rl.config.KeyGenerator(c)
		if key == "" || key == "anonymous" {
			// Allow anonymous requests to proceed (they'll fail at auth anyway)
			return c.Next()
		}

		// Get rate limit for this endpoint (or use default)
		max := rl.config.Max
		window := rl.config.Window
		path := c.Path()

		if endpointLimit, exists := rl.config.PerEndpoint[path]; exists {
			max = endpointLimit.Requests
			window = endpointLimit.Window
		}

		// Composite key: tenant + endpoint
		compositeKey := key + ":" + path

		now := time.Now()

		rl.mu.Lock()
		limiter, exists := rl.limiters[compositeKey]

		if !exists || now.After(limiter.windowEnd) {
			// Create new window
			newLimiter := &tenantLimiter{
				count:      1,
				windowEnd:  now.Add(window),
				lastAccess: now,
			}
			rl.limiters[compositeKey] = newLimiter
			rl.mu.Unlock()

			// Set rate limit headers
			c.Set("X-RateLimit-Limit", intToString(max))
			c.Set("X-RateLimit-Remaining", intToString(max-1))
			c.Set("X-RateLimit-Reset", newLimiter.windowEnd.Format(time.RFC3339))

			return c.Next()
		}

		// Increment counter
		limiter.count++
		limiter.lastAccess = now
		count := limiter.count
		remaining := max - count
		windowEnd := limiter.windowEnd
		rl.mu.Unlock()

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", intToString(max))
		if remaining < 0 {
			remaining = 0
		}
		c.Set("X-RateLimit-Remaining", intToString(remaining))
		c.Set("X-RateLimit-Reset", windowEnd.Format(time.RFC3339))

		// Check if rate limit exceeded
		if count > max {
			c.Set("Retry-After", intToString(int(time.Until(windowEnd).Seconds())))
			return domain.ErrRateLimitExceeded
		}

		return c.Next()
	}
}

// cleanup removes stale entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, limiter := range rl.limiters {
				// Remove entries that haven't been accessed in 2 windows
				if now.Sub(limiter.lastAccess) > 2*rl.config.Window {
					delete(rl.limiters, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// intToString converts int to string without fmt
func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
