package middleware

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

// MockTenantRepo is a mock implementation of TenantRepositoryInterface
type MockTenantRepo struct {
	mock.Mock
}

func (m *MockTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tenant), args.Error(1)
}

func (m *MockTenantRepo) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tenant), args.Error(1)
}

func (m *MockTenantRepo) GetByAPIKeyHash(ctx context.Context, hash string) (*domain.Tenant, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tenant), args.Error(1)
}

func (m *MockTenantRepo) Create(ctx context.Context, tenant *domain.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepo) Update(ctx context.Context, tenant *domain.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockAPIKeyRepo is a mock implementation of APIKeyRepositoryInterface
type MockAPIKeyRepo struct {
	mock.Mock
}

func (m *MockAPIKeyRepo) Create(ctx context.Context, key *domain.APIKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockAPIKeyRepo) GetByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.APIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.APIKey, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAPIKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAPIKeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestAuth(t *testing.T) {
	// Generate valid API key for testing
	validAPIKey, validHash, validPrefix, err := domain.GenerateAPIKey(domain.EnvTest)
	assert.NoError(t, err)

	tenantID := uuid.New()
	apiKeyID := uuid.New()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name           string
		authHeader     string
		setupMocks     func(*MockTenantRepo, *MockAPIKeyRepo)
		expectedStatus int
		checkTenant    bool
		checkAPIKey    bool
	}{
		{
			name:       "valid API key",
			authHeader: "Bearer " + validAPIKey,
			setupMocks: func(mt *MockTenantRepo, mk *MockAPIKeyRepo) {
				apiKey := &domain.APIKey{
					ID:        apiKeyID,
					TenantID:  tenantID,
					Name:      "Test Key",
					KeyHash:   validHash,
					KeyPrefix: validPrefix,
					IsActive:  true,
				}
				mk.On("GetByHash", mock.Anything, validHash).Return(apiKey, nil)
				mk.On("UpdateLastUsed", mock.Anything, apiKeyID).Return(nil)

				tenant := &domain.Tenant{
					ID:       tenantID,
					Name:     "Test Tenant",
					Slug:     "test-tenant",
					IsActive: true,
					Plan:     domain.PlanStarter,
				}
				mt.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
			},
			expectedStatus: 200,
			checkTenant:    true,
			checkAPIKey:    true,
		},
		{
			name:       "missing Authorization header",
			authHeader: "",
			setupMocks: func(mt *MockTenantRepo, mk *MockAPIKeyRepo) {
				// No setup needed
			},
			expectedStatus: 401,
		},
		{
			name:       "invalid API key format",
			authHeader: "Bearer invalid-key-format",
			setupMocks: func(mt *MockTenantRepo, mk *MockAPIKeyRepo) {
				// No setup needed - should fail validation before repo call
			},
			expectedStatus: 401,
		},
		{
			name:       "API key not found",
			authHeader: "Bearer " + validAPIKey,
			setupMocks: func(mt *MockTenantRepo, mk *MockAPIKeyRepo) {
				mk.On("GetByHash", mock.Anything, validHash).Return(nil, errors.New("not found"))
			},
			expectedStatus: 401,
		},
		{
			name:       "API key revoked (inactive)",
			authHeader: "Bearer " + validAPIKey,
			setupMocks: func(mt *MockTenantRepo, mk *MockAPIKeyRepo) {
				apiKey := &domain.APIKey{
					ID:        apiKeyID,
					TenantID:  tenantID,
					Name:      "Revoked Key",
					KeyHash:   validHash,
					KeyPrefix: validPrefix,
					IsActive:  false,
				}
				mk.On("GetByHash", mock.Anything, validHash).Return(apiKey, nil)
			},
			expectedStatus: 401,
		},
		{
			name:       "tenant not found",
			authHeader: "Bearer " + validAPIKey,
			setupMocks: func(mt *MockTenantRepo, mk *MockAPIKeyRepo) {
				apiKey := &domain.APIKey{
					ID:        apiKeyID,
					TenantID:  tenantID,
					Name:      "Test Key",
					KeyHash:   validHash,
					KeyPrefix: validPrefix,
					IsActive:  true,
				}
				mk.On("GetByHash", mock.Anything, validHash).Return(apiKey, nil)
				mt.On("GetByID", mock.Anything, tenantID).Return(nil, errors.New("not found"))
			},
			expectedStatus: 401,
		},
		{
			name:       "inactive tenant",
			authHeader: "Bearer " + validAPIKey,
			setupMocks: func(mt *MockTenantRepo, mk *MockAPIKeyRepo) {
				apiKey := &domain.APIKey{
					ID:        apiKeyID,
					TenantID:  tenantID,
					Name:      "Test Key",
					KeyHash:   validHash,
					KeyPrefix: validPrefix,
					IsActive:  true,
				}
				mk.On("GetByHash", mock.Anything, validHash).Return(apiKey, nil)
				// No UpdateLastUsed because tenant is inactive (middleware returns early)

				tenant := &domain.Tenant{
					ID:       tenantID,
					Name:     "Inactive Tenant",
					Slug:     "inactive-tenant",
					IsActive: false,
					Plan:     domain.PlanStarter,
				}
				mt.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
			},
			expectedStatus: 403,
		},
		{
			name:       "invalid Bearer format",
			authHeader: "Basic abc123",
			setupMocks: func(mt *MockTenantRepo, mk *MockAPIKeyRepo) {
				// No setup needed
			},
			expectedStatus: 401,
		},
		{
			name:       "empty Bearer token",
			authHeader: "Bearer ",
			setupMocks: func(mt *MockTenantRepo, mk *MockAPIKeyRepo) {
				// No setup needed
			},
			expectedStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTenantRepo := &MockTenantRepo{}
			mockAPIKeyRepo := &MockAPIKeyRepo{}
			tt.setupMocks(mockTenantRepo, mockAPIKeyRepo)

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

			deps := AuthDependencies{
				TenantRepo: mockTenantRepo,
				APIKeyRepo: mockAPIKeyRepo,
				Logger:     logger,
			}
			app.Use(Auth(deps))

			// Test endpoint
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			resp, err := app.Test(req, 1000) // 1 second timeout
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkTenant {
				body, _ := io.ReadAll(resp.Body)
				assert.Equal(t, "OK", string(body))
			}

			// Give some time for background goroutine to complete
			if tt.checkAPIKey {
				time.Sleep(100 * time.Millisecond)
			}

			mockTenantRepo.AssertExpectations(t)
			mockAPIKeyRepo.AssertExpectations(t)
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

func TestGetAPIKey(t *testing.T) {
	t.Run("api key exists", func(t *testing.T) {
		app := fiber.New()
		expectedAPIKey := &domain.APIKey{
			ID:        uuid.New(),
			TenantID:  uuid.New(),
			Name:      "Test Key",
			KeyPrefix: "rekko_test_abc123",
			IsActive:  true,
		}

		app.Get("/", func(c *fiber.Ctx) error {
			c.Locals(LocalAPIKey, expectedAPIKey)

			gotAPIKey, err := GetAPIKey(c)
			assert.NoError(t, err)
			assert.Equal(t, expectedAPIKey, gotAPIKey)
			return nil
		})

		_, err := app.Test(httptest.NewRequest("GET", "/", nil))
		assert.NoError(t, err)
	})

	t.Run("api key not set", func(t *testing.T) {
		app := fiber.New()

		app.Get("/", func(c *fiber.Ctx) error {
			_, err := GetAPIKey(c)
			assert.ErrorIs(t, err, domain.ErrUnauthorized)
			return nil
		})

		_, err := app.Test(httptest.NewRequest("GET", "/", nil))
		assert.NoError(t, err)
	})
}
