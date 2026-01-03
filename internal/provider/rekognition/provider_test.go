package rekognition

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

// TestProviderImplementsInterface verifies that Provider implements FaceProvider
func TestProviderImplementsInterface(t *testing.T) {
	var _ provider.FaceProvider = (*Provider)(nil)
}

// TestDefaultConfig verifies default configuration values
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "us-east-1", cfg.Region)
	assert.Equal(t, "rekko-", cfg.CollectionPrefix)
}

// TestCollectionName verifies collection name generation
func TestCollectionName(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		tenantID string
		want     string
	}{
		{
			name:     "default prefix",
			config:   DefaultConfig(),
			tenantID: "550e8400-e29b-41d4-a716-446655440000",
			want:     "rekko-550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "custom prefix",
			config: Config{
				Region:           "us-west-2",
				CollectionPrefix: "custom-",
			},
			tenantID: "tenant-123",
			want:     "custom-tenant-123",
		},
		{
			name: "empty prefix",
			config: Config{
				Region:           "eu-west-1",
				CollectionPrefix: "",
			},
			tenantID: "test-tenant",
			want:     "test-tenant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.CollectionName(tt.tenantID)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestErrors verifies error definitions
func TestErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "CollectionNotFound",
			err:  ErrCollectionNotFound,
			msg:  "collection not found",
		},
		{
			name: "CollectionAlreadyExists",
			err:  ErrCollectionAlreadyExists,
			msg:  "already exists",
		},
		{
			name: "InvalidCredentials",
			err:  ErrInvalidCredentials,
			msg:  "invalid",
		},
		{
			name: "NoFaceDetected",
			err:  ErrNoFaceDetected,
			msg:  "no face detected",
		},
		{
			name: "MultipleFaces",
			err:  ErrMultipleFaces,
			msg:  "multiple faces",
		},
		{
			name: "FaceNotFound",
			err:  ErrFaceNotFound,
			msg:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.err.Error(), tt.msg)
		})
	}
}

// TestParseIndexFacesError verifies error parsing from unindexed faces
func TestParseIndexFacesError(t *testing.T) {
	tests := []struct {
		name            string
		unindexedFaces  []types.UnindexedFace
		wantErr         error
		wantErrContains string
	}{
		{
			name:            "no unindexed faces",
			unindexedFaces:  []types.UnindexedFace{},
			wantErr:         nil,
			wantErrContains: "",
		},
		{
			name: "exceeds max faces",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonExceedsMaxFaces},
				},
			},
			wantErr:         ErrMultipleFaces,
			wantErrContains: "",
		},
		{
			name: "extreme pose",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonExtremePose},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "EXTREME_POSE",
		},
		{
			name: "low brightness",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonLowBrightness},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "LOW_BRIGHTNESS",
		},
		{
			name: "low sharpness",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonLowSharpness},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "LOW_SHARPNESS",
		},
		{
			name: "low confidence",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonLowConfidence},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "LOW_CONFIDENCE",
		},
		{
			name: "small bounding box",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonSmallBoundingBox},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "SMALL_BOUNDING_BOX",
		},
		{
			name: "low face quality",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonLowFaceQuality},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "LOW_FACE_QUALITY",
		},
		{
			name: "no reasons",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseIndexFacesError(tt.unindexedFaces)

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)

			if tt.wantErrContains != "" {
				assert.Contains(t, err.Error(), tt.wantErrContains)
			}
		})
	}
}

// TestCalculateQualityScore verifies quality score calculation
func TestCalculateQualityScore(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name       string
		quality    *types.ImageQuality
		want       float64
		wantMin    float64
		wantMax    float64
		exactMatch bool
	}{
		{
			name:       "nil quality",
			quality:    nil,
			want:       0.0,
			exactMatch: true,
		},
		{
			name: "perfect quality",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(100.0)),
				Sharpness:  ptr(float32(100.0)),
			},
			want:       1.0,
			exactMatch: true,
		},
		{
			name: "zero quality",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(0.0)),
				Sharpness:  ptr(float32(0.0)),
			},
			want:       0.0,
			exactMatch: true,
		},
		{
			name: "medium quality",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(50.0)),
				Sharpness:  ptr(float32(50.0)),
			},
			want:       0.5,
			exactMatch: true,
		},
		{
			name: "sharpness weighted more heavily",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(100.0)), // 30% weight
				Sharpness:  ptr(float32(0.0)),   // 70% weight
			},
			want:       0.3,
			exactMatch: true,
		},
		{
			name: "brightness low, sharpness high",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(0.0)),
				Sharpness:  ptr(float32(100.0)),
			},
			want:       0.7,
			exactMatch: true,
		},
		{
			name: "only brightness set",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(80.0)),
			},
			wantMin: 0.23,
			wantMax: 0.25,
		},
		{
			name: "only sharpness set",
			quality: &types.ImageQuality{
				Sharpness: ptr(float32(80.0)),
			},
			wantMin: 0.55,
			wantMax: 0.57,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.calculateQualityScore(tt.quality)

			if tt.exactMatch {
				assert.InDelta(t, tt.want, got, 0.0001)
			} else {
				assert.GreaterOrEqual(t, got, tt.wantMin)
				assert.LessOrEqual(t, got, tt.wantMax)
			}
		})
	}
}

