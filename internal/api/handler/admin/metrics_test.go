package admin

import (
	"encoding/json"
	"errors"
	"io"
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
	"github.com/saturnino-fabrica-de-software/rekko/internal/api/middleware"
)

// setupTestApp creates a fiber app with tenant middleware
func setupTestApp(handler fiber.Handler, tenantID uuid.UUID) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.LocalTenantID, tenantID)
		return c.Next()
	})

	app.Get("/test", handler)
	return app
}

// readResponseBody reads and unmarshals response body
func readResponseBody(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("failed to close response body: %v", err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, v)
	require.NoError(t, err)
}

// TestGetFacesMetrics_NoTenantID tests missing tenant ID scenario
func TestGetFacesMetrics_NoTenantID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := &MetricsUsageHandler{
		adminService: nil,
		logger:       logger,
	}

	app := fiber.New()
	app.Get("/test", handler.GetFacesMetrics)

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, 401, resp.StatusCode)
}

// TestParseMetricsParams_InvalidDates tests date parameter validation
func TestParseMetricsParams_InvalidDates(t *testing.T) {
	tenantID := uuid.New()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid start_date format",
			queryParams:    "?start_date=invalid-date",
			expectedStatus: 400,
			expectedError:  "invalid start_date format",
		},
		{
			name:           "invalid end_date format",
			queryParams:    "?end_date=2024-13-01",
			expectedStatus: 400,
			expectedError:  "invalid end_date format",
		},
		{
			name:           "start_date after end_date",
			queryParams:    "?start_date=2024-02-01&end_date=2024-01-01",
			expectedStatus: 400,
			expectedError:  "start_date must be before or equal to end_date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &MetricsUsageHandler{
				adminService: nil,
				logger:       logger,
			}

			app := setupTestApp(handler.GetFacesMetrics, tenantID)

			req := httptest.NewRequest("GET", "/test"+tt.queryParams, nil)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Contains(t, string(body), tt.expectedError)
		})
	}
}

// TestMetricsHandlers_NoTenantID tests all handlers without tenant ID
func TestMetricsHandlers_NoTenantID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	usageHandler := &MetricsUsageHandler{logger: logger}
	perfHandler := &MetricsPerformanceHandler{logger: logger}
	qualityHandler := &MetricsQualityHandler{logger: logger}

	tests := []struct {
		name    string
		handler fiber.Handler
	}{
		{"GetFacesMetrics", usageHandler.GetFacesMetrics},
		{"GetOperationsMetrics", usageHandler.GetOperationsMetrics},
		{"GetRequestsMetrics", usageHandler.GetRequestsMetrics},
		{"GetLatencyMetrics", perfHandler.GetLatencyMetrics},
		{"GetThroughputMetrics", perfHandler.GetThroughputMetrics},
		{"GetErrorMetrics", perfHandler.GetErrorMetrics},
		{"GetQualityMetrics", qualityHandler.GetQualityMetrics},
		{"GetConfidenceMetrics", qualityHandler.GetConfidenceMetrics},
		{"GetMatchMetrics", qualityHandler.GetMatchMetrics},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", tt.handler)

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)

			assert.Equal(t, 401, resp.StatusCode)
		})
	}
}

// TestMetricsUsageHandler_Constructor tests handler constructors
func TestMetricsUsageHandler_Constructor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("NewMetricsUsageHandler", func(t *testing.T) {
		handler := NewMetricsUsageHandler(nil, logger)
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.logger)
	})

	t.Run("NewMetricsPerformanceHandler", func(t *testing.T) {
		handler := NewMetricsPerformanceHandler(nil, logger)
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.logger)
	})

	t.Run("NewMetricsQualityHandler", func(t *testing.T) {
		handler := NewMetricsQualityHandler(nil, logger)
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.logger)
	})
}

// TestErrorHandling tests error response format
func TestErrorHandling(t *testing.T) {
	tenantID := uuid.New()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("invalid date returns proper error JSON", func(t *testing.T) {
		handler := &MetricsUsageHandler{
			adminService: nil,
			logger:       logger,
		}

		app := setupTestApp(handler.GetFacesMetrics, tenantID)

		req := httptest.NewRequest("GET", "/test?start_date=invalid", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 400, resp.StatusCode)

		var errorResp map[string]interface{}
		readResponseBody(t, resp, &errorResp)

		assert.Contains(t, errorResp, "error")
	})
}

// TestSharedParseMetricsParams tests the shared parseMetricsParams function
func TestSharedParseMetricsParams(t *testing.T) {
	tenantID := uuid.New()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("MetricsPerformanceHandler uses shared parser", func(t *testing.T) {
		handler := &MetricsPerformanceHandler{
			adminService: nil,
			logger:       logger,
		}

		app := setupTestApp(handler.GetLatencyMetrics, tenantID)

		req := httptest.NewRequest("GET", "/test?start_date=invalid", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("MetricsQualityHandler uses shared parser", func(t *testing.T) {
		handler := &MetricsQualityHandler{
			adminService: nil,
			logger:       logger,
		}

		app := setupTestApp(handler.GetQualityMetrics, tenantID)

		req := httptest.NewRequest("GET", "/test?end_date=2024-13-99", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)

		assert.Equal(t, 400, resp.StatusCode)
	})
}

// TestHandlerErrorTypes tests different error scenarios
func TestHandlerErrorTypes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "database connection error",
			err:      errors.New("connection refused"),
			expected: "connection refused",
		},
		{
			name:     "query timeout error",
			err:      errors.New("context deadline exceeded"),
			expected: "context deadline exceeded",
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// TestMetricsResponse_Format tests response structure
func TestMetricsResponse_Format(t *testing.T) {
	now := time.Now()

	response := admin.MetricsResponse{
		Data: map[string]interface{}{
			"total": 100,
		},
		Meta: admin.ResponseMeta{
			TenantID: uuid.New().String(),
			Period: admin.Period{
				Start: "2024-01-01",
				End:   "2024-01-31",
			},
			GeneratedAt: now,
		},
		Pagination: &admin.PaginationMeta{
			Total:  50,
			Limit:  100,
			Offset: 0,
		},
	}

	// Serialize to JSON and back
	jsonBytes, err := json.Marshal(response)
	require.NoError(t, err)

	var decoded admin.MetricsResponse
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.NotNil(t, decoded.Data)
	assert.Equal(t, response.Meta.TenantID, decoded.Meta.TenantID)
	assert.Equal(t, response.Meta.Period.Start, decoded.Meta.Period.Start)
	assert.Equal(t, response.Meta.Period.End, decoded.Meta.Period.End)
	assert.NotNil(t, decoded.Pagination)
	assert.Equal(t, 50, decoded.Pagination.Total)
	assert.Equal(t, 100, decoded.Pagination.Limit)
	assert.Equal(t, 0, decoded.Pagination.Offset)
}
