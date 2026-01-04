package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/saturnino-fabrica-de-software/rekko/internal/api/middleware"
	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

// MockFaceService is a mock implementation of FaceService
type MockFaceService struct {
	mock.Mock
}

func (m *MockFaceService) Register(ctx context.Context, tenantID uuid.UUID, externalID string, imageBytes []byte, requireLiveness bool, livenessThreshold float64) (*domain.Face, error) {
	args := m.Called(ctx, tenantID, externalID, imageBytes, requireLiveness, livenessThreshold)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Face), args.Error(1)
}

func (m *MockFaceService) Verify(ctx context.Context, tenantID uuid.UUID, externalID string, imageBytes []byte) (*domain.Verification, error) {
	args := m.Called(ctx, tenantID, externalID, imageBytes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Verification), args.Error(1)
}

func (m *MockFaceService) Delete(ctx context.Context, tenantID uuid.UUID, externalID string) error {
	args := m.Called(ctx, tenantID, externalID)
	return args.Error(0)
}

func (m *MockFaceService) CheckLiveness(ctx context.Context, imageBytes []byte, threshold float64) (*domain.LivenessResult, error) {
	args := m.Called(ctx, imageBytes, threshold)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LivenessResult), args.Error(1)
}

// MockUsageTracker is a mock implementation of UsageTracker
type MockUsageTracker struct {
	mock.Mock
}

func (m *MockUsageTracker) IncrementDaily(ctx context.Context, tenantID uuid.UUID, date time.Time, field string, amount int) error {
	args := m.Called(ctx, tenantID, date, field, amount)
	return args.Error(0)
}

