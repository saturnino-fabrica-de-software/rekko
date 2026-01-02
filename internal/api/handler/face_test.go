package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

func (m *MockFaceService) Register(ctx context.Context, tenantID uuid.UUID, externalID string, imageBytes []byte) (*domain.Face, error) {
	args := m.Called(ctx, tenantID, externalID, imageBytes)
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
				m.On("Register", mock.Anything, tenantID, "user_001", mock.Anything).Return(&domain.Face{
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
				m.On("Register", mock.Anything, tenantID, "user_001", mock.Anything).Return(nil, domain.ErrFaceExists)
			},
			expectedStatus: 409,
		},
		{
			name:         "no face detected",
			externalID:   "user_001",
			imageContent: make([]byte, 5000),
			contentType:  "image/jpeg",
			setupMock: func(m *MockFaceService) {
				m.On("Register", mock.Anything, tenantID, "user_001", mock.Anything).Return(nil, domain.ErrNoFaceDetected)
			},
			expectedStatus: 422,
		},
		{
			name:         "multiple faces detected",
			externalID:   "user_001",
			imageContent: make([]byte, 5000),
			contentType:  "image/jpeg",
			setupMock: func(m *MockFaceService) {
				m.On("Register", mock.Anything, tenantID, "user_001", mock.Anything).Return(nil, domain.ErrMultipleFaces)
			},
			expectedStatus: 422,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockFaceService{}
			tt.setupMock(mockService)

			handler := NewFaceHandler(mockService)
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
			tt.setupMock(mockService)

			handler := NewFaceHandler(mockService)
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
			tt.setupMock(mockService)

			handler := NewFaceHandler(mockService)
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
