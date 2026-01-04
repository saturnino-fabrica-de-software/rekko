package super

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/saturnino-fabrica-de-software/rekko/internal/admin"
)

func (m *MockAdminService) GetSystemHealth(ctx context.Context) (*admin.SystemHealth, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*admin.SystemHealth), args.Error(1)
}

func (m *MockAdminService) GetSystemMetrics(ctx context.Context) (*admin.SystemMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*admin.SystemMetrics), args.Error(1)
}

func TestGetSystemHealth(t *testing.T) {
	app := fiber.New()
	mockService := new(MockAdminService)
	logger := slog.Default()
	handler := NewSystemHandler(mockService, logger)

	mockHealth := &admin.SystemHealth{
		Status:  "healthy",
		Version: "1.0.0",
		Uptime:  "24h",
		Database: admin.ServiceHealth{
			Status:  "healthy",
			Latency: "< 1ms",
		},
		Providers: []admin.ProviderHealth{
			{
				Name:   "rekognition",
				Status: "healthy",
			},
		},
	}

	mockService.On("GetSystemHealth", mock.Anything).Return(mockHealth, nil)

	app.Get("/super/system/health", handler.GetSystemHealth)

	req := httptest.NewRequest("GET", "/super/system/health", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result admin.SystemHealth
	_ = json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "healthy", result.Status)
	assert.Equal(t, "1.0.0", result.Version)

	mockService.AssertExpectations(t)
}

func TestGetSystemHealth_Degraded(t *testing.T) {
	app := fiber.New()
	mockService := new(MockAdminService)
	logger := slog.Default()
	handler := NewSystemHandler(mockService, logger)

	mockHealth := &admin.SystemHealth{
		Status:  "degraded",
		Version: "1.0.0",
		Database: admin.ServiceHealth{
			Status:  "degraded",
			Latency: "100ms",
		},
	}

	mockService.On("GetSystemHealth", mock.Anything).Return(mockHealth, nil)

	app.Get("/super/system/health", handler.GetSystemHealth)

	req := httptest.NewRequest("GET", "/super/system/health", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetSystemMetrics(t *testing.T) {
	app := fiber.New()
	mockService := new(MockAdminService)
	logger := slog.Default()
	handler := NewSystemHandler(mockService, logger)

	mockMetrics := &admin.SystemMetrics{
		Memory: admin.MemoryMetrics{
			Alloc:      5242880,
			TotalAlloc: 104857600,
			Sys:        20971520,
			NumGC:      42,
		},
		Goroutines: 50,
		DBConnections: admin.DBConnMetrics{
			TotalConns: 10,
			IdleConns:  8,
			MaxConns:   20,
		},
		RequestsPerSecond: 125.5,
	}

	mockService.On("GetSystemMetrics", mock.Anything).Return(mockMetrics, nil)

	app.Get("/super/system/metrics", handler.GetSystemMetrics)

	req := httptest.NewRequest("GET", "/super/system/metrics", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	assert.NotNil(t, result["data"])

	mockService.AssertExpectations(t)
}