// testLogger returns a logger that discards all output
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// Helper to create multipart request
func createMultipartRequest(externalID string, imageContent []byte, contentType string) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if externalID != "" {
		_ = writer.WriteField("external_id", externalID)
	}

	if imageContent != nil {
		// Create part with custom Content-Type header
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="image"; filename="test.jpg"`)
		h.Set("Content-Type", contentType)

		part, err := writer.CreatePart(h)
		if err != nil {
			return nil, "", err
		}
		_, _ = part.Write(imageContent)
	}

	_ = writer.Close()
	return body, writer.FormDataContentType(), nil
}

// Helper to create test app with tenant in context
func createTestApp(handler *FaceHandler, tenantID uuid.UUID) *fiber.App {
	app := fiber.New()

	// Middleware that simulates authentication
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.LocalTenantID, tenantID)
		c.Locals(middleware.LocalTenant, &domain.Tenant{
			ID:       tenantID,
			Name:     "Test Tenant",
			Slug:     "test-tenant",
			IsActive: true,
			Plan:     domain.PlanStarter,
			Settings: map[string]interface{}{
				"verification_threshold": 0.8,
				"max_faces_per_user":     float64(1),
				"require_liveness":       false,
				"liveness_threshold":     0.90,
			},
		})
		return c.Next()
	})

	// Error handler
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

	return app
}

func TestFaceHandler_Register(t *testing.T) {
	tenantID := uuid.New()
	faceID := uuid.New()

	tests := []struct {
		name           string
		externalID     string
		imageContent   []byte
		contentType    string
		setupMock      func(*MockFaceService)
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:         "successful registration",
			externalID:   "user_001",
			imageContent: make([]byte, 5000),
			contentType:  "image/jpeg",
			setupMock: func(m *MockFaceService) {
				m.On("Register", mock.Anything, tenantID, "user_001", mock.Anything, mock.AnythingOfType("bool"), mock.AnythingOfType("float64")).Return(&domain.Face{
					ID:           faceID,
					ExternalID:   "user_001",
					QualityScore: 0.95,
					CreatedAt:    time.Now(),
				}, nil)
			},
			expectedStatus: 201,
			checkResponse: func(t *testing.T, body []byte) {
				var resp RegisterResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.Equal(t, faceID.String(), resp.FaceID)
				assert.Equal(t, "user_001", resp.ExternalID)
				assert.Equal(t, 0.95, resp.QualityScore)
			},
		},
		{
			name:           "missing external_id",
			externalID:     "",
			imageContent:   make([]byte, 5000),
			contentType:    "image/jpeg",
			setupMock:      func(m *MockFaceService) {},
			expectedStatus: 422,
		},
		{
			name:         "face already exists",
			externalID:   "user_001",
			imageContent: make([]byte, 5000),
			contentType:  "image/jpeg",
			setupMock: func(m *MockFaceService) {
				m.On("Register", mock.Anything, tenantID, "user_001", mock.Anything, mock.AnythingOfType("bool"), mock.AnythingOfType("float64")).Return(nil, domain.ErrFaceExists)
			},
			expectedStatus: 409,
		},
		{
			name:         "no face detected",
			externalID:   "user_001",
			imageContent: make([]byte, 5000),
			contentType:  "image/jpeg",
			setupMock: func(m *MockFaceService) {
				m.On("Register", mock.Anything, tenantID, "user_001", mock.Anything, mock.AnythingOfType("bool"), mock.AnythingOfType("float64")).Return(nil, domain.ErrNoFaceDetected)
			},
			expectedStatus: 422,
		},
		{
			name:         "multiple faces detected",
			externalID:   "user_001",
			imageContent: make([]byte, 5000),
			contentType:  "image/jpeg",
			setupMock: func(m *MockFaceService) {
				m.On("Register", mock.Anything, tenantID, "user_001", mock.Anything, mock.AnythingOfType("bool"), mock.AnythingOfType("float64")).Return(nil, domain.ErrMultipleFaces)
			},
			expectedStatus: 422,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockFaceService{}
			mockTracker := &MockUsageTracker{}
			tt.setupMock(mockService)
			// Allow any tracking calls (async, best-effort)
			mockTracker.On("IncrementDaily", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

			handler := NewFaceHandler(mockService, mockTracker, testLogger())
			app := createTestApp(handler, tenantID)
			app.Post("/v1/faces", handler.Register)

			body, contentType, _ := createMultipartRequest(tt.externalID, tt.imageContent, tt.contentType)

			req := httptest.NewRequest("POST", "/v1/faces", body)
			req.Header.Set("Content-Type", contentType)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				respBody, _ := io.ReadAll(resp.Body)
				tt.checkResponse(t, respBody)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestFaceHandler_Verify(t *testing.T) {
	tenantID := uuid.New()
	verificationID := uuid.New()

	tests := []struct {
		name           string
		externalID     string
		imageContent   []byte
		setupMock      func(*MockFaceService)
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:         "successful verification - match",
			externalID:   "user_001",
			imageContent: make([]byte, 5000),
			setupMock: func(m *MockFaceService) {
				m.On("Verify", mock.Anything, tenantID, "user_001", mock.Anything).Return(&domain.Verification{
					ID:         verificationID,
					Verified:   true,
					Confidence: 0.92,
					LatencyMs:  45,
				}, nil)
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var resp VerifyResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.True(t, resp.Verified)
				assert.Equal(t, 0.92, resp.Confidence)
				assert.Equal(t, int64(45), resp.LatencyMs)
			},
		},
		{
			name:         "successful verification - no match",
			externalID:   "user_001",
			imageContent: make([]byte, 5000),
			setupMock: func(m *MockFaceService) {
				m.On("Verify", mock.Anything, tenantID, "user_001", mock.Anything).Return(&domain.Verification{
					ID:         verificationID,
					Verified:   false,
					Confidence: 0.45,
					LatencyMs:  38,
				}, nil)
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var resp VerifyResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.False(t, resp.Verified)
				assert.Equal(t, 0.45, resp.Confidence)
			},
		},
		{
			name:         "face not found",
			externalID:   "user_999",
			imageContent: make([]byte, 5000),
			setupMock: func(m *MockFaceService) {
				m.On("Verify", mock.Anything, tenantID, "user_999", mock.Anything).Return(nil, domain.ErrFaceNotFound)
			},
			expectedStatus: 404,
		},
		{
			name:           "missing external_id",
			externalID:     "",
			imageContent:   make([]byte, 5000),
			setupMock:      func(m *MockFaceService) {},
			expectedStatus: 422,
		},
		{
			name:         "no face detected in verification",
			externalID:   "user_001",
			imageContent: make([]byte, 5000),
			setupMock: func(m *MockFaceService) {
				m.On("Verify", mock.Anything, tenantID, "user_001", mock.Anything).Return(nil, domain.ErrNoFaceDetected)
			},
			expectedStatus: 422,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockFaceService{}
			mockTracker := &MockUsageTracker{}
			tt.setupMock(mockService)
			// Allow any tracking calls (async, best-effort)
			mockTracker.On("IncrementDaily", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

			handler := NewFaceHandler(mockService, mockTracker, testLogger())
			app := createTestApp(handler, tenantID)
			app.Post("/v1/faces/verify", handler.Verify)

			body, contentType, _ := createMultipartRequest(tt.externalID, tt.imageContent, "image/jpeg")

			req := httptest.NewRequest("POST", "/v1/faces/verify", body)
			req.Header.Set("Content-Type", contentType)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				respBody, _ := io.ReadAll(resp.Body)
				tt.checkResponse(t, respBody)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestFaceHandler_Delete(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		name           string
		externalID     string
		setupMock      func(*MockFaceService)
		expectedStatus int
	}{
		{
			name:       "successful deletion",
			externalID: "user_001",
			setupMock: func(m *MockFaceService) {
				m.On("Delete", mock.Anything, tenantID, "user_001").Return(nil)
			},
			expectedStatus: 204,
		},
		{
			name:       "face not found",
			externalID: "user_999",
			setupMock: func(m *MockFaceService) {
				m.On("Delete", mock.Anything, tenantID, "user_999").Return(domain.ErrFaceNotFound)
			},
			expectedStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockFaceService{}
			mockTracker := &MockUsageTracker{}
			tt.setupMock(mockService)
			// Allow any tracking calls (async, best-effort)
			mockTracker.On("IncrementDaily", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

			handler := NewFaceHandler(mockService, mockTracker, testLogger())
			app := createTestApp(handler, tenantID)
			app.Delete("/v1/faces/:external_id", handler.Delete)

			requestURL := "/v1/faces/" + url.PathEscape(tt.externalID)

			req := httptest.NewRequest("DELETE", requestURL, nil)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			mockService.AssertExpectations(t)
		})
	}
}

func TestFaceHandler_CheckLiveness(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		name           string
		imageContent   []byte
		contentType    string
		setupMock      func(*MockFaceService)
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:         "successful liveness check - live",
			imageContent: make([]byte, 5000),
			contentType:  "image/jpeg",
			setupMock: func(m *MockFaceService) {
				m.On("CheckLiveness", mock.Anything, mock.Anything, mock.AnythingOfType("float64")).Return(&domain.LivenessResult{
					IsLive:     true,
					Confidence: 0.95,
					Checks: domain.LivenessChecks{
						EyesOpen:     true,
						FacingCamera: true,
						QualityOK:    true,
						SingleFace:   true,
					},
					Reasons: []string{},
				}, nil)
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var resp LivenessResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.True(t, resp.IsLive)
				assert.Equal(t, 0.95, resp.Confidence)
				assert.True(t, resp.Checks.EyesOpen)
				assert.True(t, resp.Checks.FacingCamera)
				assert.True(t, resp.Checks.QualityOK)
				assert.True(t, resp.Checks.SingleFace)
			},
		},
		{
			name:         "successful liveness check - not live",
			imageContent: make([]byte, 5000),
			contentType:  "image/jpeg",
			setupMock: func(m *MockFaceService) {
				m.On("CheckLiveness", mock.Anything, mock.Anything, mock.AnythingOfType("float64")).Return(&domain.LivenessResult{
					IsLive:     false,
					Confidence: 0.45,
					Checks: domain.LivenessChecks{
						EyesOpen:     false,
						FacingCamera: true,
						QualityOK:    true,
						SingleFace:   true,
					},
					Reasons: []string{"eyes closed"},
				}, nil)
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var resp LivenessResponse
				err := json.Unmarshal(body, &resp)
				assert.NoError(t, err)
				assert.False(t, resp.IsLive)
				assert.Equal(t, 0.45, resp.Confidence)
				assert.False(t, resp.Checks.EyesOpen)
				assert.Contains(t, resp.Reasons, "eyes closed")
			},
		},
		{
			name:         "no face detected",
			imageContent: make([]byte, 5000),
			contentType:  "image/jpeg",
			setupMock: func(m *MockFaceService) {
				m.On("CheckLiveness", mock.Anything, mock.Anything, mock.AnythingOfType("float64")).Return(nil, domain.ErrNoFaceDetected)
			},
			expectedStatus: 422,
		},
		{
			name:         "multiple faces detected",
			imageContent: make([]byte, 5000),
			contentType:  "image/jpeg",
			setupMock: func(m *MockFaceService) {
				m.On("CheckLiveness", mock.Anything, mock.Anything, mock.AnythingOfType("float64")).Return(nil, domain.ErrMultipleFaces)
			},
			expectedStatus: 422,
		},
		{
			name:           "missing image",
			imageContent:   nil,
			contentType:    "image/jpeg",
			setupMock:      func(m *MockFaceService) {},
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockFaceService{}
			mockTracker := &MockUsageTracker{}
			tt.setupMock(mockService)
			// Allow any tracking calls (async, best-effort)
			mockTracker.On("IncrementDaily", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

			handler := NewFaceHandler(mockService, mockTracker, testLogger())
			app := createTestApp(handler, tenantID)
			app.Post("/v1/faces/liveness", handler.CheckLiveness)

			body, contentType, _ := createMultipartRequest("", tt.imageContent, tt.contentType)

			req := httptest.NewRequest("POST", "/v1/faces/liveness", body)
			req.Header.Set("Content-Type", contentType)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				respBody, _ := io.ReadAll(resp.Body)
				tt.checkResponse(t, respBody)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestExtractAndValidateImage(t *testing.T) {
	tests := []struct {
		name          string
		imageSize     int64
		contentType   string
		expectError   bool
		expectedError *domain.AppError
	}{
		{
			name:        "valid jpeg image",
			imageSize:   5000,
			contentType: "image/jpeg",
			expectError: false,
		},
		{
			name:        "valid png image",
			imageSize:   5000,
			contentType: "image/png",
			expectError: false,
		},
		{
			name:        "valid webp image",
			imageSize:   5000,
			contentType: "image/webp",
			expectError: false,
		},
		{
			name:          "image too large",
			imageSize:     11 * 1024 * 1024,
			contentType:   "image/jpeg",
			expectError:   true,
			expectedError: domain.ErrInvalidImage,
		},
		{
			name:          "empty image",
			imageSize:     0,
			contentType:   "image/jpeg",
			expectError:   true,
			expectedError: domain.ErrInvalidImage,
		},
		{
			name:          "invalid content type",
			imageSize:     5000,
			contentType:   "image/gif",
			expectError:   true,
			expectedError: domain.ErrInvalidImage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				BodyLimit: 20 * 1024 * 1024, // 20MB to test our validation
			})

			app.Post("/test", func(c *fiber.Ctx) error {
				_, err := extractAndValidateImage(c)
				if err != nil {
					if appErr, ok := err.(*domain.AppError); ok {
						return c.Status(appErr.StatusCode).JSON(appErr)
					}
					return c.Status(500).SendString(err.Error())
				}
				return c.SendStatus(200)
			})

			body, contentType, _ := createMultipartRequest("test_user", make([]byte, tt.imageSize), tt.contentType)

			req := httptest.NewRequest("POST", "/test", body)
			req.Header.Set("Content-Type", contentType)

			resp, err := app.Test(req)
			assert.NoError(t, err)

			if tt.expectError {
				assert.NotEqual(t, 200, resp.StatusCode)
			} else {
				assert.Equal(t, 200, resp.StatusCode)
			}
		})
	}
}
