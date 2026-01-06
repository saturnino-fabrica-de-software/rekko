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

type MockFaceRepository struct {
	mock.Mock
}

func (m *MockFaceRepository) Create(ctx context.Context, face *domain.Face) error {
	args := m.Called(ctx, face)
	return args.Error(0)
}

func (m *MockFaceRepository) Update(ctx context.Context, face *domain.Face) error {
	args := m.Called(ctx, face)
	return args.Error(0)
}

func (m *MockFaceRepository) GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Face, error) {
	args := m.Called(ctx, tenantID, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Face), args.Error(1)
}

func (m *MockFaceRepository) Delete(ctx context.Context, tenantID uuid.UUID, externalID string) error {
	args := m.Called(ctx, tenantID, externalID)
	return args.Error(0)
}

func (m *MockFaceRepository) SearchByEmbedding(ctx context.Context, tenantID uuid.UUID, embedding []float64, threshold float64, limit int) ([]domain.SearchMatch, error) {
	args := m.Called(ctx, tenantID, embedding, threshold, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.SearchMatch), args.Error(1)
}

func (m *MockFaceRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

type MockVerificationRepository struct {
	mock.Mock
}

type MockSearchAuditRepository struct {
	mock.Mock
}

func (m *MockSearchAuditRepository) Create(ctx context.Context, audit *domain.SearchAudit) error {
	args := m.Called(ctx, audit)
	return args.Error(0)
}

type MockRateLimiter struct {
	mock.Mock
}

func (m *MockRateLimiter) CheckSearchLimit(ctx context.Context, tenantID uuid.UUID, limit int) error {
	args := m.Called(ctx, tenantID, limit)
	return args.Error(0)
}

func (m *MockVerificationRepository) Create(ctx context.Context, v *domain.Verification) error {
	args := m.Called(ctx, v)
	return args.Error(0)
}

type MockFaceProvider struct {
	mock.Mock
}

func (m *MockFaceProvider) DetectFaces(ctx context.Context, image []byte) ([]provider.DetectedFace, error) {
	args := m.Called(ctx, image)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]provider.DetectedFace), args.Error(1)
}

func (m *MockFaceProvider) IndexFace(ctx context.Context, image []byte) (string, []float64, error) {
	args := m.Called(ctx, image)
	return args.String(0), args.Get(1).([]float64), args.Error(2)
}

func (m *MockFaceProvider) CompareFaces(ctx context.Context, emb1, emb2 []float64) (float64, error) {
	args := m.Called(ctx, emb1, emb2)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockFaceProvider) DeleteFace(ctx context.Context, faceID string) error {
	args := m.Called(ctx, faceID)
	return args.Error(0)
}

func (m *MockFaceProvider) CheckLiveness(ctx context.Context, image []byte, threshold float64) (*provider.LivenessResult, error) {
	args := m.Called(ctx, image, threshold)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.LivenessResult), args.Error(1)
}

func (m *MockFaceProvider) AnalyzeFace(ctx context.Context, image []byte) (*provider.FaceAnalysis, error) {
	args := m.Called(ctx, image)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.FaceAnalysis), args.Error(1)
}

