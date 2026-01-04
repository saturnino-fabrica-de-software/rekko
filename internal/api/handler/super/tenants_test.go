package super

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/saturnino-fabrica-de-software/rekko/internal/admin"
)

type MockAdminService struct {
	mock.Mock
}

func (m *MockAdminService) ListAllTenants(ctx context.Context, limit, offset int) ([]admin.TenantWithMetrics, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]admin.TenantWithMetrics), args.Error(1)
}

func (m *MockAdminService) GetTenantDetailedMetrics(ctx context.Context, tenantID uuid.UUID) (*admin.TenantMetricsSummary, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*admin.TenantMetricsSummary), args.Error(1)
}

func (m *MockAdminService) UpdateTenantQuota(ctx context.Context, tenantID uuid.UUID, req admin.UpdateQuotaRequest) error {
	args := m.Called(ctx, tenantID, req)
	return args.Error(0)
}

func TestListTenants(t *testing.T) {
	app := fiber.New()
	mockService := new(MockAdminService)
	logger := slog.Default()
	handler := NewTenantsHandler(mockService, logger)

	tenantID := uuid.New()
	mockTenants := []admin.TenantWithMetrics{
		{
			ID:       tenantID.String(),
			Name:     "Test Tenant",
			PlanType: "pro",
			IsActive: true,
			Metrics: admin.TenantMetricsSummary{
				TotalFaces:    100,
				TotalRequests: 1000,
				AvgLatencyMs:  45.5,
				ErrorRate:     2.5,
			},
		},
	}

	mockService.On("ListAllTenants", mock.Anything, 50, 0).Return(mockTenants, nil)

	app.Get("/super/tenants", handler.ListTenants)

	req := httptest.NewRequest("GET", "/super/tenants", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	assert.NotNil(t, result["data"])
	assert.NotNil(t, result["meta"])

	mockService.AssertExpectations(t)
}

func TestGetTenantMetrics(t *testing.T) {
	app := fiber.New()
	mockService := new(MockAdminService)
	logger := slog.Default()
	handler := NewTenantsHandler(mockService, logger)

	tenantID := uuid.New()
	mockMetrics := &admin.TenantMetricsSummary{
		TotalFaces:    100,
		TotalRequests: 1000,
		AvgLatencyMs:  45.5,
		ErrorRate:     2.5,
	}

	mockService.On("GetTenantDetailedMetrics", mock.Anything, tenantID).Return(mockMetrics, nil)

	app.Get("/super/tenants/:id/metrics", handler.GetTenantMetrics)

	req := httptest.NewRequest("GET", "/super/tenants/"+tenantID.String()+"/metrics", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	assert.NotNil(t, result["data"])
	assert.NotNil(t, result["meta"])

	mockService.AssertExpectations(t)
}

func TestUpdateTenantQuota(t *testing.T) {
	app := fiber.New()
	mockService := new(MockAdminService)
	logger := slog.Default()
	handler := NewTenantsHandler(mockService, logger)

	tenantID := uuid.New()
	maxFaces := 5000
	quotaReq := admin.UpdateQuotaRequest{
		MaxFaces: &maxFaces,
	}

	mockService.On("UpdateTenantQuota", mock.Anything, tenantID, quotaReq).Return(nil)

	app.Post("/super/tenants/:id/quota", handler.UpdateTenantQuota)

	body, _ := json.Marshal(quotaReq)
	req := httptest.NewRequest("POST", "/super/tenants/"+tenantID.String()+"/quota", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "quota updated successfully", result["message"])
	assert.Equal(t, tenantID.String(), result["tenant_id"])

	mockService.AssertExpectations(t)
}

func TestGetTenantMetrics_InvalidID(t *testing.T) {
	app := fiber.New()
	mockService := new(MockAdminService)
	logger := slog.Default()
	handler := NewTenantsHandler(mockService, logger)

	app.Get("/super/tenants/:id/metrics", handler.GetTenantMetrics)

	req := httptest.NewRequest("GET", "/super/tenants/invalid-uuid/metrics", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