// TestCompareFaces_NotSupported verifies that CompareFaces with embeddings returns error
func TestCompareFaces_NotSupported(t *testing.T) {
	tenantID := uuid.New()
	p := &Provider{
		tenantID: tenantID,
	}

	similarity, err := p.CompareFaces(context.Background(), []float64{1.0}, []float64{1.0})

	assert.Error(t, err)
	assert.Equal(t, 0.0, similarity)
	assert.Contains(t, err.Error(), "not supported")
	assert.Contains(t, err.Error(), tenantID.String())
}

// TestParseNoFaceError verifies error parsing for no face detected scenarios
func TestParseNoFaceError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantErr        error
		wantErrMessage string
	}{
		{
			name:    "nil error",
			err:     nil,
			wantErr: nil,
		},
		{
			name:    "non-AWS error",
			err:     assert.AnError,
			wantErr: assert.AnError,
		},
		// Note: Testing with real AWS errors would require creating smithy.APIError instances
		// which is complex. The actual behavior is tested in integration tests.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseNoFaceError(tt.err)

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// skipIfNoAWSCredentials skips the test if AWS credentials are not configured
func skipIfNoAWSCredentials(t *testing.T) {
	t.Helper()

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip("Skipping integration test: AWS_ACCESS_KEY_ID not set")
	}
}

// TestIntegration_CreateDeleteCollection tests collection lifecycle
func TestIntegration_CreateDeleteCollection(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	tenantID := uuid.New().String() // Unique tenant for this test

	cfg := DefaultConfig()
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)

	// Ensure clean state
	_ = client.DeleteCollection(ctx, tenantID)

	// Create collection
	err = client.CreateCollection(ctx, tenantID)
	require.NoError(t, err)

	// Verify collection exists
	exists, err := client.CollectionExists(ctx, tenantID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify face count is zero
	count, err := client.GetCollectionFaceCount(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Delete collection
	err = client.DeleteCollection(ctx, tenantID)
	require.NoError(t, err)

	// Verify collection is deleted
	exists, err = client.CollectionExists(ctx, tenantID)
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestIntegration_EnsureCollection tests idempotent collection creation
func TestIntegration_EnsureCollection(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	tenantID := uuid.New().String()

	cfg := DefaultConfig()
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)

	// Clean up any existing collection
	_ = client.DeleteCollection(ctx, tenantID)

	// First ensure should create
	err = client.EnsureCollection(ctx, tenantID)
	require.NoError(t, err)

	exists, err := client.CollectionExists(ctx, tenantID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Second ensure should be no-op
	err = client.EnsureCollection(ctx, tenantID)
	require.NoError(t, err)

	// Clean up
	_ = client.DeleteCollection(ctx, tenantID)
}

// TestIntegration_ListCollections tests listing collections
func TestIntegration_ListCollections(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	tenantID := uuid.New().String()

	cfg := DefaultConfig()
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)

	// Create a test collection
	err = client.CreateCollection(ctx, tenantID)
	require.NoError(t, err)
	defer func() { _ = client.DeleteCollection(ctx, tenantID) }() // Clean up

	// List collections
	collections, err := client.ListCollections(ctx)
	require.NoError(t, err)

	// Should include our test collection
	expectedName := cfg.CollectionName(tenantID)
	found := false
	for _, name := range collections {
		if name == expectedName {
			found = true
			break
		}
	}
	assert.True(t, found, "expected to find collection %s in list", expectedName)
}

// TestIntegration_CollectionNotFound tests operations on non-existent collection
func TestIntegration_CollectionNotFound(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	nonExistentTenant := uuid.New().String()

	cfg := DefaultConfig()
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)

	// Ensure collection doesn't exist
	_ = client.DeleteCollection(ctx, nonExistentTenant)

	// Delete non-existent collection
	err = client.DeleteCollection(ctx, nonExistentTenant)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCollectionNotFound)

	// Get face count from non-existent collection
	_, err = client.GetCollectionFaceCount(ctx, nonExistentTenant)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCollectionNotFound)
}