func TestFaceService_Register(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   uuid.UUID
		externalID string
		imageBytes []byte
		setupMocks func(*MockFaceRepository, *MockVerificationRepository, *MockFaceProvider)
		wantErr    error
	}{
		{
			name:       "successful registration - new face",
			tenantID:   uuid.New(),
			externalID: "user_001",
			imageBytes: make([]byte, 5000),
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     make([]float64, 512),
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.90,
					FaceCount:     1,
				}, nil)
				// No existing face for this external_id
				fr.On("GetByExternalID", mock.Anything, mock.Anything, "user_001").
					Return(nil, domain.ErrFaceNotFound)
				fr.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "no face detected",
			tenantID:   uuid.New(),
			externalID: "user_001",
			imageBytes: make([]byte, 5000),
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					FaceCount: 0,
				}, nil)
			},
			wantErr: domain.ErrNoFaceDetected,
		},
		{
			name:       "multiple faces detected",
			tenantID:   uuid.New(),
			externalID: "user_001",
			imageBytes: make([]byte, 5000),
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					FaceCount: 2,
				}, nil)
			},
			wantErr: domain.ErrMultipleFaces,
		},
		{
			name:       "re-registration - updates existing face",
			tenantID:   uuid.MustParse("a6646bc1-769f-4bdc-8496-f2e0890abbd0"),
			externalID: "user_001",
			imageBytes: make([]byte, 5000),
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				existingFaceID := uuid.New()
				tenantID := uuid.MustParse("a6646bc1-769f-4bdc-8496-f2e0890abbd0")
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     make([]float64, 512),
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.90,
					FaceCount:     1,
				}, nil)
				// Face already exists for this external_id - will update
				fr.On("GetByExternalID", mock.Anything, tenantID, "user_001").Return(&domain.Face{
					ID:         existingFaceID,
					TenantID:   tenantID,
					ExternalID: "user_001",
				}, nil)
				fr.On("Update", mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			faceRepo := &MockFaceRepository{}
			verificationRepo := &MockVerificationRepository{}
			searchAuditRepo := &MockSearchAuditRepository{}
			faceProvider := &MockFaceProvider{}

			tt.setupMocks(faceRepo, verificationRepo, faceProvider)

			svc := &FaceService{
				faceRepo:         faceRepo,
				verificationRepo: verificationRepo,
				searchAuditRepo:  searchAuditRepo,
				provider:         faceProvider,
				threshold:        0.8,
			}

			face, err := svc.Register(context.Background(), tt.tenantID, tt.externalID, tt.imageBytes, false, 0.90)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, face)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, face)
				assert.Equal(t, tt.externalID, face.ExternalID)
				assert.Equal(t, tt.tenantID, face.TenantID)
			}

			faceRepo.AssertExpectations(t)
			faceProvider.AssertExpectations(t)
		})
	}
}

func TestFaceService_Register_WithLiveness(t *testing.T) {
	tests := []struct {
		name              string
		tenantID          uuid.UUID
		externalID        string
		imageBytes        []byte
		requireLiveness   bool
		livenessThreshold float64
		setupMocks        func(*MockFaceRepository, *MockVerificationRepository, *MockFaceProvider)
		wantErr           error
	}{
		{
			name:              "successful registration with liveness",
			tenantID:          uuid.New(),
			externalID:        "user_001",
			imageBytes:        make([]byte, 5000),
			requireLiveness:   true,
			livenessThreshold: 0.90,
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     make([]float64, 512),
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.95,
					FaceCount:     1,
				}, nil)
				fr.On("GetByExternalID", mock.Anything, mock.Anything, "user_001").
					Return(nil, domain.ErrFaceNotFound)
				fr.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:              "liveness check failed - low confidence",
			tenantID:          uuid.New(),
			externalID:        "user_001",
			imageBytes:        make([]byte, 5000),
			requireLiveness:   true,
			livenessThreshold: 0.90,
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     make([]float64, 512),
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.70,
					FaceCount:     1,
				}, nil)
			},
			wantErr: domain.ErrLivenessFailed,
		},
		{
			name:              "liveness not required",
			tenantID:          uuid.New(),
			externalID:        "user_001",
			imageBytes:        make([]byte, 5000),
			requireLiveness:   false,
			livenessThreshold: 0.90,
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     make([]float64, 512),
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.50,
					FaceCount:     1,
				}, nil)
				fr.On("GetByExternalID", mock.Anything, mock.Anything, "user_001").
					Return(nil, domain.ErrFaceNotFound)
				fr.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			faceRepo := &MockFaceRepository{}
			verificationRepo := &MockVerificationRepository{}
			searchAuditRepo := &MockSearchAuditRepository{}
			faceProvider := &MockFaceProvider{}

			tt.setupMocks(faceRepo, verificationRepo, faceProvider)

			svc := &FaceService{
				faceRepo:         faceRepo,
				verificationRepo: verificationRepo,
				searchAuditRepo:  searchAuditRepo,
				provider:         faceProvider,
				threshold:        0.8,
			}

			face, err := svc.Register(context.Background(), tt.tenantID, tt.externalID, tt.imageBytes, tt.requireLiveness, tt.livenessThreshold)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, face)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, face)
				assert.Equal(t, tt.externalID, face.ExternalID)
				assert.Equal(t, tt.tenantID, face.TenantID)
			}

			faceRepo.AssertExpectations(t)
			faceProvider.AssertExpectations(t)
		})
	}
}

