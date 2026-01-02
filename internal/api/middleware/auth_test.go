package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

// MockTenantRepo is a mock implementation of TenantRepository
type MockTenantRepo struct {
	mock.Mock
}

func (m *MockTenantRepo) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*domain.Tenant, error) {
	args := m.Called(ctx, apiKeyHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tenant), args.Error(1)
}

func TestAuth(t *testing.T) {
	validAPIKey := "test-api-key-12345"
	validHash := hashAPIKey(validAPIKey)
	tenantID := uuid.New()

	tests := []struct {
		name           string
		authHeader     string
		setupMock      func(*MockTenantRepo)
		expectedStatus int
		checkTenant    bool
	}{
		{
			name:       "valid API key",
			authHeader: "Bearer " + validAPIKey,
			setupMock: func(m *MockTenantRepo) {
				m.On("GetByAPIKeyHash", mock.Anything, validHash).Return(&domain.Tenant{
					ID:       tenantID,
					Name:     "Test Tenant",
					IsActive: true,
				}, nil)
			},
			expectedStatus: 200,
			checkTenant:    true,
		},
		{
			name:           "missing Authorization header",
			authHeader:     "",
			setupMock:      func(m *MockTenantRepo) {},
			expectedStatus: 401,
		},
		{
			name:       "invalid API key",
			authHeader: "Bearer invalid-key",
			setupMock: func(m *MockTenantRepo) {
				invalidHash := hashAPIKey("invalid-key")
				m.On("GetByAPIKeyHash", mock.Anything, invalidHash).Return(nil, domain.ErrTenantNotFound)
			},
			expectedStatus: 401,
		},
		{
			name:       "inactive tenant",
			authHeader: "Bearer " + validAPIKey,
			setupMock: func(m *MockTenantRepo) {
				m.On("GetByAPIKeyHash", mock.Anything, validHash).Return(&domain.Tenant{
					ID:       tenantID,
					Name:     "Inactive Tenant",
					IsActive: false,
				}, nil)
			},
			expectedStatus: 401,
		},
		{
			name:           "invalid Bearer format",
			authHeader:     "Basic abc123",
			setupMock:      func(m *MockTenantRepo) {},
			expectedStatus: 401,
		},
		{
			name:           "empty Bearer token",
			authHeader:     "Bearer ",
			setupMock:      func(m *MockTenantRepo) {},
			expectedStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockTenantRepo{}
			tt.setupMock(mockRepo)

			app := fiber.New()

			// Setup error handler to convert AppError
			app.Use(func(c *fiber.Ctx) error {
				err := c.Next()
				if err != nil {
					if appErr, ok := err.(*domain.AppError); ok {
						return c.Status(appErr.StatusCode).JSON(appErr)
					}
					return c.Status(500).SendString(err.Error())
				}
				return nil
			})

			app.Use(Auth(mockRepo))

			// Test endpoint
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkTenant {
				body, _ := io.ReadAll(resp.Body)
				assert.Equal(t, "OK", string(body))
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
	}{
		{
			name:      "valid Bearer token",
			header:    "Bearer test-token",
			wantToken: "test-token",
		},
		{
			name:      "lowercase bearer",
			header:    "bearer test-token",
			wantToken: "test-token",
		},
		{
			name:      "empty header",
			header:    "",
			wantToken: "",
		},
		{
			name:      "no Bearer prefix",
			header:    "test-token",
			wantToken: "",
		},
		{
			name:      "Basic auth (should reject)",
			header:    "Basic abc123",
			wantToken: "",
		},
		{
			name:      "Bearer with extra spaces",
			header:    "Bearer   test-token  ",
			wantToken: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			var gotToken string

			app.Get("/", func(c *fiber.Ctx) error {
				gotToken = extractBearerToken(c)
				return nil
			})

			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			_, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantToken, gotToken)
		})
	}
}

func TestHashAPIKey(t *testing.T) {
	apiKey := "my-secret-api-key" // #nosec G101 -- This is a test value, not a real credential

	// Hash must be deterministic
	hash1 := hashAPIKey(apiKey)
	hash2 := hashAPIKey(apiKey)
	assert.Equal(t, hash1, hash2)

	// Hash must have 64 characters (SHA-256 in hex)
	assert.Len(t, hash1, 64)

	// Verify it's the correct hash
	expected := sha256.Sum256([]byte(apiKey))
	expectedHex := hex.EncodeToString(expected[:])
	assert.Equal(t, expectedHex, hash1)

	// Different keys = different hashes
	differentHash := hashAPIKey("different-key")
	assert.NotEqual(t, hash1, differentHash)
}

func TestGetTenantID(t *testing.T) {
	t.Run("tenant_id exists", func(t *testing.T) {
		app := fiber.New()
		expectedID := uuid.New()

		app.Get("/", func(c *fiber.Ctx) error {
			c.Locals(LocalTenantID, expectedID)

			gotID, err := GetTenantID(c)
			assert.NoError(t, err)
			assert.Equal(t, expectedID, gotID)
			return nil
		})

		_, err := app.Test(httptest.NewRequest("GET", "/", nil))
		assert.NoError(t, err)
	})

	t.Run("tenant_id not set", func(t *testing.T) {
		app := fiber.New()

		app.Get("/", func(c *fiber.Ctx) error {
			_, err := GetTenantID(c)
			assert.ErrorIs(t, err, domain.ErrUnauthorized)
			return nil
		})

		_, err := app.Test(httptest.NewRequest("GET", "/", nil))
		assert.NoError(t, err)
	})
}

func TestGetTenant(t *testing.T) {
	t.Run("tenant exists", func(t *testing.T) {
		app := fiber.New()
		expectedTenant := &domain.Tenant{
			ID:       uuid.New(),
			Name:     "Test Tenant",
			IsActive: true,
		}

		app.Get("/", func(c *fiber.Ctx) error {
			c.Locals(LocalTenant, expectedTenant)

			gotTenant, err := GetTenant(c)
			assert.NoError(t, err)
			assert.Equal(t, expectedTenant, gotTenant)
			return nil
		})

		_, err := app.Test(httptest.NewRequest("GET", "/", nil))
		assert.NoError(t, err)
	})

	t.Run("tenant not set", func(t *testing.T) {
		app := fiber.New()

		app.Get("/", func(c *fiber.Ctx) error {
			_, err := GetTenant(c)
			assert.ErrorIs(t, err, domain.ErrUnauthorized)
			return nil
		})

		_, err := app.Test(httptest.NewRequest("GET", "/", nil))
		assert.NoError(t, err)
	})
}
