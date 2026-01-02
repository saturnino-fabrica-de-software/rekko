package service

import (
	"context"
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

type MockVerificationRepository struct {
	mock.Mock
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
			name:       "successful registration",
			tenantID:   uuid.New(),
			externalID: "user_001",
			imageBytes: make([]byte, 5000),
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fp.On("DetectFaces", mock.Anything, mock.Anything).Return([]provider.DetectedFace{
					{Confidence: 0.99, QualityScore: 0.95},
				}, nil)
				fp.On("IndexFace", mock.Anything, mock.Anything).Return("face-id", make([]float64, 512), nil)
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
				fp.On("DetectFaces", mock.Anything, mock.Anything).Return([]provider.DetectedFace{}, nil)
			},
			wantErr: domain.ErrNoFaceDetected,
		},
		{
			name:       "multiple faces detected",
			tenantID:   uuid.New(),
			externalID: "user_001",
			imageBytes: make([]byte, 5000),
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fp.On("DetectFaces", mock.Anything, mock.Anything).Return([]provider.DetectedFace{
					{Confidence: 0.99}, {Confidence: 0.95},
				}, nil)
			},
			wantErr: domain.ErrMultipleFaces,
		},
		{
			name:       "face already exists",
			tenantID:   uuid.New(),
			externalID: "user_001",
			imageBytes: make([]byte, 5000),
			setupMocks: func(fr *MockFaceRepository, vr *MockVerificationRepository, fp *MockFaceProvider) {
				fp.On("DetectFaces", mock.Anything, mock.Anything).Return([]provider.DetectedFace{
					{Confidence: 0.99, QualityScore: 0.95},
				}, nil)
				fp.On("IndexFace", mock.Anything, mock.Anything).Return("face-id", make([]float64, 512), nil)
				fr.On("Create", mock.Anything, mock.Anything).Return(domain.ErrFaceExists)
			},
			wantErr: domain.ErrFaceExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			faceRepo := &MockFaceRepository{}
			verificationRepo := &MockVerificationRepository{}
			faceProvider := &MockFaceProvider{}

			tt.setupMocks(faceRepo, verificationRepo, faceProvider)

			svc := &FaceService{
				faceRepo:         faceRepo,
				verificationRepo: verificationRepo,
				provider:         faceProvider,
				threshold:        0.8,
			}

			face, err := svc.Register(context.Background(), tt.tenantID, tt.externalID, tt.imageBytes)

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
			faceProvider := &MockFaceProvider{}

			tt.setupMocks(faceRepo, verificationRepo, faceProvider)

			svc := &FaceService{
				faceRepo:         faceRepo,
				verificationRepo: verificationRepo,
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
			faceProvider := &MockFaceProvider{}

			tt.setupMocks(faceRepo, verificationRepo, faceProvider)

			svc := &FaceService{
				faceRepo:         faceRepo,
				verificationRepo: verificationRepo,
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