func TestFaceService_Verify(t *testing.T) {
	storedFaceID := uuid.New()
	storedEmbedding := make([]float64, 512)

	tests := []struct {
		name       string
		tenantID   uuid.UUID
		externalID string
		imageBytes []byte
		setupMocks func(*MockFaceRepository, *MockVerificationRepository, *MockFaceProvider)
		wantMatch  bool
		wantErr    error
	}{
		{
			name:       "successful verification - match",
			tenantID:   uuid.New(),
			externalID: "user_001",
			imageBytes: make([]byte, 5000),
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fr.On("GetByExternalID", mock.Anything, mock.Anything, "user_001").Return(&domain.Face{
					ID:        storedFaceID,
					Embedding: storedEmbedding,
				}, nil)
				fp.On("DetectFaces", mock.Anything, mock.Anything).Return([]provider.DetectedFace{
					{Confidence: 0.99},
				}, nil)
				fp.On("IndexFace", mock.Anything, mock.Anything).Return("face-id", storedEmbedding, nil)
				fp.On("CompareFaces", mock.Anything, mock.Anything, mock.Anything).Return(0.92, nil)
				vr.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
			wantMatch: true,
			wantErr:   nil,
		},
		{
			name:       "successful verification - no match",
			tenantID:   uuid.New(),
			externalID: "user_001",
			imageBytes: make([]byte, 5000),
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fr.On("GetByExternalID", mock.Anything, mock.Anything, "user_001").Return(&domain.Face{
					ID:        storedFaceID,
					Embedding: storedEmbedding,
				}, nil)
				fp.On("DetectFaces", mock.Anything, mock.Anything).Return([]provider.DetectedFace{
					{Confidence: 0.99},
				}, nil)
				fp.On("IndexFace", mock.Anything, mock.Anything).Return("face-id", make([]float64, 512), nil)
				fp.On("CompareFaces", mock.Anything, mock.Anything, mock.Anything).Return(0.45, nil)
				vr.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
			wantMatch: false,
			wantErr:   nil,
		},
		{
			name:       "face not found",
			tenantID:   uuid.New(),
			externalID: "user_999",
			imageBytes: make([]byte, 5000),
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fr.On("GetByExternalID", mock.Anything, mock.Anything, "user_999").Return(nil, domain.ErrFaceNotFound)
			},
			wantErr: domain.ErrFaceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			faceRepo := &MockFaceRepository{}
			verificationRepo := &MockVerificationRepository{}
			searchAuditRepo := &MockSearchAuditRepository{}
			faceProvider := &MockFaceProvider{}

			tt.setupMocks(faceRepo, verificationRepo, faceProvider)

			svc := &FaceService{
				faceRepo:         faceRepo,
				verificationRepo: verificationRepo,
				searchAuditRepo:  searchAuditRepo,
				provider:         faceProvider,
				threshold:        0.8,
			}

			verification, err := svc.Verify(context.Background(), tt.tenantID, tt.externalID, tt.imageBytes)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, verification)
				assert.Equal(t, tt.wantMatch, verification.Verified)
			}

			faceRepo.AssertExpectations(t)
			verificationRepo.AssertExpectations(t)
			faceProvider.AssertExpectations(t)
		})
	}
}

func TestFaceService_Delete(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   uuid.UUID
		externalID string
		setupMocks func(*MockFaceRepository, *MockVerificationRepository, *MockFaceProvider)
		wantErr    error
	}{
		{
			name:       "successful deletion",
			tenantID:   uuid.New(),
			externalID: "user_001",
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fr.On("GetByExternalID", mock.Anything, mock.Anything, "user_001").Return(&domain.Face{
					ID: uuid.New(),
				}, nil)
				fr.On("Delete", mock.Anything, mock.Anything, "user_001").Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "face not found",
			tenantID:   uuid.New(),
			externalID: "user_999",
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fr.On("GetByExternalID", mock.Anything, mock.Anything, "user_999").Return(nil, domain.ErrFaceNotFound)
			},
			wantErr: domain.ErrFaceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			faceRepo := &MockFaceRepository{}
			verificationRepo := &MockVerificationRepository{}
			searchAuditRepo := &MockSearchAuditRepository{}
			faceProvider := &MockFaceProvider{}

			tt.setupMocks(faceRepo, verificationRepo, faceProvider)

			svc := &FaceService{
				faceRepo:         faceRepo,
				verificationRepo: verificationRepo,
				searchAuditRepo:  searchAuditRepo,
				provider:         faceProvider,
				threshold:        0.8,
			}

			err := svc.Delete(context.Background(), tt.tenantID, tt.externalID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			faceRepo.AssertExpectations(t)
		})
	}
}