// TestIntegration_CollectionAlreadyExists tests duplicate collection creation
func TestIntegration_CollectionAlreadyExists(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	tenantID := uuid.New().String()

	cfg := DefaultConfig()
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)

	// Clean up
	_ = client.DeleteCollection(ctx, tenantID)

	// Create collection
	err = client.CreateCollection(ctx, tenantID)
	require.NoError(t, err)
	defer func() { _ = client.DeleteCollection(ctx, tenantID) }()

	// Try to create again - should fail
	err = client.CreateCollection(ctx, tenantID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCollectionAlreadyExists)
}

// TestIntegration_NewProvider tests provider initialization
func TestIntegration_NewProvider(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	tenantID := uuid.New()

	cfg := DefaultConfig()
	p, err := NewProvider(ctx, cfg, tenantID)
	require.NoError(t, err)
	require.NotNil(t, p)

	// Verify provider fields
	assert.Equal(t, tenantID, p.tenantID)
	assert.NotNil(t, p.client)

	// Verify collection was created
	exists, err := p.client.CollectionExists(ctx, tenantID.String())
	require.NoError(t, err)
	assert.True(t, exists)

	// Get face count
	count, err := p.GetFaceCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Clean up
	_ = p.DeleteCollection(ctx)
}

// ptr is a helper function to get pointer to a value
func ptr[T any](v T) *T {
	return &v
}

