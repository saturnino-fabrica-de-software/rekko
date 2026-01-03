package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

// RateLimiterConfig holds configuration for rate limiting
type RateLimiterConfig struct {
	// Max requests per window
	Max int
	// Window duration
	Window time.Duration
	// Key generator function - returns tenant ID from context
	KeyGenerator func(c *fiber.Ctx) string
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
	}
}

// tenantLimiter tracks rate limiting state for a tenant
type tenantLimiter struct {
	count      int
	windowEnd  time.Time
	lastAccess time.Time
}

// RateLimiter implements per-tenant rate limiting
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

		now := time.Now()

		rl.mu.Lock()
		limiter, exists := rl.limiters[key]

		if !exists || now.After(limiter.windowEnd) {
			// Create new window
			newLimiter := &tenantLimiter{
				count:      1,
				windowEnd:  now.Add(rl.config.Window),
				lastAccess: now,
			}
			rl.limiters[key] = newLimiter
			rl.mu.Unlock()

			// Set rate limit headers
			c.Set("X-RateLimit-Limit", intToString(rl.config.Max))
			c.Set("X-RateLimit-Remaining", intToString(rl.config.Max-1))
			c.Set("X-RateLimit-Reset", newLimiter.windowEnd.Format(time.RFC3339))

			return c.Next()
		}

		// Increment counter
		limiter.count++
		limiter.lastAccess = now
		count := limiter.count
		remaining := rl.config.Max - count
		windowEnd := limiter.windowEnd
		rl.mu.Unlock()

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", intToString(rl.config.Max))
		if remaining < 0 {
			remaining = 0
		}
		c.Set("X-RateLimit-Remaining", intToString(remaining))
		c.Set("X-RateLimit-Reset", windowEnd.Format(time.RFC3339))

		// Check if rate limit exceeded
		if count > rl.config.Max {
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
