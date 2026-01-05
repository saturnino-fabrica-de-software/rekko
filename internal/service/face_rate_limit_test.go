package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

func TestFaceService_Search_RateLimiting(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		name              string
		tenant            *domain.Tenant
		rateLimitMock     func(*MockRateLimiter, uuid.UUID, int)
		expectError       bool
		expectedErrorType error
	}{
		{
			name: "rate limit allows request",
			tenant: &domain.Tenant{
				ID: tenantID,
				Settings: map[string]interface{}{
					"search_enabled":    true,
					"search_rate_limit": float64(30),
				},
			},
			rateLimitMock: func(rl *MockRateLimiter, tid uuid.UUID, limit int) {
				rl.On("CheckSearchLimit", mock.Anything, tid, limit).Return(nil)
			},
			expectError: false,
		},
		{
			name: "rate limit blocks request",
			tenant: &domain.Tenant{
				ID: tenantID,
				Settings: map[string]interface{}{
					"search_enabled":    true,
					"search_rate_limit": float64(30),
				},
			},
			rateLimitMock: func(rl *MockRateLimiter, tid uuid.UUID, limit int) {
				rl.On("CheckSearchLimit", mock.Anything, tid, limit).Return(errors.New("rate limit exceeded"))
			},
			expectError:       true,
			expectedErrorType: domain.ErrSearchRateLimitExceeded,
		},
		{
			name: "no rate limit configured (unlimited)",
			tenant: &domain.Tenant{
				ID: tenantID,
				Settings: map[string]interface{}{
					"search_enabled":    true,
					"search_rate_limit": float64(0), // 0 = unlimited
				},
			},
			rateLimitMock: func(rl *MockRateLimiter, tid uuid.UUID, limit int) {
				rl.On("CheckSearchLimit", mock.Anything, tid, 0).Return(nil)
			},
			expectError: false,
		},
		{
			name: "rate limit not in settings (uses default)",
			tenant: &domain.Tenant{
				ID: tenantID,
				Settings: map[string]interface{}{
					"search_enabled": true,
					// search_rate_limit not set, should use default (30)
				},
			},
			rateLimitMock: func(rl *MockRateLimiter, tid uuid.UUID, limit int) {
				rl.On("CheckSearchLimit", mock.Anything, tid, 30).Return(nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			faceRepo := &MockFaceRepository{}
			verificationRepo := &MockVerificationRepository{}
			searchAuditRepo := &MockSearchAuditRepository{}
			faceProvider := &MockFaceProvider{}
			rateLimiter := &MockRateLimiter{}

			// Extract settings to get actual limit
			settings := tt.tenant.GetSettings()
			tt.rateLimitMock(rateLimiter, tt.tenant.ID, settings.SearchRateLimit)

			// Only set up provider/repo mocks if rate limit passes
			if !tt.expectError {
				faceProvider.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     []float64{0.1, 0.2},
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.90,
					FaceCount:     1,
				}, nil)
				faceRepo.On("SearchByEmbedding", mock.Anything, tt.tenant.ID, mock.Anything, mock.Anything, mock.Anything).Return([]domain.SearchMatch{}, nil)
				searchAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			}

			svc := &FaceService{
				faceRepo:         faceRepo,
				verificationRepo: verificationRepo,
				searchAuditRepo:  searchAuditRepo,
				provider:         faceProvider,
				rateLimiter:      rateLimiter,
				threshold:        0.8,
			}

			result, err := svc.Search(context.Background(), tt.tenant, []byte("image"), 0.85, 10, "127.0.0.1")

			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErrorType)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}

			rateLimiter.AssertExpectations(t)
			if !tt.expectError {
				faceRepo.AssertExpectations(t)
				faceProvider.AssertExpectations(t)
			}
		})
	}
}

func TestFaceService_Search_RateLimitCheckedBeforeProviderCall(t *testing.T) {
	// This test ensures rate limit is checked BEFORE calling the provider
	// (avoiding unnecessary provider API calls when rate limited)

	tenantID := uuid.New()
	tenant := &domain.Tenant{
		ID: tenantID,
		Settings: map[string]interface{}{
			"search_enabled":    true,
			"search_rate_limit": float64(30),
		},
	}

	faceRepo := &MockFaceRepository{}
	verificationRepo := &MockVerificationRepository{}
	searchAuditRepo := &MockSearchAuditRepository{}
	faceProvider := &MockFaceProvider{}
	rateLimiter := &MockRateLimiter{}

	// Rate limiter blocks request
	rateLimiter.On("CheckSearchLimit", mock.Anything, tenantID, 30).Return(errors.New("rate limit exceeded"))

	// Provider should NOT be called
	// (no mock setup means test will fail if called)

	svc := &FaceService{
		faceRepo:         faceRepo,
		verificationRepo: verificationRepo,
		searchAuditRepo:  searchAuditRepo,
		provider:         faceProvider,
		rateLimiter:      rateLimiter,
		threshold:        0.8,
	}

	result, err := svc.Search(context.Background(), tenant, []byte("image"), 0.85, 10, "127.0.0.1")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrSearchRateLimitExceeded)
	assert.Nil(t, result)

	rateLimiter.AssertExpectations(t)

	// Ensure provider was NOT called
	faceProvider.AssertNotCalled(t, "AnalyzeFace")
	faceRepo.AssertNotCalled(t, "SearchByEmbedding")
}