func TestFaceService_Search(t *testing.T) {
	tests := []struct {
		name           string
		tenant         *domain.Tenant
		imageBytes     []byte
		threshold      float64
		maxResults     int
		clientIP       string
		setupMocks     func(*MockFaceRepository, *MockVerificationRepository, *MockFaceProvider, *MockSearchAuditRepository)
		expectedError  error
		expectedCount  int
		validateResult func(*testing.T, *domain.SearchResult)
	}{
		{
			name: "successful search with matches",
			tenant: &domain.Tenant{
				ID:   uuid.New(),
				Name: "Test Tenant",
				Settings: map[string]interface{}{
					"search_enabled": true,
				},
			},
			imageBytes: []byte("fake-image-data"),
			threshold:  0.85,
			maxResults: 10,
			clientIP:   "192.168.1.1",
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     []float64{0.1, 0.2, 0.3},
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.90,
					FaceCount:     1,
				}, nil)

				faceID1 := uuid.New()
				faceID2 := uuid.New()
				fr.On("SearchByEmbedding", mock.Anything, mock.Anything, mock.Anything, 0.85, 10).Return([]domain.SearchMatch{
					{FaceID: faceID1, ExternalID: "user1", Similarity: 0.95, Metadata: map[string]interface{}{"name": "User One"}},
					{FaceID: faceID2, ExternalID: "user2", Similarity: 0.88, Metadata: map[string]interface{}{"name": "User Two"}},
				}, nil)

				ar.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedCount: 2,
			validateResult: func(t *testing.T, result *domain.SearchResult) {
				assert.Equal(t, 2, len(result.Matches))
				assert.Equal(t, 0, result.TotalFaces) // TotalFaces removed from hot path
				assert.GreaterOrEqual(t, result.LatencyMs, int64(0))
				assert.NotEqual(t, uuid.Nil, result.SearchID)
				assert.Equal(t, "user1", result.Matches[0].ExternalID)
				assert.Equal(t, 0.95, result.Matches[0].Similarity)
			},
		},
		{
			name: "successful search with no matches",
			tenant: &domain.Tenant{
				ID: uuid.New(),
				Settings: map[string]interface{}{
					"search_enabled": true,
				},
			},
			imageBytes: []byte("fake-image-data"),
			threshold:  0.95,
			maxResults: 10,
			clientIP:   "192.168.1.2",
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     []float64{0.5, 0.6, 0.7},
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.90,
					FaceCount:     1,
				}, nil)
				fr.On("SearchByEmbedding", mock.Anything, mock.Anything, mock.Anything, 0.95, 10).Return([]domain.SearchMatch{}, nil)
				ar.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedCount: 0,
			validateResult: func(t *testing.T, result *domain.SearchResult) {
				assert.Equal(t, 0, len(result.Matches))
				assert.Equal(t, 0, result.TotalFaces) // Removed from hot path
			},
		},
		{
			name: "search not enabled",
			tenant: &domain.Tenant{
				ID: uuid.New(),
				Settings: map[string]interface{}{
					"search_enabled": false,
				},
			},
			imageBytes: []byte("fake-image-data"),
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
			},
			expectedError: domain.ErrSearchNotEnabled,
		},
		{
			name: "invalid threshold - too high",
			tenant: &domain.Tenant{
				ID: uuid.New(),
				Settings: map[string]interface{}{
					"search_enabled": true,
				},
			},
			imageBytes: []byte("fake-image-data"),
			threshold:  1.5,
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
			},
			expectedError: domain.ErrInvalidThreshold,
		},
		{
			name: "invalid threshold - negative default from tenant",
			tenant: &domain.Tenant{
				ID: uuid.New(),
				Settings: map[string]interface{}{
					"search_enabled":   true,
					"search_threshold": float64(-0.1),
				},
			},
			imageBytes: []byte("fake-image-data"),
			threshold:  0,
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
			},
			expectedError: domain.ErrInvalidThreshold,
		},
		{
			name: "invalid max results - too high",
			tenant: &domain.Tenant{
				ID: uuid.New(),
				Settings: map[string]interface{}{
					"search_enabled": true,
				},
			},
			imageBytes: []byte("fake-image-data"),
			threshold:  0.8,
			maxResults: 100,
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
			},
			expectedError: domain.ErrInvalidMaxResults,
		},
		{
			name: "invalid max results - zero with invalid default",
			tenant: &domain.Tenant{
				ID: uuid.New(),
				Settings: map[string]interface{}{
					"search_enabled":     true,
					"search_max_results": float64(0),
					"search_threshold":   0.8,
				},
			},
			imageBytes: []byte("fake-image-data"),
			threshold:  0.8,
			maxResults: 0,
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
			},
			expectedError: domain.ErrInvalidMaxResults,
		},
		{
			name: "use tenant default settings",
			tenant: &domain.Tenant{
				ID: uuid.New(),
				Settings: map[string]interface{}{
					"search_enabled":     true,
					"search_threshold":   float64(0.9),
					"search_max_results": float64(5),
				},
			},
			imageBytes: []byte("fake-image-data"),
			threshold:  0,
			maxResults: 0,
			clientIP:   "192.168.1.3",
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     []float64{0.2, 0.3, 0.4},
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.90,
					FaceCount:     1,
				}, nil)
				fr.On("SearchByEmbedding", mock.Anything, mock.Anything, mock.Anything, 0.9, 5).Return([]domain.SearchMatch{
					{FaceID: uuid.New(), ExternalID: "user3", Similarity: 0.92},
				}, nil)
				ar.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result *domain.SearchResult) {
				assert.Equal(t, 1, len(result.Matches))
				assert.Equal(t, 0, result.TotalFaces) // Removed from hot path
			},
		},
		{
			name: "liveness check required and passed - SecurityMaximum",
			tenant: &domain.Tenant{
				ID: uuid.New(),
				Settings: map[string]interface{}{
					"search_enabled":     true,
					"security_level":     "maximum",
					"liveness_threshold": float64(0.9),
				},
			},
			imageBytes: []byte("fake-image-data"),
			threshold:  0.8,
			maxResults: 10,
			clientIP:   "192.168.1.4",
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     []float64{0.3, 0.4, 0.5},
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.95,
					FaceCount:     1,
				}, nil)
				fr.On("SearchByEmbedding", mock.Anything, mock.Anything, mock.Anything, 0.8, 10).Return([]domain.SearchMatch{
					{FaceID: uuid.New(), ExternalID: "user4", Similarity: 0.85},
				}, nil)
				ar.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedCount: 1,
		},
		{
			name: "liveness check required and failed - SecurityMaximum",
			tenant: &domain.Tenant{
				ID: uuid.New(),
				Settings: map[string]interface{}{
					"search_enabled":     true,
					"security_level":     "maximum",
					"liveness_threshold": float64(0.9),
				},
			},
			imageBytes: []byte("fake-image-data"),
			threshold:  0.8,
			maxResults: 10,
			clientIP:   "192.168.1.5",
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     []float64{0.3, 0.4, 0.5},
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.4,
					FaceCount:     1,
				}, nil)
			},
			expectedError: domain.ErrLivenessFailed,
		},
		{
			name: "provider fails to analyze face",
			tenant: &domain.Tenant{
				ID: uuid.New(),
				Settings: map[string]interface{}{
					"search_enabled": true,
				},
			},
			imageBytes: []byte("invalid-image"),
			threshold:  0.8,
			maxResults: 10,
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidImage)
			},
			expectedError: domain.ErrInvalidImage,
		},
		{
			name: "repository search fails",
			tenant: &domain.Tenant{
				ID: uuid.New(),
				Settings: map[string]interface{}{
					"search_enabled": true,
				},
			},
			imageBytes: []byte("fake-image-data"),
			threshold:  0.8,
			maxResults: 10,
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider, ar *MockSearchAuditRepository) {
				fp.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
					Embedding:     []float64{0.1, 0.2},
					Confidence:    0.99,
					QualityScore:  0.95,
					LivenessScore: 0.90,
					FaceCount:     1,
				}, nil)
				fr.On("SearchByEmbedding", mock.Anything, mock.Anything, mock.Anything, 0.8, 10).Return(nil, errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			faceRepo := &MockFaceRepository{}
			verificationRepo := &MockVerificationRepository{}
			searchAuditRepo := &MockSearchAuditRepository{}
			faceProvider := &MockFaceProvider{}
			rateLimiter := &MockRateLimiter{}

			tt.setupMocks(faceRepo, verificationRepo, faceProvider, searchAuditRepo)

			// Mock rate limiter to always allow
			rateLimiter.On("CheckSearchLimit", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			svc := &FaceService{
				faceRepo:         faceRepo,
				verificationRepo: verificationRepo,
				searchAuditRepo:  searchAuditRepo,
				provider:         faceProvider,
				rateLimiter:      rateLimiter,
				threshold:        0.8,
			}

			result, err := svc.Search(context.Background(), tt.tenant, tt.imageBytes, tt.threshold, tt.maxResults, tt.clientIP)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedError, domain.ErrSearchNotEnabled) ||
					errors.Is(tt.expectedError, domain.ErrInvalidThreshold) ||
					errors.Is(tt.expectedError, domain.ErrInvalidMaxResults) ||
					errors.Is(tt.expectedError, domain.ErrLivenessFailed) ||
					errors.Is(tt.expectedError, domain.ErrInvalidImage) {
					assert.ErrorIs(t, err, tt.expectedError)
				} else {
					assert.Contains(t, err.Error(), "search faces")
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedCount, len(result.Matches))

				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
			}

			faceRepo.AssertExpectations(t)
			faceProvider.AssertExpectations(t)
		})
	}
}