// fakeImageData returns fake image data with minimum valid size
func fakeImageData() []byte {
	// Create 150 bytes of fake image data (above minimum of 100)
	data := make([]byte, 150)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

// TestDetectFaces_Success verifies successful face detection
func TestDetectFaces_Success(t *testing.T) {
	mock := &mockRekognitionAPI{
		detectFacesFunc: func(ctx context.Context, params *rekognition.DetectFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DetectFacesOutput, error) {
			return &rekognition.DetectFacesOutput{
				FaceDetails: []types.FaceDetail{
					{
						BoundingBox: &types.BoundingBox{
							Left:   ptr(float32(0.1)),
							Top:    ptr(float32(0.2)),
							Width:  ptr(float32(0.3)),
							Height: ptr(float32(0.4)),
						},
						Confidence: ptr(float32(99.5)),
						Quality: &types.ImageQuality{
							Brightness: ptr(float32(80.0)),
							Sharpness:  ptr(float32(90.0)),
						},
					},
				},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	faces, err := provider.DetectFaces(context.Background(), fakeImageData())

	require.NoError(t, err)
	require.Len(t, faces, 1)
	assert.InDelta(t, 0.1, faces[0].BoundingBox.X, 0.01)
	assert.InDelta(t, 0.2, faces[0].BoundingBox.Y, 0.01)
	assert.InDelta(t, 0.3, faces[0].BoundingBox.Width, 0.01)
	assert.InDelta(t, 0.4, faces[0].BoundingBox.Height, 0.01)
	assert.InDelta(t, 99.5, faces[0].Confidence, 0.1)
	assert.Greater(t, faces[0].QualityScore, 0.0)
}

// TestDetectFaces_NoFaces verifies handling of images with no faces
func TestDetectFaces_NoFaces(t *testing.T) {
	mock := &mockRekognitionAPI{
		detectFacesFunc: func(ctx context.Context, params *rekognition.DetectFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DetectFacesOutput, error) {
			return &rekognition.DetectFacesOutput{
				FaceDetails: []types.FaceDetail{},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	faces, err := provider.DetectFaces(context.Background(), fakeImageData())

	require.NoError(t, err)
	assert.Empty(t, faces)
}

// TestDetectFaces_Error verifies error handling in face detection
func TestDetectFaces_Error(t *testing.T) {
	expectedErr := assert.AnError
	mock := &mockRekognitionAPI{
		detectFacesFunc: func(ctx context.Context, params *rekognition.DetectFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DetectFacesOutput, error) {
			return nil, expectedErr
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	tenantID := uuid.New()
	provider := &Provider{client: client, tenantID: tenantID}

	faces, err := provider.DetectFaces(context.Background(), []byte("invalid-image"))

	require.Error(t, err)
	assert.Nil(t, faces)
	assert.Contains(t, err.Error(), tenantID.String())
}

// TestDetectFaces_MultipleFaces verifies detection of multiple faces
func TestDetectFaces_MultipleFaces(t *testing.T) {
	mock := &mockRekognitionAPI{
		detectFacesFunc: func(ctx context.Context, params *rekognition.DetectFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DetectFacesOutput, error) {
			return &rekognition.DetectFacesOutput{
				FaceDetails: []types.FaceDetail{
					{
						BoundingBox: &types.BoundingBox{
							Left:   ptr(float32(0.1)),
							Top:    ptr(float32(0.1)),
							Width:  ptr(float32(0.2)),
							Height: ptr(float32(0.2)),
						},
						Confidence: ptr(float32(95.0)),
						Quality: &types.ImageQuality{
							Brightness: ptr(float32(80.0)),
							Sharpness:  ptr(float32(90.0)),
						},
					},
					{
						BoundingBox: &types.BoundingBox{
							Left:   ptr(float32(0.5)),
							Top:    ptr(float32(0.5)),
							Width:  ptr(float32(0.2)),
							Height: ptr(float32(0.2)),
						},
						Confidence: ptr(float32(96.0)),
						Quality: &types.ImageQuality{
							Brightness: ptr(float32(85.0)),
							Sharpness:  ptr(float32(92.0)),
						},
					},
				},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	faces, err := provider.DetectFaces(context.Background(), fakeImageData())

	require.NoError(t, err)
	assert.Len(t, faces, 2)
}

// TestIndexFace_Success verifies successful face indexing
func TestIndexFace_Success(t *testing.T) {
	expectedFaceID := "face-12345678-1234-1234-1234-123456789012"
	mock := &mockRekognitionAPI{
		indexFacesFunc: func(ctx context.Context, params *rekognition.IndexFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error) {
			return &rekognition.IndexFacesOutput{
				FaceRecords: []types.FaceRecord{
					{
						Face: &types.Face{
							FaceId: ptr(expectedFaceID),
						},
					},
				},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	faceID, embedding, err := provider.IndexFace(context.Background(), fakeImageData())

	require.NoError(t, err)
	assert.Equal(t, expectedFaceID, faceID)
	assert.Nil(t, embedding) // Rekognition does not expose embeddings
}

// TestIndexFace_NoFace verifies handling when no face is detected during indexing
func TestIndexFace_NoFace(t *testing.T) {
	mock := &mockRekognitionAPI{
		indexFacesFunc: func(ctx context.Context, params *rekognition.IndexFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error) {
			return &rekognition.IndexFacesOutput{
				FaceRecords:    []types.FaceRecord{},
				UnindexedFaces: []types.UnindexedFace{},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	tenantID := uuid.New()
	provider := &Provider{client: client, tenantID: tenantID}

	faceID, embedding, err := provider.IndexFace(context.Background(), fakeImageData())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoFaceDetected)
	assert.Empty(t, faceID)
	assert.Nil(t, embedding)
	assert.Contains(t, err.Error(), tenantID.String())
}

// TestIndexFace_MultipleFaces verifies error when multiple faces are detected
func TestIndexFace_MultipleFaces(t *testing.T) {
	mock := &mockRekognitionAPI{
		indexFacesFunc: func(ctx context.Context, params *rekognition.IndexFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error) {
			return &rekognition.IndexFacesOutput{
				FaceRecords: []types.FaceRecord{},
				UnindexedFaces: []types.UnindexedFace{
					{
						Reasons: []types.Reason{types.ReasonExceedsMaxFaces},
					},
				},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	faceID, embedding, err := provider.IndexFace(context.Background(), fakeImageData())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMultipleFaces)
	assert.Empty(t, faceID)
	assert.Nil(t, embedding)
}

// TestIndexFace_LowQuality verifies handling of low quality images
func TestIndexFace_LowQuality(t *testing.T) {
	tests := []struct {
		name   string
		reason types.Reason
	}{
		{name: "extreme pose", reason: types.ReasonExtremePose},
		{name: "low brightness", reason: types.ReasonLowBrightness},
		{name: "low sharpness", reason: types.ReasonLowSharpness},
		{name: "low confidence", reason: types.ReasonLowConfidence},
		{name: "small bounding box", reason: types.ReasonSmallBoundingBox},
		{name: "low face quality", reason: types.ReasonLowFaceQuality},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRekognitionAPI{
				indexFacesFunc: func(ctx context.Context, params *rekognition.IndexFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error) {
					return &rekognition.IndexFacesOutput{
						FaceRecords: []types.FaceRecord{},
						UnindexedFaces: []types.UnindexedFace{
							{
								Reasons: []types.Reason{tt.reason},
							},
						},
					}, nil
				},
			}

			client := &Client{rekognition: mock, config: DefaultConfig()}
			provider := &Provider{client: client, tenantID: uuid.New()}

			faceID, embedding, err := provider.IndexFace(context.Background(), fakeImageData())

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrNoFaceDetected)
			assert.Empty(t, faceID)
			assert.Nil(t, embedding)
		})
	}
}

// TestIndexFace_Error verifies error handling during indexing
func TestIndexFace_Error(t *testing.T) {
	expectedErr := assert.AnError
	mock := &mockRekognitionAPI{
		indexFacesFunc: func(ctx context.Context, params *rekognition.IndexFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error) {
			return nil, expectedErr
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	tenantID := uuid.New()
	provider := &Provider{client: client, tenantID: tenantID}

	faceID, embedding, err := provider.IndexFace(context.Background(), []byte("invalid-image"))

	require.Error(t, err)
	assert.Empty(t, faceID)
	assert.Nil(t, embedding)
	assert.Contains(t, err.Error(), tenantID.String())
}

// TestDeleteFace_Success verifies successful face deletion
func TestDeleteFace_Success(t *testing.T) {
	faceIDToDelete := "face-to-delete"
	mock := &mockRekognitionAPI{
		deleteFacesFunc: func(ctx context.Context, params *rekognition.DeleteFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DeleteFacesOutput, error) {
			assert.Contains(t, params.FaceIds, faceIDToDelete)
			return &rekognition.DeleteFacesOutput{
				DeletedFaces: []string{faceIDToDelete},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	err := provider.DeleteFace(context.Background(), faceIDToDelete)

	require.NoError(t, err)
}

// TestDeleteFace_NotFound verifies error when face is not found
func TestDeleteFace_NotFound(t *testing.T) {
	mock := &mockRekognitionAPI{
		deleteFacesFunc: func(ctx context.Context, params *rekognition.DeleteFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DeleteFacesOutput, error) {
			return &rekognition.DeleteFacesOutput{
				DeletedFaces: []string{}, // No faces deleted
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	tenantID := uuid.New()
	provider := &Provider{client: client, tenantID: tenantID}

	err := provider.DeleteFace(context.Background(), "non-existent-face")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFaceNotFound)
	assert.Contains(t, err.Error(), tenantID.String())
}

// TestDeleteFace_Error verifies error handling during deletion
func TestDeleteFace_Error(t *testing.T) {
	expectedErr := assert.AnError
	mock := &mockRekognitionAPI{
		deleteFacesFunc: func(ctx context.Context, params *rekognition.DeleteFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DeleteFacesOutput, error) {
			return nil, expectedErr
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	tenantID := uuid.New()
	provider := &Provider{client: client, tenantID: tenantID}

	err := provider.DeleteFace(context.Background(), "some-face")

	require.Error(t, err)
	assert.Contains(t, err.Error(), tenantID.String())
}

// TestSearchFacesByImage_Success verifies successful face search
func TestSearchFacesByImage_Success(t *testing.T) {
	mock := &mockRekognitionAPI{
		searchFacesByImageFunc: func(ctx context.Context, params *rekognition.SearchFacesByImageInput, optFns ...func(*rekognition.Options)) (*rekognition.SearchFacesByImageOutput, error) {
			return &rekognition.SearchFacesByImageOutput{
				FaceMatches: []types.FaceMatch{
					{
						Face: &types.Face{
							FaceId:          ptr("face-1"),
							ExternalImageId: ptr("external-1"),
						},
						Similarity: ptr(float32(95.5)),
					},
					{
						Face: &types.Face{
							FaceId:          ptr("face-2"),
							ExternalImageId: nil,
						},
						Similarity: ptr(float32(88.2)),
					},
				},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	results, err := provider.SearchFacesByImage(context.Background(), fakeImageData(), 10, 0.8)

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "face-1", results[0].FaceID)
	assert.Equal(t, "external-1", results[0].ExternalImageID)
	assert.InDelta(t, 0.955, results[0].Similarity, 0.001)
	assert.Equal(t, "face-2", results[1].FaceID)
	assert.Empty(t, results[1].ExternalImageID)
	assert.InDelta(t, 0.882, results[1].Similarity, 0.001)
}

// TestSearchFacesByImage_NoMatches verifies handling when no matches are found
func TestSearchFacesByImage_NoMatches(t *testing.T) {
	mock := &mockRekognitionAPI{
		searchFacesByImageFunc: func(ctx context.Context, params *rekognition.SearchFacesByImageInput, optFns ...func(*rekognition.Options)) (*rekognition.SearchFacesByImageOutput, error) {
			return &rekognition.SearchFacesByImageOutput{
				FaceMatches: []types.FaceMatch{},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	results, err := provider.SearchFacesByImage(context.Background(), fakeImageData(), 10, 0.9)

	require.NoError(t, err)
	assert.Empty(t, results)
}

// TestSearchFacesByImage_InvalidMaxFaces verifies validation of maxFaces parameter
func TestSearchFacesByImage_InvalidMaxFaces(t *testing.T) {
	tests := []struct {
		name     string
		maxFaces int
		wantErr  bool
	}{
		{name: "negative maxFaces", maxFaces: -1, wantErr: true},
		{name: "zero maxFaces", maxFaces: 0, wantErr: false},
		{name: "valid maxFaces", maxFaces: 10, wantErr: false},
		{name: "max allowed", maxFaces: 4096, wantErr: false},
		{name: "exceeds max", maxFaces: 4097, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRekognitionAPI{
				searchFacesByImageFunc: func(ctx context.Context, params *rekognition.SearchFacesByImageInput, optFns ...func(*rekognition.Options)) (*rekognition.SearchFacesByImageOutput, error) {
					return &rekognition.SearchFacesByImageOutput{
						FaceMatches: []types.FaceMatch{},
					}, nil
				},
			}

			client := &Client{rekognition: mock, config: DefaultConfig()}
			tenantID := uuid.New()
			provider := &Provider{client: client, tenantID: tenantID}

			results, err := provider.SearchFacesByImage(context.Background(), fakeImageData(), tt.maxFaces, 0.8)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tenantID.String())
			} else {
				require.NoError(t, err)
				assert.NotNil(t, results)
			}
		})
	}
}

// TestSearchFacesByImage_Error verifies error handling during search
func TestSearchFacesByImage_Error(t *testing.T) {
	expectedErr := assert.AnError
	mock := &mockRekognitionAPI{
		searchFacesByImageFunc: func(ctx context.Context, params *rekognition.SearchFacesByImageInput, optFns ...func(*rekognition.Options)) (*rekognition.SearchFacesByImageOutput, error) {
			return nil, expectedErr
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	tenantID := uuid.New()
	provider := &Provider{client: client, tenantID: tenantID}

	results, err := provider.SearchFacesByImage(context.Background(), fakeImageData(), 10, 0.8)

	require.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), tenantID.String())
}

// TestCompareFaceImages_Success verifies successful face comparison
func TestCompareFaceImages_Success(t *testing.T) {
	mock := &mockRekognitionAPI{
		compareFacesFunc: func(ctx context.Context, params *rekognition.CompareFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.CompareFacesOutput, error) {
			return &rekognition.CompareFacesOutput{
				FaceMatches: []types.CompareFacesMatch{
					{
						Similarity: ptr(float32(92.5)),
					},
				},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	similarity, err := provider.CompareFaceImages(context.Background(), fakeImageData(), fakeImageData(), 0.8)

	require.NoError(t, err)
	assert.InDelta(t, 0.925, similarity, 0.001)
}

// TestCompareFaceImages_NoMatch verifies handling when faces don't match
func TestCompareFaceImages_NoMatch(t *testing.T) {
	mock := &mockRekognitionAPI{
		compareFacesFunc: func(ctx context.Context, params *rekognition.CompareFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.CompareFacesOutput, error) {
			return &rekognition.CompareFacesOutput{
				FaceMatches: []types.CompareFacesMatch{},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	similarity, err := provider.CompareFaceImages(context.Background(), fakeImageData(), fakeImageData(), 0.9)

	require.NoError(t, err)
	assert.Equal(t, 0.0, similarity)
}

// TestCompareFaceImages_Error verifies error handling during comparison
func TestCompareFaceImages_Error(t *testing.T) {
	expectedErr := assert.AnError
	mock := &mockRekognitionAPI{
		compareFacesFunc: func(ctx context.Context, params *rekognition.CompareFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.CompareFacesOutput, error) {
			return nil, expectedErr
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	tenantID := uuid.New()
	provider := &Provider{client: client, tenantID: tenantID}

	similarity, err := provider.CompareFaceImages(context.Background(), fakeImageData(), fakeImageData(), 0.8)

	require.Error(t, err)
	assert.Equal(t, 0.0, similarity)
	assert.Contains(t, err.Error(), tenantID.String())
}

// TestGetFaceCount_Success verifies successful retrieval of face count
func TestGetFaceCount_Success(t *testing.T) {
	expectedCount := int64(42)
	mock := &mockRekognitionAPI{
		describeCollectionFunc: func(ctx context.Context, params *rekognition.DescribeCollectionInput, optFns ...func(*rekognition.Options)) (*rekognition.DescribeCollectionOutput, error) {
			return &rekognition.DescribeCollectionOutput{
				FaceCount: ptr(expectedCount),
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	count, err := provider.GetFaceCount(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expectedCount, count)
}

// TestCreateCollection_Success verifies successful collection creation
func TestCreateCollection_Success(t *testing.T) {
	mock := &mockRekognitionAPI{
		createCollectionFunc: func(ctx context.Context, params *rekognition.CreateCollectionInput, optFns ...func(*rekognition.Options)) (*rekognition.CreateCollectionOutput, error) {
			return &rekognition.CreateCollectionOutput{}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	err := provider.CreateCollection(context.Background())

	require.NoError(t, err)
}

// TestDeleteCollection_Success verifies successful collection deletion
func TestDeleteCollection_Success(t *testing.T) {
	mock := &mockRekognitionAPI{
		deleteCollectionFunc: func(ctx context.Context, params *rekognition.DeleteCollectionInput, optFns ...func(*rekognition.Options)) (*rekognition.DeleteCollectionOutput, error) {
			return &rekognition.DeleteCollectionOutput{}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	err := provider.DeleteCollection(context.Background())

	require.NoError(t, err)
}

// TestIndexFace_Success_FullFlow verifies complete face indexing workflow
func TestIndexFace_Success_FullFlow(t *testing.T) {
	expectedFaceID := "aws-face-id-123"
	indexCalled := false

	mock := &mockRekognitionAPI{
		indexFacesFunc: func(ctx context.Context, params *rekognition.IndexFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error) {
			indexCalled = true
			assert.NotNil(t, params.CollectionId)
			assert.NotNil(t, params.Image)
			assert.Equal(t, int32(1), *params.MaxFaces)
			return &rekognition.IndexFacesOutput{
				FaceRecords: []types.FaceRecord{
					{
						Face: &types.Face{
							FaceId: ptr(expectedFaceID),
						},
					},
				},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	faceID, embedding, err := provider.IndexFace(context.Background(), fakeImageData())

	require.NoError(t, err)
	assert.Equal(t, expectedFaceID, faceID)
	assert.Nil(t, embedding)
	assert.True(t, indexCalled, "IndexFaces should have been called")
}

// TestValidateImage_EmptyImage verifies validation of empty images
func TestValidateImage_EmptyImage(t *testing.T) {
	mock := &mockRekognitionAPI{}
	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	_, err := provider.DetectFaces(context.Background(), []byte{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidImage)
}

// TestValidateImage_TooSmall verifies validation of images that are too small
func TestValidateImage_TooSmall(t *testing.T) {
	mock := &mockRekognitionAPI{}
	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	smallImage := make([]byte, 50) // Less than minImageSize (100)

	_, err := provider.DetectFaces(context.Background(), smallImage)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidImage)
	assert.Contains(t, err.Error(), "too small")
}

// TestValidateImage_TooLarge verifies validation of images that are too large
func TestValidateImage_TooLarge(t *testing.T) {
	mock := &mockRekognitionAPI{}
	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}

	// Create image larger than maxImageSize (5MB = 5 * 1024 * 1024)
	largeImage := make([]byte, 6*1024*1024)

	_, err := provider.DetectFaces(context.Background(), largeImage)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidImage)
	assert.Contains(t, err.Error(), "too large")
}
