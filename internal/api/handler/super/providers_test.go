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

func (m *MockAdminService) GetProvidersStatus(ctx context.Context) ([]admin.ProviderHealth, error) {
	args := m.Called(ctx)
	return args.Get(0).([]admin.ProviderHealth), args.Error(1)
}

func TestGetProvidersStatus(t *testing.T) {
	app := fiber.New()
	mockService := new(MockAdminService)
	logger := slog.Default()
	handler := NewProvidersHandler(mockService, logger)

	mockProviders := []admin.ProviderHealth{
		{
			Name:   "rekognition",
			Status: "healthy",
		},
		{
			Name:   "deepface",
			Status: "healthy",
		},
	}

	mockService.On("GetProvidersStatus", mock.Anything).Return(mockProviders, nil)

	app.Get("/super/providers", handler.GetProvidersStatus)

	req := httptest.NewRequest("GET", "/super/providers", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	assert.NotNil(t, result["data"])
	data := result["data"].([]interface{})
	assert.Equal(t, 2, len(data))

	mockService.AssertExpectations(t)
}