func TestFaceService_Search_WithCustomThreshold(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		name              string
		requestThreshold  float64
		tenantThreshold   float64
		expectedThreshold float64
	}{
		{
			name:              "use request threshold when provided",
			requestThreshold:  0.9,
			tenantThreshold:   0.8,
			expectedThreshold: 0.9,
		},
		{
			name:              "use tenant default when request is zero",
			requestThreshold:  0,
			tenantThreshold:   0.85,
			expectedThreshold: 0.85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := &domain.Tenant{
				ID: tenantID,
				Settings: map[string]interface{}{
					"search_enabled":    true,
					"search_threshold":  tt.tenantThreshold,
					"search_rate_limit": float64(30),
				},
			}

			faceRepo := &MockFaceRepository{}
			faceProvider := &MockFaceProvider{}
			searchAuditRepo := &MockSearchAuditRepository{}
			rateLimiter := &MockRateLimiter{}

			rateLimiter.On("CheckSearchLimit", mock.Anything, tenantID, 30).Return(nil)
			faceProvider.On("AnalyzeFace", mock.Anything, mock.Anything).Return(&provider.FaceAnalysis{
				Embedding:     []float64{0.1, 0.2},
				Confidence:    0.99,
				QualityScore:  0.95,
				LivenessScore: 0.90,
				FaceCount:     1,
			}, nil)
			faceRepo.On("SearchByEmbedding", mock.Anything, tenantID, mock.Anything, tt.expectedThreshold, mock.Anything).Return([]domain.SearchMatch{}, nil)
			searchAuditRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()

			svc := &FaceService{
				faceRepo:        faceRepo,
				searchAuditRepo: searchAuditRepo,
				provider:        faceProvider,
				rateLimiter:     rateLimiter,
				threshold:       0.8,
			}

			_, err := svc.Search(context.Background(), tenant, []byte("image"), tt.requestThreshold, 10, "127.0.0.1")

			require.NoError(t, err)
			faceRepo.AssertExpectations(t)
			faceProvider.AssertExpectations(t)
		})
	}
}
